#!/usr/bin/env python3
"""GTT - Bootstrap 1.0 contract engine.

The CLI is told WHAT the Bootstrap offers and HOW to invoke it; it never embeds GTT methodology.
This engine reads the contracts in .gtt/contract/ (data only) and provides:

  release                 the Bootstrap release identity, distinct from the scaffold version
  negotiate               compatibility decision for a CLI: PASS, or REFUSE with reasons (no side effects)
  capabilities            the capability registry
  operations              the operation registry
  show <contract>         profiles | export-policy | recovery | elicitation | ade-registry | initial-design
  run <op> [name=value]   execute ONE declared operation: fixed argv, typed arguments, no shell,
                          implementation confined to .gtt/scripts/; operations with human authority
                          need confirmed_by_human=true; mutating operations are dry runs without apply=true
  check                   integrity of the contracts themselves (used by gtt-check-contract.sh)

Exit codes: 0 ok, 1 violation, 2 usage / cannot run, 3 REFUSED (incompatible), 4 refused: human authority
required, 5 refused: invalid arguments or undeclared operation.
"""

import argparse
import json
import os
import re
import subprocess
import sys

import gtt_manifest as gm

CONTRACT_DIR = ".gtt/contract"
FILES = {"release": "release.json", "capabilities": "capabilities.json", "operations": "operations.json",
         "profiles": "profiles.json", "export-policy": "export-policy.json", "recovery": "recovery.json",
         "elicitation": "elicitation.json"}
ID_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9_.-]*$")
CONTROL = re.compile(r"[\x00-\x1f]")
SHELL_META = re.compile(r"[;&|`$<>\n]")
EXIT_REFUSED, EXIT_HUMAN, EXIT_ARGS = 3, 4, 5


def die(message, code=2):
    print(f"gtt-contract: {message}", file=sys.stderr)
    sys.exit(code)


def load(name):
    path = os.path.join(CONTRACT_DIR, FILES[name])
    try:
        with open(path, encoding="utf-8") as handle:
            return json.load(handle)
    except OSError:
        die(f"contract file missing: {path}")
    except ValueError as exc:
        die(f"{path} is not valid JSON: {exc}")


def emit(obj):
    print(json.dumps(obj, indent=2, sort_keys=True))


def semver(text):
    match = re.fullmatch(r"(\d+)\.(\d+)\.(\d+)", str(text or ""))
    return tuple(int(x) for x in match.groups()) if match else None


# ---------------------------------------------------------------- negotiate

def negotiate_result(release, cli_version, cli_capabilities, cli_schemas):
    """Fail closed: anything missing or unknown is a reason to refuse."""
    reasons = []
    compat = release["compatibility"]
    version = semver(cli_version)
    if version is None:
        reasons.append("CLI version missing or not MAJOR.MINOR.PATCH")
    else:
        low, high = semver(compat["cli"]["min_version"]), semver(compat["cli"].get("max_version"))
        if low and version < low:
            reasons.append(f"CLI {cli_version} is older than the minimum {compat['cli']['min_version']}")
        if compat["cli"].get("max_version") and high is not None and version > high:
            reasons.append(f"CLI {cli_version} is newer than the maximum {compat['cli']['max_version']}")
    wanted_schema = compat["schema"]["version"]
    if cli_schemas is None:
        reasons.append("CLI did not declare which Bootstrap schema versions it supports")
    elif wanted_schema not in cli_schemas:
        reasons.append(f"Bootstrap schema {wanted_schema} is not supported by the CLI (supports {sorted(cli_schemas)})")
    have = set(cli_capabilities or [])
    for need in release.get("requires_cli_capabilities", []):
        if need not in have:
            reasons.append(f"Bootstrap requires the CLI capability `{need}`, which the CLI does not declare")
    return {"schema": 1, "compatible": not reasons, "bootstrap": release["bootstrap"],
            "cli": {"version": cli_version, "capabilities": sorted(have), "schemas": sorted(cli_schemas or [])},
            "reasons": reasons,
            "on_refusal": "the CLI must not modify the project; this command never does"}


