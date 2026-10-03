#!/usr/bin/env python3
"""GTT - ADE integration engine (Multi-ADE).

One GTT governance model, several ADE integration surfaces, one Primary ADE.
This engine is the Bootstrap-owned contract a CLI (or a human) uses to ask
"which ADEs exist, which are here, which did the human choose, is each
integration intact, and what exactly did GTT install" - so that no caller
invents an ADE-specific path.

What it reads
  * the ADE integration registry: the `overlays:` section of
    .gtt/scaffold/manifest.yaml (data only; see gtt_manifest.py)
  * per-ADE Session Memory declarations, .gtt/session-adapters/<id>.json
    (only `adapter_status`, `ade_binary` and `files[].install_path`)
  * the per-project state, .gtt/ade.json (written only by this engine)

What the state means - and does not
  detected       an observation (never stored): candidates only
  participating  the ADEs the human explicitly chose to govern
  primary        ONE participating ADE: the principal workflow identifier.
                 It carries NO authority over any governed artifact.
  excluded       ADEs the human declined; a re-run never reintroduces them
  installed      a ledger of the files GTT itself copied (path -> sha256), so
                 clean / export --clean remove exactly those and nothing else

Every mutating command is a dry run unless --apply is given, refuses to
overwrite a file it did not install (report, never merge), and rolls back on
any failure. The engine holds no governance logic and grants no ADE any
authority: instruction files and overlays are integration surfaces.

Exit 0 = ok, 1 = violation / conflict, 2 = cannot run (usage, missing input).
"""

import argparse
import copy
import hashlib
import json
import os
import posixpath
import shutil
import sys

import gtt_manifest as gm

SCHEMA = 1
STATE = ".gtt/ade.json"
SESSION_DIR = ".gtt/session-adapters"
IGNORE_DIRS = {"__pycache__", ".git"}
IGNORE_SUFFIX = (".pyc",)
NOTICE = ("Declaration of the ADEs the human chose for this project, plus the ledger of "
          "files GTT installed for them. Written only by .gtt/scripts/gtt-ade.sh; never "
          "hand-edited. Not governed context and not authority: the Primary ADE "
          "identifies the principal workflow and holds no authority over any artifact.")


# ------------------------------------------------------------------ helpers

def die(message, code=2):
    print(f"gtt-ade: {message}", file=sys.stderr)
    sys.exit(code)


def norm(path):
    return posixpath.normpath(path.replace("\\", "/"))


def sha(path):
    digest = hashlib.sha256()
    with open(path, "rb") as handle:
        for chunk in iter(lambda: handle.read(65536), b""):
            digest.update(chunk)
    return digest.hexdigest()


def split(text):
    return [x.strip() for x in (text or "").split(",") if x.strip()]


def registry(root):
    try:
        manifest = gm.load(os.path.join(root, gm.MANIFEST))
        return manifest, gm.overlays(manifest)
    except gm.ManifestError as exc:
        die(str(exc))


def session_decl(root, ade):
    path = os.path.join(root, SESSION_DIR, ade + ".json")
    try:
        with open(path, encoding="utf-8") as handle:
            return json.load(handle)
    except (OSError, ValueError):
        return None


def declared_install_paths(root, ade):
    """Install paths of a Session Memory adapter that is actually installed."""
    decl = session_decl(root, ade)
    if not decl or decl.get("adapter_status") != "installed":
        return []
    return sorted(f["install_path"] for f in decl.get("files", []) if f.get("install_path"))


def load_state():
    if not os.path.isfile(STATE):
        return None
    try:
        with open(STATE, encoding="utf-8") as handle:
            return json.load(handle)
    except ValueError as exc:
        die(f"{STATE} is not valid JSON: {exc}")


def state_text(state):
    return json.dumps(state, indent=2, sort_keys=True) + "\n"


