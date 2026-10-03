#!/usr/bin/env python3
"""GTT Bootstrap 1.0 - acceptance tests for the CLI v1.0 contract.

Every scenario runs on a DISPOSABLE copy of this project in a temp directory; the project itself is never
modified. The tests drive the Bootstrap only through its contracts (gtt-contract.sh: release, negotiate,
show, run <declared operation>) exactly as a CLI would, and cover the acceptance list of the task:
release identity, compatibility (compatible -> PASS, incompatible -> REFUSE with the project unchanged),
profiles, ADE, questionnaire, export, recovery, evolution without CLI change, unsupported capability.

  python .gtt/tests/bootstrap-acceptance.py [--project .] [--keep]

Exit 0 = every check held, 1 = at least one failed.
"""

import argparse
import glob
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile

FAILS, COUNT = [], 0


def check(cond, label, detail=""):
    global COUNT
    COUNT += 1
    if cond:
        print(f"  ok    {label}")
    else:
        print(f"  FAIL  {label} {detail}")
        FAILS.append(label)


def bash(cwd, *cmd):
    p = subprocess.run(["bash", *cmd], cwd=cwd, capture_output=True, text=True, encoding="utf-8", errors="replace")
    return p.returncode, p.stdout, p.stderr


def contract(cwd, *args):
    return bash(cwd, ".gtt/scripts/gtt-contract.sh", *args)


def op(cwd, name, **kw):
    """Run a declared operation the way a CLI would: name + typed name=value arguments, envelope out."""
    args = [f"{k}={('true' if v is True else 'false' if v is False else v)}" for k, v in kw.items()]
    rc, out, err = contract(cwd, "run", name, *args, "--envelope")
    try:
        return rc, json.loads(out), err
    except ValueError:
        return rc, {"exit_code": rc, "stdout": out, "stderr": err}, err


def data(env):
    try:
        return json.loads(env["stdout"])
    except ValueError:
        return None


def tree_hash(root):
    digest = hashlib.sha256()
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = sorted(d for d in dirnames if d not in ("__pycache__", ".git"))
        for name in sorted(filenames):
            path = os.path.join(dirpath, name)
            digest.update(os.path.relpath(path, root).encode())
            digest.update(open(path, "rb").read())
    return digest.hexdigest()


def write(path, text):
    os.makedirs(os.path.dirname(path) or ".", exist_ok=True)
    with open(path, "w", encoding="utf-8", newline="\n") as handle:
        handle.write(text)


def read_json(path):
    with open(path, encoding="utf-8") as handle:
        return json.load(handle)


def save_json(path, obj):
    write(path, json.dumps(obj, indent=2) + "\n")


def fresh(project, tmp, name):
    dst = os.path.join(tmp, name)
    shutil.copytree(project, dst, ignore=shutil.ignore_patterns("__pycache__", ".git"))
    return dst


def make_host(project, tmp, name):
    """A host project: portable core + Engine + domain, no overlay installed."""
    host = os.path.join(tmp, name)
    os.makedirs(host)
    for item in ("AGENTS.md",):
        shutil.copy2(os.path.join(project, item), host)
    for item in (".gtt", "gtt-domain"):
        shutil.copytree(os.path.join(project, item), os.path.join(host, item), ignore=shutil.ignore_patterns("__pycache__"))
    return host


# ------------------------------------------------------------------ scenarios

def release_identity(project, tmp):
    print("[release identity] the CLI can identify the exact Bootstrap release")
    p = fresh(project, tmp, "rel")
    rc, out, _ = contract(p, "release", "--json")
    rel = json.loads(out)
    b = rel["bootstrap"]
    check(rc == 0 and b["id"] == "gtt-bootstrap" and b["version"] == "1.2.0" and b["schema_version"] == 1 and b["channel"] == "stable",
          "id, version, schema_version and channel are machine-readable", out[:200])
    check(rel["scaffold"]["version"] == 2 and rel["scaffold"]["version"] != b["version"], "scaffold version is distinct from the release version")
    check(all(isinstance(v, int) for v in rel["contracts"].values()) and "export_policy" in rel["contracts"], "every contract has a version")
    rc, out, _ = contract(p, "capabilities", "--json")
    ids = {c["id"] for c in json.loads(out)["capabilities"]}
    need = {"project.detect", "ade.detect", "ade.install", "template.materialize", "methodology.profile", "validation",
            "freeze", "session-context", "export-policy", "recovery", "developer-experience"}
    check(need <= ids, "the capability registry lists the required capabilities", str(need - ids))
    rc, out, _ = contract(p, "operations", "--json")
    ops = json.loads(out)["operations"]
    check(all(isinstance(o["version"], int) for o in ops.values()), "the operation registry is versioned")
    rc, out, err = contract(p, "check")
    check(rc == 0, "the contract check passes", out + err)


