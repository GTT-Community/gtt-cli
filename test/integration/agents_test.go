package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Case 1: one ADE only -> detected, context OK.
func TestAgentsSingleADE(t *testing.T) {
	root := installed(t, nil)
	out := gtt(t, root, "agents", "check").must(t, 0).out
	if !strings.Contains(out, "Declared agents:\n  Claude Code") || !strings.Contains(out, "OK") || !strings.Contains(out, "No changes were made.") {
		t.Errorf("got:\n%s", out)
	}
}

// Cases 2 and 6: a declared agent's context is missing -> detected, reported,
// synchronised from the authorised source; repeating it changes nothing.
func TestAgentsMissingContextIsSynchronisedIdempotently(t *testing.T) {
	root := installed(t, nil, "--ade", "claude,copilot", "--primary", "claude", "--language", "en", "--method", "medium", "--no-sources", "--no-questionnaire")
	managed := ".copilot/copilot-instructions.md"
	original := read(t, root, managed)
	os.Remove(filepath.Join(root, managed))

	r := gtt(t, root, "agents", "check", "--ci").must(t, 1)
	if !strings.Contains(r.out, "INCONSISTENT") || exists(root, managed) {
		t.Fatalf("check is read-only and must fail on a declared agent without context:\n%s", r.out)
	}
	if s := gtt(t, root, "status", "--fast").must(t, 0).out; !strings.Contains(s, "INCONSISTENT") {
		t.Errorf("status must show it:\n%s", s)
	}
	if m := gtt(t, root, "agents", "manifest").out; !strings.Contains(m, "missing       "+managed) || !strings.Contains(m, "synchronized  .claude/CLAUDE.md") {
		t.Errorf("the manifest must tell a missing managed file from a synchronized one:\n%s", m)
	}
	// Medium: synchronising agent context is confirmed by the human.
	gtt(t, root, "agents", "sync").must(t, 3)
	gttIn(t, root, "n\n", "agents", "sync").must(t, 4)
	if exists(root, managed) {
		t.Fatal("declined: nothing may be written")
	}
	r = gttIn(t, root, "y\n", "agents", "sync").must(t, 0)
	if read(t, root, managed) != original || !strings.Contains(r.out, "copilot") {
		t.Fatalf("the managed context must be restored:\n%s", r.out)
	}
	gtt(t, root, "agents", "check", "--ci").must(t, 0)

	if trace := read(t, root, ".gtt/cli/operations.log"); !strings.Contains(trace, `"agent":"copilot"`) ||
		!strings.Contains(trace, `"adapter_version":1`) || !strings.Contains(trace, `"destination":["`+managed+`"]`) {
		t.Errorf("each synchronisation leaves agent, adapter version, source and destination:\n%s", trace)
	}
	log := filepath.Join(root, ".gtt/cli/operations.log")
	os.Remove(log)
	stable := treeHash(t, root)
	for i := 0; i < 3; i++ {
		if out := gtt(t, root, "agents", "sync").must(t, 0).out; !strings.Contains(out, "Nothing to do") {
			t.Errorf("run %d: %s", i, out)
		}
	}
	os.Remove(log)
	if treeHash(t, root) != stable {
		t.Error("repeated sync must be stable: no duplication, no rewrite")
	}
	if state := read(t, root, ".gtt/cli/agents.json"); !strings.Contains(state, "\"synchronized\"") {
		t.Errorf("the sync state must be recorded:\n%s", state)
	}
}

// Light: the same synchronisation needs no question.
func TestAgentsSyncIsAutomaticUnderLight(t *testing.T) {
	root := installed(t, nil, "--ade", "claude", "--language", "en", "--method", "light", "--no-sources", "--no-questionnaire")
	managed := ".claude/rules/implementation.md"
	os.Remove(filepath.Join(root, managed))
	gtt(t, root, "agents", "sync").must(t, 0)
	if !exists(root, managed) {
		t.Error("Light delegates safe agent context synchronisation")
	}
	if log := read(t, root, ".gtt/cli/operations.log"); !strings.Contains(log, "agents.sync") || strings.Contains(strings.ToLower(log), "token") {
		t.Errorf("the synchronisation must be traceable and carry no secrets:\n%s", log)
	}
}

