#!/usr/bin/env python3
"""GTT - provenance, gaps, sources and working agreements (engine).

One deterministic engine behind ONE gate (gtt-check-provenance.sh) and the
governance view of status/query. It reads artifacts; it owns none of them, and
nothing it prints is authority - the Markdown it points at is.

What it reads (all optional; a project without them is unchanged)
  provenance tags   in governed context and ADRs (code spans and fences are ignored):
                      [FUENTE: ref]      evidence retrieved from a source (id[:loc] or a repo path[:line])
                      [VACÍO: GAP-id]    the authorised sources do not determine it; classified by a gap
                      [CONFLICTO: a vs b]  two or more authorised sources disagree (sources named)
                      [PROPUESTA]        reasoning output - never allowed inside governed context
  gap register      block `gtt-gaps` in gtt-domain/context/stack.md
                      KIND | ID | topic | scope: ... | affects: ...        KIND = OPEN | BLOCKING
                      RESOLVED | ID | topic | was: OPEN | by: ADR-NNN        (append-only trace)
  source manifest   block `gtt-sources` in gtt-domain/context/sources.md
                      policy: provenance=advisory|required
                      id | path | version | authority | precedence | status
  working agreements  block `gtt-preferences` in gtt-domain/working-agreements.md (team, versioned)
                      and .gtt/local/preferences.md (user, local, never committed)
                      id | scope | applies-to | text

Semantics fixed here, enforced by the gate
  BLOCKING  a gap that must be decided before freeze: refuses freeze; in a frozen design it is a violation.
  OPEN      known and undecided, NOT needed for the current design: crosses freeze, stays visible, has an
            explicit scope, and is never an authorisation to change the frozen design.
  Precedence orders sources; it never erases a [CONFLICTO].
  A working preference sits below governed context and can never override it.

Exit 0 = no FAIL (WARN allowed), 1 = at least one FAIL, 2 = cannot run.
"""

import argparse
import glob
import json
import os
import re
import subprocess
import sys

STACK = "gtt-domain/context/stack.md"
SOURCES = "gtt-domain/context/sources.md"
TEAM = "gtt-domain/working-agreements.md"
LOCAL = ".gtt/local/preferences.md"
FROZEN = "gtt-domain/.frozen"
CTX_GLOB = "gtt-domain/context/*.md"
ADR_GLOB = "gtt-domain/adr/ADR-*.md"

AUTHORITIES = {"primary", "secondary", "reference", "evidence"}
SOURCE_STATES = {"active", "superseded", "retired"}
OVERRIDE = re.compile(
    r"(?i)\b(overrid\w*|supersed\w*|ignore[sd]?|bypass\w*|anula\w*|reemplaza\w*|ignora\w*|sobrescrib\w*)\b"
    r".{0,60}\b(governed|context|adr|constraint\w*|decision\w*|gobernad\w*|contexto|restricci\w+|decisi\w+)\b")
AUTHORIZES = re.compile(
    r"(?i)\b(authori[sz]\w*|permit\w*|allowed to|free to|unrestricted|any change|anything|"
    r"autoriza\w*|permite\w*|libre\w*|cualquier cambio)\b")
TAG = {
    "FUENTE": re.compile(r"\[FUENTE(?::([^\]]*))?\]"),
    "VACIO": re.compile(r"\[VAC[ÍI]O(?::([^\]]*))?\]"),
    "CONFLICTO": re.compile(r"\[CONFLICTO(?::([^\]]*))?\]"),
    "PROPUESTA": re.compile(r"\[PROPUESTA(?::[^\]]*)?\]"),
}


# ------------------------------------------------------------------ helpers

def read(path):
    try:
        with open(path, encoding="utf-8") as handle:
            return handle.read()
    except OSError:
        return None


def frozen():
    return os.path.isfile(FROZEN)


DEFAULT_GATES = {"provenance_policy": "advisory", "warnings_block_freeze": False,
                 "sources_manifest_required_for_freeze": False, "open_gap_requires_affects": False}