def expand(root, owned):
    """Files (project-relative, posix) under every owned path of `root`."""
    files = []
    for entry in owned:
        full = os.path.join(root, entry)
        if entry.endswith("/") or os.path.isdir(full):
            for dirpath, dirnames, filenames in os.walk(full):
                dirnames[:] = sorted(d for d in dirnames if d not in IGNORE_DIRS)
                for name in sorted(filenames):
                    if not name.endswith(IGNORE_SUFFIX):
                        files.append(norm(os.path.relpath(os.path.join(dirpath, name), root)))
        elif os.path.isfile(full):
            files.append(norm(entry))
    return sorted(set(files))


def emit(args, obj, text):
    if getattr(args, "json", False):
        print(json.dumps(obj, indent=2, sort_keys=True))
    else:
        print(text)


def new_state(primary, participating, excluded, installed):
    return {"schema": SCHEMA, "notice": NOTICE, "primary": primary,
            "participating": sorted(set(participating)),
            "excluded": sorted(set(excluded)), "installed": installed}


class Txn:
    """File changes that can all be undone if a later step fails."""

    def __init__(self):
        self.undo = []

    def _mkdirs(self, directory):
        made, probe = [], directory
        while probe and not os.path.isdir(probe):
            made.append(probe)
            probe = os.path.dirname(probe)
        if directory:
            os.makedirs(directory, exist_ok=True)
        return made

    def copy(self, src, dst):
        old = open(dst, "rb").read() if os.path.isfile(dst) else None
        made = self._mkdirs(os.path.dirname(dst))
        shutil.copyfile(src, dst)
        try:
            shutil.copymode(src, dst)
        except OSError:
            pass
        self.undo.append(("copy", dst, old, made))

    def write_text(self, dst, text):
        old = open(dst, "rb").read() if os.path.isfile(dst) else None
        made = self._mkdirs(os.path.dirname(dst))
        with open(dst, "w", encoding="utf-8", newline="\n") as handle:
            handle.write(text)
        self.undo.append(("copy", dst, old, made))

    def delete(self, path):
        old = open(path, "rb").read()
        os.remove(path)
        self.undo.append(("delete", path, old, []))
        parent = os.path.dirname(path)
        while parent and parent != ".":
            try:
                os.rmdir(parent)
            except OSError:
                break
            self.undo.append(("mkdir", parent, None, []))
            parent = os.path.dirname(parent)

    def rollback(self):
        for kind, path, old, made in reversed(self.undo):
            try:
                if kind == "mkdir":
                    os.makedirs(path, exist_ok=True)
                elif kind == "delete":
                    os.makedirs(os.path.dirname(path) or ".", exist_ok=True)
                    with open(path, "wb") as handle:
                        handle.write(old)
                elif old is None:
                    if os.path.isfile(path):
                        os.remove(path)
                    for directory in made:
                        try:
                            os.rmdir(directory)
                        except OSError:
                            pass
                else:
                    with open(path, "wb") as handle:
                        handle.write(old)
            except OSError as exc:
                print(f"gtt-ade: rollback could not restore {path}: {exc}", file=sys.stderr)
        self.undo = []


# --------------------------------------------------------------- validation

