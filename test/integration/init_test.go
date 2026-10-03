package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func methodology(t *testing.T, root string) map[string]any {
	return gtt(t, root, "method", "show", "--json").must(t, 0).json(t)["plan"].(map[string]any)
}

func TestInitInstallsOnlyWhatTheHumanChose(t *testing.T) {
	// Kiro is detectable (its directory exists) but is not selected.
	root := project(t, map[string]string{".kiro/settings.json": "{}\n", "docs/requirements.md": "# requirements\n"})
	initProject(t, root, pristine, "--ade", "claude,copilot", "--primary", "copilot", "--language", "es", "--method", "hard", "--sources", "docs/requirements.md")

	for _, p := range []string{".gtt/contract/release.json", "gtt-domain/context/stack.md", "AGENTS.md", ".claude/CLAUDE.md", ".copilot/copilot-instructions.md", ".gtt/cli/state.json"} {
		if !exists(root, p) {
			t.Errorf("missing %s", p)
		}
	}
	if exists(root, ".kiro/steering") {
		t.Error("detected is not participating: the Kiro overlay must not be installed")
	}
	if read(t, root, ".kiro/settings.json") != "{}\n" {
		t.Error("a host file was touched")
	}
	st := gtt(t, root, "status", "--fast", "--json").must(t, 0).json(t)
	ade := st["status"].(map[string]any)["ade"].(map[string]any)
	if ade["primary"] != "copilot" || len(ade["participating"].([]any)) != 2 {
		t.Errorf("exactly one Primary among the participating ADEs: %v", ade)
	}
	if !strings.Contains(strings.Join(toStrings(ade["excluded"]), ","), "kiro") {
		t.Errorf("declined ADEs are recorded as excluded: %v", ade["excluded"])
	}
	plan := methodology(t, root)
	if plan["profile"] != "hard" || plan["language"] != "es" || plan["selected"] != true {
		t.Errorf("the selection is passed to the Bootstrap as chosen: %v %v", plan["profile"], plan["language"])
	}
	if got := gtt(t, root, "sources").must(t, 0).out; !strings.Contains(got, "docs/requirements.md") || strings.Contains(got, "Pendiente") {
		t.Errorf("the selected source must be applied and validated: %s", got)
	}
	gtt(t, root, "validate", "--ci").must(t, 0)
	if exists(root, ".gtt-init-transaction.json") {
		t.Error("the transaction journal must be gone after a successful init")
	}
}

func toStrings(v any) []string {
	var out []string
	for _, x := range v.([]any) {
		out = append(out, x.(string))
	}
	return out
}

func TestEveryPlanIsPassedThroughUninterpreted(t *testing.T) {
	for _, plan := range []string{"light", "medium", "hard", "team"} {
		t.Run(plan, func(t *testing.T) {
			root := project(t, nil)
			initProject(t, root, pristine, "--ade", "claude", "--language", "en", "--method", plan, "--no-sources", "--no-questionnaire")
			if got := methodology(t, root)["profile"]; got != plan {
				t.Errorf("got %v", got)
			}
		})
	}
}

func TestInitNeverInfersAHumanDecision(t *testing.T) {
	cases := map[string][]string{
		"participating ADEs": {"--language", "en", "--method", "light", "--no-sources"},
		"Method Plan":        {"--ade", "claude", "--language", "en", "--no-sources"},
		"language":           {"--ade", "claude", "--method", "light", "--no-sources"},
		"Primary ADE":        {"--ade", "claude,copilot", "--language", "en", "--method", "light", "--no-sources"},
		"sources":            {"--ade", "claude", "--language", "en", "--method", "light"},
	}
	for name, flags := range cases {
		t.Run(name, func(t *testing.T) {
			root := project(t, map[string]string{"design.md": "# design\n"})
			before := treeHash(t, root)
			r := gtt(t, root, append([]string{"init", "--bootstrap", pristine, "--yes"}, flags...)...).must(t, 3)
			if !strings.Contains(r.err, "No project files were modified") {
				t.Errorf("the refusal must say the project is untouched:\n%s", r.err)
			}
			if treeHash(t, root) != before {
				t.Error("the project was modified")
			}
		})
	}
}

