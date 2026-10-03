#!/usr/bin/env python3
"""GTT - artifact identity, repository integrity, and technical index engine.

GTT v2.1 capability (see GTT-v2.1-Artifact-Identity-Technical-Index.md):

  * IDENTITY   .gtt/index/artifacts.json maps a stable artifact id to its
               current path, historical paths, and aliases. The path is a
               locator; the id is the reference. A move is reconciled, never
               treated as delete + create.
  * INDEX      .gtt/index/technical-index.json is a DERIVED accelerator -
               documents, sections, concepts, relationships, provenance,
               authority, version. Rebuildable from the Markdown alone,
               never a second source of truth (the Markdown is).
  * INTEGRITY  broken links, unresolved [[ID]] references, duplicate logical
               identity, unreconciled moves, and a stale index all fail the
               build - independent of whether `git status` is clean.

Everything is deterministic and stdlib-only (like gtt_guard.py, whose
python-resolution wrapper convention the .sh entry points share). Git is used
only as a hint for rename detection; correctness never depends on it.

Commands:
  index                     register new artifacts, rebuild the technical index
  check                     integrity gate (exit 0 pass, 1 violation)
  reconcile [--apply]       detect moved/renamed artifacts, update identity and
        [--map OLD=NEW]     rewrite references. Dry-run unless --apply.
        [--retire ID]
  query TERM|ID[#anchor]    section-level retrieval from the index
        [--show] [--deep]
  summary                   one-line-per-fact state for gtt-status.sh
"""

import difflib
import hashlib
import json
import os
import posixpath
import re
import subprocess
import sys

MANIFEST = ".gtt/index/artifacts.json"
INDEX = ".gtt/index/technical-index.json"
FROZEN = "gtt-domain/.frozen"
SCHEMA = 1

# A file under these is governed: once frozen this tool will not rewrite it
# (the Human Promotion Boundary). Unresolved references there are reported.
GOVERNED_PREFIXES = ("gtt-domain/context/", "gtt-domain/adr/")

CANONICAL_RE = re.compile(r"GTT-CANONICAL-(v\d+(?:\.\d+)*)\.md")
ADR_ID_RE = re.compile(r"^(ADR-(?:\d+|TEMPLATE))(?=$|[-_.\s])")
ID_REF_RE = re.compile(r"\[\[([A-Za-z0-9][A-Za-z0-9._-]*)\]\]")
LINK_RE = re.compile(r"(?<!\!)\[[^\]\n]*\]\(([^)\s]+)(?:\s+\"[^\"]*\")?\)")
HEADING_RE = re.compile(r"^(#{1,6})[ \t]+(.+?)[ \t]*#*[ \t]*$")

TYPE_AUTHORITY = {
    "governed-context": "L0",
    "architecture-decision": "L1",
    "backlog": "development-line",
    "change-request": "input",
    "proposal": "draft",
    "skill": "instruction",
    "instruction": "instruction",
    "documentation": "human-reference",
    "source-brief": "source",
    "template": "template",
}


# --------------------------------------------------------------- utilities

def die(msg, code=2):
    print(f"gtt-artifacts: {msg}", file=sys.stderr)
    sys.exit(code)


def norm(path):
    return posixpath.normpath(path.replace("\\", "/"))


def read_text(path):
    with open(path, "r", encoding="utf-8", errors="replace", newline="") as fh:
        return fh.read()


def sha(text):
    # CRLF-normalised so the index is identical on every OS / autocrlf setting.
    return hashlib.sha256(text.replace("\r\n", "\n").encode("utf-8")).hexdigest()


def dump_json(obj):
    return json.dumps(obj, indent=2, ensure_ascii=False, sort_keys=False) + "\n"


def write_json(path, obj):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8", newline="\n") as fh:
        fh.write(dump_json(obj))


def slug_id(text):
    return re.sub(r"[^A-Z0-9]+", "-", text.upper()).strip("-")


def git(*args):
    try:
        out = subprocess.run(["git", *args], capture_output=True, text=True,
                             encoding="utf-8", errors="replace")
    except OSError:
        return None
    return out.stdout if out.returncode == 0 else None


# --------------------------------------------------------- repository scope

