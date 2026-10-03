#!/usr/bin/env python3
"""GTT - project-facing Bootstrap contracts (engine).

Structured, deterministic views and the small per-project state the CLI needs, derived from the
actual project (never from an agent's recollection, never authority):

  detect                     project detection: GTT state, layout, freeze, design-source candidates
  status [--with-validation] structured status: bootstrap, ade, methodology, sources, governance, freeze, validation, session
  session                    structured session context (freeze, change request, proposals, git, operational state, artifacts)
  validation                 gtt-validate.sh as one structured result
  interaction                what the selected plan does without asking and what it asks for (derived from its policy)
  next-id                    the next free id of a kind (adr|epic|story); deterministic, never reuses a retired id
  profile get|set            Method Plan (light|medium|hard|team) and language; meaning lives in .gtt/contract/profiles.json
  source select|list         initial sources: a SELECTED source is never a governed authority
  export-policy              clean-export policy resolved against this project (ADE overlays from the ownership ledger)
  clean-plan                 what `clean` would remove; removes nothing
  recovery snapshot|restore  preserve / re-apply GTT configuration (not the governed domain content)

Every mutating command is a dry run unless --apply is given and never overwrites existing state.
Exit 0 ok, 1 violation / refused, 2 cannot run.
"""

import argparse
import datetime
import glob
import json
import os
import re
import subprocess
import sys

import gtt_manifest as gm
import gtt_provenance as gp

CONTRACT = ".gtt/contract"
METHOD = ".gtt/methodology.json"
SELECTED = ".gtt/selected-sources.json"
FROZEN = "gtt-domain/.frozen"
CONTEXT_FILES = ["stack", "architecture", "constraints", "principles", "solution-vision", "glossary"]
PLACEHOLDER = re.compile(r"TODO|PLACEHOLDER|REPLACE ME|<[a-z][^>]*>")
KIT = {"AGENTS.md", "LICENSE", "README.md", "readme-gtt.md", "readme-gtt.es.md"}
SOURCE_SUFFIX = (".md", ".txt", ".rst", ".docx", ".pdf", ".odt")


def die(message, code=2):
    print(f"gtt-project: {message}", file=sys.stderr)
    sys.exit(code)


def emit(obj):
    print(json.dumps(obj, indent=2, sort_keys=True))


def now():
    return os.environ.get("GTT_NOW") or datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def read_json(path, default=None):
    try:
        with open(path, encoding="utf-8") as handle:
            return json.load(handle)
    except OSError:
        return default
    except ValueError as exc:
        die(f"{path} is not valid JSON: {exc}")


def write_json(path, data):
    os.makedirs(os.path.dirname(path) or ".", exist_ok=True)
    with open(path, "w", encoding="utf-8", newline="\n") as handle:
        handle.write(json.dumps(data, indent=2, sort_keys=True) + "\n")


def sh(*cmd):
    proc = subprocess.run(["bash", *cmd], capture_output=True, text=True, encoding="utf-8", errors="replace")
    return proc.returncode, proc.stdout, proc.stderr


def release():
    data = read_json(os.path.join(CONTRACT, "release.json"))
    if data is None:
        die(f"{CONTRACT}/release.json missing")
    return data


def profiles():
    data = read_json(os.path.join(CONTRACT, "profiles.json"))
    if data is None:
        die(f"{CONTRACT}/profiles.json missing")
    return data


def frozen_since():
    if not os.path.isfile(FROZEN):
        return None
    with open(FROZEN, encoding="utf-8") as handle:
        return handle.readline().strip() or "unknown"


def ade_json(op, *extra):
    if not os.path.isfile(".gtt/ade.json") and op in ("state", "owned", "validate"):
        return {"configured": False}
    rc, out, err = sh(".gtt/scripts/gtt-ade.sh", op, "--json", *extra)
    try:
        return json.loads(out)
    except ValueError:
        return {"error": (err or out).strip()[:200], "exit_code": rc}


# --------------------------------------------------------------- methodology