func TestInteractiveInitAsksAndRecords(t *testing.T) {
	root := project(t, map[string]string{"docs/requirements.md": "# r\n", "docs/architecture.md": "# a\n"})
	// not a project? yes; ADEs 1,3 (claude, copilot); primary 2 (copilot); language 2 (es);
	// sources 1,2; plan 1 (light); proceed; then exit the handoff menu.
	answers := "y\n1,3\n2\n2\n1,2\n1\ny\n2\n"
	r := gttIn(t, root, answers, "init", "--bootstrap", pristine).must(t, 0)
	for _, want := range []string{"detection is not participation", "Select the Primary ADE", "Light Method", "Candidate source documents"} {
		if !strings.Contains(r.err, want) {
			t.Errorf("prompt %q was not shown", want)
		}
	}
	plan := methodology(t, root)
	if plan["profile"] != "light" || plan["language"] != "es" {
		t.Errorf("recorded %v %v", plan["profile"], plan["language"])
	}
	if got := gtt(t, root, "sources", "--json").must(t, 0).json(t)["sources"].([]any); len(got) != 2 {
		t.Errorf("both selected sources must be recorded: %v", got)
	}
}

func TestNoDocumentOffersTheBootstrapQuestionnaire(t *testing.T) {
	root := project(t, nil)
	r := initProject(t, root, pristine, "--ade", "claude", "--language", "en", "--method", "medium", "--questionnaire")
	out := "gtt-domain/proposals/bootstrap/initial-design-questionnaire.md"
	if !strings.Contains(r.out, "Questionnaire created:\n"+out) {
		t.Errorf("the location must be reported:\n%s", r.out)
	}
	tpl := read(t, pristine, ".gtt/scaffold/templates/gtt-initial-design-questionnaire.md")
	if read(t, root, out) != tpl {
		t.Error("the working copy must be the Bootstrap's template, byte for byte")
	}
	gtt(t, root, "validate").must(t, 0)
}

func TestDecliningTheQuestionnaireDoesNotPretendADesignExists(t *testing.T) {
	root := project(t, nil)
	r := initProject(t, root, pristine, defaults...)
	if !strings.Contains(r.out, "no initial design yet") {
		t.Errorf("got:\n%s", r.out)
	}
	if exists(root, "gtt-domain/proposals/bootstrap/initial-design-questionnaire.md") || exists(root, "SOURCE-BRIEF.md") {
		t.Error("nothing may be created to stand in for a design")
	}
}

func TestInitIsIdempotentAndNeverReinstalls(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	before := treeHash(t, root)
	r := gtt(t, root, "init", "--bootstrap", pristine, "--ade", "kiro", "--method", "light", "--language", "es", "--no-sources", "--yes").must(t, 0)
	if !strings.Contains(r.out, "already initialized") || treeHash(t, root) != before {
		t.Errorf("a second init must change nothing:\n%s", r.out)
	}
}

func TestNamespaceConflictIsReportedNotMerged(t *testing.T) {
	root := project(t, map[string]string{"AGENTS.md": "my own agents file\n", "gtt-domain/mine.txt": "x\n"})
	before := treeHash(t, root)
	r := gtt(t, root, append([]string{"init", "--bootstrap", pristine, "--yes"}, defaults...)...).must(t, 5)
	if !strings.Contains(r.err, "AGENTS.md") || !strings.Contains(r.err, "gtt-domain") || treeHash(t, root) != before {
		t.Errorf("conflict handling:\n%s", r.err)
	}
}

