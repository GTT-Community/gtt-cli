#!/usr/bin/env python3
"""GTT - Session Memory adapter conformance check (engine).

Checks one ADE adapter against .gtt/docs/session-adapter-contract.md using its
declaration, .gtt/session-adapters/<ade>.json. Everything ADE-specific is DATA
in that declaration; this engine has no per-ADE code path, and it never runs
an ADE - so it is deterministic and works with no ADE installed.

Each item prints PASS, FAIL, or SKIPPED. SKIPPED is never counted as PASS.
Whether an ADE actually loads the context is not checkable here; that is the
declared verification.runtime, and it is reported as SKIPPED.

Exit 0 = no FAIL, 1 = at least one FAIL, 2 = cannot run (usage, no manifest).
"""

import json
import os
import re
import shutil
import subprocess
import sys
import tempfile

SERVICE = "gtt-session-context.sh"
MARKERS = ["operational-only", "NOT authority", "NOT evidence",
           "NOT a decision record", "NOT a grounding source"]
EVENTS = ["startup", "resume", "clear", "compact", "fork"]
EVENT_VALUES = {"verified", "documented", "unsupported", "unknown"}
RUNTIME_VALUES = {"runtime-verified", "not-verified",
                  "requires-interactive-session", "not-installed"}
COVERAGE = {"N1", "N2", "N3"}
OUTPUTS = {"text", "json-hookSpecificOutput", "json-additionalContext"}

# GTT logic that belongs to the service, never to an adapter.
DUPLICATED_LOGIC = [
    (r"gtt-status\.sh", "runs gtt-status.sh itself"),
    (r"gtt_artifacts", "calls the identity/index engine"),
    (r"technical-index|artifacts\.json", "reads the index/manifest"),
    (r"gtt-index\.sh|gtt-reconcile\.sh", "runs index/reconcile tooling"),
    (r"registry\.yaml", "reads the protection registry"),
    (r"\bgit\s+(log|status|rev-parse|diff|branch)\b", "interprets repository state via git"),
    (r"(?i)session\.md", "touches gtt-domain/session.md directly (only an N2 adapter may name it)"),
]
SILENCERS = [
    (r"\|\|\s*true\b", "'|| true' swallows failure"),
    (r"2>\s*/dev/null", "'2>/dev/null' hides errors"),
    (r">\s*/dev/null\s+2>&1", "'>/dev/null 2>&1' hides errors"),
    (r"^\s*except\s*:", "bare 'except:'"),
    (r"except\s+Exception\s*:\s*\n\s*(pass|sys\.exit\(0\)|return)\b",
     "'except Exception' that swallows"),
    (r"set\s+\+e\b", "'set +e' disables error stopping"),
]

results = []  # (status, message)


def item(status, message):
    results.append((status, message))
    print(f"{status:8} {message}")


def read(path):
    with open(path, "r", encoding="utf-8", errors="replace") as fh:
        return fh.read()


def load_manifest(ade):
    path = f".gtt/session-adapters/{ade}.json"
    if not os.path.isfile(path):
        print(f"gtt-check-session-adapter: no declaration {path}", file=sys.stderr)
        sys.exit(2)
    try:
        return path, json.loads(read(path))
    except ValueError as exc:
        print(f"gtt-check-session-adapter: {path} is not valid JSON: {exc}", file=sys.stderr)
        sys.exit(2)


def walk_strings(node, trail=()):
    """Yield (ancestor_keys, string) for every string in a JSON tree."""
    if isinstance(node, dict):
        for k, v in node.items():
            yield from walk_strings(v, trail + (k,))
    elif isinstance(node, list):
        for v in node:
            yield from walk_strings(v, trail)
    elif isinstance(node, str):
        yield trail, node


SCHEMA_KEYS = {
    "": {"schema", "ade", "display_name", "adapter_status", "coverage", "files",
         "registration", "command", "output", "events", "verification",
         "ade_binary", "limitations"},
    "files[]": {"role", "install_path", "staged_path"},
    "registration": {"file", "event", "token"},
    "verification": {"static", "runtime", "notes"},
}


def unknown_keys(m):
    """A declaration is data, never implementation: any key outside the schema
    is a place logic could hide, so it is rejected."""
    found = [k for k in m if k not in SCHEMA_KEYS[""]]
    for f in m.get("files", []):
        found += [f"files[].{k}" for k in f if k not in SCHEMA_KEYS["files[]"]]
    for section in ("registration", "verification"):
        found += [f"{section}.{k}" for k in m.get(section, {}) if k not in SCHEMA_KEYS[section]]
    return found