def current_profile():
    state = read_json(METHOD) or {}
    spec = profiles()
    profile = state.get("profile")
    # No selection: the gates of spec["default"] apply, but that is a fallback, never a selection - the plan is
    # reported as not selected and the Bootstrap still has to ask the human (profiles.json -> selection).
    return {"profile": profile or spec["default"], "source": "project" if profile else "default",
            "selected": bool(profile), "language": state.get("language"), "state": state}


ASKS = ("confirm", "propose_confirm", "team_policy", "required")


def interaction(profile_id, spec=None):
    """What the selected plan makes GTT do without asking and what it asks for - derived from the plan's own
    policy, never a second definition. This is how the developer-experience rule differs from plan to plan."""
    spec = spec or profiles()
    policy = spec["profiles"][profile_id]["plan"]["policy"]
    values = dict(policy["automation"], relevant_change=policy["human"]["confirmation"]["relevant_change"],
                  governed_decision=policy["human"]["confirmation"]["governed_decision"],
                  destructive=policy["human"]["confirmation"]["destructive"])
    values.pop("level", None)
    return {"plan": profile_id, "automation_level": policy["automation"]["level"],
            "without_asking": sorted(k for k, v in values.items() if v not in ASKS),
            "asks_for": {k: v for k, v in sorted(values.items()) if v in ASKS},
            "maintain_reconciles_moves": policy["automation"]["reference_updates"] in ("automatic", "automatic_safe"),
            "id_resolution": {"automatic": "use", "automatic_safe": "use"}.get(policy["automation"]["identity_resolution"], "propose")}


def cmd_interaction(args):
    cur = current_profile()
    out = dict(interaction(cur["profile"]), schema=1, selected=cur["selected"])
    if args.key:
        if args.key not in out:
            die(f"unknown key `{args.key}`")
        value = out[args.key]
        print(str(value).lower() if isinstance(value, bool) else value)
    else:
        emit(out)
    return 0


def cmd_profile(args):
    spec = profiles()
    order = spec["strictness_order"]
    cur = current_profile()
    if args.action == "get":
        prof = spec["profiles"][cur["profile"]]
        out = {"schema": 1, "contract_version": spec["contract_version"], "profile": cur["profile"], "source": cur["source"],
               "language": cur["language"], "supported": [p["id"] for p in spec["supported"]], "default": spec["default"],
               "selected": cur["selected"], "selection": spec["selection"], "plan": prof["plan"],
               "developer_experience": spec["developer_experience"], "interaction": interaction(cur["profile"], spec),
               "gates": prof["gates"], "relaxes": prof["relaxes"], "semantics": prof["semantics"],
               "invariants": [i["id"] for i in spec["invariants"]], "frozen": os.path.isfile(FROZEN)}
        if args.json:
            emit(out)
        elif cur["selected"]:
            print(f"plan: {prof['plan']['label']} ({out['profile']}); language: {out['language'] or 'unset'}; frozen: {out['frozen']}")
        else:
            print(f"plan: not selected - the human must choose one of {', '.join(out['supported'])} "
                  f"({out['profile']} gates apply meanwhile); language: {out['language'] or 'unset'}; frozen: {out['frozen']}")
        return 0
    if not args.profile and not args.language:
        die("give --profile and/or --language")
    new = dict(cur["state"])
    if args.profile:
        if args.profile not in order:
            die(f"unknown profile `{args.profile}` (supported: {', '.join(order)})")
        if os.path.isfile(FROZEN) and order.index(args.profile) < order.index(cur["profile"]):
            print(f"REFUSED: the project is frozen; moving from `{cur['profile']}` to the less strict `{args.profile}` is a "
                  "governed change (change request -> proposal -> human decision). Nothing was written.", file=sys.stderr)
            return 1
        new["profile"] = args.profile
    if args.language:
        if args.language not in release()["supported_languages"]:
            die(f"unsupported language `{args.language}` (supported: {', '.join(release()['supported_languages'])})")
        new["language"] = args.language
    new.update({"schema": 1, "set_at": now()})
    before = cur["profile"] if cur["selected"] else "not selected"
    print(f"methodology: plan {before} -> {new.get('profile', before)}; "
          f"language {cur['language'] or 'unset'} -> {new.get('language') or 'unset'}")
    if not args.apply:
        print("(dry run - nothing written; re-run with --apply)")
        return 0
    write_json(METHOD, new)
    print(f"gtt-project: written {METHOD}")
    return 0