def profile_gates():
    """(plan id, gates, selected) of the project's Method Plan. Without a selection in .gtt/methodology.json the
    gates are those of `medium` (the behaviour before plans existed) and `selected` is False: a fallback, never a
    choice. What each plan MEANS is defined by the Bootstrap in .gtt/contract/profiles.json; this module only
    applies its machine-enforced gates."""
    profile, selected = "medium", False
    try:
        try:
            with open(".gtt/methodology.json", encoding="utf-8") as handle:
                chosen = json.load(handle).get("profile")
        except OSError:
            chosen = None
        with open(".gtt/contract/profiles.json", encoding="utf-8") as handle:
            spec = json.load(handle)
        profile, selected = chosen or spec.get("default", "medium"), bool(chosen)
        return profile, dict(DEFAULT_GATES, **spec["profiles"][profile]["gates"]), selected
    except (OSError, ValueError, KeyError):
        return profile, dict(DEFAULT_GATES), selected


def mask(text):
    """Blank fenced blocks and inline code, keeping offsets and line numbers."""
    def blank(match):
        return re.sub(r"[^\n]", " ", match.group(0))
    text = re.sub(r"^```.*?^```", blank, text, flags=re.S | re.M)
    return re.sub(r"`[^`\n]*`", blank, text)


def block(text, name):
    """[(line number, raw line)] of the non-blank, non-comment lines of ```gtt-<name>."""
    out, inside = [], False
    for number, line in enumerate(text.split("\n"), 1):
        if not inside:
            inside = re.match(r"^```gtt-%s\s*$" % re.escape(name), line) is not None
        elif line.startswith("```"):
            inside = False
        elif line.strip() and not line.lstrip().startswith("#"):
            out.append((number, line.strip()))
    return out


def has_block(text, name):
    return text is not None and re.search(r"^```gtt-%s\s*$" % re.escape(name), text, re.M) is not None


def attr(value, key):
    match = re.match(r"(?i)^%s\s*:\s*(.*)$" % key, value.strip())
    return match.group(1).strip() if match else None


class Findings:
    def __init__(self):
        self.items = []

    def add(self, level, sub, where, message):
        self.items.append((level, sub, where, message))

    def fails(self):
        return [i for i in self.items if i[0] == "FAIL"]


# --------------------------------------------------------------------- load

def load_gaps(findings):
    text, gaps = read(STACK), []
    if text is None:
        return gaps, False
    seen = set()
    for number, raw in block(text, "gaps"):
        where = f"{STACK}:{number}"
        cells = [c.strip() for c in raw.split("|")]
        if len(cells) != 5:
            findings.add("FAIL", "gaps", where, "expected 5 fields: KIND | ID | topic | attr | attr")
            continue
        kind, gid, topic, first, second = cells
        if kind not in ("OPEN", "BLOCKING", "RESOLVED"):
            findings.add("FAIL", "gaps", where, f"unknown kind `{kind}` (OPEN, BLOCKING, RESOLVED)")
            continue
        if not re.fullmatch(r"GAP-\d{3,}", gid):
            findings.add("FAIL", "gaps", where, f"id `{gid}` must look like GAP-001")
            continue
        if gid in seen and kind != "RESOLVED":
            findings.add("FAIL", "gaps", where, f"duplicate id {gid}")
        seen.add(gid)
        gaps.append({"kind": kind, "id": gid, "topic": topic, "line": number, "where": where,
                     "scope": attr(first, "scope"), "affects": attr(second, "affects"),
                     "was": attr(first, "was"), "by": attr(second, "by")})
    return gaps, has_block(text, "gaps")