def compatibility(project, tmp):
    print("[compatibility] compatible CLI -> PASS; incompatible -> REFUSE with the project unchanged")
    p = fresh(project, tmp, "compat")
    before = tree_hash(p)
    good = ("--cli-version", "1.0.0", "--cli-capabilities", "contract.negotiate.v1,operation.execute.v1", "--cli-schemas", "1")
    rc, out, _ = contract(p, "negotiate", *good, "--json")
    check(rc == 0 and json.loads(out)["compatible"], "a compatible CLI passes")
    cases = [("older CLI", ("--cli-version", "0.9.9", "--cli-capabilities", "contract.negotiate.v1,operation.execute.v1", "--cli-schemas", "1")),
             ("unsupported schema", ("--cli-version", "1.0.0", "--cli-capabilities", "contract.negotiate.v1,operation.execute.v1", "--cli-schemas", "2")),
             ("missing capability", ("--cli-version", "1.0.0", "--cli-capabilities", "contract.negotiate.v1", "--cli-schemas", "1")),
             ("no version declared", ("--cli-capabilities", "contract.negotiate.v1,operation.execute.v1", "--cli-schemas", "1")),
             ("no schemas declared", ("--cli-version", "1.0.0", "--cli-capabilities", "contract.negotiate.v1,operation.execute.v1")),
             ("garbage version", ("--cli-version", "one", "--cli-capabilities", "contract.negotiate.v1,operation.execute.v1", "--cli-schemas", "1"))]
    for label, args in cases:
        rc, out, _ = contract(p, "negotiate", *args, "--json")
        check(rc == 3 and not json.loads(out)["compatible"] and json.loads(out)["reasons"], f"{label}: REFUSE with reasons")
    rel = read_json(os.path.join(p, ".gtt/contract/release.json"))
    rel["compatibility"]["cli"]["max_version"] = "1.4.0"
    save_json(os.path.join(p, ".gtt/contract/release.json"), rel)
    rc, out, _ = contract(p, "negotiate", "--cli-version", "2.0.0", "--cli-capabilities", "contract.negotiate.v1,operation.execute.v1", "--cli-schemas", "1")
    check(rc == 3, "a CLI newer than max_version is refused")
    rel["compatibility"]["cli"]["max_version"] = None
    save_json(os.path.join(p, ".gtt/contract/release.json"), rel)
    check(tree_hash(p) != before, "(sanity) the copy was edited only by this test")
    q = fresh(project, tmp, "compat2")
    h = tree_hash(q)
    contract(q, "negotiate", "--cli-version", "0.1.0", "--cli-schemas", "9")
    check(tree_hash(q) == h, "a refusal never modifies the project")


def profiles(project, tmp):
    print("[profiles] Light, Medium, Hard and Team are Bootstrap-defined; the human selects, the CLI records")
    p = fresh(project, tmp, "prof")
    rc, out, _ = contract(p, "show", "profiles")
    spec = json.loads(out)
    check([x["id"] for x in spec["supported"]] == ["light", "medium", "hard", "team"] and spec["strictness_order"] == ["light", "medium", "hard", "team"],
          "four supported plans, ordered light < medium < hard < team")
    check(spec["selection"]["policy"] == "ask" and spec["selection"]["unselected"]["gates_from"] == spec["default"] == "medium",
          "the plan is asked for, never inferred; medium is only the gate fallback of an unselected project")
    check(all(spec["profiles"][x]["plan"]["policy"]["human"]["confirmation"][k] == "required"
              for x in spec["strictness_order"] for k in ("governed_decision", "destructive")),
          "no plan delegates a governed decision or a destructive operation")
    check(spec["profiles"]["team"]["gates"] == spec["profiles"]["hard"]["gates"] and spec["profiles"]["team"]["plan"]["policy"]["collaboration"]["ci"] == "required",
          "Team is the Hard gates plus collaboration requirements (CI required)")
    check(all(not i["relaxable"] for i in spec["invariants"]) and {"human-decision-authority", "provenance-tags", "freeze-semantics"} <= {i["id"] for i in spec["invariants"]},
          "the mandatory invariants are defined and none is relaxable")
    check(all(spec["profiles"][x]["relaxes"] == [] for x in ("medium", "hard", "team")) and spec["profiles"]["light"]["relaxes"],
          "only Light relaxes anything, and says exactly how")
    rc, env, _ = op(p, "methodology.profile.get")
    got = data(env)
    check(got["profile"] == "medium" and got["source"] == "default" and got["selected"] is False,
          "no selection -> reported as not selected; medium gates apply as a fallback")
    rc, text, _ = bash(p, ".gtt/scripts/gtt-project.sh", "profile", "get")
    check("plan: not selected" in text, "the unselected state is stated in plain words", text)
    for choice in ("light", "medium", "team", "hard"):
        rc, env, _ = op(p, "methodology.profile.set", profile=choice, language="es", apply=True)
        state = read_json(os.path.join(p, ".gtt/methodology.json"))
        check(rc == 0 and state["profile"] == choice and state["language"] == "es", f"{choice} is valid and recorded")
    rc, env, _ = op(p, "methodology.profile.set", profile="extreme", apply=True)
    check(rc == 5, "an unsupported profile is refused by the argument schema")
    rc, env, _ = op(p, "methodology.profile.set", profile="light")
    check(rc == 0 and read_json(os.path.join(p, ".gtt/methodology.json"))["profile"] == "hard", "without apply it is a dry run")
    # the Bootstrap applies the semantics: gates differ by profile
    results = {}
    for choice in ("light", "medium", "team", "hard"):
        op(p, "methodology.profile.set", profile=choice, apply=True)
        results[choice] = bash(p, ".gtt/scripts/gtt-check-provenance.sh", "--pre-freeze")[0]
    check(results["light"] == 0 and results["medium"] == 0 and results["hard"] == 1 and results["team"] == 1,
          "Hard and Team refuse freeze where Light and Medium accept (warnings block, manifest required)", str(results))
    got = data(op(p, "methodology.profile.get")[1])
    check(got["selected"] is True and got["plan"]["label"] == "Hard Method", "a selected plan is reported with its label and policy")
    op(p, "methodology.profile.set", profile="hard", apply=True)
    hard = bash(p, ".gtt/scripts/gtt-check-provenance.sh", "--pre-freeze")
    out = hard[1] + hard[2]
    check("requires a source manifest" in out and "blocks freeze" in out, "the Hard failures cite the profile", out[-300:])
    # frozen: relaxing is a governed change
    write(os.path.join(p, "gtt-domain/.frozen"), "2026-01-01T00:00:00Z\n")
    rc, env, err = op(p, "methodology.profile.set", profile="light", apply=True)
    check(env["exit_code"] == 1 and "governed change" in env["stderr"] and read_json(os.path.join(p, ".gtt/methodology.json"))["profile"] == "hard",
          "a frozen project refuses a less strict profile", env["stderr"][:200])
    op(p, "methodology.profile.set", profile="team", apply=True)
    rc, env, err = op(p, "methodology.profile.set", profile="hard", apply=True)
    check(env["exit_code"] == 1 and read_json(os.path.join(p, ".gtt/methodology.json"))["profile"] == "team",
          "a frozen project accepts a stricter plan (team) and then refuses to step back to hard")
    # the contracts themselves refuse to weaken an invariant
    bad = fresh(project, tmp, "prof-bad")
    spec = read_json(os.path.join(bad, ".gtt/contract/profiles.json"))
    spec["profiles"]["light"]["relaxes"].append({"control": "freeze-semantics", "how": "skip it"})
    save_json(os.path.join(bad, ".gtt/contract/profiles.json"), spec)
    rc, out, _ = contract(bad, "check")
    check(rc == 1 and "relaxes the invariant" in out, "a profile that relaxes an invariant fails the contract check", out)
    spec = read_json(os.path.join(bad, ".gtt/contract/profiles.json"))
    spec["profiles"]["light"]["relaxes"].pop()
    spec["invariants"][0]["relaxable"] = True
    save_json(os.path.join(bad, ".gtt/contract/profiles.json"), spec)
    rc, out, _ = contract(bad, "check")
    check(rc == 1 and "must not be relaxable" in out, "an invariant marked relaxable fails the check", out)
    spec = read_json(os.path.join(bad, ".gtt/contract/profiles.json"))
    spec["invariants"][0]["relaxable"] = False
    spec["profiles"]["light"]["plan"]["policy"]["human"]["confirmation"]["destructive"] = "not_required"
    spec["selection"]["policy"] = "infer"
    save_json(os.path.join(bad, ".gtt/contract/profiles.json"), spec)
    rc, out, _ = contract(bad, "check")
    check(rc == 1 and "no plan delegates it" in out and "never inferred" in out,
          "a plan that delegates a destructive operation, or a selection that is inferred, fails the check", out)