def validate_manifest(m):
    ok = True
    problems = []
    extra = unknown_keys(m)
    if extra:
        problems.append(f"keys outside the declaration schema (data only): {extra}")
    for key in ("ade", "adapter_status", "coverage", "files", "registration",
                "command", "output", "events", "verification"):
        if key not in m:
            problems.append(f"missing '{key}'")
    if m.get("adapter_status") not in ("installed", "staged"):
        problems.append("adapter_status must be installed|staged")
    if m.get("coverage") not in COVERAGE:
        problems.append("coverage must be N1|N2|N3")
    if m.get("output") not in OUTPUTS:
        problems.append(f"output must be one of {sorted(OUTPUTS)}")
    events = m.get("events", {})
    for name in EVENTS:
        if name not in events:
            problems.append(f"event '{name}' not declared")
        elif events[name] not in EVENT_VALUES:
            problems.append(f"event '{name}' must be one of {sorted(EVENT_VALUES)}")
    for name in events:
        if name not in EVENTS:
            problems.append(f"unknown event '{name}'")
    runtime = m.get("verification", {}).get("runtime")
    if runtime not in RUNTIME_VALUES:
        problems.append(f"verification.runtime must be one of {sorted(RUNTIME_VALUES)}")
    if not m.get("verification", {}).get("static"):
        problems.append("verification.static missing")
    if not isinstance(m.get("limitations"), list):
        problems.append("limitations[] missing")
    if problems:
        ok = False
        item("FAIL", "declaration: " + "; ".join(problems))
    else:
        item("PASS", f"declaration well-formed (coverage {m['coverage']}, "
                     f"{m['adapter_status']}, runtime '{runtime}')")
    return ok


def effective(m, f):
    return f["install_path"] if m["adapter_status"] == "installed" else f.get("staged_path")


def check_files(m):
    missing = []
    for f in m["files"]:
        p = effective(m, f)
        if not p or not os.path.isfile(p):
            missing.append(p or f"(no staged_path for {f['install_path']})")
    if missing:
        item("FAIL", f"adapter files missing ({m['adapter_status']}): {', '.join(missing)}")
        return False
    where = "native paths" if m["adapter_status"] == "installed" else "staged under gtt-domain/proposals/"
    item("PASS", f"adapter files present ({len(m['files'])}, {where})")
    return True


def check_registration(m):
    reg = m["registration"]
    cfg = next((f for f in m["files"] if f["install_path"] == reg["file"]), None)
    if cfg is None:
        item("FAIL", f"registration file {reg['file']} is not listed in files[]")
        return
    try:
        tree = json.loads(read(effective(m, cfg)))
    except ValueError as exc:
        item("FAIL", f"registration file is not valid JSON: {exc}")
        return
    hits = [(trail, s) for trail, s in walk_strings(tree) if reg["token"] in s]
    if not hits:
        item("FAIL", f"{reg['file']} never runs '{reg['token']}'")
        return
    wrong = [t for t, _ in hits if reg["event"] not in t]
    if wrong:
        item("FAIL", f"adapter is registered under an event other than '{reg['event']}': "
                     f"{'/'.join(map(str, wrong[0]))}")
        return
    if not any(m["command"] in s for _, s in hits):
        item("FAIL", f"registered command does not contain the declared command '{m['command']}'")
        return
    item("PASS", f"registered only under '{reg['event']}' in {reg['file']}")


def check_logic_and_silence(m):
    scripts = [effective(m, f) for f in m["files"] if f["role"] == "script"]
    every = [effective(m, f) for f in m["files"]]
    if not any(re.search(re.escape(SERVICE), read(p)) for p in every):
        item("FAIL", f"no adapter file references {SERVICE}")
    else:
        item("PASS", f"adapter references {SERVICE}")

    dup = []
    for p in every:
        text = read(p)
        for pattern, why in DUPLICATED_LOGIC:
            if pattern == r"(?i)session\.md" and m["coverage"] == "N2":
                continue
            if re.search(pattern, text):
                dup.append(f"{p}: {why}")
    item("FAIL" if dup else "PASS",
         "duplicated GTT logic: " + "; ".join(dup) if dup else "no duplicated GTT logic")

    bad = []
    for p in scripts:
        text = read(p)
        for pattern, why in SILENCERS:
            if re.search(pattern, text, re.M):
                bad.append(f"{p}: {why}")
        if re.search(r"\bexit\s+2\b|sys\.exit\(2\)", text):
            bad.append(f"{p}: exits 2 (blocks session start in some ADEs)")
        if not re.search(r"\bexit\s+1\b|sys\.exit\(1\)", text):
            bad.append(f"{p}: has no explicit non-zero failure exit")
    item("FAIL" if bad else "PASS",
         "error handling: " + "; ".join(bad) if bad else "errors are surfaced, not silenced (exit 1, never 2)")


def extract_context(m, stdout):
    kind = m["output"]
    if kind == "text":
        return stdout
    try:
        data = json.loads(stdout)
        if kind == "json-hookSpecificOutput":
            return data["hookSpecificOutput"]["additionalContext"]
        return data["additionalContext"]
    except (ValueError, KeyError, TypeError):
        return None