// Case 3: an ADE is detected but not declared -> reported, nothing modified.
func TestAgentsUndeclaredIsReportedNotAdded(t *testing.T) {
	root := installed(t, nil)
	write(t, root, ".kiro/settings.json", "{}\n")
	r := gtt(t, root, "agents", "check", "--ci").must(t, 0)
	if !strings.Contains(r.out, "Undeclared detected agent:\n  kiro") {
		t.Errorf("got:\n%s", r.out)
	}
	gtt(t, root, "agents", "sync").must(t, 0)
	if exists(root, ".kiro/steering") {
		t.Error("an undeclared agent must not be installed")
	}
	// Incorporating it is explicit and confirmed, in every plan.
	gtt(t, root, "agents", "sync", "--add", "kiro", "--bootstrap", pristine).must(t, 3)
	gtt(t, root, "agents", "sync", "--add", "nonexistent", "--bootstrap", pristine, "--yes").must(t, 3)
	gtt(t, root, "agents", "sync", "--add", "kiro", "--bootstrap", pristine, "--yes").must(t, 0)
	if !exists(root, ".kiro/steering") || read(t, root, ".kiro/settings.json") != "{}\n" {
		t.Error("the added agent gets its overlay and the host file stays")
	}
	gtt(t, root, "agents", "check", "--ci").must(t, 0)
}

// Case 4: managed context changed locally -> detected, never overwritten silently.
func TestAgentsLocalChangeIsNotOverwritten(t *testing.T) {
	root := installed(t, nil, "--ade", "claude", "--language", "en", "--method", "light", "--no-sources", "--no-questionnaire")
	managed := ".claude/rules/implementation.md"
	write(t, root, managed, "my rewritten rules\n")
	gtt(t, root, "agents", "sync", "--bootstrap", pristine)
	if read(t, root, managed) != "my rewritten rules\n" {
		t.Fatal("a locally changed managed file was overwritten")
	}
	if m := gtt(t, root, "agents", "manifest").out; !strings.Contains(m, "changed       "+managed) {
		t.Errorf("the manifest must report the local change:\n%s", m)
	}
}

// Case 5: the human switches Primary -> re-detected, governed context untouched.
func TestSwitchingADEDoesNotAlterGovernedContext(t *testing.T) {
	root := installed(t, nil, "--ade", "claude,codex", "--primary", "claude", "--language", "en", "--method", "light", "--no-sources", "--no-questionnaire")
	governed := treeHash(t, filepath.Join(root, "gtt-domain"))
	gtt(t, root, "config", "primary-ade", "codex", "--yes").must(t, 0)
	gtt(t, root, "agents", "sync").must(t, 0)
	gtt(t, root, "agents", "check", "--ci").must(t, 0)
	if treeHash(t, filepath.Join(root, "gtt-domain")) != governed {
		t.Error("changing the ADE must not alter governed context")
	}
}