# -------------------------------------------------------------------- next id

ID_KINDS = {"adr": "ADR", "epic": "EPIC", "story": "STORY"}


def occupied_ids(prefix):
    """Every number already taken for `prefix`: ids in the identity manifest (active AND retired - a retired id is
    never reused), file names under gtt-domain/, and ids written in gtt-domain/backlog.md. Template placeholder
    lines do not count. Deterministic: the same repository always gives the same answer."""
    token = re.compile(rf"(?<![A-Za-z0-9]){prefix}-(\d+)(?![0-9])")
    taken = {}

    def note(text, where):
        for match in token.finditer(text):
            taken.setdefault(int(match.group(1)), where)

    manifest = read_json(".gtt/index/artifacts.json") or {}
    for entry in manifest.get("artifacts", []):
        note(str(entry.get("id", "")), f"identity ({entry.get('status', 'active')})")
    for root, _dirs, files in os.walk("gtt-domain"):
        for name in files:
            note(name, os.path.join(root, name).replace(os.sep, "/"))
    try:
        with open("gtt-domain/backlog.md", encoding="utf-8") as handle:
            for line in handle:
                if not re.search(r"<[A-Za-z][^>]*>", line):      # a template example is not a real Epic/Story
                    note(line, "gtt-domain/backlog.md")
    except OSError:
        pass
    return taken


def cmd_next_id(args):
    prefix = ID_KINDS[args.kind]
    taken = occupied_ids(prefix)
    number = max(taken, default=0) + 1
    cur = current_profile()
    out = {"schema": 1, "kind": args.kind, "id": f"{prefix}-{number:03d}",
           "plan": cur["profile"], "plan_selected": cur["selected"],
           "resolution": interaction(cur["profile"])["id_resolution"],
           "occupied": [f"{prefix}-{n:03d}" for n in sorted(taken)],
           "rule": "highest occupied number + 1; retired ids are never reused"}
    if args.requested:
        match = re.fullmatch(rf"{prefix}-(\d+)", args.requested)
        if not match:
            die(f"`{args.requested}` is not a {prefix} id ({prefix}-NNN)")
        free = int(match.group(1)) not in taken
        out["requested"] = {"id": args.requested, "free": free,
                            "occupied_by": None if free else taken[int(match.group(1))]}
        if free:
            out["id"] = f"{prefix}-{int(match.group(1)):03d}"
    if args.json:
        emit(out)
    else:
        print(out["id"])
    return 0


# ------------------------------------------------------------------- sources