def load_sources(findings):
    text = read(SOURCES)
    if text is None:
        return None
    sources, policy, seen_prec = [], "advisory", {}
    for number, raw in block(text, "sources"):
        where = f"{SOURCES}:{number}"
        match = re.match(r"(?i)^policy\s*:\s*provenance\s*=\s*(advisory|required)\s*$", raw)
        if match:
            policy = match.group(1).lower()
            continue
        cells = [c.strip() for c in raw.split("|")]
        if len(cells) != 6:
            findings.add("FAIL", "sources", where, "expected: id | path | version | authority | precedence | status")
            continue
        sid, path, version, authority, precedence, state = cells
        if any(s["id"] == sid for s in sources):
            findings.add("FAIL", "sources", where, f"duplicate source id `{sid}`")
        if authority not in AUTHORITIES:
            findings.add("FAIL", "sources", where, f"authority `{authority}` must be one of {sorted(AUTHORITIES)}")
        if state not in SOURCE_STATES:
            findings.add("FAIL", "sources", where, f"status `{state}` must be one of {sorted(SOURCE_STATES)}")
        prec = int(precedence) if precedence.isdigit() else None
        if prec is None and authority in ("primary", "secondary") and state == "active":
            findings.add("FAIL", "sources", where, f"`{sid}`: an active {authority} source needs an integer precedence")
        if prec is not None:
            if prec in seen_prec:
                findings.add("FAIL", "sources", where,
                             f"precedence {prec} is shared by `{seen_prec[prec]}` and `{sid}` - ambiguous")
            seen_prec[prec] = sid
        if not version:
            findings.add("WARN", "sources", where, f"`{sid}` declares no version")
        if not glob.glob(path):
            findings.add("FAIL", "sources", where, f"`{sid}`: path does not exist: {path}")
        if frozen() and os.path.basename(path).upper().startswith("SOURCE-BRIEF") \
                and authority in ("primary", "secondary") and state == "active":
            findings.add("FAIL", "sources", where,
                         "SOURCE-BRIEF is evidence/history after freeze; declare it authority: evidence")
        sources.append({"id": sid, "path": path, "version": version, "authority": authority,
                        "precedence": prec, "status": state, "where": where})
    return {"policy": policy, "sources": sources}


def governed_files():
    files = sorted(glob.glob(CTX_GLOB))
    files += [f for f in sorted(glob.glob(ADR_GLOB)) if "ADR-TEMPLATE" not in f]
    return [f.replace("\\", "/") for f in files]


def scan_tags():
    """[(file, line, tag, argument)] over governed files, code ignored."""
    found = []
    for path in governed_files():
        text = read(path)
        if text is None:
            continue
        masked = mask(text)
        for name, regex in TAG.items():
            for match in regex.finditer(masked):
                line = masked.count("\n", 0, match.start()) + 1
                arg = match.group(1).strip() if regex.groups and match.group(1) is not None else ""
                found.append((path, line, name, arg))
    return found


def stack_rows():
    """Technology rows of stack.md section 1: [(layer, technology, locked_by, line)]."""
    text = read(STACK)
    if text is None:
        return []
    rows, inside = [], False
    for number, line in enumerate(text.split("\n"), 1):
        if line.startswith("## 1."):
            inside = True
            continue
        if inside and line.startswith("## "):
            break
        if inside and line.startswith("|") and not re.match(r"^\|[\s:|-]+\|?\s*$", line):
            cells = [c.strip() for c in line.strip().strip("|").split("|")]
            if len(cells) >= 4 and cells[0].lower() != "layer":
                rows.append((cells[0], cells[1], cells[3], number))
    return rows


def load_agreements(findings):
    out = []
    for path, want in ((TEAM, "team"), (LOCAL, "user")):
        text = read(path)
        if text is None:
            continue
        seen = set()
        for number, raw in block(text, "preferences"):
            where = f"{path}:{number}"
            cells = [c.strip() for c in raw.split("|")]
            if len(cells) != 4:
                findings.add("FAIL", "preferences", where, "expected: id | scope | applies-to | text")
                continue
            pid, scope, applies, body = cells
            if scope != want:
                findings.add("FAIL", "preferences", where, f"scope `{scope}` must be `{want}` in this file")
            if pid in seen:
                findings.add("FAIL", "preferences", where, f"duplicate id {pid}")
            seen.add(pid)
            if OVERRIDE.search(body) or OVERRIDE.search(applies):
                findings.add("FAIL", "preferences", where,
                             f"{pid}: a working preference cannot override governed context or decisions")
            out.append({"id": pid, "scope": scope, "applies": applies, "text": body, "file": path})
    if os.path.isfile(LOCAL):
        tracked = subprocess.run(["git", "ls-files", "--error-unmatch", LOCAL], capture_output=True)
        if tracked.returncode == 0:
            findings.add("WARN", "preferences", LOCAL, "user-level preferences are tracked by git; keep them local")
    return out