func TestFailedInstallRollsBackCompletely(t *testing.T) {
	// An overlay file already exists: the Bootstrap refuses to overwrite it,
	// and the core that was already committed must be removed again.
	root := project(t, map[string]string{".claude/CLAUDE.md": "my own instructions\n"})
	before := treeHash(t, root)
	r := gtt(t, root, append([]string{"init", "--bootstrap", pristine, "--yes"}, defaults...)...)
	if r.code == 0 {
		t.Fatal("must fail")
	}
	if treeHash(t, root) != before {
		t.Errorf("a failed init must leave no half-installed project:\n%s", r.err)
	}
	if read(t, root, ".claude/CLAUDE.md") != "my own instructions\n" {
		t.Error("the host file was overwritten")
	}
}

func TestInterruptedInitIsDetectedFromRecordedState(t *testing.T) {
	root := project(t, nil)
	write(t, root, ".gtt-init-transaction.json", `{"schema":1,"state":"STAGING","core_paths":[".gtt","gtt-domain","AGENTS.md","readme-gtt.md","readme-gtt.es.md"],"completed":[]}`)
	write(t, root, ".gtt/half", "x")
	r := gtt(t, root, append([]string{"init", "--bootstrap", pristine, "--yes"}, defaults...)...).must(t, 5)
	if !strings.Contains(r.err, "Incomplete GTT initialization") || !strings.Contains(r.err, "STAGING") {
		t.Errorf("got:\n%s", r.err)
	}
	gtt(t, root, "init", "--rollback").must(t, 0)
	if exists(root, ".gtt") || exists(root, ".gtt-init-transaction.json") || !exists(root, "src/main.go") {
		t.Error("rollback must remove the partial installation and nothing else")
	}
}

func TestUnsafeSourcePathIsRefused(t *testing.T) {
	root := project(t, nil)
	os.WriteFile(filepath.Join(filepath.Dir(root), "outside.md"), []byte("x"), 0o644)
	gtt(t, root, "init", "--bootstrap", pristine, "--yes", "--ade", "claude", "--language", "en", "--method", "light", "--sources", "../outside.md").must(t, 3)
	if exists(root, ".gtt") {
		t.Error("nothing may be installed")
	}
}

// The ADE matrix: whatever combination the human chooses is installed, with
// exactly one Primary, and nothing that was not chosen.
func TestADEMatrix(t *testing.T) {
	overlay := map[string]string{"claude": ".claude/CLAUDE.md", "copilot": ".copilot/copilot-instructions.md", "kiro": ".kiro/steering"}
	for name, c := range map[string]struct{ ades, primary string }{
		"copilot alone":                  {"copilot", "copilot"},
		"claude and codex":               {"claude,codex", "claude"},
		"claude, copilot and codex":      {"claude,copilot,codex", "claude"},
		"several with a copilot primary": {"claude,copilot,codex", "copilot"},
	} {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(c.ades, "claude") {
				// Bootstrap 1.2.0 validates every session adapter under
				// .gtt/session-adapters/ whether or not its ADE participates,
				// so a project without Claude Code fails validation. The check
				// is the Bootstrap's; the CLI must not patch around it.
				t.Skip("blocked by the Bootstrap: gtt-validate.sh checks the Claude session adapter of a non-participating ADE")
			}
			root := project(t, nil)
			initProject(t, root, pristine, "--ade", c.ades, "--primary", c.primary, "--language", "en", "--method", "light", "--no-sources", "--no-questionnaire")
			chosen := strings.Split(c.ades, ",")
			st := gtt(t, root, "status", "--fast", "--json").must(t, 0).json(t)
			ade := st["status"].(map[string]any)["ade"].(map[string]any)
			if got := toStrings(ade["participating"]); ade["primary"] != c.primary || len(got) != len(chosen) {
				t.Errorf("primary %v, participating %v; want %s among %v", ade["primary"], got, c.primary, chosen)
			}
			if !exists(root, "AGENTS.md") {
				t.Error("the portable core is always installed")
			}
			for id, path := range overlay {
				if want := strings.Contains(","+c.ades+",", ","+id+","); exists(root, path) != want {
					t.Errorf("overlay of %s (%s): installed %v, chosen %v", id, path, !want, want)
				}
			}
			gtt(t, root, "validate", "--ci").must(t, 0)
		})
	}
}