def validate(reg, state):
    """[(level, ade|None, message)] with level PASS | WARN | FAIL."""
    out = []
    if state is None:
        return None
    add = lambda level, ade, msg: out.append((level, ade, msg))
    if state.get("schema") != SCHEMA:
        add("FAIL", None, f"{STATE}: unsupported schema {state.get('schema')!r}")
        return out
    part, prim = state.get("participating"), state.get("primary")
    excl, inst = state.get("excluded", []), state.get("installed")
    if not (isinstance(part, list) and part and all(isinstance(a, str) for a in part)):
        add("FAIL", None, "participating must be a non-empty list of ADE ids")
    if not isinstance(prim, str):
        add("FAIL", None, "primary must be exactly one ADE id")
    if not isinstance(excl, list) or not isinstance(inst, dict):
        add("FAIL", None, "excluded must be a list and installed a mapping")
    if out:
        return out
    if len(set(part)) != len(part):
        add("FAIL", None, "participating lists an ADE twice")
    for ade in sorted(set(part) | set(excl) | set(inst)):
        if ade not in reg:
            add("FAIL", ade, f"unknown ADE `{ade}` (not in the registry)")
    if prim not in part:
        add("FAIL", prim, f"primary `{prim}` is not a participating ADE")
    for ade in sorted(set(part) & set(excl)):
        add("FAIL", ade, "is both participating and excluded")
    for ade in sorted(set(part) - set(inst)):
        add("FAIL", ade, "participating but has no install record")
    for ade in sorted(set(inst) - set(part)):
        add("FAIL", ade, "has an install record but is not participating")
    if not os.path.isfile("AGENTS.md"):
        add("FAIL", None, "AGENTS.md (portable core) is missing")

    failed = {a for level, a, _ in out if level == "FAIL"}
    for ade in part:
        if ade not in reg or ade in failed:
            continue
        entry, record, bad = reg[ade], inst.get(ade, {}), False
        for label, path in (("overlay path", entry["path"]), ("instruction entry point", entry["entry"])):
            if not os.path.exists(path):
                add("FAIL", ade, f"{label} missing: {path}")
                bad = True
        files = record.get("files", {}) if isinstance(record, dict) else {}
        for rel in sorted(files):
            if not os.path.isfile(rel):
                add("FAIL", ade, f"installed file missing: {rel}")
                bad = True
            elif sha(rel) != files[rel]:
                add("WARN", ade, f"modified since install: {rel}")
        for rel in declared_install_paths(".", ade):
            if not os.path.isfile(rel):
                add("FAIL", ade, f"declared session adapter file missing: {rel}")
                bad = True
        if not bad:
            add("PASS", ade, f"integration valid (enforcement: {entry['enforcement']})")
    for ade in sorted(set(reg) - set(part)):
        seen = [p for p in reg[ade]["detect"] if os.path.exists(p)]
        if seen:
            label = "explicitly excluded but present" if ade in excl else "detected, not participating"
            add("WARN", ade, f"{label}: {', '.join(seen)} - not governed by GTT")
    return out


def render_validation(results):
    lines = []
    for level, ade, msg in results:
        lines.append(f"{level:5}  {(ade or '-'):9} {msg}")
    return "\n".join(lines)


# ----------------------------------------------------------------- commands

def cmd_list(args):
    root = args.frm or "."
    manifest, reg = registry(root)
    rows = []
    for ade, entry in sorted(reg.items()):
        decl = session_decl(root, ade) or {}
        rows.append(dict(entry, session_adapter={
            "declared": bool(decl), "status": decl.get("adapter_status"),
            "runtime": (decl.get("verification") or {}).get("runtime")}))
    text = ["GTT ADE integration registry (layout version %s)" % gm.layout_version(manifest)]
    for row in rows:
        owned = ", ".join(row["owned"]) or "(none beyond the core entry point)"
        text.append(f"  {row['id']:8} {row['name']:16} entry: {row['entry']:38} owned: {owned}")
        text.append(f"           enforcement: {row['enforcement']}; session adapter: "
                    f"{row['session_adapter']['status'] or 'not declared'}")
    emit(args, {"schema": SCHEMA, "layout": gm.layout_version(manifest), "ades": rows}, "\n".join(text))
    return 0


def detect_rows(root, reg, state):
    part = set((state or {}).get("participating", []))
    excl = set((state or {}).get("excluded", []))
    rows = []
    for ade, entry in sorted(reg.items()):
        signals = ["path:" + p for p in entry["detect"] if os.path.exists(p)]
        decl = session_decl(root, ade) or {}
        binary = decl.get("ade_binary")
        if binary and shutil.which(binary):
            signals.append("binary:" + binary)
        status = ("primary" if state and state.get("primary") == ade else
                  "participating" if ade in part else "excluded" if ade in excl else "candidate")
        rows.append({"id": ade, "name": entry["name"], "signals": signals, "status": status})
    return rows


def cmd_detect(args):
    root = args.frm or "."
    _, reg = registry(root)
    rows = detect_rows(root, reg, load_state())
    lines = ["GTT ADE detection - candidates only. Detection is not installation,",
             "authorization, governance or participation: the human chooses."]
    for row in rows:
        sig = ", ".join(row["signals"]) or "no signal"
        lines.append(f"  {row['id']:8} {row['name']:16} {row['status']:13} signals: {sig}")
    emit(args, {"schema": SCHEMA, "candidates": rows}, "\n".join(lines))
    return 0