def tracked_files():
    """Every Markdown file in the GTT kit (same scope as gtt-check-markdown.sh,
    plus SOURCE-BRIEF.md). gtt-domain/session.md is derived, so it is never an artifact."""
    found = []
    for f in ("AGENTS.md", "readme-gtt.md", "readme-gtt.es.md", "SOURCE-BRIEF.md",
              "gtt-domain/backlog.md", "gtt-domain/change-request.md"):
        if os.path.isfile(f):
            found.append(f)
    # Scope, not semantics: these directories hold kit files (instructions,
    # skills, rules) that must be indexed like any other artifact. Nothing below
    # branches on which tool owns a directory.
    for base in (".gtt", "gtt-domain", ".claude", ".kiro", ".copilot"):
        for dirpath, dirnames, filenames in os.walk(base):
            dirnames[:] = sorted(d for d in dirnames if d not in (".git", "__pycache__"))
            for name in sorted(filenames):
                if not name.lower().endswith(".md"):
                    continue
                if name == "session.md" and norm(dirpath) == "gtt-domain":
                    continue                # derived, never an artifact
                found.append(norm(os.path.join(dirpath, name)))
    return sorted(set(found))


def derive(path):
    """(id, type) a new file would be registered as. ADR ids are
    path-independent on purpose: ADR-007 is ADR-007 wherever it lives."""
    stem = posixpath.basename(path)[:-3]
    m = ADR_ID_RE.match(stem)
    if m:
        aid = m.group(1)
        return aid, ("template" if aid == "ADR-TEMPLATE" else "architecture-decision")
    if path == "AGENTS.md":
        return "AGENTS", "instruction"
    if "/" not in path and path.lower().startswith("readme-gtt"):
        return slug_id(stem), "documentation"
    if path.startswith("SOURCE-BRIEF"):
        return "SOURCE-BRIEF", "source-brief"
    if path.startswith("gtt-domain/context/"):
        return "CTX-" + slug_id(stem), "governed-context"
    if path.startswith("gtt-domain/proposals/"):
        return "PROP-" + slug_id(stem), "proposal"
    if path == "gtt-domain/backlog.md":
        return "BACKLOG", "backlog"
    if path == "gtt-domain/change-request.md":
        return "CHANGE-REQUEST", "change-request"
    if path.startswith(".gtt/docs/"):
        return "DOC-" + slug_id(stem), "documentation"
    if path.startswith(".gtt/"):
        return "GTT-" + slug_id(stem), "documentation"
    if "/skills/" in path and posixpath.basename(path) == "SKILL.md":
        return "SKILL-" + slug_id(posixpath.basename(posixpath.dirname(path))), "skill"
    if path.startswith("."):
        # A tool-owned top-level directory: registered like any other file. The
        # engine attaches no meaning to which tool owns it.
        return "ART-" + slug_id(path[:-3]), "instruction"
    return "ART-" + slug_id(path[:-3]), "documentation"


# ---------------------------------------------------------------- manifest

def load_manifest():
    if not os.path.isfile(MANIFEST):
        return {"schema": SCHEMA, "artifacts": []}
    try:
        data = json.loads(read_text(MANIFEST))
    except ValueError as exc:
        die(f"{MANIFEST} is not valid JSON: {exc}")
    if data.get("schema") != SCHEMA or not isinstance(data.get("artifacts"), list):
        die(f"{MANIFEST}: unsupported schema")
    return data


def save_manifest(data):
    data["artifacts"].sort(key=lambda a: a["id"])
    write_json(MANIFEST, data)


def entry_paths(entry):
    return [entry["path"]] + list(entry.get("history", [])) + list(entry.get("aliases", []))


# ------------------------------------------------------ markdown analysis

def mask_code(text):
    """Blank out fenced and inline code (same length, newlines kept) so links,
    headings and [[ID]] examples inside code never count."""
    out, fence = [], None
    for line in text.split("\n"):
        stripped = line.lstrip()
        if fence:
            out.append(" " * len(line))
            if stripped.startswith(fence):
                fence = None
            continue
        if stripped.startswith("```") or stripped.startswith("~~~"):
            fence = stripped[:3]
            out.append(" " * len(line))
            continue
        out.append(re.sub(r"`[^`\n]*`", lambda m: " " * len(m.group(0)), line))
    return "\n".join(out)


