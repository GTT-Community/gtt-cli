#!/usr/bin/env python3
"""GTT - GTTGuard: shared, ADE-independent engine.

GTTGuard lets a developer mark a file, class, or method with a @GTTGuard
marker so an AI agent may read/analyze/propose changes to it but never
modify it autonomously (GTT Method canonical spec v2.1, sections 13-17
"GTTGuard"). This module is the
one place that logic lives - the sync/check CLI below and the Claude Code
PreToolUse hook (.claude/hooks/protect-guard.py) both import it, so the
detection/resolution rules can never drift between "what gets written to the
registry" and "what actually gets blocked".

Everything here is deterministic, stdlib-only (no PyYAML - the project's
only stated requirement is python3) and intentionally scoped to the language
families the proposal itself illustrates: Java, C#, Python, and
brace/comment-based languages (JS/TS and friends). A marker that does not
precede a recognizable class/method declaration falls back to protecting
the whole file rather than guessing at a narrower scope - ambiguity always
resolves toward MORE protection, never less. A marker that precedes what
looks like a declaration but whose body genuinely can't be resolved (a
truncated file, for example) is reported as an error instead.

.gtt/protection/registry.yaml is a DERIVED artifact, the same way a lockfile
is: `sync` regenerates it in full from source markers every time, and
`check` fails the build if the committed file has drifted from a fresh
regeneration. That is what makes hand-editing the registry self-defeating
without needing a write-block on the file itself.
"""

import glob
import os
import re
import subprocess
import sys

LANG_BY_EXT = {
    ".java": ("java", "brace"),
    ".cs": ("csharp", "brace"),
    ".py": ("python", "indent"),
    ".js": ("javascript", "brace"),
    ".jsx": ("javascript", "brace"),
    ".ts": ("typescript", "brace"),
    ".tsx": ("typescript", "brace"),
}

EXCLUDE_DIRS = {
    ".git", ".claude", ".kiro", ".copilot",
    "node_modules", ".venv", "venv", "__pycache__",
    "dist", "build", "target", ".next", ".pytest_cache",
}

# GTT's own root-level directory (the governed domain; .gtt/ is a dot-directory, skipped anyway). Skipped
# at the project root ONLY: a host project's own src/context/ or src/docs/ holds
# real code whose @GTTGuard markers must stay visible.
ROOT_ONLY_EXCLUDE_DIRS = {"gtt-domain"}

VALID_POLICIES = {"HUMAN_APPROVAL"}

KEYWORDS = {
    "if", "for", "while", "switch", "catch", "return", "new", "throw",
    "synchronized", "using", "foreach", "function", "class", "super",
    "this", "typeof", "instanceof", "yield", "await", "async", "else",
    "do", "try", "finally", "in", "of", "with", "lock", "case", "delete",
}

CLASS_RE = re.compile(r"\bclass\s+([A-Za-z_]\w*)")
DEF_RE = re.compile(r"^\s*(?:async\s+)?def\s+([A-Za-z_]\w*)\s*\(")
FUNCTION_RE = re.compile(r"\bfunction\s+([A-Za-z_]\w*)\s*\(")
ARROW_ASSIGN_RE = re.compile(
    r"\b([A-Za-z_]\w*)\s*=\s*(?:async\s*)?\([^)]*\)\s*=>\s*\{"
)

DECORATOR_RE = re.compile(r"^\s*@GTTGuard\b(?:\((?P<args>.*)\))?\s*$")
ATTRIBUTE_RE = re.compile(r"^\s*\[\s*GTTGuard\b(?:\((?P<args>.*)\))?\s*\]\s*$")
COMMENT_SLASH_RE = re.compile(r"^\s*//\s*@GTTGuard\b\s*(?P<rest>.*)$")
COMMENT_HASH_RE = re.compile(r"^\s*#\s*@GTTGuard\b\s*(?P<rest>.*)$")
OTHER_MARKER_RE = re.compile(r"^\s*(@\w+(\(.*\))?|\[\s*\w+(\(.*\))?\s*\])\s*$")
KV_RE = re.compile(r"(\w+)\s*=\s*\"([^\"]*)\"")


# --- scrubbing: strip string/comment content so brace/keyword matching ----
# only ever sees real code, not text that happens to look like code inside a
# literal or a docstring.