def cmd_source(args):
    data = read_json(SELECTED) or {"schema": 1, "sources": []}
    if args.action == "select":
        path = os.path.normpath(args.path).replace("\\", "/")
        root = os.path.realpath(".")
        if not os.path.exists(path) or not os.path.realpath(path).startswith(root):
            die(f"`{args.path}` does not exist inside the project; nothing recorded.", 1)
        if any(s["path"] == path for s in data["sources"]):
            die(f"`{path}` is already selected; nothing recorded.", 1)
        entry = {"path": path, "type": args.type, "version": args.version or None, "selected_by": args.by or "unspecified",
                 "selected_at": now(), "authority_status": "unassigned", "precedence_status": "unassigned"}
        print(f"select {path} ({args.type}): authority and precedence stay UNASSIGNED - a selected source is not a governed authority")
        if not args.apply:
            print("(dry run - nothing written; re-run with --apply)")
            return 0
        data["sources"].append(entry)
        write_json(SELECTED, data)
        return 0
    findings = gp.Findings()
    manifest = gp.load_sources(findings)
    declared = {s["path"]: s for s in (manifest["sources"] if manifest else [])}
    rows = []
    for s in data["sources"]:
        d = declared.get(s["path"])
        rows.append(dict(s, governed=bool(d), authority_status=d["authority"] if d else "unassigned",
                         precedence_status=(d["precedence"] if d and d["precedence"] is not None else "unassigned"),
                         declared_id=d["id"] if d else None, version=s["version"] or (d["version"] if d else None)))
    selected_paths = {s["path"] for s in data["sources"]}
    out = {"schema": 1, "rule": "selected != governed authority: authority and precedence are declared in "
           "gtt-domain/context/sources.md by a human, never inferred by the CLI or the Bootstrap",
           "grounding": "only declared sources with authority primary or secondary enter grounding as authority; "
                        "selected-only sources are candidates for Confirmation A",
           "selected": rows,
           "declared_not_selected": [d for p, d in declared.items() if p not in selected_paths],
           "policy": manifest["policy"] if manifest else "none"}
    if args.json:
        emit(out)
    else:
        for r in rows:
            print(f"{r['path']}  [{r['type']}]  authority: {r['authority_status']}  precedence: {r['precedence_status']}")
        if not rows:
            print("no sources selected")
    return 0


# -------------------------------------------------------------------- detect

def context_state():
    states = {}
    for name in CONTEXT_FILES:
        path = f"gtt-domain/context/{name}.md"
        if not os.path.isfile(path):
            states[name] = "missing"
            continue
        with open(path, encoding="utf-8") as handle:
            text = handle.read()
        states[name] = "placeholders" if (PLACEHOLDER.search(text) or len([l for l in text.split("\n") if l.strip()]) <= 5) else "no-placeholders"
    return states


def design_candidates():
    out = []
    for name in sorted(os.listdir(".")):
        if not os.path.isfile(name) or name in KIT or name.startswith(".") or name.startswith("SOURCE-BRIEF"):
            continue
        if name.lower().endswith(SOURCE_SUFFIX) and not re.match(r"(?i)^(license|changelog|contributing|code_of_conduct)", name):
            out.append(name)
    return out


def cmd_detect(args):
    engine, domain = os.path.isdir(".gtt"), os.path.isdir("gtt-domain")
    layout = None
    overlays = {}
    if os.path.isfile(gm.MANIFEST):
        m = gm.load()
        layout = gm.layout_version(m)
        overlays = gm.overlays(m)
    ctx = context_state()
    if not engine:
        state = "no-gtt"
    elif os.path.isfile(FROZEN):
        state = "frozen"
    elif all(v == "no-placeholders" for v in ctx.values()):
        state = "populated"
    else:
        state = "virgin"
    candidates = design_candidates()
    has_brief = bool(glob.glob("SOURCE-BRIEF.*"))
    tpl = gm.templates(gm.load()).get("initial-design-questionnaire") if engine and layout else None
    offer = bool(tpl) and state == "virgin" and not candidates and not has_brief
    out = {"schema": 1, "kind": "gtt-project-detection", "state": state,
           "gtt": {"engine": engine, "domain": domain, "scaffold_layout": layout,
                   "bootstrap": release()["bootstrap"] if engine else None},
           "catalog": bool(overlays) and all(os.path.exists(o["path"]) for o in overlays.values()) and not os.path.isfile(".gtt/ade.json"),
           "frozen": frozen_since(), "context": ctx,
           "design_sources": {"candidates_at_root": candidates, "source_brief": has_brief,
                              "note": "candidates only; which one is authoritative is a human decision (Confirmation A)"},
           "questionnaire": {"offer": offer, "operation": "template.materialize", "template": "initial-design-questionnaire",
                             "working_copy": tpl["materialize_to"] if tpl else None,
                             "working_copy_exists": bool(tpl) and os.path.isfile(tpl["materialize_to"])},
           "ade_configured": os.path.isfile(".gtt/ade.json"),
           "methodology": {k: current_profile()[k] for k in ("profile", "source", "language")}}
    if args.json:
        emit(out)
    else:
        print(f"state: {state}; frozen: {out['frozen'] or 'no'}; design candidates: {', '.join(candidates) or 'none'}; "
              f"questionnaire offered: {offer}")
    return 0