# -------------------------------------------------------------------- checks

def run_checks(only=None, prefreeze=False):
    findings = Findings()
    gaps, gaps_declared = load_gaps(findings)
    manifest = load_sources(findings)
    tags = scan_tags()
    agreements = load_agreements(findings)
    is_frozen = frozen()
    policy = manifest["policy"] if manifest else "none"
    profile, gates, plan_selected = profile_gates()
    strict = policy == "required" or gates["provenance_policy"] == "required"
    known_ids = {s["id"] for s in manifest["sources"]} if manifest else set()
    gap_ids = {g["id"] for g in gaps if g["kind"] != "RESOLVED"}

    # ---- gaps
    for gap in gaps:
        if gap["kind"] == "OPEN":
            if not gap["scope"]:
                findings.add("FAIL", "gaps", gap["where"], f"{gap['id']}: an OPEN needs an explicit `scope:`")
            elif AUTHORIZES.search(gap["scope"]):
                findings.add("FAIL", "gaps", gap["where"],
                             f"{gap['id']}: OPEN is 'not decided yet', never an authorisation - narrow the scope")
            if not gap["affects"]:
                findings.add("FAIL" if gates["open_gap_requires_affects"] else "WARN", "gaps", gap["where"],
                             f"{gap['id']}: no `affects:` (context it touches)"
                             + (f" - required by the {profile} profile" if gates["open_gap_requires_affects"] else ""))
        elif gap["kind"] == "BLOCKING":
            if not gap["scope"]:
                findings.add("FAIL", "gaps", gap["where"], f"{gap['id']}: a BLOCKING needs an explicit `scope:`")
            level = "FAIL" if (is_frozen or prefreeze) else "WARN"
            why = "a frozen design cannot carry an unresolved BLOCKING" if is_frozen \
                else "freeze is refused while it is unresolved"
            findings.add(level, "gaps", gap["where"], f"{gap['id']}: BLOCKING pending - {why}")
        else:
            if gap["was"] not in ("OPEN", "BLOCKING"):
                findings.add("FAIL", "gaps", gap["where"], f"{gap['id']}: RESOLVED must state `was: OPEN|BLOCKING`")
            adr = gap["by"] or ""
            if not re.fullmatch(r"ADR-\d{3,}", adr) or not glob.glob(f"gtt-domain/adr/{adr}*.md"):
                findings.add("FAIL", "gaps", gap["where"],
                             f"{gap['id']}: RESOLVED must cite an existing ADR (`by: ADR-NNN`), got `{adr}`")
    resolved = {g["id"] for g in gaps if g["kind"] == "RESOLVED"}
    live = {g["id"] for g in gaps if g["kind"] in ("OPEN", "BLOCKING")}
    for gid in sorted(resolved & live):
        findings.add("FAIL", "gaps", STACK, f"{gid} is both RESOLVED and still OPEN/BLOCKING")
    classified = {g["topic"].lower() for g in gaps if g["kind"] in ("OPEN", "BLOCKING")}
    for layer, tech, _, number in stack_rows():
        if not tech and layer.lower() not in classified:
            findings.add("WARN", "gaps", f"{STACK}:{number}",
                         f"`{layer}` is empty and not classified as an OPEN or BLOCKING gap")

    # ---- tags
    for path, line, name, arg in tags:
        where = f"{path}:{line}"
        if name == "PROPUESTA":
            findings.add("FAIL", "tags", where, "[PROPUESTA] inside governed context: a proposal is not a decision "
                         "(keep it in gtt-domain/proposals/)")
        elif name == "FUENTE":
            token = arg.split()[0] if arg else ""
            if not token:
                findings.add("FAIL", "tags", where, "[FUENTE] without a reference")
                continue
            head = token.split(":")[0]
            ok = head in known_ids or bool(glob.glob(head))
            if not ok:
                level = "FAIL" if (manifest or strict) else "WARN"
                findings.add(level, "tags", where, f"[FUENTE: {arg}] does not resolve to a declared source or an existing path")
        elif name == "VACIO":
            if not arg:
                level = "FAIL" if strict else "WARN"
                findings.add(level, "tags", where, "[VACÍO] is not classified: cite the gap it belongs to ([VACÍO: GAP-001])")
            elif arg not in gap_ids:
                findings.add("FAIL", "tags", where, f"[VACÍO: {arg}] names a gap that is not OPEN or BLOCKING in the register")
        elif name == "CONFLICTO":
            parts = [p.strip() for p in re.split(r"\s+vs\.?\s+|,|\|", arg) if p.strip()]
            if len(parts) < 2:
                findings.add("FAIL", "tags", where, "[CONFLICTO] must name the participating sources ([CONFLICTO: a vs b])")
            elif manifest:
                for part in parts:
                    if part.split(":")[0] not in known_ids:
                        findings.add("FAIL", "tags", where, f"[CONFLICTO]: `{part}` is not a declared source")
            level = "FAIL" if (is_frozen or prefreeze) else "WARN"
            findings.add(level, "tags", where, "unresolved [CONFLICTO] in governed context: "
                         "resolve it through the governed path; precedence orders sources but never erases a conflict")
    if strict:
        for layer, tech, locked, number in stack_rows():
            has_tag = any(p == STACK and l == number and n == "FUENTE" for p, l, n, _ in tags)
            if tech and not has_tag and not re.search(r"ADR-\d{3,}", locked):
                findings.add("FAIL", "tags", f"{STACK}:{number}",
                             f"provenance required ({'manifest policy' if policy == 'required' else profile + ' profile'}): "
                             f"`{layer}` has neither a [FUENTE] nor a `Locked by` ADR")

    if prefreeze and gates["sources_manifest_required_for_freeze"] and not manifest:
        findings.add("FAIL", "sources", SOURCES, f"the {profile} profile requires a source manifest before freeze")
    if prefreeze and gates["warnings_block_freeze"]:
        findings.items = [("FAIL" if lvl == "WARN" else lvl, sub, where, msg + (" [warning blocks freeze: " + profile + " profile]" if lvl == "WARN" else ""))
                          for lvl, sub, where, msg in findings.items]
    if only:
        findings.items = [i for i in findings.items if i[1] == only]
    return findings, {"gaps": gaps, "gaps_declared": gaps_declared, "manifest": manifest,
                      "tags": tags, "agreements": agreements, "policy": policy, "profile": profile, "plan_selected": plan_selected,
                      "gates": gates}