def scrub_brace_line(line, state):
    out = []
    i, n = 0, len(line)
    while i < n:
        c = line[i]
        if state["in_block_comment"]:
            if c == "*" and i + 1 < n and line[i + 1] == "/":
                state["in_block_comment"] = False
                i += 2
                continue
            i += 1
            continue
        if state["in_string"]:
            if c == "\\" and i + 1 < n:
                i += 2
                continue
            if c == state["in_string"]:
                state["in_string"] = None
            i += 1
            continue
        if c == "/" and i + 1 < n and line[i + 1] == "/":
            break
        if c == "/" and i + 1 < n and line[i + 1] == "*":
            state["in_block_comment"] = True
            i += 2
            continue
        if c in ("\"", "'", "`"):
            state["in_string"] = c
            i += 1
            continue
        out.append(c)
        i += 1
    return "".join(out)


def scrub_python_line(line, state):
    out = []
    i, n = 0, len(line)
    while i < n:
        if state["in_triple"]:
            if line[i:i + 3] == state["in_triple"]:
                state["in_triple"] = None
                i += 3
                continue
            i += 1
            continue
        if line[i:i + 3] in ('"""', "'''"):
            state["in_triple"] = line[i:i + 3]
            i += 3
            continue
        if line[i] == "#":
            break
        if line[i] in ("\"", "'"):
            q = line[i]
            i += 1
            while i < n and line[i] != q:
                i += 2 if line[i] == "\\" else 1
            i += 1
            continue
        out.append(line[i])
        i += 1
    return "".join(out)


def scrub_all(lines, mode):
    state = {"in_block_comment": False, "in_string": None, "in_triple": None}
    scrub_line = scrub_python_line if mode == "python" else scrub_brace_line
    return [scrub_line(line, state) for line in lines]


# --- declaration matching --------------------------------------------------

def match_class_decl(scrubbed_line):
    m = CLASS_RE.search(scrubbed_line)
    return m.group(1) if m else None


def _find_matching_paren(line, open_idx):
    depth = 0
    for j in range(open_idx, len(line)):
        if line[j] == "(":
            depth += 1
        elif line[j] == ")":
            depth -= 1
            if depth == 0:
                return j
    return None


def _find_call_like(line):
    """(name, close_paren_index) for top-level identifier(...) not preceded
    by '.' (a method call on an object) and not a control-flow keyword."""
    results = []
    i, n = 0, len(line)
    ident_re = re.compile(r"[A-Za-z_]\w*")
    while i < n:
        m = ident_re.match(line, i)
        if m and m.end() < n and line[m.end()] == "(":
            name = m.group(0)
            preceded_by_dot = i > 0 and line[i - 1] == "."
            close_idx = _find_matching_paren(line, m.end())
            if name not in KEYWORDS and not preceded_by_dot and close_idx is not None:
                results.append((name, close_idx))
            i = m.end() + 1
            continue
        i += 1
    return results


def match_method_decl(scrubbed_line):
    m = DEF_RE.match(scrubbed_line)
    if m:
        return m.group(1)
    m = FUNCTION_RE.search(scrubbed_line)
    if m:
        return m.group(1)
    m = ARROW_ASSIGN_RE.search(scrubbed_line)
    if m:
        return m.group(1)

    candidates = _find_call_like(scrubbed_line)
    if not candidates:
        return None
    name, close_idx = candidates[-1]
    trailing = scrubbed_line[close_idx + 1:].strip()
    if trailing.startswith("{") or trailing == ";":
        return name
    if re.match(r"^(throws\s+[\w,.\s]+)?:?\s*[\w\[\]<>,.\s]*\{", trailing):
        return name
    return None


# --- span resolution ---------------------------------------------------

def resolve_brace_span(scrubbed, decl_idx):
    depth = 0
    started = False
    for i in range(decl_idx, len(scrubbed)):
        for ch in scrubbed[i]:
            if ch == "{":
                depth += 1
                started = True
            elif ch == "}":
                depth -= 1
                if started and depth == 0:
                    return (decl_idx, i)
        if not started and ";" in scrubbed[i]:
            return (decl_idx, i)
    return None


def resolve_indent_span(lines, decl_idx):
    decl_line = lines[decl_idx]
    decl_indent = len(decl_line) - len(decl_line.lstrip(" \t"))
    end = decl_idx
    for i in range(decl_idx + 1, len(lines)):
        stripped = lines[i].strip()
        if stripped == "" or stripped.startswith("#"):
            end = i
            continue
        indent = len(lines[i]) - len(lines[i].lstrip(" \t"))
        if indent > decl_indent:
            end = i
        else:
            break
    return (decl_idx, end)


