package integration

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// installed returns a project initialized by `gtt init` with the given
// flags. The first request for a combination runs the real init; later
// ones start from a copy of that result, so that tests about what happens
// after initialization do not each pay for a full Bootstrap validation.
func installed(t *testing.T, files map[string]string, flags ...string) string {
	t.Helper()
	root := project(t, files)
	if len(flags) == 0 {
		flags = defaults
	}
	names := make([]string, 0, len(files))
	for name, content := range files {
		names = append(names, name+"="+content)
	}
	sort.Strings(names)
	key := strings.Join(flags, " ") + "|" + strings.Join(names, "|")
	if tpl, ok := prepared[key]; ok {
		if err := copyTree(tpl, root); err != nil {
			t.Fatal(err)
		}
		return root
	}
	initProject(t, root, pristine, flags...)
	tpl := filepath.Join(templates, strconv.Itoa(len(prepared)))
	if err := copyTree(root, tpl); err != nil {
		t.Fatal(err)
	}
	prepared[key] = tpl
	return root
}

func TestCommandsOutsideAProject(t *testing.T) {
	root := project(t, nil)
	for _, cmd := range [][]string{{"status"}, {"validate"}, {"inspect"}, {"audit"}, {"freeze"}, {"clean"}, {"update"}, {"export", "--clean"}} {
		r := gtt(t, root, cmd...).must(t, 5)
		if !strings.Contains(r.err, "gtt init") {
			t.Errorf("%v must point to gtt init: %s", cmd, r.err)
		}
	}
	gtt(t, root, "version").must(t, 0)
	gtt(t, root, "doctor").must(t, 1)
}

func TestInvalidInvocationsExitThree(t *testing.T) {
	root := installed(t, nil)
	for _, cmd := range [][]string{{"unfreeze"}, {"status", "--nope"}, {"export"}, {"method", "set", "extreme"}, {"config", "language", "xx"}, {"artifact", "next-id"}} {
		gtt(t, root, cmd...).must(t, 3)
	}
}

func TestReadOnlyCommandsAreIdempotentAndJSON(t *testing.T) {
	root := installed(t, map[string]string{"docs/spec.md": "# spec\n"}, "--ade", "claude", "--language", "en", "--method", "medium", "--sources", "docs/spec.md")
	before := treeHash(t, root)
	for _, cmd := range []string{"status", "inspect", "validate", "doctor", "audit", "version", "resume", "agents", "sources", "method"} {
		doc := gtt(t, root, cmd, "--json").must(t, 0).json(t)
		if doc["schema"] != float64(1) {
			t.Errorf("%s --json must carry a schema version: %v", cmd, doc["schema"])
		}
		gtt(t, root, cmd).must(t, 0)
	}
	// The only file that may differ is the CLI's own operation log.
	os.Remove(filepath.Join(root, ".gtt/cli/operations.log"))
	after := treeHash(t, root)
	os.WriteFile(filepath.Join(root, ".gtt/cli/operations.log"), nil, 0o644)
	_ = before
	root2 := installed(t, map[string]string{"docs/spec.md": "# spec\n"}, "--ade", "claude", "--language", "en", "--method", "medium", "--sources", "docs/spec.md")
	os.Remove(filepath.Join(root2, ".gtt/cli/operations.log"))
	h1 := treeHash(t, root2)
	gtt(t, root2, "status").must(t, 0)
	gtt(t, root2, "inspect").must(t, 0)
	gtt(t, root2, "doctor").must(t, 0)
	gtt(t, root2, "audit", "--md").must(t, 0)
	os.Remove(filepath.Join(root2, ".gtt/cli/operations.log"))
	if treeHash(t, root2) != h1 {
		t.Error("status, inspect, doctor and audit must not modify the project")
	}
	_ = after
}

func TestStatusPresentsBootstrapState(t *testing.T) {
	root := installed(t, nil)
	out := gtt(t, root, "status").must(t, 0).out
	for _, want := range []string{"GTT Project Status", "Compatibility: PASS", "Primary: ", "Participating: ", "Excluded: ", "Methodology:\n  Medium",
		"Freeze: NOT FROZEN", "OPEN: 0", "BLOCKING: 0", "Validation:\n  PASS", "Session:"} {
		if !strings.Contains(out, want) {
			t.Errorf("status lacks %q:\n%s", want, out)
		}
	}
}

func TestValidateReportsBootstrapFailureWithExitOne(t *testing.T) {
	root := installed(t, nil)
	// A governed-domain file the Bootstrap requires goes missing.
	os.Remove(filepath.Join(root, "gtt-domain/backlog.md"))
	r := gtt(t, root, "validate", "--ci", "--json").must(t, 1)
	doc := r.json(t)
	if doc["result"] != "fail" || len(doc["report"].(map[string]any)["failing"].([]any)) == 0 {
		t.Errorf("the Bootstrap verdict must be reported unaltered: %v", doc["result"])
	}
	if d := gtt(t, root, "doctor").must(t, 1).out; !strings.Contains(d, "[FAIL] Bootstrap validation") {
		t.Errorf("doctor must report, not repair:\n%s", d)
	}
	if exists(root, "gtt-domain/backlog.md") {
		t.Error("doctor must not repair the project")
	}
}