# ------------------------------------------------------------------- outputs

def summary_lines(state):
    gaps, tags, manifest = state["gaps"], state["tags"], state["manifest"]
    count = lambda kind: sum(1 for g in gaps if g["kind"] == kind)
    n = lambda name: sum(1 for t in tags if t[2] == name)
    unclassified = sum(1 for t in tags if t[2] == "VACIO" and not t[3])
    team = sum(1 for a in state["agreements"] if a["scope"] == "team")
    user = sum(1 for a in state["agreements"] if a["scope"] == "user")
    return [
        (f"method plan: {state['profile']}" if state["plan_selected"]
         else f"method plan: not selected ({state['profile']} gates apply until the human selects one)")
        + f" (provenance gate: {state['gates']['provenance_policy']})",
        f"sources: {len(manifest['sources']) if manifest else 0} declared (provenance policy: {state['policy']})",
        f"gaps: OPEN {count('OPEN')}, BLOCKING {count('BLOCKING')}, RESOLVED {count('RESOLVED')}"
        + ("" if state["gaps_declared"] else " (no gap register in stack.md)"),
        f"unresolved conflicts in governed context: {n('CONFLICTO')}",
        f"provenance tags in governed context: FUENTE {n('FUENTE')}, VACIO {n('VACIO')} "
        f"({unclassified} unclassified), PROPUESTA {n('PROPUESTA')} (must be 0)",
        f"working agreements: team {team}, user {user} (never authority; below governed context)",
    ]