def resolve_span(lines, scrubbed, decl_idx, mode):
    if mode == "brace":
        return resolve_brace_span(scrubbed, decl_idx)
    return resolve_indent_span(lines, decl_idx)


def find_all_classes(lines, scrubbed, mode):
    results = []
    for i, s in enumerate(scrubbed):
        name = match_class_decl(s)
        if name:
            span = resolve_span(lines, scrubbed, i, mode)
            if span:
                results.append((name, i, span[0], span[1]))
    return results


def enclosing_class(class_decls, decl_idx):
    best = None
    for name, cidx, start, end in class_decls:
        if cidx == decl_idx:
            continue
        if start <= decl_idx <= end and (best is None or start > best[1]):
            best = (name, start)
    return best[0] if best else None


def resolve_symbol_span(root, artifact, symbol):
    """Fresh, live resolution against the file on disk - never cached, so a
    hook checking this at edit-time always sees the current file, not a
    stale span from the last sync."""
    ext = os.path.splitext(artifact)[1]
    lang = LANG_BY_EXT.get(ext)
    full = os.path.join(root, artifact)
    if lang is None or not os.path.isfile(full):
        return None
    _, mode = lang
    with open(full, "r", encoding="utf-8", errors="replace") as f:
        lines = f.read().splitlines()
    if symbol == "":
        return (0, max(len(lines) - 1, 0))

    scrubbed = scrub_all(lines, mode)
    class_decls = find_all_classes(lines, scrubbed, mode)

    if "." not in symbol:
        for name, cidx, start, end in class_decls:
            if name == symbol:
                return (start, end)
        search_range = range(len(scrubbed))
        cls_span = None
    else:
        cls_name, symbol = symbol.split(".", 1)
        cls_span = next(((s, e) for n, _, s, e in class_decls if n == cls_name), None)
        if cls_span is None:
            return None
        search_range = range(cls_span[0], cls_span[1] + 1)

    for i in search_range:
        s = scrubbed[i]
        if match_class_decl(s):
            continue
        if match_method_decl(s) == symbol:
            span = resolve_span(lines, scrubbed, i, mode)
            if span:
                return span
    return None


# --- marker scanning ---------------------------------------------------

def _parse_kv(text):
    kv = dict((k.lower(), v) for k, v in KV_RE.findall(text or ""))
    return kv.get("reason", ""), kv.get("source", "")


def find_markers(lines):
    out = []
    for i, line in enumerate(lines):
        m = DECORATOR_RE.match(line) or ATTRIBUTE_RE.match(line)
        if m:
            reason, source = _parse_kv(m.group("args"))
            out.append((i, reason, source))
            continue
        m = COMMENT_SLASH_RE.match(line) or COMMENT_HASH_RE.match(line)
        if m:
            reason, source = _parse_kv(m.group("rest"))
            out.append((i, reason, source))
    return out


def next_declaration_line(lines, marker_idx, lookahead=6):
    i = marker_idx + 1
    steps = 0
    while i < len(lines) and steps < lookahead:
        line = lines[i]
        if not line.strip() or OTHER_MARKER_RE.match(line):
            i += 1
            steps += 1
            continue
        return i
    return None


def make_entry(artifact, symbol, reason, source):
    return {
        "artifact": artifact,
        "symbol": symbol,
        "protection": "HUMAN_APPROVAL",
        "reason": reason or "",
        "source": source or "",
    }