def ade(project, tmp):
    print("[ADE] detect, participate, primary, secondary, validate, change primary")
    host = make_host(project, tmp, "ade-host")
    write(os.path.join(host, ".kiro", "user-notes.md"), "the host's own\n")
    rc, out, _ = contract(host, "show", "ade-registry")
    reg = json.loads(out)
    row = {a["id"]: a for a in reg["ades"]}
    need = {"id", "name", "detect", "install", "validate", "owned_paths", "handoff", "invoke", "version"}
    check(sorted(row) == ["claude", "codex", "copilot", "kiro"] and all(need <= set(a) for a in reg["ades"]),
          "the ADE registry is machine-readable with the required fields")
    check(reg["participation_states"] == ["detected", "participating", "primary", "excluded"], "participation states are defined")
    rc, env, _ = op(host, "ade.detect")
    cand = {c["id"]: c for c in data(env)["candidates"]}
    check(rc == 0 and cand["kiro"]["status"] == "candidate", "detect yields candidates, never participation")
    rc, env, _ = op(host, "ade.install", **{"from": project, "participating": "claude,codex", "primary": "claude"})
    check(rc == 0 and not os.path.exists(os.path.join(host, ".gtt/ade.json")), "install is a dry run by default")
    rc, env, _ = op(host, "ade.install", **{"from": project, "participating": "claude,codex", "primary": "claude", "apply": True})
    check(rc == 0 and os.path.isfile(os.path.join(host, ".claude/CLAUDE.md")), "participating ADEs are installed", env["stderr"])
    rc, env, _ = op(host, "ade.state")
    st = data(env)
    check(st["primary"] == "claude" and st["participating"] == ["claude", "codex"] and st["excluded"] == ["copilot", "kiro"], "primary + secondary + excluded recorded")
    rc, env, _ = op(host, "ade.validate")
    check(rc == 0, "the resulting state validates")
    before_claude = tree_hash(os.path.join(host, ".claude"))
    rc, env, _ = op(host, "ade.set-primary", ade="copilot", apply=True)
    check(rc != 0, "the Primary must be a participating ADE")
    rc, env, _ = op(host, "ade.set-primary", ade="codex", apply=True)
    st = data(op(host, "ade.state")[1])
    check(rc == 0 and st["primary"] == "codex" and st["participating"] == ["claude", "codex"], "changing the Primary preserves the secondary ADEs")
    check(tree_hash(os.path.join(host, ".claude")) == before_claude and os.path.exists(os.path.join(host, ".kiro/user-notes.md")),
          "changing the Primary reinstalls and removes nothing")
    rc, env, _ = op(host, "ade.validate")
    check(rc == 0, "the state validates after the change")
    state = read_json(os.path.join(host, ".gtt/ade.json"))
    state["primary"] = "kiro"
    save_json(os.path.join(host, ".gtt/ade.json"), state)
    rc, env, _ = op(host, "ade.validate")
    check(rc == 1, "exactly one Primary, and it participates: a violating state fails validation")