def cmd_state(args):
    _, reg = registry(".")
    state = load_state()
    if state is None:
        emit(args, {"schema": SCHEMA, "configured": False},
             "ADE state: not configured (no .gtt/ade.json - single-ADE legacy install or the catalog).\n"
             "Adopt it with: bash .gtt/scripts/gtt-ade.sh adopt")
        return 0
    results = validate(reg, state) or []
    rows = detect_rows(".", reg, state)
    pending = [r for r in rows if r["status"] in ("candidate", "excluded")
               and any(s.startswith("path:") for s in r["signals"])]
    lines = [f"primary: {state.get('primary')}",
             "participating: " + ", ".join(state.get("participating", [])),
             "excluded: " + (", ".join(state.get("excluded", [])) or "none"),
             "detected, not participating: " + (", ".join(r["id"] for r in pending) or "none"),
             "integrations:"]
    for level, ade, msg in results:
        lines.append(f"  {level:5} {(ade or '-'):9} {msg}")
    emit(args, {"schema": SCHEMA, "configured": True, "primary": state.get("primary"),
                "participating": state.get("participating"), "excluded": state.get("excluded"),
                "detected_not_participating": [r["id"] for r in pending],
                "integrations": [{"level": l, "ade": a, "message": m} for l, a, m in results]},
         "\n".join(lines))
    return 0


def cmd_validate(args):
    _, reg = registry(".")
    state = load_state()
    if state is None:
        print("gtt-ade: cannot determine - no .gtt/ade.json (run `gtt-ade.sh adopt` or `install`).",
              file=sys.stderr)
        return 2
    results = validate(reg, state)
    emit(args, {"schema": SCHEMA, "results": [{"level": l, "ade": a, "message": m} for l, a, m in results]},
         render_validation(results))
    failed = [r for r in results if r[0] == "FAIL"]
    if failed:
        print(f"gtt-ade: FAILED - {len(failed)} integration violation(s).", file=sys.stderr)
        return 1
    return 0


def check_ids(reg, ids, what):
    unknown = [a for a in ids if a not in reg]
    if unknown:
        die(f"unknown ADE id(s) for {what}: {', '.join(unknown)} (registry: {', '.join(sorted(reg))})")


def finish(args, txn, state, reg, done_message, check=True):
    """Write state, validate the result, roll everything back on failure.
    `remove` passes check=False: a removal's postcondition is absence, not integrity."""
    txn.write_text(STATE, state_text(state))
    failed = [r for r in validate(reg, state) if r[0] == "FAIL"] if check else []
    if failed:
        txn.rollback()
        for level, ade, msg in failed:
            print(f"FAIL   {(ade or '-'):9} {msg}", file=sys.stderr)
        die("post-change validation failed; every change was rolled back.", 1)
    print(done_message)
    return 0


