package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Simple execution: `gtt run` lists what the agent prepared, `gtt run <n>`
// shows the steps, asks the human once when the run changes something, runs
// the sealed script through the Bootstrap's run.execute and archives it. The
// plan, the script and the review are the Bootstrap's; a Bootstrap without
// them is skipped.

func runProject(t *testing.T) string {
	t.Helper()
	if !exists(pristine, ".gtt/contract/run.json") {
		t.Skip("this Bootstrap does not bring simple execution")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	return root
}

// prepare does what the agent does: write the plan, then generate the script.
func prepare(t *testing.T, root string, n, changes, steps string) {
	t.Helper()
	write(t, root, "gtt-domain/run/RUN-00"+n+"-demo.md",
		"# RUN-00"+n+"\n\n- **What it does:** Prints a greeting\n- **Why:** STORY-001\n- **Changes:** "+changes+"\n\n```gtt-run\n"+steps+"\n```\n")
	cmd := exec.Command("bash", ".gtt/scripts/gtt-run.sh", "generate", n, "--apply")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate failed: %v\n%s", err, out)
	}
}

func TestRunListsPendingAndShowsWithoutRunning(t *testing.T) {
	root := runProject(t)
	if out := gtt(t, root, "run").must(t, 0).out; !strings.Contains(out, "No pending runs") {
		t.Errorf("nothing pending:\n%s", out)
	}
	prepare(t, root, "1", "environment", "step: Greet | echo hello-from-run")
	out := gtt(t, root, "run").must(t, 0).out
	if !strings.Contains(out, "gtt run 1") || !strings.Contains(out, "Prints a greeting") || !strings.Contains(out, "environment") {
		t.Errorf("one line per pending run:\n%s", out)
	}
	show := gtt(t, root, "run", "show", "1").must(t, 0).out
	if !strings.Contains(show, "1. Greet") || !strings.Contains(show, "echo hello-from-run") {
		t.Errorf("show lists the steps:\n%s", show)
	}
	if exists(root, "gtt-domain/run/log.jsonl") {
		t.Error("listing and showing run nothing")
	}
	gtt(t, root, "run", "check").must(t, 0)
	gtt(t, root, "run", "9").must(t, 3) // there is no such run
	gtt(t, root, "run", "banana").must(t, 3)
}

func TestRunThatChangesSomethingNeedsTheHumanAtATerminal(t *testing.T) {
	root := runProject(t)
	prepare(t, root, "1", "environment", "step: Greet | echo hello-from-run\nstep: Bye | echo bye")
	before := treeHash(t, root)
	gtt(t, root, "run", "1").must(t, 3)          // no terminal: refused
	gtt(t, root, "run", "1", "--yes").must(t, 3) // --yes does not stand in for the human
	if treeHash(t, root) != before {
		t.Error("a refused run changes nothing")
	}
	gttIn(t, root, "n\n", "run", "1").must(t, 4) // declined
	if exists(root, "gtt-domain/run/log.jsonl") {
		t.Error("a declined run leaves no log")
	}
	r := gttIn(t, root, "y\n", "run", "RUN-001").must(t, 0)
	if !strings.Contains(r.out, "hello-from-run") || !strings.Contains(r.out, "[2/2] Bye") {
		t.Errorf("the steps ran and were announced:\n%s", r.out)
	}
	if log := read(t, root, "gtt-domain/run/log.jsonl"); !strings.Contains(log, `"id": "RUN-001"`) || !strings.Contains(log, `"exit": 0`) {
		t.Errorf("one line in the log:\n%s", log)
	}
	if !exists(root, "gtt-domain/run/done/RUN-001-demo.md") || !exists(root, "gtt-domain/run/done/RUN-001-demo.sh") || exists(root, "gtt-domain/run/RUN-001-demo.md") {
		t.Error("the plan and the script move to done/")
	}
}

func TestRunThatChangesNothingRunsWithoutATerminal(t *testing.T) {
	root := runProject(t)
	prepare(t, root, "1", "none", "step: Look | echo only-reading")
	r := gtt(t, root, "run", "1").must(t, 0)
	if !strings.Contains(r.out, "only-reading") || !exists(root, "gtt-domain/run/done/RUN-001-demo.md") {
		t.Errorf("a read-only run needs no question:\n%s\n%s", r.out, r.err)
	}
}

func TestRunRefusesAnEditedScriptAndAFailingStepStaysPending(t *testing.T) {
	root := runProject(t)
	prepare(t, root, "1", "none", "step: Look | echo fine")
	f, err := os.OpenFile(filepath.Join(root, "gtt-domain/run/RUN-001-demo.sh"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("echo edited\n")
	f.Close()
	if r := gtt(t, root, "run", "1").must(t, 5); !strings.Contains(r.err, "RUN-05") {
		t.Errorf("the seal is named:\n%s", r.err)
	}
	gtt(t, root, "run", "check").must(t, 1)
	prepare(t, root, "1", "none", "step: Break | false\nstep: Never | echo unreachable")
	r := gtt(t, root, "run", "1").must(t, 1)
	if strings.Contains(r.out, "[2/2]") || !exists(root, "gtt-domain/run/RUN-001-demo.md") {
		t.Errorf("a failing step stops the run and the plan stays pending:\n%s", r.out)
	}
}

func TestRunOnABootstrapWithoutItSaysSoInOneLine(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	os.Remove(filepath.Join(root, ".gtt/contract/run.json"))
	edit(t, filepath.Join(root, ".gtt/contract/operations.json"), func(d map[string]any) {
		ops := d["operations"].(map[string]any)
		for k := range ops {
			if strings.HasPrefix(k, "run.") {
				delete(ops, k)
			}
		}
	})
	r := gtt(t, root, "run")
	if r.code != 2 || !strings.Contains(r.err, "does not offer simple execution") {
		t.Errorf("one line, and the way out:\nexit %d\n%s", r.code, r.err)
	}
}

// Criterion 13: under Windows process creation the Bootstrap starts the bash
// the CLI resolved (GTT_BASH), so `gtt run` works where a bare "bash" is the
// WSL launcher.
func TestWindowsEmulationRunExecutesThroughTheResolvedBash(t *testing.T) {
	root := runProject(t)
	emulateWindowsProcessCreation(t)
	prepare(t, root, "1", "none", "step: Greet | echo resolved-bash-ran")
	r := gtt(t, root, "run", "1").must(t, 0)
	if !strings.Contains(r.out, "resolved-bash-ran") || strings.Contains(r.out+r.err, "Windows Subsystem for Linux") {
		t.Errorf("gtt run uses the bash the CLI handed over:\n%s\n%s", r.out, r.err)
	}
}