def gh_slug(heading):
    s = re.sub(r"[`*_\[\]]", "", heading.strip().lower())
    s = re.sub(r"[^\w\- ]", "", s)
    return s.replace(" ", "-")


def parse_sections(masked):
    lines = masked.split("\n")
    heads, seen = [], {}
    for i, line in enumerate(lines, 1):
        m = HEADING_RE.match(line)
        if not m:
            continue
        level, title = len(m.group(1)), m.group(2).strip()
        base = gh_slug(title)
        n = seen.get(base, 0)
        seen[base] = n + 1
        heads.append({"heading": title, "level": level,
                      "anchor": base if n == 0 else f"{base}-{n}", "line_start": i})
    total = len(lines)
    stack = []
    for idx, h in enumerate(heads):
        end = total
        for nxt in heads[idx + 1:]:
            if nxt["level"] <= h["level"]:
                end = nxt["line_start"] - 1
                break
        h["line_end"] = end
        while stack and stack[-1]["level"] >= h["level"]:
            stack.pop()
        stack.append(h)
        h["path"] = " > ".join(s["heading"] for s in stack)
    return heads


def find_links(masked):
    """[(start, end, raw_target)] for relative-or-absolute markdown links."""
    return [(m.start(1), m.end(1), m.group(1)) for m in LINK_RE.finditer(masked)]


def split_target(raw):
    path, _, frag = raw.partition("#")
    return path, frag


def is_external(raw):
    return bool(re.match(r"^[a-zA-Z][a-zA-Z0-9+.-]*:", raw)) or raw.startswith("#")


def resolve(from_file, raw_path):
    if not raw_path:
        return from_file
    return norm(posixpath.join(posixpath.dirname(from_file), raw_path))


# ------------------------------------------------------------------ state

class State:
    """Everything derived from one read of the repository."""

    def __init__(self):
        self.manifest = load_manifest()
        self.entries = {e["id"]: e for e in self.manifest["artifacts"]}
        self.by_path = {}
        for e in self.manifest["artifacts"]:
            if e.get("status", "active") == "active":
                self.by_path[e["path"]] = e
        self.old_paths = {}
        for e in self.manifest["artifacts"]:
            for p in e.get("history", []):
                self.old_paths.setdefault(p, e["id"])
        self.alias_paths = {p for e in self.manifest["artifacts"] for p in e.get("aliases", [])}
        self.disk = tracked_files()
        self.disk_set = set(self.disk)
        self.missing = sorted(p for p in self.by_path if p not in self.disk_set)
        known = set(self.by_path) | self.alias_paths
        self.unregistered = sorted(p for p in self.disk if p not in known)
        self._text = {}

    def text(self, path):
        if path not in self._text:
            self._text[path] = read_text(path)
        return self._text[path]


# -------------------------------------------------------------- building

def analyse(state, path):
    """Structural facts for one registered document."""
    text = state.text(path)
    masked = mask_code(text)
    sections = parse_sections(masked)
    links, id_refs, broken, stale, bad_anchor = [], [], [], [], []
    for _, _, raw in find_links(masked):
        if is_external(raw):
            if raw.startswith("#") and raw[1:] and raw[1:] not in {s["anchor"] for s in sections}:
                bad_anchor.append(raw)
            continue
        p, frag = split_target(raw)
        target = resolve(path, p)
        if target in state.by_path:
            links.append(state.by_path[target]["id"])
            if frag and target.endswith(".md"):
                tsecs = {s["anchor"] for s in parse_sections(mask_code(state.text(target)))} \
                    if target in state.disk_set else set()
                if frag not in tsecs:
                    bad_anchor.append(raw)
        elif target in state.old_paths:
            stale.append((raw, state.old_paths[target]))
        elif os.path.exists(target.rstrip("/")):
            continue  # non-artifact file or directory that exists
        else:
            broken.append(raw)
    unresolved = []
    for m in ID_REF_RE.finditer(masked):
        ref = m.group(1)
        target = state.entries.get(ref)
        if target is None:
            unresolved.append(ref)
        else:
            id_refs.append(target["id"])
    return {"text": text, "masked": masked, "sections": sections,
            "refs": sorted(set(links) | set(id_refs)),
            "broken": broken, "stale": stale, "unresolved": unresolved,
            "bad_anchor": bad_anchor}