def questionnaire(project, tmp):
    print("[questionnaire] no design -> contract -> materialize -> ADE handoff")
    rc, out, _ = contract(project, "show", "initial-design")
    q = json.loads(out)["scaffold"]["initial_design"]["questionnaire"]
    check(q["template"] == ".gtt/scaffold/templates/gtt-initial-design-questionnaire.md" and q["output"] and q["version"] and q["contract_version"],
          "the CLI discovers template, output, version and contract_version from the Bootstrap")
    check(q["handoff"]["reader"] == "the Primary ADE", "the handoff names the Primary ADE")
    rc, out, _ = contract(project, "show", "elicitation")
    eli = json.loads(out)
    ids = {d["id"] for d in eli["directives"]}
    need = {"inspect-existing-evidence", "ask-progressively", "avoid-redundant-questions", "identify-missing-information", "identify-conflicts",
            "mark-proposals", "preserve-provenance", "ask-human-for-decisions", "do-not-invent-architecture"}
    check(need <= ids, "the guided-elicitation contract has every required directive", str(need - ids))
    host = make_host(project, tmp, "q-host")
    rc, env, _ = op(host, "project.detect")
    det = data(env)
    check(det["questionnaire"]["offer"] and det["design_sources"]["candidates_at_root"] == [], "no design document -> the questionnaire is offered")
    write(os.path.join(host, "design-notes.md"), "# notes\n")
    det = data(op(host, "project.detect")[1])
    check(not det["questionnaire"]["offer"] and det["design_sources"]["candidates_at_root"] == ["design-notes.md"],
          "a candidate document is reported, not assumed, and the questionnaire is not offered")
    os.remove(os.path.join(host, "design-notes.md"))
    rc, env, _ = op(host, "template.materialize", id="initial-design-questionnaire", apply=True)
    target = os.path.join(host, q["output"])
    same = os.path.isfile(target) and open(target, "rb").read() == open(os.path.join(host, q["template"]), "rb").read()
    check(rc == 0 and same, "materialize creates the working copy verbatim (the CLI holds no copy)", env["stderr"])
    rc, env, _ = op(host, "template.materialize", id="initial-design-questionnaire", apply=True)
    check(rc == 1 and "CONFLICT" in env["stderr"], "a filled working copy is never overwritten")
    bad = fresh(project, tmp, "q-bad")
    tpl = os.path.join(bad, q["template"])
    text = open(tpl, encoding="utf-8").read().replace("Operating Contract for the ADE", "Renamed")
    write(tpl, text)
    rc, out, _ = contract(bad, "check")
    check(rc == 1 and "elicitation directive" in out, "the elicitation contract stays tied to the questionnaire's own structure", out[-200:])


def sources(project, tmp):
    print("[initial sources] selected source != governed authority")
    p = fresh(project, tmp, "src")
    write(os.path.join(p, "docs/spec.md"), "# spec\n")
    rc, env, _ = op(p, "source.select", path="docs/spec.md", type="design-document", version="2.1", by="cli")
    check(rc == 0 and not os.path.exists(os.path.join(p, ".gtt/selected-sources.json")), "select is a dry run by default")
    rc, env, _ = op(p, "source.select", path="docs/spec.md", type="design-document", version="2.1", by="cli", apply=True)
    rows = data(op(p, "source.list")[1])["selected"]
    r = rows[0]
    check(rc == 0 and r["path"] == "docs/spec.md" and r["type"] == "design-document" and r["version"] == "2.1" and r["selected_by"] == "cli"
          and r["authority_status"] == "unassigned" and r["precedence_status"] == "unassigned",
          "path, type, selection metadata and version are kept; authority and precedence are unassigned")
    rc, env, _ = op(p, "source.select", path="docs/spec.md", type="design-document", apply=True)
    check(rc == 1, "a source is selected once")
    rc, env, _ = op(p, "source.select", path="docs/none.md", type="other", apply=True)
    check(rc == 1, "a source that does not exist is refused")
    rc, env, _ = op(p, "source.select", path="../outside.md", type="other", apply=True)
    check(rc != 0, "a path outside the project is refused")
    write(os.path.join(p, "gtt-domain/context/sources.md"),
          "# Sources\n\n> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md\n\n"
          "```gtt-sources\nspec | docs/spec.md | 2.1 | primary | 1 | active\n```\n")
    r = data(op(p, "source.list")[1])["selected"][0]
    check(r["governed"] and r["authority_status"] == "primary" and r["precedence_status"] == 1,
          "authority and precedence appear only once a human declares them in the governed manifest")
    check("never inferred" in data(op(p, "source.list")[1])["rule"], "the Bootstrap states that authority is never inferred")


def export_and_clean(project, tmp):
    print("[export policy] Bootstrap policy -> CLI export -> correct GTT exclusions")
    host = make_host(project, tmp, "exp-host")
    write(os.path.join(host, ".claude", "keep-me.txt"), "the host's own claude file\n")
    op(host, "ade.install", **{"from": project, "participating": "claude,codex", "primary": "claude", "apply": True})
    pol = data(op(host, "export-policy")[1])["clean_export"]
    check(all(x in pol["exclude"] for x in (".gtt/", "gtt-domain/", "AGENTS.md", "readme-gtt.md", "SOURCE-BRIEF.*")), "GTT-owned content is excluded")
    check(".claude/CLAUDE.md" in pol["exclude"] and ".claude/keep-me.txt" not in pol["exclude"],
          "ADE overlays come from the ownership ledger; the host's own ADE files are not excluded")
    check(".claude" not in pol["exclude"] and ".claude/" not in pol["exclude"], "an ADE directory is never excluded by name")
    with open(os.path.join(host, ".claude/CLAUDE.md"), "a", encoding="utf-8") as handle:
        handle.write("local edit\n")
    pol = data(op(host, "export-policy")[1])["clean_export"]
    check(any(".claude/CLAUDE.md" in c for c in pol["requires_confirmation"]) and ".claude/CLAUDE.md" not in pol["exclude"],
          "a modified GTT file needs confirmation instead of being dropped silently")
    before = tree_hash(host)
    rc, env, _ = op(host, "clean.plan")
    plan = data(env)
    check(rc == 0 and plan["removes_nothing"] and ".gtt/" in plan["would_remove"] and "export --clean" in plan["distinct_from"], "clean.plan is distinct from export --clean")
    check(tree_hash(host) == before, "neither policy nor plan modified the project")


