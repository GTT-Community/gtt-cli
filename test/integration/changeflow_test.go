package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The change flow during development: gtt change, gtt proposals, gtt promote,
// gtt undo. Every step goes through the Bootstrap; the CLI decides nothing.

func changeID(t *testing.T, root string) string {
	t.Helper()
	id, _ := gtt(t, root, "change", "show", "--json").must(t, 0).json(t)["id"].(string)
	return id
}

func answer(t *testing.T, root, id string) {
	t.Helper()
	write(t, root, "gtt-domain/proposals/PROPOSAL-sessions.md", "# PROPOSAL - Sessions in Redis\n\n"+
		"> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md\n\nRequest: "+id+"\n")
}

func TestAChangeRequestIsRecordedAndTheAgentIsToldToProcessIt(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	if r := gtt(t, root, "change", "Sessions in Redis").must(t, 3); !strings.Contains(r.err, "what to change and why") {
		t.Errorf("without a terminal the change and its reason are required:\n%s", r.err)
	}
	before := read(t, root, "gtt-domain/change-request.md")
	gtt(t, root, "change", "Sessions in Redis", "--reason", "Scale out").must(t, 3) // a human confirmation is needed
	if read(t, root, "gtt-domain/change-request.md") != before {
		t.Error("nothing is written before the human confirms")
	}
	gtt(t, root, "change", "Sessions in Redis", "--reason", "Scale out", "--priority", "high", "--yes").must(t, 0)
	st := gtt(t, root, "change", "show", "--json").must(t, 0).json(t)
	id, _ := st["id"].(string)
	if st["state"] != "pending" || !strings.HasPrefix(id, "CR-") {
		t.Fatalf("the request is pending with its id: %v", st)
	}
	if out := gtt(t, root, "resume").must(t, 0).out; !strings.Contains(out, "Change request "+id+" pending") || !strings.Contains(out, "never apply it") {
		t.Errorf("the session tells the agent to process it, never to apply it:\n%s", out)
	}
	if r := gtt(t, root, "change", "Something else", "--reason", "x", "--yes").must(t, 5); !strings.Contains(r.err, "still pending") {
		t.Errorf("a pending request is never replaced in silence:\n%s", r.err)
	}
	answer(t, root, id)
	if st := gtt(t, root, "change", "show", "--json").must(t, 0).json(t); st["state"] != "processed" {
		t.Errorf("a proposal citing the id answers the request: %v", st["state"])
	}
	if out := gtt(t, root, "resume").must(t, 0).out; strings.Contains(out, "Change request "+id) {
		t.Errorf("once answered, the agent is no longer told to process it:\n%s", out)
	}
	list := gtt(t, root, "proposals").must(t, 0).out
	if !strings.Contains(list, "PROPOSAL-sessions.md") || !strings.Contains(list, id) {
		t.Errorf("gtt proposals lists the proposal and the request it answers:\n%s", list)
	}
}

func TestInteractiveChangeAsksWhatAndWhy(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	// what; why; trigger, scope, impact, risk empty; priority 2 (high); record it; do not open the ADE.
	r := gttIn(t, root, "Sessions in Redis\nScale out\n\n\n\n\n2\ny\nn\n", "change").must(t, 0)
	if !strings.Contains(r.err, "What needs to change?") || !strings.Contains(r.err, "Record this change request?") {
		t.Errorf("the request is asked, then confirmed:\n%s", r.err)
	}
	st := gtt(t, root, "change", "show", "--json").must(t, 0).json(t)
	fields := st["fields"].(map[string]any)
	if fields["Change"] != "Sessions in Redis" || fields["Reason"] != "Scale out" || fields["Priority"] != "high" {
		t.Errorf("the answers are recorded as given: %v", fields)
	}
}

func TestPromoteAndOneStepBack(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	glossary := read(t, root, "gtt-domain/context/glossary.md")
	write(t, root, "staged-glossary.md", glossary+"\nterm: a decided term\n")
	if r := gtt(t, root, "promote", "nothing-staged", "--yes"); r.code == 0 {
		t.Error("an unknown set is refused")
	}
	stage(t, root, "glossary", "staged-glossary.md=gtt-domain/context/glossary.md")
	if out := gtt(t, root, "proposals").must(t, 0).out; !strings.Contains(out, "ready to promote  promotion-set  glossary") {
		t.Errorf("a staged set is listed as ready to promote:\n%s", out)
	}
	gtt(t, root, "promote", "glossary").must(t, 3) // the human confirms a promotion
	if read(t, root, "gtt-domain/context/glossary.md") != glossary {
		t.Fatal("nothing is promoted without the human's confirmation")
	}
	gtt(t, root, "promote", "glossary", "--yes").must(t, 0)
	if !strings.Contains(read(t, root, "gtt-domain/context/glossary.md"), "a decided term") {
		t.Fatal("the confirmed set is applied")
	}
	gtt(t, root, "undo", "--yes").must(t, 0)
	if read(t, root, "gtt-domain/context/glossary.md") != glossary {
		t.Error("undo puts back what the last promotion replaced")
	}
	if r := gtt(t, root, "undo", "--yes").must(t, 5); !strings.Contains(r.err, "no promotion to undo") || !strings.Contains(r.err, "gtt change") {
		t.Errorf("going further back is a change request:\n%s", r.err)
	}
}

func TestAnEditedPromotionIsNotUndone(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	glossary := read(t, root, "gtt-domain/context/glossary.md")
	write(t, root, "staged-glossary.md", glossary+"\nterm: a decided term\n")
	stage(t, root, "glossary", "staged-glossary.md=gtt-domain/context/glossary.md")
	gtt(t, root, "promote", "glossary", "--yes").must(t, 0)
	write(t, root, "gtt-domain/context/glossary.md", read(t, root, "gtt-domain/context/glossary.md")+"edited later\n")
	if r := gtt(t, root, "undo", "--yes").must(t, 5); !strings.Contains(r.err, "changed after the promotion") {
		t.Errorf("undoing would discard a later edit: refused\n%s", r.err)
	}
	if !strings.Contains(read(t, root, "gtt-domain/context/glossary.md"), "edited later") {
		t.Error("the later edit is kept")
	}
}

func TestAWithdrawnProposalIsArchived(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	gtt(t, root, "change", "Sessions in Redis", "--reason", "Scale out", "--yes").must(t, 0)
	answer(t, root, changeID(t, root))
	gtt(t, root, "proposals", "withdraw", "PROPOSAL-sessions.md", "--yes").must(t, 0)
	if exists(root, "gtt-domain/proposals/PROPOSAL-sessions.md") {
		t.Error("a withdrawn proposal leaves the drafts")
	}
	matches, _ := filepath.Glob(filepath.Join(root, "gtt-domain", "proposals", "withdrawn", "*PROPOSAL-sessions.md"))
	if len(matches) != 1 {
		t.Fatalf("it is archived, never deleted: %v", matches)
	}
	if data, _ := os.ReadFile(matches[0]); !strings.Contains(string(data), "Withdrawn:") {
		t.Error("the archive says it was withdrawn")
	}
	if st := gtt(t, root, "change", "show", "--json").must(t, 0).json(t); st["state"] != "pending" {
		t.Errorf("the request it answered is open again: %v", st["state"])
	}
}

// stage prepares a promotion set the way an agent does.
func stage(t *testing.T, root, name string, pairs ...string) {
	t.Helper()
	args := append([]string{filepath.Join(root, ".gtt", "scripts", "gtt-stage.sh"), name, "--reason", "test set"}, pairs...)
	cmd := exec.Command("bash", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("staging failed: %v\n%s", err, out)
	}
}