def cmd_release(args):
    release = load("release")
    manifest = gm.load()
    out = dict(release, scaffold=dict(release["scaffold"], version=gm.layout_version(manifest)))
    if args.json:
        emit(out)
    else:
        b = release["bootstrap"]
        print(f"{b['id']} {b['version']} (schema {b['schema_version']}, channel {b['channel']}; "
              f"scaffold layout {gm.layout_version(manifest)}; canon {release['canon']})")
    return 0


def cmd_negotiate(args):
    caps = [c for c in (args.cli_capabilities or "").split(",") if c] if args.cli_capabilities is not None else None
    schemas = [int(s) for s in args.cli_schemas.split(",") if s.strip().isdigit()] if args.cli_schemas is not None else None
    result = negotiate_result(load("release"), args.cli_version, caps, schemas)
    if args.json:
        emit(result)
    else:
        print("COMPATIBLE" if result["compatible"] else "REFUSE")
        for reason in result["reasons"]:
            print("  - " + reason)
    return 0 if result["compatible"] else EXIT_REFUSED


# ----------------------------------------------------------------- registries

def cmd_capabilities(args):
    data = load("capabilities")
    if args.json:
        emit(data)
    else:
        for cap in data["capabilities"]:
            print(f"{cap['id']:24} v{cap['version']}  ops: {', '.join(cap['operations'])}")
    return 0


def cmd_operations(args):
    data = load("operations")
    if args.json:
        emit(data)
    else:
        for op_id, op in sorted(data["operations"].items()):
            flags = ("mutates " if op["mutates"] else "") + ("human-authority" if op["human_authority"] else "")
            print(f"{op_id:26} v{op['version']}  {op['implementation']}  {flags}".rstrip())
    return 0


def ade_registry():
    manifest = gm.load()
    rows = []
    for ade, entry in sorted(gm.overlays(manifest).items()):
        decl = {}
        try:
            with open(os.path.join(".gtt", "session-adapters", ade + ".json"), encoding="utf-8") as handle:
                decl = json.load(handle)
        except (OSError, ValueError):
            pass
        rows.append({
            "id": ade, "name": entry["name"], "version": entry["version"],
            "detect": {"paths": entry["detect"], "binary": decl.get("ade_binary")},
            "install": {"operation": "ade.install", "owned_paths": entry["owned"]},
            "validate": {"operation": "ade.validate"},
            "owned_paths": entry["owned"],
            "handoff": {"instruction_entry": entry["entry"]},
            "invoke": {"binary": decl.get("ade_binary")},
            "enforcement": entry["enforcement"],
        })
    return {"schema": 1, "contract_version": load("release")["contracts"]["ade_integration"],
            "participation_states": ["detected", "participating", "primary", "excluded"],
            "rules": "zero or more participating secondary ADEs and exactly one Primary ADE; the CLI collects the "
                     "human choice, the Bootstrap validates the state (ade.validate)",
            "ades": rows}


def initial_design():
    manifest = gm.load()
    tpl = gm.templates(manifest).get("initial-design-questionnaire")
    if not tpl:
        die("the manifest declares no initial-design-questionnaire template", 1)
    return {"schema": 1, "scaffold": {"initial_design": {"questionnaire": {
        "template": tpl["path"], "output": tpl["materialize_to"], "version": tpl.get("version"),
        "contract_version": tpl.get("contract_version"), "becomes": tpl.get("becomes"),
        "operation": "template.materialize", "handoff": load("elicitation")["handoff"]}}}}


def cmd_show(args):
    kind = args.kind
    if kind == "profiles":
        data = load("profiles")
    elif kind == "export-policy":
        data = load("export-policy")
    elif kind == "recovery":
        data = load("recovery")
    elif kind == "elicitation":
        data = load("elicitation")
    elif kind == "ade-registry":
        data = ade_registry()
    elif kind == "initial-design":
        data = initial_design()
    else:
        die(f"unknown contract `{kind}`")
    emit(data)
    return 0


# ------------------------------------------------------------------------ run