def cmd_install(args):
    cat = args.frm or "."
    manifest, reg = registry(cat)
    part, primary = split(args.participating), args.primary
    check_ids(reg, part + [primary], "--participating/--primary")
    if not part or primary not in part:
        die("--primary must be one of --participating (exactly one Primary ADE).")
    state = load_state()
    reinclude = set(split(args.reinclude))
    if state:
        current = set(state["participating"])
        if current - set(part):
            die("install never drops an installed ADE (" + ", ".join(sorted(current - set(part))) +
                "); use `remove` if that is what you mean.")
        if primary != state["primary"]:
            die(f"the Primary ADE is `{state['primary']}`; changing it is `set-primary`, never a side effect of install.")
        blocked = [a for a in part if a not in current and a in state.get("excluded", []) and a not in reinclude]
        if blocked:
            die("explicitly excluded by an earlier run: " + ", ".join(blocked) +
                "; pass --reinclude <ade> to say you want it now.")
    current = set(state["participating"]) if state else set()
    new = [a for a in sorted(set(part)) if a not in current]
    layout = gm.layout_version(manifest)
    for ade in new:
        if reg[ade]["scaffold"] is not None and reg[ade]["scaffold"] != layout:
            die(f"overlay `{ade}` declares scaffold layout {reg[ade]['scaffold']}, catalog is {layout}.")
    plan, conflicts = [], []
    for ade in new:
        for owned in reg[ade]["owned"]:
            if not os.path.exists(os.path.join(cat, owned)):
                die(f"catalog {cat} does not contain `{owned}` (registry says overlay `{ade}` owns it).")
        for rel in expand(cat, reg[ade]["owned"]):
            (conflicts if os.path.exists(rel) else plan).append((ade, rel))
    print(f"GTT ADE install - primary: {primary}; participating: {', '.join(sorted(set(part)))}")
    for ade, rel in plan:
        print(f"  COPY      {ade:8} {rel}")
    for ade in new:
        if not reg[ade]["owned"]:
            print(f"  NOTHING   {ade:8} (its integration is the core entry point {reg[ade]['entry']})")
    for ade, rel in conflicts:
        print(f"  CONFLICT  {ade:8} {rel} already exists - GTT will not overwrite or merge it")
    if conflicts:
        print("gtt-ade: nothing was written; resolve the conflict(s) deliberately, then re-run.", file=sys.stderr)
        return 1
    if not args.apply:
        print("(dry run - nothing written; re-run with --apply)")
        return 0
    installed = copy.deepcopy(state["installed"]) if state else {}
    txn = Txn()
    try:
        for ade in new:
            installed[ade] = {"origin": "bootstrap", "files": {}}
        for ade, rel in plan:
            txn.copy(os.path.join(cat, rel), rel)
            installed[ade]["files"][rel] = sha(rel)
        excluded = ((set(state.get("excluded", [])) if state else set()) | (set(reg) - set(part))) - set(part)
        return finish(args, txn, new_state(primary, part, excluded, installed), reg,
                      f"gtt-ade: installed {len(plan)} file(s); state written to {STATE}.")
    except OSError as exc:
        txn.rollback()
        die(f"install failed and was rolled back: {exc}", 1)


def present_overlays(reg):
    have = [a for a, e in sorted(reg.items()) if e["owned"] and any(os.path.exists(p) for p in e["owned"])]
    if have:
        return have
    bare = [a for a, e in sorted(reg.items()) if not e["owned"] and os.path.exists(e["entry"])]
    return bare if len(bare) == 1 else []


def cmd_adopt(args):
    _, reg = registry(".")
    if load_state() is not None:
        die("already configured (.gtt/ade.json exists); adopt is for projects that predate it.")
    if args.participating:
        part, primary = split(args.participating), args.primary
        check_ids(reg, part + [primary or ""], "--participating/--primary")
        if not primary:
            die("--primary is required with --participating.")
    else:
        have = present_overlays(reg)
        if len(have) != 1:
            die("cannot infer which ADE participates: overlays present = " +
                (", ".join(have) or "none") + ". Authorization is not inferred; declare it with "
                "--participating a,b --primary a.", 1)
        part, primary = have, have[0]
    if primary not in part:
        die("--primary must be one of --participating.")
    state = new_state(primary, part, [], {a: {"origin": "adopted", "files": {}} for a in part})
    results = validate(reg, state)
    print(f"GTT ADE adopt - primary: {primary}; participating: {', '.join(sorted(part))}")
    print(render_validation(results))
    if any(r[0] == "FAIL" for r in results):
        print("gtt-ade: nothing was written; the declared integration is not intact.", file=sys.stderr)
        return 1
    print("Adopted ADEs carry no install record: clean / export --clean keep their files unless "
          "--assume-registry-owned says the registry's paths are GTT's.")
    if not args.apply:
        print("(dry run - nothing written; re-run with --apply)")
        return 0
    txn = Txn()
    return finish(args, txn, state, reg, f"gtt-ade: state written to {STATE}.")