def cmd_check(args):
    findings, state = run_checks(args.only, args.pre_freeze)
    for level, sub, where, message in findings.items:
        print(f"{level:5}  {sub:11} {where}  {message}")
    for line in summary_lines(state):
        print("INFO   " + line)
    bad = findings.fails()
    warn = len(findings.items) - len(bad)
    print(f"SUMMARY  {len(bad)} FAIL, {warn} WARN" + ("  (pre-freeze rules)" if args.pre_freeze else ""))
    if bad:
        print("gtt-check-provenance: FAILED - see the FAIL line(s) above.", file=sys.stderr)
        return 1
    return 0


def cmd_summary(_args):
    _, state = run_checks()
    print("\n".join(summary_lines(state)))
    return 0


def cmd_list(args):
    _, state = run_checks()
    kind = args.kind
    if kind in ("open", "blocking", "resolved"):
        rows = [g for g in state["gaps"] if g["kind"] == kind.upper()]
        if not rows:
            print(f"no {kind.upper()} gaps")
        for g in rows:
            extra = (f"scope: {g['scope']}; affects: {g['affects']}" if g["kind"] != "RESOLVED"
                     else f"was {g['was']}; resolved by {g['by']}")
            print(f"{g['id']}  {g['topic']}  [{extra}]  ({g['where']})")
    elif kind == "conflicts":
        precedence = {s["id"]: s["precedence"] for s in (state["manifest"] or {"sources": []})["sources"]}
        rows = [t for t in state["tags"] if t[2] == "CONFLICTO"]
        if not rows:
            print("no unresolved conflicts in governed context")
        for path, line, _, arg in rows:
            ids = [p.strip().split(":")[0] for p in re.split(r"\s+vs\.?\s+|,|\|", arg) if p.strip()]
            order = " > ".join(f"{i}({precedence[i]})" for i in sorted(ids, key=lambda x: precedence.get(x) or 10**6)
                               if i in precedence and precedence[i] is not None)
            print(f"{path}:{line}  {arg or '(no sources named)'}  "
                  f"[declared precedence: {order or 'none'} - orders the sources, does not resolve the conflict]")
    elif kind == "sources":
        manifest = state["manifest"]
        if not manifest:
            print("no source manifest (gtt-domain/context/sources.md)")
        else:
            print(f"policy: provenance={manifest['policy']}")
            for s in sorted(manifest["sources"], key=lambda x: (x["precedence"] is None, x["precedence"] or 0)):
                print(f"{s['precedence'] if s['precedence'] is not None else '-'}  {s['id']}  {s['path']}  "
                      f"v{s['version'] or '?'}  {s['authority']}  {s['status']}")
    else:
        rows = state["agreements"]
        if not rows:
            print("no working agreements")
        for a in rows:
            print(f"{a['id']}  [{a['scope']}]  applies to: {a['applies']}  - {a['text']}  ({a['file']})")
    return 0


def main(argv):
    if not os.path.isdir(".gtt"):
        print("gtt-provenance: run from the project root (no .gtt/ directory here)", file=sys.stderr)
        return 2
    parser = argparse.ArgumentParser(prog="gtt-provenance", description=__doc__.split("\n")[0])
    sub = parser.add_subparsers(dest="command", required=True)
    p = sub.add_parser("check")
    p.add_argument("--pre-freeze", action="store_true", help="apply the rules that must hold before freeze")
    p.add_argument("--only", choices=["tags", "gaps", "sources", "preferences"])
    p.set_defaults(func=cmd_check)
    sub.add_parser("summary").set_defaults(func=cmd_summary)
    p = sub.add_parser("list")
    p.add_argument("kind", choices=["open", "blocking", "resolved", "conflicts", "sources", "agreements"])
    p.set_defaults(func=cmd_list)
    args = parser.parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