def recovery(project, tmp):
    print("[recovery] snapshot -> clean -> detect -> restore -> validate")
    host = make_host(project, tmp, "rec-host")
    op(host, "ade.install", **{"from": project, "participating": "claude,codex", "primary": "codex", "apply": True})
    op(host, "methodology.profile.set", profile="hard", language="es", apply=True)
    write(os.path.join(host, "docs/spec.md"), "# spec\n")
    op(host, "source.select", path="docs/spec.md", type="design-document", by="cli", apply=True)
    snap_path = os.path.join(tmp, "snapshot.json")
    rc, env, _ = op(host, "recovery.snapshot", output=snap_path)
    snap = read_json(snap_path)
    for key in ("bootstrap", "compatibility", "ade", "methodology", "sources", "operational", "recovery"):
        check(key in snap, f"snapshot carries `{key}`")
    check(snap["bootstrap"]["version"] == "1.2.0" and snap["ade"]["primary"] == "codex" and snap["methodology"] == {"profile": "hard", "language": "es"}
          and snap["sources"]["selected"][0]["path"] == "docs/spec.md", "identity, Primary, profile, language and selected sources are preserved")
    rc, env, _ = op(host, "recovery.snapshot", output=snap_path)
    check(env["exit_code"] == 1, "a snapshot is never overwritten")
    # clean the GTT configuration
    op(host, "ade.remove", all=True, apply=True)
    for f in (".gtt/methodology.json", ".gtt/selected-sources.json"):
        if os.path.exists(os.path.join(host, f)):
            os.remove(os.path.join(host, f))
    det = data(op(host, "project.detect")[1])
    check(not det["ade_configured"] and det["methodology"]["source"] == "default", "detect sees the cleaned project")
    rc, env, _ = op(host, "recovery.restore", snapshot=snap_path)
    check(rc == 0 and not os.path.exists(os.path.join(host, ".gtt/methodology.json")), "restore is a dry run by default")
    rc, env, _ = op(host, "recovery.restore", snapshot=snap_path, **{"from": project, "apply": True})
    st = data(op(host, "ade.state")[1])
    check(rc == 0 and st["primary"] == "codex" and st["participating"] == ["claude", "codex"], "the ADE state is restored (overlays reinstalled from the catalog)", env["stderr"])
    check(data(op(host, "methodology.profile.get")[1])["profile"] == "hard" and data(op(host, "methodology.profile.get")[1])["language"] == "es",
          "profile and language are restored")
    check(data(op(host, "source.list")[1])["selected"][0]["path"] == "docs/spec.md", "selected sources are restored")
    rc, env, _ = op(host, "ade.validate")
    check(rc == 0, "the restored ADE state validates")
    rc, env, _ = op(host, "recovery.restore", snapshot=snap_path, apply=True)
    check(not os.path.exists(os.path.join(host, "gtt-domain/.frozen")), "restore never writes a freeze marker")
    # refusals
    for label, mutate in (("newer schema", lambda s: s.update(schema=2)),
                          ("another bootstrap", lambda s: s["bootstrap"].update(id="someone-else")),
                          ("missing field", lambda s: s.pop("ade")),
                          ("unknown profile", lambda s: s["methodology"].update(profile="extreme"))):
        bad = json.loads(json.dumps(snap))
        mutate(bad)
        path = os.path.join(tmp, f"bad-{label.replace(' ', '-')}.json")
        save_json(path, bad)
        rc, env, _ = op(host, "recovery.restore", snapshot=path, apply=True)
        check(env["exit_code"] == 1 and "REFUSED" in env["stderr"], f"restore refuses: {label}")
    write(os.path.join(host, "gtt-domain/.frozen"), "2026-01-01T00:00:00Z\n")
    relax = json.loads(json.dumps(snap))
    relax["methodology"]["profile"] = "light"
    path = os.path.join(tmp, "relax.json")
    save_json(path, relax)
    rc, env, _ = op(host, "recovery.restore", snapshot=path, apply=True)
    check(env["exit_code"] == 1 and "relax" in env["stderr"], "a frozen project refuses a snapshot that would relax the profile")