def cmd_set_primary(args):
    _, reg = registry(".")
    state = load_state() or die("no .gtt/ade.json; nothing to change.")
    if args.ade not in state["participating"]:
        die(f"`{args.ade}` is not participating; the Primary ADE must be one of: {', '.join(state['participating'])}.")
    print(f"primary: {state['primary']} -> {args.ade}   (workflow identifier only; no authority moves)")
    if not args.apply:
        print("(dry run - nothing written; re-run with --apply)")
        return 0
    state["primary"] = args.ade
    return finish(args, Txn(), state, reg, f"gtt-ade: primary is now {args.ade}.")


def cmd_record(args):
    _, reg = registry(".")
    state = load_state() or die("no .gtt/ade.json.")
    if args.ade not in state["participating"]:
        die(f"`{args.ade}` is not participating.")
    allowed = set(expand(".", reg[args.ade]["owned"])) | set(declared_install_paths(".", args.ade))
    for path in args.paths:
        rel = norm(path)
        if rel not in allowed or not os.path.isfile(rel):
            die(f"`{rel}` is not an existing file the registry or a declaration attributes to `{args.ade}`.")
        print(f"  RECORD    {args.ade:8} {rel}")
    if not args.apply:
        print("(dry run - nothing written; re-run with --apply)")
        return 0
    for path in args.paths:
        state["installed"][args.ade]["files"][norm(path)] = sha(norm(path))
    return finish(args, Txn(), state, reg, "gtt-ade: recorded.")


def owned_rows(reg, state):
    rows = []
    for ade in sorted(state["participating"]):
        record = state["installed"].get(ade, {})
        files = record.get("files", {})
        for rel in sorted(files):
            status = ("missing" if not os.path.isfile(rel) else
                      "unmodified" if sha(rel) == files[rel] else "modified")
            rows.append({"ade": ade, "path": rel, "status": status, "basis": "ledger"})
        extra = {p: "session-declaration" for p in declared_install_paths(".", ade)}
        if record.get("origin") == "adopted":
            extra.update({p: "registry" for p in expand(".", reg[ade]["owned"])})
        for rel in sorted(extra):
            if rel not in files and os.path.isfile(rel):
                rows.append({"ade": ade, "path": rel, "status": "unrecorded", "basis": extra[rel]})
    return rows


def cmd_owned(args):
    manifest, reg = registry(".")
    state = load_state() or die("no .gtt/ade.json; nothing is recorded as GTT-installed.")
    rows = owned_rows(reg, state)
    core = [e["path"] for e in manifest["entries"].get("entry_points", [])]
    text = ["GTT-installed ADE integration surfaces (the only ADE paths clean / export --clean may remove):"]
    text += [f"  {r['status']:11} {r['ade']:8} {r['path']}   [{r['basis']}]" for r in rows]
    text.append("core entry points (removed with the core, never with an ADE): " + ", ".join(core))
    text.append("`unrecorded` files are never removed without --assume-registry-owned; anything not "
                "listed here is the host project's own and must be kept.")
    emit(args, {"schema": SCHEMA, "surfaces": rows, "core_entry_points": core}, "\n".join(text))
    return 0