def validate_value(spec, value):
    """The argv fragment (list) for one argument value, or raise ValueError."""
    kind = spec["type"]
    if kind == "bool":
        if value not in ("true", "false"):
            raise ValueError("must be true or false")
        return value == "true"
    if CONTROL.search(value):
        raise ValueError("control characters are not allowed")
    if kind == "enum":
        if value not in spec["values"]:
            raise ValueError(f"must be one of {spec['values']}")
        return [value]
    if kind in ("id", "list"):
        parts = [value] if kind == "id" else [p for p in value.split(",") if p]
        if not parts:
            raise ValueError("empty")
        for part in parts:
            if not ID_RE.match(part):
                raise ValueError(f"`{part}` is not a valid identifier")
        return [value]
    if kind in ("string", "path"):
        if not value or value.startswith("-"):
            raise ValueError("must be non-empty and must not start with '-'")
        return [value]
    if kind == "path-list":
        parts = [p for p in value.split(",") if p]
        if not parts or any(p.startswith("-") for p in parts):
            raise ValueError("must be non-empty paths not starting with '-'")
        return parts
    raise ValueError(f"unknown argument type {kind}")


def build_argv(op, supplied):
    specs = op["args"]
    known = {s["name"] for s in specs}
    unknown = sorted(set(supplied) - known - {"confirmed_by_human"})
    if unknown:
        raise ValueError("undeclared argument(s): " + ", ".join(unknown))
    argv, positional = [], []
    for spec in specs:
        name = spec["name"]
        if name not in supplied:
            if spec.get("required"):
                raise ValueError(f"missing required argument `{name}`")
            continue
        try:
            value = validate_value(spec, supplied[name])
        except ValueError as exc:
            raise ValueError(f"argument `{name}`: {exc}")
        if spec["type"] == "bool":
            if value:
                argv.append(spec["flag"])
        elif spec.get("positional"):
            positional.append(value)
        else:
            argv += [spec["flag"], value[0]]
    for group in positional:
        argv += group
    return list(op["argv"]) + argv


def implementation_path(impl):
    norm = os.path.normpath(impl).replace("\\", "/")
    if not norm.startswith(".gtt/scripts/") or ".." in norm.split("/"):
        return None
    return norm if os.path.isfile(norm) else None


def cmd_run(args):
    ops = load("operations")["operations"]
    if args.operation not in ops:
        print(f"gtt-contract: operation `{args.operation}` is not declared by this Bootstrap; nothing was run.", file=sys.stderr)
        return EXIT_ARGS
    op = ops[args.operation]
    supplied = {}
    for item in args.arguments:
        if "=" not in item:
            print(f"gtt-contract: argument `{item}` must be name=value", file=sys.stderr)
            return EXIT_ARGS
        name, value = item.split("=", 1)
        supplied[name] = value
    if op["human_authority"] and supplied.get("confirmed_by_human") != "true":
        print(f"gtt-contract: `{args.operation}` needs a human decision; pass confirmed_by_human=true only after the human "
              "confirmed it. Nothing was run.", file=sys.stderr)
        return EXIT_HUMAN
    try:
        argv = build_argv(op, supplied)
    except ValueError as exc:
        print(f"gtt-contract: invalid arguments for `{args.operation}`: {exc}. Nothing was run.", file=sys.stderr)
        return EXIT_ARGS
    impl = implementation_path(op["implementation"])
    if impl is None:
        print(f"gtt-contract: `{op['implementation']}` is not a trusted implementation (must exist under .gtt/scripts/). "
              "Nothing was run.", file=sys.stderr)
        return EXIT_ARGS
    command = ["bash", impl] + argv
    proc = subprocess.run(command, capture_output=True, text=True, encoding="utf-8", errors="replace", shell=False)
    if args.envelope:
        emit({"operation": args.operation, "version": op["version"], "exit_code": proc.returncode,
              "mutates": op["mutates"], "applied": supplied.get("apply") == "true" if op["mutates"] else False,
              "stdout": proc.stdout, "stderr": proc.stderr, "outputs": op["outputs"]})
    else:
        sys.stdout.write(proc.stdout)
        sys.stderr.write(proc.stderr)
    return proc.returncode


# ----------------------------------------------------------------------- check