# ----------------------------------------------------------- status / session

def change_request_state():
    path = "gtt-domain/change-request.md"
    if not os.path.isfile(path):
        return "missing"
    with open(path, encoding="utf-8") as handle:
        return "empty" if re.search(r"(?m)^Change: <", handle.read()) else "filled"


def proposals():
    try:
        return sorted(f for f in os.listdir("gtt-domain/proposals") if f != "README.md")
    except OSError:
        return []


def governance():
    findings, state = gp.run_checks()
    gaps = state["gaps"]
    count = lambda kind: sum(1 for g in gaps if g["kind"] == kind)
    return {"gaps": {"open": count("OPEN"), "blocking": count("BLOCKING"), "resolved": count("RESOLVED")},
            "unresolved_conflicts": sum(1 for t in state["tags"] if t[2] == "CONFLICTO"),
            "proposals_in_governed_context": sum(1 for t in state["tags"] if t[2] == "PROPUESTA"),
            "provenance_policy": state["policy"], "provenance_findings": {"fail": len(findings.fails()),
                                                                             "warn": len(findings.items) - len(findings.fails())},
            "pending_proposals": proposals(), "change_request": change_request_state(),
            "working_agreements": {"team": sum(1 for a in state["agreements"] if a["scope"] == "team"),
                                   "user": sum(1 for a in state["agreements"] if a["scope"] == "user")}}


def run_validation():
    rc, out, err = sh(".gtt/scripts/gtt-validate.sh")
    checks, messages = [], []
    for line in out.splitlines():
        # the summary lines gtt-validate.sh prints per check are fixed-width ("PASS" + 15 spaces, ...); the detail
        # lines a failing check adds start with the level and TWO spaces and must not be mistaken for checks
        m = re.match(r"^(PASS {15}|FAIL {15}|SKIPPED {12}|CANNOT-DETERMINE {3})(\S.*)$", line)
        if m:
            checks.append({"result": m.group(1).strip().lower(), "check": m.group(2).strip()})
        elif re.match(r"^(FAIL|WARN) {2,}\S", line):
            messages.append(line.strip())
    result = "pass" if rc == 0 else "fail"
    return {"schema": 1, "kind": "gtt-validation", "result": result, "exit_code": rc, "checks": checks,
            "failing": [c["check"] for c in checks if c["result"] == "fail"], "messages": messages}, rc


def cmd_validation(args):
    out, rc = run_validation()
    if args.json:
        emit(out)
    else:
        for c in out["checks"]:
            print(f"{c['result'].upper():18} {c['check']}")
    return rc


def selected_summary():
    data = read_json(SELECTED) or {"sources": []}
    findings = gp.Findings()
    manifest = gp.load_sources(findings)
    return {"selected": len(data["sources"]), "declared": len(manifest["sources"]) if manifest else 0,
            "policy": manifest["policy"] if manifest else "none"}


def cmd_status(args):
    cur = current_profile()
    val = None
    if args.with_validation:
        val, _ = run_validation()
    out = {"schema": 1, "kind": "gtt-status",
           "bootstrap": dict(release()["bootstrap"], scaffold_layout=gm.layout_version(gm.load())),
           "ade": ade_json("state"),
           "methodology": {"profile": cur["profile"], "source": cur["source"], "language": cur["language"]},
           "sources": selected_summary(),
           "governance": governance(),
           "freeze": {"frozen": os.path.isfile(FROZEN), "since": frozen_since()},
           "validation": val or {"run": False, "operation": "validation.run", "note": "pass --with-validation to run it"},
           "session": {"file": "gtt-domain/session.md", "exists": os.path.isfile("gtt-domain/session.md"),
                       "authority": "none - derived operational state"}}
    if args.json:
        emit(out)
    else:
        b = out["bootstrap"]
        print(f"{b['id']} {b['version']}; profile {cur['profile']}; frozen: {out['freeze']['frozen']}; "
              f"gaps open/blocking: {out['governance']['gaps']['open']}/{out['governance']['gaps']['blocking']}")
    return 0