def cmd_remove(args):
    _, reg = registry(".")
    state = load_state() or die("no .gtt/ade.json; nothing is recorded as GTT-installed.")
    targets = list(state["participating"]) if args.all else split(args.ades)
    if not targets:
        die("name the ADE(s) to remove, or --all.")
    check_ids(reg, targets, "remove")
    absent = [a for a in targets if a not in state["participating"]]
    if absent:
        die("not participating: " + ", ".join(absent))
    remaining = [a for a in state["participating"] if a not in targets]
    if not args.all:
        if not remaining:
            die("that would leave no participating ADE; use --all to remove GTT's ADE integrations entirely.")
        if state["primary"] in targets and args.new_primary not in remaining:
            die(f"`{state['primary']}` is the Primary ADE; pass --new-primary <one of: {', '.join(remaining)}>.")
    rows = [r for r in owned_rows(reg, state) if r["ade"] in targets]
    removable, kept = [], []
    for row in rows:
        ok = (row["status"] == "unmodified" or
              (row["status"] == "modified" and args.include_modified) or
              (row["status"] == "unrecorded" and args.assume_registry_owned))
        (removable if ok else kept).append(row)
    print(f"GTT ADE remove - {', '.join(targets)}")
    for row in removable:
        print(f"  REMOVE    {row['ade']:8} {row['path']}")
    for row in kept:
        if row["status"] != "missing":
            print(f"  KEEP      {row['ade']:8} {row['path']}   ({row['status']}: "
                  "pass --include-modified / --assume-registry-owned to remove it)")
    if not args.apply:
        print("(dry run - nothing written; re-run with --apply)")
        return 0
    txn = Txn()
    try:
        for row in removable:
            txn.delete(row["path"])
        new = copy.deepcopy(state)
        gone = {r["path"] for r in removable} | {r["path"] for r in rows if r["status"] == "missing"}
        stay = []
        for ade in targets:
            left = [r for r in kept if r["ade"] == ade and r["status"] != "missing"]
            new["installed"][ade]["files"] = {p: h for p, h in new["installed"][ade]["files"].items() if p not in gone}
            if left:
                stay.append(ade)
                print(f"gtt-ade: {ade} was only partly removed; it stays recorded.")
            else:
                new["participating"].remove(ade)
                new["installed"].pop(ade)
                if ade not in new["excluded"]:
                    new["excluded"].append(ade)
        if not new["participating"]:
            if os.path.isfile(STATE):
                txn.delete(STATE)
            print("gtt-ade: no participating ADE left; .gtt/ade.json removed.")
            return 0
        if new["primary"] not in new["participating"]:
            new["primary"] = args.new_primary or sorted(new["participating"])[0]
        new["excluded"] = sorted(set(new["excluded"]))
        return finish(args, txn, new, reg, "gtt-ade: removed.", check=False)
    except OSError as exc:
        txn.rollback()
        die(f"remove failed and was rolled back: {exc}", 1)


def cmd_update(args):
    cat = args.frm or die("--from <catalog root> is required.")
    m_new, reg_new = registry(cat)
    m_cur, reg = registry(".")
    state = load_state() or die("no .gtt/ade.json; run `adopt` first.")
    if gm.layout_version(m_new) != gm.layout_version(m_cur):
        die(f"catalog scaffold layout {gm.layout_version(m_new)} differs from the installed "
            f"{gm.layout_version(m_cur)}: run the core migration first; overlays are never "
            "migrated across a layout change.", 1)
    actions, problems = [], []
    for ade in sorted(state["participating"]):
        if ade not in reg_new:
            problems.append(f"{ade}: no longer in the catalog registry")
            continue
        record = state["installed"][ade]
        files = record["files"]
        if record.get("origin") == "adopted":
            actions.append(("UNRECORDED", ade, "-", "adopted ADE has no install record; overlay not evaluated file by file"))
            continue
        wanted = expand(cat, reg_new[ade]["owned"])
        for rel in wanted:
            src = os.path.join(cat, rel)
            if rel in files:
                if not os.path.isfile(rel):
                    actions.append(("RESTORE", ade, rel, ""))
                elif sha(rel) == files[rel]:
                    if sha(src) != files[rel]:
                        actions.append(("UPDATE", ade, rel, ""))
                elif sha(src) == sha(rel):
                    actions.append(("REFRESH", ade, rel, "already equals the catalog; ledger refreshed"))
                elif sha(src) != files[rel]:
                    actions.append(("CONFLICT", ade, rel, "modified locally and changed in the catalog"))
                else:
                    actions.append(("KEEP", ade, rel, "modified locally; catalog unchanged"))
            elif os.path.exists(rel):
                actions.append(("CONFLICT", ade, rel, "exists but was not installed by GTT"))
            else:
                actions.append(("ADD", ade, rel, ""))
        for rel in sorted(set(files) - set(wanted)):
            if not os.path.isfile(rel):
                actions.append(("DROP", ade, rel, "already gone; ledger entry dropped"))
            elif sha(rel) == files[rel]:
                actions.append(("STALE", ade, rel, "no longer in the catalog; removed"))
            else:
                actions.append(("KEEP", ade, rel, "no longer in the catalog but modified locally"))
    print("GTT ADE update - evaluating every participating overlay against the catalog")
    for kind, ade, rel, note in actions:
        print(f"  {kind:10} {ade:8} {rel}{('   (' + note + ')') if note else ''}")
    for problem in problems:
        print(f"  PROBLEM    {problem}")
    if problems or any(a[0] == "CONFLICT" for a in actions):
        print("gtt-ade: nothing was written; resolve the conflict(s) first.", file=sys.stderr)
        return 1
    if not args.apply:
        print("(dry run - nothing written; re-run with --apply)")
        return 0
    txn = Txn()
    try:
        new = copy.deepcopy(state)
        for kind, ade, rel, _ in actions:
            files = new["installed"][ade]["files"]
            if kind in ("UPDATE", "RESTORE", "ADD"):
                txn.copy(os.path.join(cat, rel), rel)
                files[rel] = sha(rel)
            elif kind == "REFRESH":
                files[rel] = sha(rel)
            elif kind == "STALE":
                txn.delete(rel)
                files.pop(rel)
            elif kind == "DROP":
                files.pop(rel)
        return finish(args, txn, new, reg,
                      "gtt-ade: overlays updated; now run .gtt/scripts/gtt-validate.sh.")
    except OSError as exc:
        txn.rollback()
        die(f"update failed and was rolled back: {exc}", 1)