def build_sandbox(m):
    """A tree laid out as an installed project: .gtt/ and gtt-domain/ plus the adapter files at
    their install paths. Lets a staged adapter be exercised as installed."""
    root = tempfile.mkdtemp(prefix="gtt-adapter-")
    shutil.copytree(".gtt", os.path.join(root, ".gtt"),
                    ignore=shutil.ignore_patterns("__pycache__"))
    # The governed domain sits at gtt-domain/; its staged drafts (gtt-domain/proposals/)
    # are deliberately left out (adapters run from install paths).
    if os.path.isdir("gtt-domain"):
        shutil.copytree("gtt-domain", os.path.join(root, "gtt-domain"),
                        ignore=shutil.ignore_patterns("__pycache__", "proposals"))
    for f in m["files"]:
        dest = os.path.join(root, f["install_path"])
        os.makedirs(os.path.dirname(dest), exist_ok=True)
        shutil.copy2(effective(m, f), dest)
    return root


def run(command, cwd):
    return subprocess.run(["bash", "-c", command], cwd=cwd, stdin=subprocess.DEVNULL,
                          capture_output=True, text=True, encoding="utf-8",
                          errors="replace", timeout=90)


def check_execution(m):
    if shutil.which("bash") is None:
        item("SKIPPED", "command execution: bash not found on PATH")
        return
    root = build_sandbox(m)
    try:
        try:
            proc = run(m["command"], root)
        except subprocess.TimeoutExpired:
            item("FAIL", "command timed out after 90s")
            return
        if proc.returncode != 0:
            item("FAIL", f"command exited {proc.returncode}: "
                         f"{(proc.stderr or proc.stdout).strip()[:200]}")
            return
        item("PASS", f"command runs (exit 0): {m['command']}")
        context = extract_context(m, proc.stdout)
        if context is None:
            item("FAIL", f"output is not the declared format '{m['output']}'")
        else:
            missing = [x for x in MARKERS if x not in context]
            item("FAIL" if missing else "PASS",
                 f"authority markers missing from delivered context: {missing}" if missing
                 else f"delivered context ({len(context)} chars, format '{m['output']}') "
                      "carries every non-authority marker")

        # Failure path: break the service; the adapter must fail visibly.
        os.remove(os.path.join(root, ".gtt", "scripts", "gtt_artifacts.py"))
        bad = run(m["command"], root)
        problems = []
        if bad.returncode == 0:
            problems.append("exit 0 despite a broken service (failure swallowed)")
        if bad.returncode == 2:
            problems.append("exit 2 (blocks session start in some ADEs)")
        if not bad.stderr.strip():
            problems.append("nothing on stderr")
        if any(x in bad.stdout for x in MARKERS):
            problems.append("emitted context despite the failure")
        item("FAIL" if problems else "PASS",
             "failure path: " + "; ".join(problems) if problems
             else f"failure path: exit {bad.returncode}, message on stderr, no context emitted")
    finally:
        shutil.rmtree(root, ignore_errors=True)


def check_claims(m):
    runtime = m["verification"]["runtime"]
    verified = [e for e, v in m["events"].items() if v == "verified"]
    if verified and runtime != "runtime-verified":
        item("FAIL", f"events {verified} claim 'verified' but verification.runtime is '{runtime}'")
    else:
        item("PASS", "no event claims more than the declared runtime verification")
    if m["events"].get("startup") not in ("verified", "documented") and m["coverage"] == "N1":
        item("FAIL", "coverage N1 requires a start event that is at least documented")
    if m["coverage"] == "N2" and "gtt-domain/session.md" not in json.dumps(m):
        item("FAIL", "coverage N2 must name the loaded session file")


def report_runtime(m):
    binary = m.get("ade_binary")
    present = shutil.which(binary) is not None if binary else False
    item("SKIPPED",
         f"runtime verification: declared '{m['verification']['runtime']}'; "
         f"ADE binary '{binary}' {'present' if present else 'not installed'} here; "
         "this check never runs the ADE")


def main(argv):
    if len(argv) != 2:
        print("usage: gtt-check-session-adapter.sh <ade>", file=sys.stderr)
        return 2
    if not os.path.isdir(".gtt"):
        print("gtt-check-session-adapter: run from the project root", file=sys.stderr)
        return 2
    ade = argv[1]
    _, m = load_manifest(ade)
    print(f"GTT session adapter check: {ade}")
    if not validate_manifest(m):
        print(f"SUMMARY ade={ade} adapter=? coverage=? static=FAIL runtime-declared=?")
        return 1
    if check_files(m):
        check_registration(m)
        check_logic_and_silence(m)
        check_execution(m)
    check_claims(m)
    report_runtime(m)
    failed = any(s == "FAIL" for s, _ in results)
    print(f"SUMMARY ade={ade} adapter={m['adapter_status']} coverage={m['coverage']} "
          f"static={'FAIL' if failed else 'PASS'} runtime-declared={m['verification']['runtime']}")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