def session_status_validation(project, tmp):
    print("[session / status / validation] deterministic, structured, derived from the project")
    p = fresh(project, tmp, "ssv")
    subprocess.run(["git", "init", "-q"], cwd=p)
    subprocess.run(["git", "-c", "user.email=t@t", "-c", "user.name=t", "add", "-A"], cwd=p)
    subprocess.run(["git", "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "-m", "test commit"], cwd=p)
    write(os.path.join(p, "untracked.txt"), "x\n")
    rc, env, _ = op(p, "session-context")
    s = data(env)
    need = {"freeze", "change_request", "proposals", "validation", "git", "operational", "artifacts"}
    check(rc == 0 and need <= set(s), "the session contract identifies freeze, change request, proposals, validation, git, operational state and artifacts", str(need - set(s)))
    check(s["authority"] == "none" and s["operational_only"] and "NOT authority" in s["markers"], "it declares itself non-authoritative")
    check(s["git"]["available"] and s["git"]["uncommitted_paths"] >= 1 and s["git"]["recent_commits"][0].endswith("test commit"), "git state comes from the repository")
    check(s["change_request"] == "empty" and s["freeze"]["frozen"] is False, "freeze and change request come from the actual files")
    rc, env, _ = op(p, "status")
    st = data(env)
    need = {"bootstrap", "ade", "methodology", "sources", "governance", "freeze", "validation", "session"}
    check(rc == 0 and need <= set(st) and st["bootstrap"]["version"] == "1.2.0", "the status contract has every section", str(need - set(st)))
    rc, env, _ = op(p, "validation.run")
    v = data(env)
    check(rc == 0 and v["result"] == "pass" and any(c["check"].startswith("gtt-check-contract.sh") and c["result"] == "pass" for c in v["checks"]),
          "validation.run is one structured entry point that includes the contract gate", str(v)[:300])
    write(os.path.join(p, "gtt-domain/context/architecture.md"), "# Architecture\n\n[PROPUESTA] something\n")
    rc, env, _ = op(p, "validation.run")
    v = data(env)
    check(rc == 1 and v["result"] == "fail" and "gtt-check-provenance.sh" in v["failing"], "a violation surfaces as a failing check", str(v)[:200])
    check(any("[PROPUESTA]" in m for m in v["messages"]) and not any(c["check"].startswith(("FAIL", "WARN")) for c in v["checks"]),
          "detail lines are reported as messages, never mistaken for checks", str(v)[:200])
    rc, env, _ = op(p, "session-context.text")
    check(rc == 0 and "GTT-SESSION-CONTEXT" in env["stdout"], "the text session context remains available")


def developer_experience(project, tmp):
    print("[developer experience] deterministic work is done, not asked; reports are brief; protected operations still need the human")
    p = fresh(project, tmp, "dx")
    spec = json.loads(contract(p, "show", "profiles")[1])
    dx = spec["developer_experience"]
    check(all(dx[k] is True for k in ("minimize_interruption", "automatic_deterministic_operations", "concise_reports", "details_on_demand")),
          "the policy is declared for every plan")
    check(dx["protected_operations"]["agent_executes"] is False and len(dx["stop_conditions"]) == 5,
          "protected operations stay with the human; STOP has a closed list of reasons")
    check(data(op(p, "methodology.profile.get")[1])["developer_experience"]["report"]["default"] == "brief",
          "the CLI receives the policy with the plan")
    # the next free id is determined, never asked for
    h = tree_hash(p)
    rc, env, _ = op(p, "artifact.next-id", kind="adr")
    first = data(env)
    check(rc == 0 and first["id"] not in first["occupied"] and "ADR-001" in first["occupied"], "the next free ADR id is determined from the project", str(first))
    check(data(op(p, "artifact.next-id", kind="epic")[1])["id"] == "EPIC-001", "a template example Epic does not occupy an id")
    got = data(op(p, "artifact.next-id", kind="adr", requested="ADR-001")[1])
    check(got["requested"]["free"] is False and got["id"] == first["id"], "an occupied id resolves to the next free one, with no question")
    check(tree_hash(p) == h, "resolving an id reserves and writes nothing")
    write(os.path.join(p, "gtt-domain/adr", first["id"] + "-example.md"), "# " + first["id"] + " - example\n")
    after = data(op(p, "artifact.next-id", kind="adr")[1])
    check(int(after["id"].split("-")[1]) == int(first["id"].split("-")[1]) + 1, "a newly taken id moves the answer on")
    os.remove(os.path.join(p, "gtt-domain/adr", first["id"] + "-example.md"))
    rc, env, _ = op(p, "artifact.next-id", kind="widget")
    check(rc == 5, "an unknown kind is refused by the argument schema")
    # post-operation maintenance: one run, brief by default, detail on demand
    rc, out, err = bash(p, ".gtt/scripts/gtt-maintain.sh")
    lines = [l for l in out.splitlines() if l.strip()]
    check(rc == 0 and len(lines) <= 4 and not any(l.startswith("PASS") for l in lines) and "Validation:" in out,
          "maintain reports in a few lines", out + err)
    rc, out, _ = bash(p, ".gtt/scripts/gtt-maintain.sh", "--verbose")
    check(rc == 0 and "PASS" in out and "gtt-validate: OK" in out, "the full detail is available on demand")
    rc, env, _ = op(p, "maintain")
    check(rc == 0 and env["exit_code"] == 0, "maintain runs through the operation registry")
    for cmd in (["init", "-q", "."], ["add", "-A"], ["-c", "user.name=gtt", "-c", "user.email=gtt@example.invalid", "commit", "-qm", "base"],
                ["mv", ".gtt/docs/usage.md", ".gtt/docs/usage-moved.md"]):       # a real rename, as a developer would do it
        subprocess.run(["git", *cmd], cwd=p, capture_output=True, check=True)
    rc, out, _ = bash(p, ".gtt/scripts/gtt-maintain.sh")
    check(rc == 1 and "need reconciling" in out and "gtt-reconcile.sh" in out and os.path.isfile(os.path.join(p, ".gtt/docs/usage-moved.md")),
          "maintain does not rewrite references on its own: it stops with the exact command", out)
    # the same rule, a different effect per plan
    expect = {"light": (True, "use", []), "medium": (False, "use", ["agent_context_sync", "reference_updates", "relevant_change"]),
              "hard": (False, "propose", ["agent_context_sync", "identity_resolution", "reference_updates", "relevant_change"]),
              "team": (False, "propose", ["agent_context_sync", "identity_resolution", "reference_updates", "relevant_change"])}
    for choice, (reconciles, resolution, extra) in expect.items():
        op(p, "methodology.profile.set", profile=choice, apply=True)
        got = data(op(p, "methodology.profile.get")[1])["interaction"]
        check(got["maintain_reconciles_moves"] is reconciles and got["id_resolution"] == resolution
              and sorted(set(got["asks_for"]) - {"governed_decision", "destructive"}) == extra
              and {"governed_decision", "destructive"} <= set(got["asks_for"]),
              f"{choice}: asks for governed decisions, destructive operations and {extra or 'nothing else'}", str(got))
        check(data(op(p, "artifact.next-id", kind="adr")[1])["resolution"] == resolution, f"{choice}: an id is to {resolution}")
        if not reconciles:
            rc, out, _ = bash(p, ".gtt/scripts/gtt-maintain.sh")
            check(rc == 1 and "need reconciling" in out and choice in out, f"{choice}: maintain stops before rewriting references", out)
    op(p, "methodology.profile.set", profile="light", apply=True)
    rc, out, err = bash(p, ".gtt/scripts/gtt-maintain.sh")
    check(rc == 0 and "Reconciled 1 moved artifact(s)" in out and "usage-moved.md" in out,
          "light: maintain reconciles the unambiguous move, says so, and validates", out + err)
    os.remove(os.path.join(p, ".gtt/docs/usage-moved.md"))
    rc, out, _ = bash(p, ".gtt/scripts/gtt-maintain.sh")
    check(rc == 1 and "need your decision" in out, "light: a missing artifact is still a decision, never resolved automatically", out)
    check(all(os.access(f, os.X_OK) for f in glob.glob(os.path.join(p, ".gtt/scripts/*.sh")) + glob.glob(os.path.join(p, "gtt-domain/proposals/*.sh"))),
          "every shipped script is directly executable")
    # lower friction can never hand a protected operation to the agent
    bad = fresh(project, tmp, "dx-bad")
    spec = read_json(os.path.join(bad, ".gtt/contract/profiles.json"))
    spec["developer_experience"]["protected_operations"]["agent_executes"] = True
    spec["developer_experience"]["automatic_operations"].append({"what": "freeze", "operation": "freeze"})
    save_json(os.path.join(bad, ".gtt/contract/profiles.json"), spec)
    rc, out, _ = contract(bad, "check")
    check(rc == 1 and "agent_executes must be false" in out and "needs human authority" in out,
          "a policy that automates a human-authority operation fails the contract check", out)


def fresh_host(project, tmp):
    print("[fresh host] install -> reconcile the catalog identity -> index -> validation.run")
    host = make_host(project, tmp, "fresh-host")
    for kit in ("readme-gtt.md", "readme-gtt.es.md"):
        shutil.copy2(os.path.join(project, kit), host)
    rc, env, _ = op(host, "ade.install", **{"from": project, "participating": "claude", "primary": "claude", "apply": True})
    rc, env, _ = op(host, "validation.run")
    check(env["exit_code"] == 1 and "gtt-check-integrity.sh" in data(env)["failing"],
          "a fresh host still carries the catalog's identity manifest: integrity fails until it is reconciled")
    before = tree_hash(host)
    rc, env, _ = op(host, "reconcile", retire_missing=True)
    check(rc == 0 and "dry run" in env["stdout"] and tree_hash(host) == before, "reconcile retire_missing is a dry run by default")
    rc, env, _ = op(host, "reconcile", retire_missing=True, apply=True)
    check(rc == 0, "reconcile retire_missing apply=true retires only registered paths that are gone", env["stderr"][:200])
    rc, env, _ = op(host, "index")
    rc, env, _ = op(host, "validation.run")
    check(env["exit_code"] == 0 and data(env)["result"] == "pass", "after reconcile and index the fresh host validates", str(data(env))[:200])


def registry_safety(project, tmp):
    print("[operations] declared only, no arbitrary shell, no unfreeze, fail closed")
    p = fresh(project, tmp, "safe")
    canary = os.path.join(p, "canary")
    rc, env, _ = op(p, "query", term=f"$(touch {canary})")
    check(not os.path.exists(canary), "shell metacharacters in an argument are data, never executed")
    rc, env, _ = op(p, "query", term=f"x; touch {canary}")
    check(not os.path.exists(canary), "a command separator in an argument is not executed")
    rc, out, err = contract(p, "run", "nosuch", "--envelope")
    check(rc == 5, "an undeclared operation is refused")
    rc, env, _ = op(p, "ade.set-primary", ade="--apply")
    check(rc == 5, "an argument that looks like a flag is refused")
    rc, env, _ = op(p, "ade.set-primary", ade="claude", surprise="1")
    check(rc == 5, "an undeclared argument is refused")
    rc, env, _ = op(p, "ade.install", **{"from": project})
    check(rc == 5, "a missing required argument is refused")
    rc, env, _ = op(p, "freeze")
    check(rc == 4, "freeze needs a human decision from the CLI")
    ops = read_json(os.path.join(p, ".gtt/contract/operations.json"))
    check(not any("unfreeze" in k.lower() for k in ops["operations"]), "there is no unfreeze operation")
    def mutated(name, fn, expect):
        q = fresh(project, tmp, name)
        doc = read_json(os.path.join(q, ".gtt/contract/operations.json"))
        fn(doc)
        save_json(os.path.join(q, ".gtt/contract/operations.json"), doc)
        rc, out, _ = contract(q, "check")
        check(rc == 1 and expect in out, f"the contract check rejects: {expect}", out[-200:])
        return q
    mutated("m1", lambda d: d["operations"].update({"unfreeze": dict(d["operations"]["freeze"])}), "no unfreeze operation")
    mutated("m2", lambda d: d["operations"]["index"].update(implementation="/bin/echo"), "existing file under .gtt/scripts/")
    mutated("m3", lambda d: d["operations"]["index"].update(argv=["-c", "rm -rf /; echo"]), "shell syntax")
    mutated("m4", lambda d: d["operations"]["freeze"].update(human_authority=False), "human_authority")
    q = mutated("m5", lambda d: d["operations"]["index"].update(implementation=".gtt/scripts/../../etc/passwd"), "existing file under .gtt/scripts/")
    rc, env, _ = op(q, "index")
    check(rc == 5, "the runner also refuses an untrusted implementation")
    mutated("m6", lambda d: d["operations"]["template.materialize"]["args"].pop(2), "must offer `apply`")


def evolution(project, tmp):
    print("[evolution] Bootstrap content changes work with the CLI source unchanged")
    p = fresh(project, tmp, "evo")
    # new template
    write(os.path.join(p, ".gtt/scaffold/templates/gtt-new-template.md"), "# New template\n\n> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md\n")
    manifest = os.path.join(p, ".gtt/scaffold/manifest.yaml")
    text = open(manifest, encoding="utf-8", newline="").read()
    line = "  - { id: new-template, path: .gtt/scaffold/templates/gtt-new-template.md, kind: file, required: false, use: test, when: test, materialize_to: gtt-domain/proposals/new-template.md, becomes: draft, scaffold: 2, version: 1, contract_version: 1, role: new template }\n"
    anchor = "\nnaming:"
    text = text.replace(anchor, "\n" + line.rstrip("\n") + "\n" + anchor, 1) if "\nnaming:" in text else text
    write(manifest, text)
    rc, env, _ = op(p, "template.list")
    check(any(t["id"] == "new-template" for t in data(env)["templates"]), "a new template is offered with no code change")
    rc, env, _ = op(p, "template.materialize", id="new-template", apply=True)
    check(rc == 0 and os.path.isfile(os.path.join(p, "gtt-domain/proposals/new-template.md")), "and can be materialized through the same operation")
    # new profile semantics
    spec = read_json(os.path.join(p, ".gtt/contract/profiles.json"))
    spec["profiles"]["medium"]["semantics"]["audit_requirements"] = "gtt-audit every sprint (new semantics)"
    save_json(os.path.join(p, ".gtt/contract/profiles.json"), spec)
    check("every sprint" in data(op(p, "methodology.profile.get")[1])["semantics"]["audit_requirements"], "new profile semantics are read from the contract")
    # new ADE using existing primitives
    text = open(manifest, encoding="utf-8", newline="").read()
    ade_line = "  - { id: newade, name: New ADE, path: .newade/, entry: .newade/rules.md, owned: [.newade/], detect: [.newade/], enforcement: ci-gate, scaffold: 2, version: 1, kind: dir, required: false, role: test overlay }\n"
    text = text.replace("\ntemplates:", "\n" + ade_line.rstrip("\n") + "\n\ntemplates:", 1)
    write(manifest, text)
    write(os.path.join(p, ".newade/rules.md"), "# rules\n")
    reg = {a["id"] for a in json.loads(contract(p, "show", "ade-registry")[1])["ades"]}
    check("newade" in reg, "a new ADE appears in the registry with no CLI change")
    host = make_host(p, tmp, "evo-host")
    rc, env, _ = op(host, "ade.install", **{"from": p, "participating": "newade", "primary": "newade", "apply": True})
    check(rc == 0 and os.path.isfile(os.path.join(host, ".newade/rules.md")), "and installs through the existing ADE primitives", env["stderr"])
    # new operation on an existing implementation
    doc = read_json(os.path.join(p, ".gtt/contract/operations.json"))
    doc["operations"]["guard.check"] = dict(doc["operations"]["guard.validate"])
    save_json(os.path.join(p, ".gtt/contract/operations.json"), doc)
    rc, env, _ = op(p, "guard.check")
    check(rc == 0, "a new operation declared over an existing implementation runs through the same runner", env["stderr"])
    rc, out, _ = contract(p, "check")
    check(rc == 0, "the evolved contracts still pass the contract check", out)


def unsupported(project, tmp):
    print("[unsupported capability] Bootstrap requires X, CLI lacks X -> REFUSE -> project unchanged")
    p = fresh(project, tmp, "unsup")
    rel = read_json(os.path.join(p, ".gtt/contract/release.json"))
    rel["requires_cli_capabilities"].append("future.capability.v2")
    save_json(os.path.join(p, ".gtt/contract/release.json"), rel)
    before = tree_hash(p)
    rc, out, _ = contract(p, "negotiate", "--cli-version", "1.0.0", "--cli-capabilities", "contract.negotiate.v1,operation.execute.v1", "--cli-schemas", "1", "--json")
    r = json.loads(out)
    check(rc == 3 and not r["compatible"] and any("future.capability.v2" in x for x in r["reasons"]), "the CLI is told exactly which capability it lacks")
    check(tree_hash(p) == before, "the project is unchanged")
    rc, out, _ = contract(p, "negotiate", "--cli-version", "1.0.0", "--cli-capabilities", "contract.negotiate.v1,operation.execute.v1,future.capability.v2", "--cli-schemas", "1")
    check(rc == 0, "a CLI that has the capability is compatible")


def guard_and_retrieval(project, tmp):
    print("[guard / index / reconcile / query] stable operations")
    p = fresh(project, tmp, "gr")
    for name, kw in (("guard.validate", {}), ("guard.sync", {}), ("index", {}), ("reconcile", {}), ("query", {"term": "Human Promotion Boundary"}),
                     ("query.governance", {"kind": "open"})):
        rc, env, _ = op(p, name, **kw)
        check(rc == 0, f"{name} runs through the operation registry", env["stderr"][:200])
    rc, env, _ = op(p, "query.governance", kind="everything")
    check(rc == 5, "an enum outside its values is refused")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--project", default=".")
    parser.add_argument("--keep", action="store_true")
    args = parser.parse_args()
    project = os.path.abspath(args.project)
    tmp = tempfile.mkdtemp(prefix="gtt-bootstrap-acceptance-")
    try:
        for scenario in (release_identity, compatibility, profiles, developer_experience, ade, questionnaire, sources, export_and_clean, recovery,
                         session_status_validation, fresh_host, registry_safety, evolution, unsupported, guard_and_retrieval):
            scenario(project, tmp)
    finally:
        if args.keep:
            print("kept:", tmp)
        else:
            shutil.rmtree(tmp, ignore_errors=True)
    print(f"\n{COUNT - len(FAILS)}/{COUNT} checks held")
    if FAILS:
        print("FAILED:", "; ".join(FAILS))
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