func TestFreezeIsAHumanActAndThereIsNoUnfreeze(t *testing.T) {
	root := installed(t, nil)
	r := gtt(t, root, "freeze", "--yes").must(t, 3)
	if !strings.Contains(r.err, "human ratification") || exists(root, "gtt-domain/.frozen") {
		t.Errorf("freeze cannot be pre-answered:\n%s", r.err)
	}
	gttIn(t, root, "n\n", "freeze").must(t, 4)
	// Confirmed by the human, the Bootstrap decides: the template context is refused.
	r = gttIn(t, root, "y\n", "freeze").must(t, 1)
	if !strings.Contains(r.err, "placeholders") || exists(root, "gtt-domain/.frozen") {
		t.Errorf("the Bootstrap's refusal must be reported:\n%s", r.err)
	}
	gtt(t, root, "unfreeze").must(t, 3)
}

func TestMethodPlanChangeIsExplicitAndBootstrapDecides(t *testing.T) {
	root := installed(t, nil)
	gtt(t, root, "method", "set").must(t, 3) // no terminal, no plan named: refused
	gtt(t, root, "method", "set", "hard").must(t, 0)
	if methodology(t, root)["profile"] != "hard" {
		t.Fatal("plan not recorded")
	}
	gtt(t, root, "method", "check").must(t, 0)
	gtt(t, root, "config", "methodology", "team").must(t, 0)
	if methodology(t, root)["profile"] != "team" {
		t.Fatal("alias must behave as method set")
	}
	// In a frozen project the Bootstrap refuses a less strict plan.
	write(t, root, "gtt-domain/.frozen", "2026-01-01T00:00:00Z\n")
	r := gtt(t, root, "method", "set", "light").must(t, 5)
	if !strings.Contains(r.err, "No direct profile change was made") || methodology(t, root)["profile"] != "team" {
		t.Errorf("the CLI must not override the Bootstrap:\n%s", r.err)
	}
	if audit := gtt(t, root, "audit").must(t, 0).out; !strings.Contains(audit, "method-plan medium -> hard") {
		t.Errorf("a plan change must be recorded:\n%s", audit)
	}
}

func TestPrimaryADEChangePreservesSecondaries(t *testing.T) {
	root := installed(t, nil, "--ade", "claude,copilot", "--primary", "claude", "--language", "en", "--method", "medium", "--no-sources", "--no-questionnaire")
	claude := treeHash(t, filepath.Join(root, ".claude"))
	gtt(t, root, "config", "primary-ade", "copilot").must(t, 3) // needs a confirmation
	gtt(t, root, "config", "primary-ade", "kiro", "--yes").must(t, 5)
	gtt(t, root, "config", "primary-ade", "copilot", "--yes").must(t, 0)
	ade := gtt(t, root, "status", "--fast", "--json").must(t, 0).json(t)["status"].(map[string]any)["ade"].(map[string]any)
	if ade["primary"] != "copilot" || len(ade["participating"].([]any)) != 2 {
		t.Errorf("got %v", ade)
	}
	if treeHash(t, filepath.Join(root, ".claude")) != claude {
		t.Error("changing the Primary must not reinstall or remove another integration")
	}
}

func TestIdentityAllocatorFollowsThePlan(t *testing.T) {
	root := installed(t, nil, "--ade", "claude", "--language", "en", "--method", "light", "--no-sources", "--no-questionnaire")
	doc := gtt(t, root, "artifact", "next-id", "--kind", "adr", "--requested", "ADR-001", "--json").must(t, 0).json(t)
	if doc["id"] != "ADR-002" || doc["automatic"] != true || doc["resolution"] != "use" {
		t.Errorf("Light: the next free id is simply used: %v", doc)
	}
	if log := read(t, root, ".gtt/cli/operations.log"); !strings.Contains(log, `"operation":"artifact.next-id"`) ||
		!strings.Contains(log, `"requested":"ADR-001"`) || !strings.Contains(log, `"id":"ADR-002"`) {
		t.Errorf("a resolved collision is recorded in the operation log: %s", log)
	}
	gtt(t, root, "method", "set", "hard").must(t, 0)
	doc = gtt(t, root, "artifact", "next-id", "--kind", "adr", "--json").must(t, 0).json(t)
	if doc["automatic"] != false || doc["resolution"] != "propose" {
		t.Errorf("Hard: the id is proposed for confirmation: %v", doc)
	}
	gtt(t, root, "artifact", "next-id", "--kind", "nonsense").must(t, 2)
}

// Team: what the Bootstrap leaves to team policy follows the team's working
// agreements; with nothing declared it is proposed and confirmed.
func TestTeamPolicyFollowsTheWorkingAgreements(t *testing.T) {
	root := installed(t, nil, "--ade", "claude", "--language", "en", "--method", "team", "--no-sources", "--no-questionnaire")
	if doc := gtt(t, root, "artifact", "next-id", "--kind", "adr", "--json").must(t, 0).json(t); doc["automatic"] != false {
		t.Errorf("Team with nothing declared proposes the id: %v", doc)
	}
	write(t, root, "gtt-domain/working-agreements.md", "# Working agreements\n\n"+
		"> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md\n\n```gtt-preferences\n"+
		"TA-01 | team | policy.identity_resolution | automatic\n"+
		"TA-02 | team | code reviews | two reviewers for every change\n```\n")
	if doc := gtt(t, root, "artifact", "next-id", "--kind", "adr", "--json").must(t, 0).json(t); doc["automatic"] != true {
		t.Errorf("the team declared automatic id resolution: %v", doc)
	}
	if out := gtt(t, root, "method", "show").must(t, 0).out; !strings.Contains(out, "identity_resolution: automatic") {
		t.Errorf("method show must list what the team declared:\n%s", out)
	}
	// A new governed artifact is registered by the deterministic upkeep.
	gtt(t, root, "sync").must(t, 0)
	gtt(t, root, "validate", "--ci").must(t, 0)
}