def adr_meta(text):
    meta = {}
    for key, field in (("Status", "status"), ("Date", "date"),
                       ("Approved by", "approved_by"), ("Supersedes", "supersedes")):
        m = re.search(rf"^-\s*{key}:\s*(.+?)\s*$", text, re.M)
        if m:
            meta[field] = m.group(1)
    return meta


def build_index(state):
    docs, facts = {}, {}
    for path, entry in sorted(state.by_path.items(), key=lambda kv: kv[1]["id"]):
        if path not in state.disk_set:
            continue
        facts[entry["id"]] = analyse(state, path)

    referenced_by = {i: [] for i in facts}
    for aid, f in facts.items():
        for ref in f["refs"]:
            if ref in referenced_by and ref != aid:
                referenced_by[ref].append(aid)

    concepts = {}
    for aid, entry in sorted(state.entries.items()):
        if aid not in facts:
            continue
        f = facts[aid]
        rels, prov = [], {"history": list(entry.get("history", []))}
        head = "\n".join(f["text"].split("\n")[:15])
        version = (CANONICAL_RE.search(head) or [None, None])[1]
        if entry["type"] == "architecture-decision":
            meta = adr_meta(f["text"])
            prov["adr"] = meta
            for target in re.findall(r"ADR-\d+", meta.get("supersedes", "")):
                if target in state.entries:
                    rels.append({"rel": "supersedes", "target": target})
            sm = re.search(r"Superseded by\s+(ADR-\d+)", meta.get("status", ""))
            if sm and sm.group(1) in state.entries:
                rels.append({"rel": "superseded-by", "target": sm.group(1)})
        prov["referenced_by_proposals"] = sorted(
            r for r in referenced_by[aid] if state.entries[r]["type"] == "proposal")
        title = next((s["heading"] for s in f["sections"] if s["level"] == 1), None)
        docs[aid] = {
            "id": aid,
            "type": entry["type"],
            "path": entry["path"],
            "title": title,
            "authority": TYPE_AUTHORITY.get(entry["type"], "unclassified"),
            "version": version,
            "sha256": sha(f["text"]),
            "aliases": sorted(entry.get("aliases", [])),
            "sections": [{k: s[k] for k in ("heading", "level", "anchor", "path",
                                            "line_start", "line_end")}
                         for s in f["sections"]],
            "references": f["refs"],
            "referenced_by": sorted(referenced_by[aid]),
            "relationships": sorted(rels, key=lambda r: (r["rel"], r["target"])),
            "provenance": prov,
        }
        for s in f["sections"]:
            term = s["heading"].lower()
            if 2 <= len(term) <= 60:
                concepts.setdefault(term, []).append(f"{aid}#{s['anchor']}")
        if entry["type"] == "governed-context" and entry["path"].endswith("glossary.md"):
            for m in re.finditer(r"^\|\s*([^|\n]+?)\s*\|", f["masked"], re.M):
                t = m.group(1).strip().lower()
                if t and t not in ("term", "---") and not set(t) <= set("-: "):
                    concepts.setdefault(t, []).append(f"{aid}#glossary")
    return {
        "schema": SCHEMA,
        "notice": "DERIVED by .gtt/scripts/gtt-index.sh from the Markdown artifacts. "
                  "Never hand-edit; never authoritative. Delete and rebuild at will.",
        "documents": [docs[k] for k in sorted(docs)],
        "concepts": {k: sorted(set(v)) for k, v in sorted(concepts.items())},
    }, facts


# ----------------------------------------------------------------- index