def git(*args):
    try:
        proc = subprocess.run(["git", *args], capture_output=True, text=True, encoding="utf-8", errors="replace")
    except OSError:
        return None
    return proc.stdout.strip() if proc.returncode == 0 else None


def stories(status):
    path = "gtt-domain/backlog.md"
    if not os.path.isfile(path):
        return []
    with open(path, encoding="utf-8") as handle:
        lines = handle.read().replace("\r\n", "\n").split("\n")
    out = []
    for i, line in enumerate(lines):
        if line.startswith("##### STORY-") and any(re.match(r"^\s*- \*\*Status:\*\* " + status + r"\s*$", l) for l in lines[i + 1:i + 4]):
            out.append(line[6:].strip())
    return out


def cmd_session(args):
    branch = git("rev-parse", "--abbrev-ref", "HEAD")
    dirty = git("status", "--porcelain")
    recent = git("log", "-5", "--format=%h %s")
    cur = current_profile()
    ctx_files = sorted(glob.glob("gtt-domain/context/*.md"))
    out = {"schema": 1, "kind": "gtt-session-context", "authority": "none", "operational_only": True,
           "markers": ["operational-only", "NOT authority", "NOT evidence", "NOT a decision record", "NOT a grounding source"],
           "precedence": "gtt-domain/context/ (L0) > gtt-domain/adr/ (L1) > gtt-domain/backlog.md > this",
           "derived_from": "repository state at generation time; never from an agent's recollection",
           "freeze": {"frozen": os.path.isfile(FROZEN), "since": frozen_since()},
           "change_request": change_request_state(),
           "proposals": proposals(),
           "validation": {"operation": "validation.run", "result": None,
                          "note": "session-context does not execute validation; run validation.run"},
           "git": {"available": branch is not None, "branch": branch,
                   "uncommitted_paths": len(dirty.splitlines()) if dirty else 0,
                   "recent_commits": recent.splitlines() if recent else []},
           "operational": {"ade": ade_json("state"), "methodology": {"profile": cur["profile"], "language": cur["language"]},
                           "stories_in_progress": stories("In Progress"), "stories_blocked": stories("Blocked"),
                           "governance": governance()},
           "artifacts": {"governed_context": ctx_files,
                         "adrs": sorted(glob.glob("gtt-domain/adr/ADR-[0-9]*.md")),
                         "backlog": "gtt-domain/backlog.md", "proposals_dir": "gtt-domain/proposals/",
                         "identity_index": ".gtt/index/artifacts.json"}}
    emit(out) if args.json else print("\n".join(f"{k}: {v}" for k, v in out.items() if k in ("freeze", "change_request", "proposals")))
    return 0


# ------------------------------------------------------------- export / clean

def ownership():
    policy = read_json(os.path.join(CONTRACT, "export-policy.json"))
    if policy is None:
        die(f"{CONTRACT}/export-policy.json missing")
    static = [e["pattern"] for e in policy["ownership"]["static"]]
    owned = ade_json("owned")
    surfaces = owned.get("surfaces", []) if isinstance(owned, dict) else []
    clean = sorted({s["path"] for s in surfaces if s["status"] == "unmodified"})
    confirm = sorted({s["path"] + f"  ({s['status']}, {s['basis']})" for s in surfaces if s["status"] in ("modified", "unrecorded")})
    return policy, static, clean, confirm, owned


def cmd_export_policy(args):
    policy, static, clean, confirm, owned = ownership()
    out = {"schema": 1, "contract_version": policy["contract_version"], "kind": "gtt-export-policy",
           "clean_export": {"creates": policy["export"]["clean"]["creates"], "include": ["**"],
                            "exclude": static + clean, "exclude_static": static, "exclude_from_ledger": clean,
                            "requires_confirmation": confirm},
           "ownership": policy["ownership"], "ade_state_configured": bool(owned.get("configured", True)) if isinstance(owned, dict) else False,
           "note": "everything not matched belongs to the host project; ADE directories are never matched by name"}
    emit(out) if args.json else print("\n".join(out["clean_export"]["exclude"]))
    return 0