def scan_file(root, relpath):
    ext = os.path.splitext(relpath)[1]
    _, mode = LANG_BY_EXT[ext]
    full = os.path.join(root, relpath)
    try:
        with open(full, "r", encoding="utf-8", errors="replace") as f:
            lines = f.read().splitlines()
    except OSError:
        return [], []

    scrubbed = scrub_all(lines, mode)
    class_decls = find_all_classes(lines, scrubbed, mode)

    entries, errors = [], []
    for marker_idx, reason, source in find_markers(lines):
        decl_idx = next_declaration_line(lines, marker_idx)
        symbol = None

        if decl_idx is not None:
            cname = match_class_decl(scrubbed[decl_idx])
            if cname:
                span = resolve_span(lines, scrubbed, decl_idx, mode)
                if span is None:
                    errors.append(
                        f"{relpath}:{decl_idx + 1}: could not resolve the "
                        f"body of class '{cname}' - fix the marker's "
                        f"placement or the class body"
                    )
                    continue
                symbol = cname
            else:
                mname = match_method_decl(scrubbed[decl_idx])
                if mname is not None:
                    span = resolve_span(lines, scrubbed, decl_idx, mode)
                    if span is None:
                        errors.append(
                            f"{relpath}:{decl_idx + 1}: could not resolve "
                            f"the body of method '{mname}' - fix the "
                            f"marker's placement or the method body"
                        )
                        continue
                    enclosing = enclosing_class(class_decls, decl_idx)
                    symbol = f"{enclosing}.{mname}" if enclosing else mname

        if symbol is None:
            # No recognizable class/method declaration follows the marker -
            # GTTGuard falls back to protecting the whole file. Ambiguity
            # resolves toward MORE protection, never less, the same
            # fail-safe posture protect-guard.py takes at edit-time.
            entries.append(make_entry(relpath, "", reason, source))
        else:
            entries.append(make_entry(relpath, symbol, reason, source))

    return entries, errors


def iter_source_files(root):
    for dirpath, dirnames, filenames in os.walk(root):
        at_root = os.path.abspath(dirpath) == os.path.abspath(root)
        dirnames[:] = [
            d for d in dirnames
            if d not in EXCLUDE_DIRS and not d.startswith(".")
            and not (at_root and d in ROOT_ONLY_EXCLUDE_DIRS)
        ]
        for name in filenames:
            if os.path.splitext(name)[1] in LANG_BY_EXT:
                rel = os.path.relpath(os.path.join(dirpath, name), root)
                yield rel.replace("\\", "/")


def scan_repository(root):
    entries, errors = [], []
    for relpath in sorted(iter_source_files(root)):
        e, err = scan_file(root, relpath)
        entries.extend(e)
        errors.extend(err)
    entries.sort(key=lambda e: (e["artifact"], e["symbol"]))
    return entries, errors


# --- registry read/write -------------------------------------------------

REGISTRY_HEADER = """\
# GTT - GTTGuard protection registry
#
# Generated by .gtt/scripts/gtt-guard-sync.sh from @GTTGuard markers in
# source. Do not hand-edit: gtt-check-protection.sh fails the build if this
# file drifts from what the markers in source actually declare. Change the
# marker in source instead, then re-run the sync script.
"""


def _yaml_quote(value):
    escaped = value.replace("\\", "\\\\").replace('"', '\\"')
    return f'"{escaped}"'


def format_registry(entries):
    lines = [REGISTRY_HEADER.rstrip("\n"), ""]
    if not entries:
        lines.append("protected: []")
        return "\n".join(lines) + "\n"

    lines.append("protected:")
    for e in entries:
        lines.append(f"  - artifact: {_yaml_quote(e['artifact'])}")
        lines.append(f"    symbol: {_yaml_quote(e['symbol'])}")
        lines.append(f"    protection: {e['protection']}")
        lines.append(f"    reason: {_yaml_quote(e.get('reason', ''))}")
        lines.append(f"    source: {_yaml_quote(e.get('source', ''))}")
    return "\n".join(lines) + "\n"


def _unquote(value):
    value = value.strip()
    if value.startswith('"') and value.endswith('"') and len(value) >= 2:
        value = value[1:-1].replace('\\"', '"').replace("\\\\", "\\")
    return value


def parse_registry(text):
    entries = []
    cur = None
    for raw in text.splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        if line == "protected: []":
            return []
        if line == "protected:":
            continue
        m = re.match(r"^-\s*artifact:\s*(.+)$", line)
        if m:
            if cur:
                entries.append(cur)
            cur = {"artifact": _unquote(m.group(1))}
            continue
        for key in ("symbol", "protection", "reason", "source"):
            m = re.match(rf"^{key}:\s*(.*)$", line)
            if m and cur is not None:
                value = _unquote(m.group(1))
                cur[key] = value if key != "protection" else value.strip()
                break
    if cur:
        entries.append(cur)
    return entries


# --- CI gate ---------------------------------------------------------------

def git_diff_names(root, base_ref):
    try:
        r = subprocess.run(
            ["git", "-C", root, "diff", "--name-only", f"{base_ref}...HEAD"],
            capture_output=True, text=True,
        )
    except OSError:
        return None
    if r.returncode != 0:
        return None
    return {line.strip().replace("\\", "/") for line in r.stdout.splitlines() if line.strip()}