def register_new(state):
    """Assign identity to unregistered files. Refuses on duplicate identity."""
    added = []
    for path in state.unregistered:
        aid, typ = derive(path)
        if aid in state.entries:
            other = state.entries[aid]
            die(f"duplicate logical identity {aid}: {path} collides with "
                f"{other['path']}. If they are the same artifact, run "
                f"reconcile; if the copy is intentional, list it under "
                f"\"aliases\" in {MANIFEST}.", 1)
        entry = {"id": aid, "type": typ, "path": path, "status": "active",
                 "history": [], "aliases": []}
        state.manifest["artifacts"].append(entry)
        state.entries[aid] = entry
        state.by_path[path] = entry
        added.append(path)
    state.unregistered = []
    return added


def cmd_index(_args):
    state = State()
    if state.missing:
        print("gtt-index: refusing to register new artifacts - manifest paths are "
              "missing on disk (moved, renamed, or deleted):", file=sys.stderr)
        for p in state.missing:
            print(f"  {p}  ({state.by_path[p]['id']})", file=sys.stderr)
        print("Run .gtt/scripts/gtt-reconcile.sh first.", file=sys.stderr)
        return 1
    added = register_new(state)
    if added:
        save_manifest(state.manifest)
        for p in added:
            print(f"registered  {state.by_path[p]['id']}  {p}")
    index, _ = build_index(state)
    write_json(INDEX, index)
    print(f"gtt-index: OK - {len(index['documents'])} artifact(s), "
          f"{sum(len(d['sections']) for d in index['documents'])} section(s) indexed "
          f"-> {INDEX}")
    return 0


# ----------------------------------------------------------------- check

def cmd_check(_args):
    state = State()
    problems, warnings = [], []

    if not os.path.isfile(MANIFEST):
        print(f"gtt-check-integrity: cannot determine - {MANIFEST} missing. "
              "Run .gtt/scripts/gtt-index.sh.", file=sys.stderr)
        return 2

    seen_paths = {}
    for e in state.manifest["artifacts"]:
        for key in ("id", "type", "path"):
            if not e.get(key):
                problems.append(f"manifest entry without '{key}': {e}")
        if e.get("status", "active") not in ("active", "retired"):
            problems.append(f"{e['id']}: invalid status '{e.get('status')}'")
        if e.get("status", "active") == "active":
            if e["path"] in seen_paths:
                problems.append(f"duplicate path {e['path']} registered to "
                                f"{seen_paths[e['path']]} and {e['id']}")
            seen_paths[e["path"]] = e["id"]
    ids = [e["id"] for e in state.manifest["artifacts"]]
    for dup in sorted({i for i in ids if ids.count(i) > 1}):
        problems.append(f"duplicate artifact id in manifest: {dup}")

    for p in state.missing:
        problems.append(f"unreconciled path: {p} ({state.by_path[p]['id']}) no longer "
                        "exists - moved? run .gtt/scripts/gtt-reconcile.sh")
    for p in state.unregistered:
        aid, _ = derive(p)
        if aid in state.entries and state.entries[aid]["path"] in state.disk_set:
            problems.append(f"duplicate logical identity {aid}: {p} and "
                            f"{state.entries[aid]['path']} - same artifact twice? "
                            "(list intentional copies under \"aliases\")")
        else:
            problems.append(f"unregistered artifact: {p} - run .gtt/scripts/gtt-index.sh")

    index, facts = build_index(state)
    for aid, f in sorted(facts.items()):
        path = state.entries[aid]["path"]
        for raw in f["broken"]:
            problems.append(f"{path}: broken link -> {raw}")
        for raw, oid in f["stale"]:
            problems.append(f"{path}: link -> {raw} uses an old path of {oid}; "
                            "run .gtt/scripts/gtt-reconcile.sh --apply")
        for ref in f["unresolved"]:
            problems.append(f"{path}: unresolved artifact reference [[{ref}]]")
        for raw in f["bad_anchor"]:
            warnings.append(f"{path}: anchor not found -> {raw}")
        for ref in f["refs"]:
            if state.entries[ref].get("status") == "retired":
                warnings.append(f"{path}: references retired artifact {ref}")

    if not state.missing and not state.unregistered:
        if not os.path.isfile(INDEX):
            problems.append(f"technical index missing ({INDEX}) - run .gtt/scripts/gtt-index.sh")
        elif read_text(INDEX).replace("\r\n", "\n") != dump_json(index):
            problems.append(f"technical index is stale ({INDEX}) - run .gtt/scripts/gtt-index.sh")

    for w in warnings:
        print(f"WARN  {w}")
    if problems:
        for p in problems:
            print(f"FAIL  {p}")
        print(f"\ngtt-check-integrity: FAILED - {len(problems)} problem(s).", file=sys.stderr)
        return 1
    print(f"gtt-check-integrity: OK - {len(state.by_path)} artifact(s), identity, "
          "references, and technical index consistent.")
    return 0