def cmd_clean_plan(args):
    policy, static, clean, confirm, _ = ownership()
    out = {"schema": 1, "kind": "gtt-clean-plan", "removes_nothing": True,
           "would_remove": static + clean, "plus": policy["clean"]["plus"],
           "requires_confirmation": confirm, "confirmation": policy["clean"]["confirmation"],
           "keeps": policy["clean"]["keeps"],
           "distinct_from": "export --clean, which creates a separate delivery artifact and leaves this project untouched",
           "ade_operations": ["ade.remove"]}
    emit(out) if args.json else print("\n".join(out["would_remove"]))
    return 0


# ------------------------------------------------------------------ recovery

def snapshot():
    rel = release()
    cur = current_profile()
    ade = read_json(".gtt/ade.json") or {}
    sel = read_json(SELECTED) or {"sources": []}
    return {"schema": 1, "contract_version": read_json(os.path.join(CONTRACT, "recovery.json"))["contract_version"],
            "kind": "gtt-recovery-snapshot",
            "bootstrap": rel["bootstrap"], "compatibility": rel["compatibility"],
            "ade": {"primary": ade.get("primary"), "participating": ade.get("participating", []), "excluded": ade.get("excluded", []),
                    "configured": bool(ade)},
            "methodology": {"profile": cur["profile"], "language": cur["language"]},
            "sources": {"selected": sel["sources"]},
            "operational": {"frozen": os.path.isfile(FROZEN), "frozen_marker_time": frozen_since()},
            "recovery": {"created_at": now(), "note": "GTT configuration only; the governed domain content lives in version control",
                         "restore_rules": "recovery.json"}}


def cmd_recovery(args):
    if args.action == "snapshot":
        snap = snapshot()
        if args.output:
            if os.path.exists(args.output):
                die(f"{args.output} exists; snapshots are never overwritten.", 1)
            write_json(args.output, snap)
            print(f"snapshot written: {args.output}")
        else:
            emit(snap)
        return 0
    spec = read_json(os.path.join(CONTRACT, "recovery.json"))
    snap = read_json(args.snapshot)
    if snap is None:
        die(f"snapshot not readable: {args.snapshot}", 1)
    rel = release()["bootstrap"]
    refusals = []
    for key in spec["snapshot"]["required"]:
        if key not in snap:
            refusals.append(f"snapshot lacks required field `{key}`")
    if not refusals:
        if snap["kind"] != spec["snapshot"]["kind"]:
            refusals.append("not a gtt-recovery-snapshot")
        if snap["schema"] > 1:
            refusals.append(f"snapshot schema {snap['schema']} is newer than this Bootstrap understands")
        if snap["bootstrap"]["id"] != rel["id"]:
            refusals.append(f"snapshot was produced by `{snap['bootstrap']['id']}`, not `{rel['id']}`")
        order = profiles()["strictness_order"]
        prof = snap["methodology"]["profile"]
        if prof not in order:
            refusals.append(f"snapshot names an unknown profile `{prof}`")
        elif os.path.isfile(FROZEN) and order.index(prof) < order.index(current_profile()["profile"]):
            refusals.append("the project is frozen and the snapshot would relax the methodology profile")
    if refusals:
        for r in refusals:
            print("REFUSED: " + r, file=sys.stderr)
        print("Nothing was written.", file=sys.stderr)
        return 1
    plan = []
    if not os.path.exists(METHOD):
        plan.append(("write", METHOD, {"schema": 1, "profile": snap["methodology"]["profile"],
                                       "language": snap["methodology"]["language"], "set_at": now(), "restored": True}))
    else:
        plan.append(("keep", METHOD, "already exists - never overwritten"))
    if not os.path.exists(SELECTED) and snap["sources"]["selected"]:
        plan.append(("write", SELECTED, {"schema": 1, "sources": snap["sources"]["selected"]}))
    elif os.path.exists(SELECTED):
        plan.append(("keep", SELECTED, "already exists - never overwritten"))
    ade = snap["ade"]
    if ade.get("configured") and not os.path.exists(".gtt/ade.json"):
        plan.append(("ade", ".gtt/ade.json", ade))
    for kind, target, detail in plan:
        print(f"  {kind.upper():6} {target}" + (f"   ({detail})" if isinstance(detail, str) else ""))
    print("  NOTE   the freeze marker is never restored: freeze is a human act")
    if not args.apply:
        print("(dry run - nothing written; re-run with --apply)")
        return 0
    for kind, target, detail in plan:
        if kind == "write":
            write_json(target, detail)
        elif kind == "ade":
            if not args.frm:
                print("  ADE state needs --from <catalog> to re-install the recorded ADEs; skipped (run ade.install yourself)")
                continue
            rc, out, err = sh(".gtt/scripts/gtt-ade.sh", "install", "--from", args.frm, "--participating",
                              ",".join(detail["participating"]), "--primary", detail["primary"], "--apply")
            sys.stdout.write(out); sys.stderr.write(err)
            if rc != 0:
                return rc
    print("restored. Next: validation.run")
    return 0