def check_contracts():
    problems = []
    add = problems.append
    data = {}
    for name in FILES:
        path = os.path.join(CONTRACT_DIR, FILES[name])
        try:
            with open(path, encoding="utf-8") as handle:
                data[name] = json.load(handle)
        except (OSError, ValueError) as exc:
            add(f"{path}: {exc}")
    if problems:
        return problems, data
    release, caps, ops = data["release"], data["capabilities"], data["operations"]
    prof, exp, rec, eli = data["profiles"], data["export-policy"], data["recovery"], data["elicitation"]
    b = release.get("bootstrap", {})
    for key in ("id", "version", "schema_version", "channel"):
        if key not in b:
            add(f"release.bootstrap.{key} missing")
    if semver(b.get("version")) is None:
        add("release.bootstrap.version must be MAJOR.MINOR.PATCH")
    if semver(release["compatibility"]["cli"].get("min_version")) is None:
        add("release.compatibility.cli.min_version must be MAJOR.MINOR.PATCH")
    if release["compatibility"]["schema"]["version"] != b.get("schema_version"):
        add("release.compatibility.schema.version differs from bootstrap.schema_version")
    try:
        gm.layout_version(gm.load())          # the scaffold layout version lives ONLY in the manifest
    except gm.ManifestError as exc:
        add(f"manifest: {exc}")
    if "version" in release.get("scaffold", {}):
        add("release.scaffold must not carry a version: the scaffold layout version is read from the manifest only")
    for contract, version in release.get("contracts", {}).items():
        if not isinstance(version, int) or version < 1:
            add(f"release.contracts.{contract} must be a positive integer")
    for required in ("release", "compatibility", "capability", "operation", "methodology_profile", "questionnaire",
                     "recovery", "export_policy", "ade_integration"):
        if required not in release.get("contracts", {}):
            add(f"release.contracts.{required} missing (every externally consumed contract is versioned)")
    for name, doc in (("capabilities", caps), ("operations", ops), ("profiles", prof), ("export-policy", exp),
                      ("recovery", rec), ("elicitation", eli)):
        if doc.get("schema") != 1 or not isinstance(doc.get("contract_version"), int):
            add(f"{FILES[name]}: needs schema 1 and an integer contract_version")
    # operations
    operations = ops["operations"]
    for op_id, op in operations.items():
        if "unfreeze" in op_id.lower() or "thaw" in op_id.lower():
            add(f"operation `{op_id}`: there must be no unfreeze operation (freeze is a human ratification)")
        if not isinstance(op.get("version"), int):
            add(f"operation `{op_id}`: integer version required")
        if implementation_path(op.get("implementation", "")) is None:
            add(f"operation `{op_id}`: implementation must be an existing file under .gtt/scripts/")
        for token in op.get("argv", []):
            if SHELL_META.search(token) or token.startswith("sh -c"):
                add(f"operation `{op_id}`: argv token `{token}` looks like shell syntax (no arbitrary shell)")
        if op.get("implementation") in ("bash", "sh", "/bin/sh", "/bin/bash"):
            add(f"operation `{op_id}`: an interpreter is not a trusted implementation")
        names = set()
        for spec in op.get("args", []):
            if spec.get("name") in names:
                add(f"operation `{op_id}`: duplicate argument `{spec.get('name')}`")
            names.add(spec.get("name"))
            if spec.get("type") not in ("id", "string", "path", "path-list", "list", "enum", "bool"):
                add(f"operation `{op_id}`: argument `{spec.get('name')}` has an unknown type")
            if spec.get("type") == "enum" and not spec.get("values"):
                add(f"operation `{op_id}`: enum argument `{spec.get('name')}` needs values")
            if not spec.get("positional") and not spec.get("flag"):
                add(f"operation `{op_id}`: argument `{spec.get('name')}` needs a flag or must be positional")
        if op.get("mutates") and op_id not in ("freeze", "guard.sync", "index", "maintain") \
                and not any(s.get("name") == "apply" for s in op.get("args", [])) and op_id != "ade.remove":
            add(f"operation `{op_id}`: a mutating operation must offer `apply` (dry run by default)")
    if not operations.get("freeze", {}).get("human_authority"):
        add("operation `freeze` must declare human_authority")
    # capabilities
    seen = set()
    for cap in caps["capabilities"]:
        if cap["id"] in seen:
            add(f"capability `{cap['id']}` declared twice")
        seen.add(cap["id"])
        for op_id in cap["operations"]:
            if op_id not in operations:
                add(f"capability `{cap['id']}` names an undeclared operation `{op_id}`")
    for required in ("project.detect", "ade.detect", "ade.install", "template.materialize", "methodology.profile",
                     "validation", "freeze", "session-context", "export-policy", "recovery", "developer-experience"):
        if required not in seen:
            add(f"required capability `{required}` is not registered")
    # profiles
    supported = [p["id"] for p in prof["supported"]]
    if sorted(supported) != ["hard", "light", "medium", "team"]:
        add("profiles.supported must be exactly light, medium, hard and team")
    if prof.get("default") not in supported:
        add("profiles.default must be a supported profile")
    if sorted(prof.get("strictness_order", [])) != sorted(supported):
        add("profiles.strictness_order must order every supported profile exactly once")
    selection = prof.get("selection", {})
    if selection.get("policy") != "ask":
        add("profiles.selection.policy must be `ask`: the plan is the human's choice and is never inferred")
    if selection.get("unselected", {}).get("gates_from") != prof.get("default"):
        add("profiles.selection.unselected.gates_from must equal profiles.default (one fallback, stated once)")
    values = prof.get("policy_values", {})
    for term in ("deterministic_operation", "relevant_change", "destructive_operation", "governed_decision"):
        if not prof.get("definitions", {}).get(term):
            add(f"profiles.definitions.{term} missing (a plan must not use an undefined term)")
    for pid in supported:
        p = prof["profiles"].get(pid)
        if not p:
            add(f"profile `{pid}` has no definition")
            continue
        for key in ("proposal_rigor", "confirmation_requirements", "documentation_depth", "adr_expectations", "change_process",
                    "validation_gates", "freeze_expectations", "audit_requirements", "promotion_rigor", "evidence_requirements"):
            if not p["semantics"].get(key):
                add(f"profile `{pid}`: semantics.{key} missing")
        for gate in ("provenance_policy", "warnings_block_freeze", "sources_manifest_required_for_freeze", "open_gap_requires_affects"):
            if gate not in p["gates"]:
                add(f"profile `{pid}`: gates.{gate} missing")
        plan = p.get("plan", {})
        for key in ("label", "summary", "delegates", "policy_enforcement"):
            if not plan.get(key):
                add(f"profile `{pid}`: plan.{key} missing (every plan states plainly what it is)")
        policy = plan.get("policy", {})
        for group, keys in (("automation", ("identity_resolution", "reference_updates", "index_rebuild", "validation",
                                            "agent_context_sync")),
                            ("collaboration", ("ci", "multi_user", "traceability"))):
            for key in keys:
                value = policy.get(group, {}).get(key)
                if value not in values:
                    add(f"profile `{pid}`: plan.policy.{group}.{key} must be a value defined in profiles.policy_values")
        confirmation = policy.get("human", {}).get("confirmation", {})
        for key in ("governed_decision", "destructive"):
            if confirmation.get(key) != "required":
                add(f"profile `{pid}`: human confirmation of `{key}` must be required (no plan delegates it)")
        if confirmation.get("relevant_change") not in values:
            add(f"profile `{pid}`: plan.policy.human.confirmation.relevant_change must be a value defined in profiles.policy_values")
        invariant_ids = {i["id"] for i in prof["invariants"]}
        for relax in p.get("relaxes", []):
            if relax["control"] in invariant_ids:
                add(f"profile `{pid}` relaxes the invariant `{relax['control']}`")
    dx = prof.get("developer_experience", {})
    for flag in ("minimize_interruption", "automatic_deterministic_operations", "concise_reports", "details_on_demand"):
        if dx.get(flag) is not True:
            add(f"profiles.developer_experience.{flag} must be true")
    if len(dx.get("stop_conditions", [])) < 1 or not dx.get("decision_rule", {}).get("resolve_from"):
        add("profiles.developer_experience needs its decision rule and its stop conditions")
    if dx.get("protected_operations", {}).get("agent_executes") is not False:
        add("profiles.developer_experience.protected_operations.agent_executes must be false (lower friction never lets "
            "an agent run a protected operation)")
    for auto in dx.get("automatic_operations", []):
        target = operations.get(auto.get("operation"))
        if target is None:
            add(f"developer_experience names an undeclared operation `{auto.get('operation')}`")
        elif target.get("human_authority"):
            add(f"developer_experience makes `{auto['operation']}` automatic, but it needs human authority")
    for inv in prof["invariants"]:
        if inv.get("relaxable"):
            add(f"invariant `{inv['id']}` must not be relaxable")
    for pid in ("medium", "hard", "team"):
        if prof["profiles"].get(pid, {}).get("relaxes"):
            add(f"profile `{pid}` must not relax anything (only Light may)")
    hard_gates, team_gates = (prof["profiles"].get(pid, {}).get("gates", {}) for pid in ("hard", "team"))
    if team_gates != hard_gates:
        add("profile `team`: gates must equal the Hard gates (Team is Hard plus collaboration requirements)")
    # export policy / recovery / elicitation
    if not exp["ownership"]["static"]:
        add("export-policy.ownership.static is empty")
    if not any(d.get("source") == "ade.owned" for d in exp["ownership"]["dynamic"]):
        add("export-policy must include the dynamic ade.owned source (ADE overlays are ledger-owned)")
    for key in rec["snapshot"]["required"]:
        if key not in rec["snapshot"]["fields"] and key not in ("schema", "contract_version", "kind"):
            add(f"recovery.snapshot.fields lacks `{key}`")
    template = None
    try:
        template = gm.templates(gm.load()).get("initial-design-questionnaire")
    except gm.ManifestError as exc:
        add(f"manifest: {exc}")
    if template is None:
        add("the manifest declares no initial-design-questionnaire template")
    else:
        for key in ("version", "contract_version", "materialize_to"):
            if template.get(key) in (None, ""):
                add(f"initial-design-questionnaire template lacks `{key}`")
        try:
            with open(template["path"], encoding="utf-8") as handle:
                headings = handle.read()
        except OSError:
            headings = ""
            add(f"questionnaire template file missing: {template['path']}")
        for directive in eli["directives"]:
            if headings and directive["questionnaire"] not in headings:
                add(f"elicitation directive `{directive['id']}` cites a questionnaire heading that does not exist: "
                    f"{directive['questionnaire']}")
        if headings and eli["minimum_viable_design"]["section"] not in headings:
            add("elicitation.minimum_viable_design.section does not exist in the questionnaire")
    # ade registry
    try:
        for ade in gm.overlays(gm.load()).values():
            if not isinstance(ade["version"], int):
                add(f"ADE `{ade['id']}`: integration version must be an integer")
    except gm.ManifestError as exc:
        add(f"manifest: {exc}")
    return problems, data