# ------------------------------------------------------------- reconcile

def git_rename_hints():
    hints = {}
    out = git("diff", "--name-status", "-M", "HEAD")
    for line in (out or "").splitlines():
        parts = line.split("\t")
        if len(parts) == 3 and parts[0].startswith("R"):
            hints[norm(parts[1])] = (norm(parts[2]), int(parts[0][1:] or 100) / 100.0)
    return hints


def old_content(path):
    out = git("show", f"HEAD:{path}")
    return out


def match_moves(state, forced):
    """Returns (pairs {old: (new, how)}, ambiguous {old: [candidates]}, gone [old])."""
    pairs, ambiguous = {}, {}
    candidates = set(state.unregistered)
    hints = git_rename_hints()
    for old in state.missing:
        entry = state.by_path[old]
        if old in forced:
            pairs[old] = (forced[old], "forced --map")
            candidates.discard(forced[old])
            continue
        if old in hints and hints[old][0] in candidates:
            pairs[old] = (hints[old][0], "git rename")
            candidates.discard(hints[old][0])
            continue
        # Same logical id derived from a new path (ADR-007 in a new directory).
        same = [c for c in sorted(candidates) if derive(c)[0] == entry["id"]]
        if len(same) == 1:
            pairs[old] = (same[0], "same identity")
            candidates.discard(same[0])
            continue
        prev = old_content(old)
        if prev is None:
            continue
        scored = []
        for c in sorted(candidates):
            new_text = state.text(c)
            if sha(prev) == sha(new_text):
                scored.append((1.0, c, "identical content"))
            else:
                ratio = difflib.SequenceMatcher(
                    None, prev.replace("\r\n", "\n").splitlines(),
                    new_text.replace("\r\n", "\n").splitlines()).ratio()
                same_name = posixpath.basename(c) == posixpath.basename(old)
                if ratio >= 0.8 or (same_name and ratio >= 0.5):
                    scored.append((ratio, c, f"{int(ratio * 100)}% similar content"))
        scored.sort(reverse=True)
        if scored and (len(scored) == 1 or scored[0][0] > scored[1][0]) and scored[0][0] >= 0.8:
            pairs[old] = (scored[0][1], scored[0][2])
            candidates.discard(scored[0][1])
        elif scored:
            ambiguous[old] = [c for _, c, _ in scored]
    gone = [o for o in state.missing if o not in pairs and o not in ambiguous]
    return pairs, ambiguous, gone


def rewrite_links(state, moves):
    """Rewrite relative links so they keep resolving to the same artifact,
    accounting for both moved targets and moved source files."""
    frozen = os.path.isfile(FROZEN)
    changed, skipped = [], []
    new_of = {o: n for o, n in moves.items()}
    old_of = {n: o for o, n in moves.items()}
    for new_path in state.disk:
        old_path = old_of.get(new_path, new_path)
        text = state.text(new_path)
        masked = mask_code(text)
        edits = []
        for start, end, raw in find_links(masked):
            if is_external(raw):
                continue
            p, frag = split_target(raw)
            if not p:
                continue
            target_old = resolve(old_path, p)
            target_new = new_of.get(target_old, target_old)
            if resolve(new_path, p) == target_new:
                continue
            if not (target_old in new_of or os.path.exists(target_new.rstrip("/"))):
                continue  # was already broken before; not ours to guess
            rel = posixpath.relpath(target_new, posixpath.dirname(new_path) or ".")
            if p.endswith("/"):
                rel += "/"
            edits.append((start, end, rel + (("#" + frag) if frag else "")))
        if not edits:
            continue
        if frozen and new_path.startswith(GOVERNED_PREFIXES):
            skipped.append((new_path, len(edits)))
            continue
        for start, end, repl in sorted(edits, reverse=True):
            text = text[:start] + repl + text[end:]
        with open(new_path, "w", encoding="utf-8", newline="") as fh:
            fh.write(text)
        changed.append((new_path, len(edits)))
    return changed, skipped


