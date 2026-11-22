package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// sentinel runs the real governed-sentinel hook of the installed project the way Claude Code does around an agent's command.
func sentinel(t *testing.T, root, mode string) string {
	t.Helper()
	py, err := exec.LookPath("python3")
	if err != nil {
		if py, err = exec.LookPath("python"); err != nil {
			t.Skip("no python to run the sentinel hook")
		}
	}
	cmd := exec.Command(py, filepath.Join(root, ".claude/hooks/governed-sentinel.py"), mode)
	cmd.Dir = root
	cmd.Stdin = strings.NewReader("{}")
	cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+root)
	out, _ := cmd.CombinedOutput()
	return strings.TrimSpace(string(out))
}

// An update writes the scripts and hooks that the sentinel of a running agent session watches. When an agent's command is in flight
// meanwhile, the sentinel compares before and after: without the update's window it read the new scripts as the agent's, put some of
// them back and archived the rest, and the project was left half updated (the new contracts beside the old scripts, the ledger ahead of
// the files). The CLI records the window itself, in the Bootstrap's format and place.
func TestUpdateIsNotRevertedByTheSentinelOfARunningAgentSession(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the sentinel's state lives under LOCALAPPDATA there; covered on Linux")
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", "")
	root := installed(t, nil, "--ade", "claude", "--primary", "claude", "--language", "es", "--method", "hard")
	if !exists(root, ".claude/hooks/governed-sentinel.py") {
		t.Skip("this Bootstrap ships no sentinel")
	}
	write(t, root, "gtt-domain/.frozen", "frozen\n") // the sentinel watches a frozen project

	script := ".gtt/scripts/gtt-validate.sh"
	before := read(t, root, script)

	// Control: the same window of time with an edit that is nobody's act is put back. The test would be vacuous if it were not.
	sentinel(t, root, "pre")
	write(t, root, script, before+"\n# an agent command wrote this\n")
	if out := sentinel(t, root, "post"); !strings.Contains(out, "put it back") && !strings.Contains(out, "GTT put it back") {
		t.Fatalf("control: the sentinel must put back a write outside any human act, it said:\n%s", out)
	}
	if read(t, root, script) != before {
		t.Fatal("control: the sentinel did not restore the script")
	}

	archived := func() int {
		entries, _ := filepath.Glob(filepath.Join(root, "gtt-domain/proposals/withdrawn/*-reverted"))
		return len(entries)
	}
	controlArchived := archived() // the control's write is archived, as designed
	if controlArchived != 1 {
		t.Fatalf("control: its write must be archived once, got %d", controlArchived)
	}

	// The update: an agent command begins, the human's `gtt update` runs and writes a script, the agent's command ends.
	to := next(t, catalogVersion(t), 1)
	target := newer(t, to)
	data, _ := os.ReadFile(filepath.Join(target, script))
	if err := os.WriteFile(filepath.Join(target, script), append(data, []byte("\n# a line the new release ships\n")...), 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel(t, root, "pre")
	gtt(t, root, "update", "--bootstrap", target).must(t, 0)
	if out := sentinel(t, root, "post"); out != "" {
		t.Errorf("the update is the human's act: the sentinel must stay silent, it said:\n%s", out)
	}
	if !strings.Contains(read(t, root, script), "a line the new release ships") {
		t.Error("the script the update wrote was put back by the sentinel")
	}
	if archived() != controlArchived {
		t.Error("nothing of the update may be archived as reverted")
	}
	if exists(root, ".gtt-update-backup") {
		t.Error("the update left its backup")
	}
}

// An update that was cut short by the sentinel leaves the ledger ahead of a script: it holds the release's hash and the file is the old
// one. That is not an edit of the person's, and the next update finishes it - the evidence is the copy of the release's file that the
// sentinel archived as a write it reverted. An edit of the person's, with no such copy, is still kept as theirs.
func TestUpdateFinishesWhatTheSentinelCutShort(t *testing.T) {
	root := installed(t, nil, "--ade", "claude", "--primary", "claude", "--language", "es", "--method", "hard")
	script, mine := ".gtt/scripts/gtt-validate.sh", ".gtt/scripts/gtt-skills.sh"
	if !exists(root, script) || !exists(root, mine) {
		t.Fatal("the scripts the test edits are not in this Bootstrap")
	}
	oldValidate, oldMine := read(t, root, script), read(t, root, mine)

	to := next(t, catalogVersion(t), 1)
	target := newer(t, to)
	for _, rel := range []string{script, mine} {
		data, _ := os.ReadFile(filepath.Join(target, rel))
		if err := os.WriteFile(filepath.Join(target, rel), append(data, []byte("\n# shipped by "+to+"\n")...), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	gtt(t, root, "update", "--bootstrap", target).must(t, 0)
	shippedValidate := read(t, root, script)
	if !strings.Contains(shippedValidate, "shipped by "+to) {
		t.Fatal("the update did not install the new script")
	}

	// What the sentinel of a running agent session did to the first one: archived the release's file and put the old one back. The second
	// is the person's own edit of a script (nothing archived): its ledger entry is the release's and the file is not.
	write(t, root, "gtt-domain/proposals/withdrawn/2026-11-21T231759-reverted/"+script, shippedValidate)
	write(t, root, script, oldValidate)
	write(t, root, mine, oldMine+"\n# my own change\n")

	gtt(t, root, "update", "--bootstrap", target).must(t, 0)
	if got := read(t, root, script); got != shippedValidate {
		t.Errorf("the next update must finish what the sentinel cut short; the script is:\n%s", got)
	}
	if got := read(t, root, mine); !strings.Contains(got, "my own change") || strings.Contains(got, "shipped by "+to) {
		t.Errorf("an edit of the person's, with nothing archived, stays theirs:\n%s", got)
	}
}