# --------------------------------------------------------------------- main

def build_parser():
    parser = argparse.ArgumentParser(prog="gtt-ade.sh", description=__doc__.split("\n")[0])
    sub = parser.add_subparsers(dest="command", required=True)

    def command(name, func, help_text, frm=False, json_out=False, apply=False):
        p = sub.add_parser(name, help=help_text)
        p.set_defaults(func=func)
        if frm:
            p.add_argument("--from", dest="frm", metavar="CATALOG",
                           help="root of the GTT Bootstrap catalog (default: this project)")
        if json_out:
            p.add_argument("--json", action="store_true", help="machine-readable output")
        if apply:
            p.add_argument("--apply", action="store_true", help="write (default is a dry run)")
        return p

    command("list", cmd_list, "the ADE integration registry", frm=True, json_out=True)
    command("detect", cmd_detect, "candidate ADEs (observation only)", frm=True, json_out=True)
    command("state", cmd_state, "primary / participating / integration state", json_out=True)
    command("validate", cmd_validate, "every participating integration is intact", json_out=True)
    command("owned", cmd_owned, "the GTT-installed ADE surfaces (clean / export --clean)", json_out=True)
    p = command("install", cmd_install, "copy the chosen overlays and record them", frm=True, apply=True)
    p.add_argument("--participating", required=True, help="comma-separated ADE ids the human chose")
    p.add_argument("--primary", required=True, help="the one Primary ADE (a participating one)")
    p.add_argument("--reinclude", help="explicitly re-admit an ADE an earlier run excluded")
    p = command("adopt", cmd_adopt, "record an already-installed ADE integration", apply=True)
    p.add_argument("--participating")
    p.add_argument("--primary")
    p = command("set-primary", cmd_set_primary, "change the Primary ADE", apply=True)
    p.add_argument("ade")
    p = command("record", cmd_record, "add files to an ADE's install ledger", apply=True)
    p.add_argument("ade")
    p.add_argument("paths", nargs="+")
    p = command("remove", cmd_remove, "remove GTT-installed integration of ADE(s)", apply=True)
    p.add_argument("ades", nargs="?", default="")
    p.add_argument("--all", action="store_true")
    p.add_argument("--new-primary")
    p.add_argument("--include-modified", action="store_true")
    p.add_argument("--assume-registry-owned", action="store_true")
    p = command("update", cmd_update, "evaluate and migrate every participating overlay", frm=True, apply=True)
    return parser


def main(argv):
    if not os.path.isdir(".gtt") and not (len(argv) > 1 and "--from" in argv):
        die("run from the project root (no .gtt/ directory here)")
    args = build_parser().parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