def cmd_reconcile(args):
    apply = "--apply" in args
    retire_all = "--retire-missing" in args      # explicit: every registered path that is gone (fresh host project)
    forced, retire = {}, set()
    it = iter(args)
    for a in it:
        if a == "--map":
            spec = next(it, "")
            if "=" not in spec:
                die("--map expects OLD=NEW")
            o, n = spec.split("=", 1)
            forced[norm(o)] = norm(n)
        elif a == "--retire":
            retire.add(next(it, ""))
    state = State()
    if not state.missing:
        print("gtt-reconcile: nothing to reconcile - every registered path exists.")
        return 0
    for o, n in forced.items():
        if o not in state.by_path:
            die(f"--map: {o} is not a registered path")
        if n not in state.disk_set:
            die(f"--map: {n} does not exist (or is outside the GTT kit)")
    pairs, ambiguous, gone = match_moves(state, forced)
    retire_paths = set(gone) if retire_all else {p for p in gone if state.by_path[p]["id"] in retire}

    for old, (new, how) in sorted(pairs.items()):
        print(f"MOVE       {state.by_path[old]['id']}: {old} -> {new}  [{how}]")
    for old, cands in sorted(ambiguous.items()):
        print(f"AMBIGUOUS  {state.by_path[old]['id']}: {old} -> one of {', '.join(cands)}"
              f"  (decide with --map {old}=<path>)")
    for old in gone:
        tag = "RETIRE    " if old in retire_paths else "MISSING   "
        print(f"{tag} {state.by_path[old]['id']}: {old}"
              + ("" if old in retire_paths else
                 f"  (deleted? --retire {state.by_path[old]['id']}; moved? --map {old}=<path>)"))

    if not apply:
        print("\ngtt-reconcile: dry run. Re-run with --apply to update identity and references.")
        return 1 if (ambiguous or (gone and not (retire or retire_all))) else 0

    moves = {o: n for o, (n, _) in pairs.items()}
    for old, (new, _) in pairs.items():
        e = state.by_path[old]
        e["history"] = sorted(set(e.get("history", [])) | {old})
        e["path"] = new
    for old in retire_paths:
        state.by_path[old]["status"] = "retired"
    save_manifest(state.manifest)
    changed, skipped = rewrite_links(state, moves)
    for path, n in changed:
        print(f"rewrote    {n} link(s) in {path}")
    for path, n in skipped:
        print(f"SKIPPED    {n} link(s) in {path} - governed file, frozen: fix through the "
              "change process (prefer [[ID]] references, which survive moves)")
    if ambiguous or (set(gone) - retire_paths):
        print("\ngtt-reconcile: partial - unresolved items remain; index not rebuilt.")
        return 1
    return cmd_index([])


# ------------------------------------------------------------------ query

def load_index():
    if not os.path.isfile(INDEX):
        die(f"{INDEX} missing - run .gtt/scripts/gtt-index.sh", 2)
    return json.loads(read_text(INDEX))


def section_text(doc, sec):
    lines = read_text(doc["path"]).split("\n")
    return "\n".join(lines[sec["line_start"] - 1: sec["line_end"]]).rstrip()


