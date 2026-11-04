package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Pending events: what waits for the human is one small file in
// gtt-domain/pending/. The Bootstrap creates and checks the files; the CLI lists
// them (`gtt pending`) and writes the human's answer (`gtt answer`) through the
// declared operations. A Bootstrap without them is skipped.

func pendingProject(t *testing.T) string {
	t.Helper()
	if !exists(pristine, ".gtt/contract/pending.json") {
		t.Skip("this Bootstrap does not bring pending events")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	return root
}

// agentAsks does what the agent does: create an event through the Bootstrap.
func agentAsks(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{".gtt/scripts/gtt-pending.sh"}, append(args, "--apply")...)...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gtt-pending.sh failed: %v\n%s", err, out)
	}
}

func askBilling(t *testing.T, root string) {
	agentAsks(t, root, "ask", "--type", "money", "--question", "Manual billing with an invoice in v1?", "--why", "Decides if online billing enters v1.",
		"--option", "Yes, manual with invoice", "--option", "No, online billing from v1", "--recommended", "1")
}

func decideUnencrypted(t *testing.T, root string) {
	write(t, root, "src/sync/doc.go", "package sync\n")
	agentAsks(t, root, "decide", "--affects", "security", "--title", "Phone data goes unencrypted in v1", "--what", "Local data stays unencrypted until v1.1.",
		"--why", "Encrypting needs a native build.", "--where", "src/sync/")
}

func TestPendingIsEmptyByDefaultAndListsWhatWaits(t *testing.T) {
	root := pendingProject(t)
	if out := gtt(t, root, "pending").must(t, 0).out; !strings.Contains(out, "Nothing pending") {
		t.Errorf("an empty folder says so:\n%s", out)
	}
	askBilling(t, root)
	decideUnencrypted(t, root)
	out := gtt(t, root, "pending").must(t, 0).out
	if !strings.Contains(out, "ASK") || !strings.Contains(out, "Manual billing with an invoice in v1?") || !strings.Contains(out, "recommended 1") ||
		!strings.Contains(out, "DEC") || !strings.Contains(out, "Phone data goes unencrypted in v1") {
		t.Errorf("one line per event, the recommended option shown:\n%s", out)
	}
}

func TestAnswerShowsWhatItWritesAndWritesTheHumansChoice(t *testing.T) {
	root := pendingProject(t)
	askBilling(t, root)
	r := gtt(t, root, "answer", "1", "1").must(t, 0)
	if !strings.Contains(r.out, "ASK-001") || !strings.Contains(r.out, "1. Yes, manual with invoice") {
		t.Errorf("it shows the option it writes:\n%s", r.out)
	}
	if text := read(t, root, "gtt-domain/pending/ASK-001.md"); !strings.Contains(text, "Answer: 1\n") {
		t.Errorf("the answer is in the file:\n%s", text)
	}
	if out := gtt(t, root, "pending").must(t, 0).out; !strings.Contains(out, "answered, not applied yet") {
		t.Errorf("an answer nobody applied is shown as such:\n%s", out)
	}
}

func TestAnswerRefusesWhatTheBootstrapRefuses(t *testing.T) {
	root := pendingProject(t)
	askBilling(t, root)
	before := read(t, root, "gtt-domain/pending/ASK-001.md")
	gtt(t, root, "answer", "1", "3").must(t, 1)                                 // no such option
	gtt(t, root, "answer", "1", "sk-abcdefghijklmnopqrstuvwxyz0123").must(t, 1) // a key
	gtt(t, root, "answer", "9", "1").must(t, 3)                                 // no such event
	gtt(t, root, "answer", "banana", "1").must(t, 3)
	gtt(t, root, "answer", "1").must(t, 3) // an answer is missing
	if read(t, root, "gtt-domain/pending/ASK-001.md") != before {
		t.Error("a refused answer changes nothing")
	}
}

func TestAnswerOkApprovesEveryDecisionAndMarksThem(t *testing.T) {
	root := pendingProject(t)
	askBilling(t, root)
	decideUnencrypted(t, root)
	r := gtt(t, root, "answer", "ok").must(t, 0)
	if !strings.Contains(r.out, "DEC-002") || strings.Contains(r.out, "ASK-001") {
		t.Errorf("only decisions are approved:\n%s", r.out)
	}
	if text := read(t, root, "gtt-domain/pending/DEC-002.md"); !strings.Contains(text, "Attended:") {
		t.Errorf("an approved decision is marked at once:\n%s", text)
	}
	if text := read(t, root, "gtt-domain/pending/ASK-001.md"); strings.Contains(text, "Answer: ") {
		t.Errorf("a question is not answered by `ok`:\n%s", text)
	}
	if out := gtt(t, root, "answer", "ok").must(t, 0).out; !strings.Contains(out, "No decision is pending") {
		t.Errorf("nothing left to approve:\n%s", out)
	}
}

func TestPendingOnABootstrapWithoutItSaysSoInOneLine(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	os.Remove(filepath.Join(root, ".gtt/contract/pending.json"))
	edit(t, filepath.Join(root, ".gtt/contract/operations.json"), func(d map[string]any) {
		ops := d["operations"].(map[string]any)
		for k := range ops {
			if strings.HasPrefix(k, "pending.") {
				delete(ops, k)
			}
		}
	})
	for _, args := range [][]string{{"pending"}, {"answer", "1", "1"}} {
		r := gtt(t, root, args...)
		if r.code != 2 || !strings.Contains(r.err, "does not offer pending events") {
			t.Errorf("%v: one line, and the way out:\nexit %d\n%s", args, r.code, r.err)
		}
	}
}