def check_governed_path_routing(root, base_ref, entries):
    changed = git_diff_names(root, base_ref)
    if changed is None:
        return [], True
    if any(p.startswith("gtt-domain/proposals/") or p.startswith("gtt-domain/adr/") for p in changed):
        return [], False
    protected_files = {e["artifact"] for e in entries if e.get("protection") == "HUMAN_APPROVAL"}
    violations = [
        f"{f} changed with no accompanying change under gtt-domain/proposals/ or gtt-domain/adr/"
        for f in sorted(changed & protected_files)
    ]
    return violations, False


def cmd_sync(root, registry_path):
    entries, errors = scan_repository(root)
    if errors:
        for e in errors:
            print(f"gtt-guard: {e}", file=sys.stderr)
        print("gtt-guard: sync FAILED - fix the unresolved markers above.", file=sys.stderr)
        return 1

    os.makedirs(os.path.dirname(registry_path), exist_ok=True)
    with open(registry_path, "w", encoding="utf-8", newline="\n") as f:
        f.write(format_registry(entries))
    rel = os.path.relpath(registry_path, root).replace("\\", "/")
    print(f"gtt-guard: sync OK - {len(entries)} protected artifact(s) recorded in {rel}")
    return 0


def cmd_check(root, registry_path, base_ref):
    fresh_entries, errors = scan_repository(root)
    fail = False
    for e in errors:
        print(f"gtt-check-protection: FAILED - {e}", file=sys.stderr)
        fail = True

    try:
        with open(registry_path, "r", encoding="utf-8") as f:
            committed_text = f.read()
    except OSError:
        print(f"gtt-check-protection: {registry_path} not found.", file=sys.stderr)
        return 2

    if format_registry(fresh_entries).strip() != committed_text.strip():
        print(
            "gtt-check-protection: FAILED - .gtt/protection/registry.yaml is out "
            "of date with the @GTTGuard markers in source. Run "
            ".gtt/scripts/gtt-guard-sync.sh and commit the result.",
            file=sys.stderr,
        )
        fail = True

    committed_entries = parse_registry(committed_text)
    for e in committed_entries:
        artifact = e.get("artifact", "")
        protection = e.get("protection", "")
        if protection not in VALID_POLICIES:
            print(
                f"gtt-check-protection: FAILED - {artifact}: invalid protection "
                f"value '{protection}'", file=sys.stderr,
            )
            fail = True

        if not os.path.isfile(os.path.join(root, artifact)):
            print(
                f"gtt-check-protection: FAILED - {artifact}: artifact does not "
                f"exist", file=sys.stderr,
            )
            fail = True
            continue

        symbol = e.get("symbol", "")
        if symbol and resolve_symbol_span(root, artifact, symbol) is None:
            print(
                f"gtt-check-protection: FAILED - {artifact}: symbol '{symbol}' "
                f"could not be resolved", file=sys.stderr,
            )
            fail = True

        source = e.get("source", "")
        m = re.match(r"^(ADR-\d+)", source)
        if m and not glob.glob(os.path.join(root, "gtt-domain", "adr", m.group(1) + "*.md")):
            print(
                f"gtt-check-protection: FAILED - {artifact}: source '{source}' "
                f"does not exist under gtt-domain/adr/", file=sys.stderr,
            )
            fail = True

    violations, skipped = check_governed_path_routing(root, base_ref, committed_entries)
    if skipped:
        print(
            f"gtt-check-protection: base ref '{base_ref}' unavailable - "
            f"skipping the governed-path-routing check.", file=sys.stderr,
        )
    for v in violations:
        print(
            f"gtt-check-protection: FAILED - {v} - route it through "
            f"gtt-propose-change (Form 5) instead.", file=sys.stderr,
        )
        fail = True

    if fail:
        return 1
    print("gtt-check-protection: OK")
    return 0


def main(argv):
    if len(argv) < 2 or argv[1] not in ("sync", "check"):
        print("usage: gtt_guard.py <sync|check> [base-ref]", file=sys.stderr)
        return 2

    root = os.getcwd()
    registry_path = os.path.join(root, ".gtt", "protection", "registry.yaml")

    if argv[1] == "sync":
        return cmd_sync(root, registry_path)

    base_ref = argv[2] if len(argv) > 2 else "origin/main"
    return cmd_check(root, registry_path, base_ref)


if __name__ == "__main__":
    sys.exit(main(sys.argv))