def cmd_check(args):
    problems, _ = check_contracts()
    if args.json:
        emit({"schema": 1, "ok": not problems, "problems": problems})
    else:
        for problem in problems:
            print("FAIL  " + problem)
        print("gtt-check-contract: " + ("OK - every contract is consistent, versioned and fails closed." if not problems
                                        else f"FAILED - {len(problems)} problem(s)."))
    return 1 if problems else 0


# ------------------------------------------------------------------------ main

def build_parser():
    parser = argparse.ArgumentParser(prog="gtt-contract.sh", description=__doc__.split("\n")[0])
    sub = parser.add_subparsers(dest="command", required=True)
    p = sub.add_parser("release"); p.add_argument("--json", action="store_true"); p.set_defaults(func=cmd_release)
    p = sub.add_parser("negotiate")
    p.add_argument("--cli-version"); p.add_argument("--cli-capabilities"); p.add_argument("--cli-schemas")
    p.add_argument("--json", action="store_true"); p.set_defaults(func=cmd_negotiate)
    for name, func in (("capabilities", cmd_capabilities), ("operations", cmd_operations)):
        p = sub.add_parser(name); p.add_argument("--json", action="store_true"); p.set_defaults(func=func)
    p = sub.add_parser("show"); p.add_argument("kind"); p.set_defaults(func=cmd_show)
    p = sub.add_parser("run"); p.add_argument("operation"); p.add_argument("arguments", nargs="*")
    p.add_argument("--envelope", action="store_true"); p.set_defaults(func=cmd_run)
    p = sub.add_parser("check"); p.add_argument("--json", action="store_true"); p.set_defaults(func=cmd_check)
    return parser


def main(argv):
    if not os.path.isdir(".gtt"):
        die("run from the project root (no .gtt/ directory here)")
    args = build_parser().parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