# ---------------------------------------------------------------------- main

def build_parser():
    parser = argparse.ArgumentParser(prog="gtt-project.sh", description=__doc__.split("\n")[0])
    sub = parser.add_subparsers(dest="command", required=True)
    p = sub.add_parser("detect"); p.add_argument("--json", action="store_true"); p.set_defaults(func=cmd_detect)
    p = sub.add_parser("status"); p.add_argument("--json", action="store_true"); p.add_argument("--with-validation", action="store_true"); p.set_defaults(func=cmd_status)
    p = sub.add_parser("session"); p.add_argument("--json", action="store_true"); p.set_defaults(func=cmd_session)
    p = sub.add_parser("validation"); p.add_argument("--json", action="store_true"); p.set_defaults(func=cmd_validation)
    p = sub.add_parser("interaction"); p.add_argument("--key"); p.set_defaults(func=cmd_interaction)
    p = sub.add_parser("next-id"); p.add_argument("--kind", required=True, choices=sorted(ID_KINDS)); p.add_argument("--requested")
    p.add_argument("--json", action="store_true"); p.set_defaults(func=cmd_next_id)
    p = sub.add_parser("profile"); p.add_argument("action", choices=["get", "set"]); p.add_argument("--json", action="store_true")
    p.add_argument("--profile"); p.add_argument("--language"); p.add_argument("--apply", action="store_true"); p.set_defaults(func=cmd_profile)
    p = sub.add_parser("source"); p.add_argument("action", choices=["select", "list"]); p.add_argument("--json", action="store_true")
    p.add_argument("--path"); p.add_argument("--type"); p.add_argument("--version"); p.add_argument("--by"); p.add_argument("--apply", action="store_true")
    p.set_defaults(func=cmd_source)
    p = sub.add_parser("export-policy"); p.add_argument("--json", action="store_true"); p.set_defaults(func=cmd_export_policy)
    p = sub.add_parser("clean-plan"); p.add_argument("--json", action="store_true"); p.set_defaults(func=cmd_clean_plan)
    p = sub.add_parser("recovery"); p.add_argument("action", choices=["snapshot", "restore"]); p.add_argument("--output")
    p.add_argument("--snapshot"); p.add_argument("--from", dest="frm"); p.add_argument("--apply", action="store_true"); p.set_defaults(func=cmd_recovery)
    return parser


def main(argv):
    if not os.path.isdir(".gtt"):
        die("run from the project root (no .gtt/ directory here)")
    args = build_parser().parse_args(argv)
    if args.command == "source" and args.action == "select" and not (args.path and args.type):
        die("source select needs --path and --type")
    if args.command == "recovery" and args.action == "restore" and not args.snapshot:
        die("recovery restore needs --snapshot")
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