func TestSourceTrackingLifecycle(t *testing.T) {
	root := installed(t, map[string]string{"INITIAL-001.md": "# directive v1\n", "INITIAL-002.md": "# other\n"},
		"--ade", "claude", "--language", "en", "--method", "light", "--sources", "INITIAL-001.md")
	out := gtt(t, root, "sources").must(t, 0).out
	if !strings.Contains(out, "✓ Sin cambios (ya aplicada y validada): INITIAL-001.md") || !strings.Contains(out, "⚠ Pendiente: INITIAL-002.md") {
		t.Fatalf("applied by init without a second command; unselected stays pending:\n%s", out)
	}
	// Reprocessing unchanged content does nothing.
	if out = gtt(t, root, "sources", "apply").must(t, 0).out; !strings.Contains(out, "Sin cambios (ya aplicada y validada): INITIAL-001.md") {
		t.Errorf("unchanged content must not be reprocessed:\n%s", out)
	}
	// Same name, new content: pending again, then applied again.
	write(t, root, "INITIAL-001.md", "# directive v2\n")
	if out = gtt(t, root, "sources").must(t, 0).out; !strings.Contains(out, "⚠ Pendiente: INITIAL-001.md (content changed") {
		t.Fatalf("a modified MD must be detected:\n%s", out)
	}
	if out = gtt(t, root, "sources", "apply").must(t, 0).out; !strings.Contains(out, "✓ Directiva aplicada y validada: INITIAL-001.md") {
		t.Fatalf("got:\n%s", out)
	}
	if tr := read(t, root, ".gtt/cli/tracking.json"); !strings.Contains(tr, "\"previous\"") || !strings.Contains(tr, "VALIDATED") {
		t.Errorf("the processed version must be traceable:\n%s", tr)
	}
	// Naming a path selects it.
	if out = gtt(t, root, "sources", "apply", "INITIAL-002.md:requirements").must(t, 0).out; !strings.Contains(out, "✓ Directiva aplicada y validada: INITIAL-002.md") {
		t.Fatalf("got:\n%s", out)
	}
	// A failure is never recorded as applied.
	r := gtt(t, root, "sources", "apply", "MISSING.md").must(t, 1)
	if !strings.Contains(r.out, "✗ Directiva no aplicada: MISSING.md") {
		t.Errorf("got:\n%s", r.out)
	}
	if tr := read(t, root, ".gtt/cli/tracking.json"); strings.Contains(tr, "MISSING.md") {
		t.Error("a document that could not be read must not be tracked as processed")
	}
	// When validation fails, nothing is marked.
	write(t, root, "INITIAL-001.md", "# directive v3\n")
	os.Remove(filepath.Join(root, "gtt-domain", "backlog.md"))
	r = gtt(t, root, "sources", "apply").must(t, 1)
	if !strings.Contains(r.out, "✗ Directiva no aplicada: INITIAL-001.md") {
		t.Errorf("an unvalidated application must not be marked:\n%s", r.out)
	}
	if tr := read(t, root, ".gtt/cli/tracking.json"); !strings.Contains(tr, "\"state\": \"FAILED\"") {
		t.Errorf("the failed state must be recorded, never APPLIED:\n%s", tr)
	}
}

func TestSyncDoesTheDeterministicUpkeepInOneRun(t *testing.T) {
	root := installed(t, nil, "--ade", "claude", "--language", "en", "--method", "light", "--no-sources", "--no-questionnaire")
	// A new document in the proposals area is not in the index yet.
	write(t, root, "gtt-domain/proposals/PROPOSAL-x.md", "# Proposal\n\n> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md\n\ntext\n")
	gtt(t, root, "validate").must(t, 1)
	out := gtt(t, root, "sync").must(t, 0).out
	if !strings.Contains(out, "Index rebuilt") || !strings.Contains(out, "Validation") {
		t.Errorf("got:\n%s", out)
	}
	gtt(t, root, "validate").must(t, 0)
}

// When the Bootstrap declares where initial-source tracking lives (and
// creates it at install), the CLI keeps its state there instead of under
// .gtt/cli/: the Bootstrap prepares the infrastructure, the CLI keeps the state.
func TestSourceTrackingUsesTheRecordTheBootstrapDeclares(t *testing.T) {
	c := catalogCopy(t)
	manifest := filepath.Join(c, ".gtt/scaffold/manifest.yaml")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	declared := "\ntracking:\n  initial_sources: { path: .gtt/tracking/initial-sources.json, schema: 1, maintained_by: cli }\n"
	if err := os.WriteFile(manifest, append(data, declared...), 0o644); err != nil {
		t.Fatal(err)
	}
	write(t, c, ".gtt/tracking/initial-sources.json", "{\"schema\": 1, \"entries\": {}}\n")
	root := project(t, map[string]string{"INITIAL-001.md": "# directive\n"})
	initProject(t, root, c, "--ade", "claude", "--language", "en", "--method", "light", "--sources", "INITIAL-001.md", "--no-questionnaire")
	if tr := read(t, root, ".gtt/tracking/initial-sources.json"); !strings.Contains(tr, "INITIAL-001.md") || !strings.Contains(tr, "VALIDATED") {
		t.Errorf("the declared record holds the state:\n%s", tr)
	}
	if exists(root, ".gtt/cli/tracking.json") {
		t.Error("no second tracking record may be created")
	}
	if out := gtt(t, root, "sources").must(t, 0).out; !strings.Contains(out, "INITIAL-001.md") {
		t.Errorf("sources reads the declared record:\n%s", out)
	}
}