def cmd_query(args):
    show = "--show" in args
    deep = "--deep" in args
    words = [a for a in args if not a.startswith("--")]
    if not words:
        die("usage: query TERM|ID[#anchor] [--show] [--deep]")
    term = " ".join(words)
    index = load_index()
    docs = {d["id"]: d for d in index["documents"]}
    stale = []

    def fresh(doc):
        ok = os.path.isfile(doc["path"]) and sha(read_text(doc["path"])) == doc["sha256"]
        if not ok and doc["id"] not in stale:
            stale.append(doc["id"])
        return ok

    ref, _, anchor = term.partition("#")
    if ref.upper() in docs or ref in docs:
        doc = docs.get(ref) or docs[ref.upper()]
        if not fresh(doc):
            print(f"WARN  {doc['id']} changed since indexing - run .gtt/scripts/gtt-index.sh",
                  file=sys.stderr)
            return 1
        print(f"{doc['id']}  [{doc['type']} / {doc['authority']}]  {doc['path']}")
        for s in doc["sections"]:
            if anchor and s["anchor"] != anchor:
                continue
            print(f"{'  ' * (s['level'] - 1)}#{s['anchor']}  {s['heading']}  "
                  f"(lines {s['line_start']}-{s['line_end']})")
            if show and anchor:
                print("\n" + section_text(doc, s))
        if not anchor and doc["referenced_by"]:
            print(f"referenced-by: {', '.join(doc['referenced_by'])}")
        return 0

    needle = term.lower()
    hits = []  # (rank, doc, section)
    for d in index["documents"]:
        for s in d["sections"]:
            h = s["heading"].lower()
            if h == needle:
                hits.append((0, d, s))
            elif needle in h:
                hits.append((1, d, s))
    for concept, locs in index["concepts"].items():
        if needle == concept or needle in concept:
            for loc in locs:
                did, _, anc = loc.partition("#")
                d = docs[did]
                s = next((x for x in d["sections"] if x["anchor"] == anc), None)
                if s and not any(h[1]["id"] == did and h[2] is s for h in hits):
                    hits.append((2, d, s))
    if deep or not hits:
        for d in index["documents"]:
            if not fresh(d):
                continue
            lines = read_text(d["path"]).split("\n")
            for s in d["sections"]:
                body = "\n".join(lines[s["line_start"]: s["line_end"]]).lower()
                if needle in body and not any(h[1]["id"] == d["id"] and h[2] is s for h in hits):
                    hits.append((3, d, s))
    hits.sort(key=lambda h: (h[0], h[1]["id"], h[2]["line_start"]))
    if not hits:
        print(f"no match for '{term}'")
        return 1
    for rank, d, s in hits[:15]:
        print(f"{d['id']}#{s['anchor']}  {d['path']}:{s['line_start']}-{s['line_end']}  "
              f"[{d['authority']}]  {s['path']}")
    if show:
        _, d, s = hits[0]
        if fresh(d):
            print(f"\n--- {d['id']}#{s['anchor']} ---\n{section_text(d, s)}")
    for did in stale:
        print(f"WARN  {did} changed since indexing - run .gtt/scripts/gtt-index.sh",
              file=sys.stderr)
    return 0


# ---------------------------------------------------------------- summary

def cmd_summary(_args):
    if not os.path.isfile(MANIFEST):
        print("identity: not initialised (run .gtt/scripts/gtt-index.sh)")
        return 0
    state = State()
    active = len(state.by_path)
    retired = sum(1 for e in state.manifest["artifacts"] if e.get("status") == "retired")
    print(f"identity: {active} active artifact(s), {retired} retired")
    if state.missing:
        print(f"unreconciled paths: {len(state.missing)} ({', '.join(state.missing)})")
    if state.unregistered:
        print(f"unregistered artifacts: {len(state.unregistered)} ({', '.join(state.unregistered)})")
    if not state.missing and not state.unregistered:
        index, facts = build_index(state)
        broken = sum(len(f["broken"]) + len(f["stale"]) + len(f["unresolved"])
                     for f in facts.values())
        if not os.path.isfile(INDEX):
            print("technical index: missing")
        elif read_text(INDEX).replace("\r\n", "\n") == dump_json(index):
            print("technical index: fresh")
        else:
            print("technical index: STALE (run .gtt/scripts/gtt-index.sh)")
        print(f"unresolved references: {broken}")
    return 0


COMMANDS = {"index": cmd_index, "check": cmd_check, "reconcile": cmd_reconcile,
            "query": cmd_query, "summary": cmd_summary}


def main(argv):
    if len(argv) < 2 or argv[1] not in COMMANDS:
        print(__doc__)
        return 2
    if not os.path.isdir(".gtt"):
        die("run from the project root (no .gtt/ directory here)")
    return COMMANDS[argv[1]](argv[2:])


if __name__ == "__main__":
    sys.exit(main(sys.argv))
