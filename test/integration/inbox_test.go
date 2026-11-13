package integration

import (
	"os/exec"
	"strings"
	"testing"
)

// The inbox (capability inbox.v1): one pipeline for every change. The Bootstrap owns the files and the rules; the
// CLI writes, lists and decides only through the declared operations. A Bootstrap without it is skipped.

func inboxProject(t *testing.T) string {
	t.Helper()
	if !exists(pristine, ".gtt/contract/inbox.json") {
		t.Skip("this Bootstrap does not bring the inbox")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	return root
}

const form1 = `Proposed Architecture Change

Current decision: sessions live in memory
Suggested change: sessions live in Redis
Reason: horizontal scaling
Impact: one new service
Risk: operational
Affected files: gtt-domain/context/architecture.md
Alternatives considered: sticky sessions
`

// agentProposes does what the agent does: register a proposal through the engine.
func agentProposes(t *testing.T, root, slug, origin string) {
	t.Helper()
	write(t, root, "gtt-domain/proposals/drafts/"+slug+".md", form1)
	args := []string{".gtt/scripts/gtt-inbox.sh", "propose", "--file", "gtt-domain/proposals/drafts/" + slug + ".md", "--level", "arch", "--form", "1", "--slug", slug, "--apply"}
	if origin != "" {
		args = append(args, "--origin", origin)
	}
	cmd := exec.Command("bash", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gtt-inbox.sh propose failed: %v\n%s", err, out)
	}
}

func entries(t *testing.T, root string, args ...string) map[string]map[string]any {
	t.Helper()
	res := gtt(t, root, append([]string{"inbox", "--json"}, args...)...).must(t, 0).json(t)
	out := map[string]map[string]any{}
	for _, e := range res["entries"].([]any) {
		m := e.(map[string]any)
		out[m["ref"].(string)] = m
	}
	return out
}

func TestTheHumansThreeWaysToWriteEnterTheInbox(t *testing.T) {
	root := inboxProject(t)
	// an idea: the preview comes first, a refusal writes nothing, --yes writes
	if r := gtt(t, root, "idea", "add", "sincronizar parcelas"); r.code == 0 || exists(root, "docs/ideas/IDEA-001.md") {
		t.Errorf("without a terminal and without --yes nothing is written:\n%s%s", r.out, r.err)
	}
	r := gtt(t, root, "idea", "add", "sincronizar parcelas", "--yes", "--now").must(t, 0)
	if !strings.Contains(r.out, "docs/ideas/IDEA-001.md") {
		t.Errorf("the preview names what it creates:\n%s", r.out)
	}
	// from a file
	write(t, root, "nota.txt", "una nota larga\ncon varias líneas")
	gtt(t, root, "idea", "add", "--file", "nota.txt", "--yes").must(t, 0)
	if body := read(t, root, "docs/ideas/IDEA-002.md"); !strings.Contains(body, "con varias líneas") {
		t.Errorf("the file is the idea:\n%s", body)
	}
	// a request: through idea add and through the alias
	gtt(t, root, "idea", "add", "--type", "solicitud", "--change", "Sesiones en Redis", "--reason", "escalar", "--priority", "high", "--yes").must(t, 0)
	gtt(t, root, "change", "Cola en Rabbit", "--reason", "desacoplar", "--yes").must(t, 0)
	for _, ref := range []string{"IDEA-003", "IDEA-004"} {
		if body := read(t, root, "docs/ideas/"+ref+".md"); !strings.Contains(body, "Reason:") || !strings.Contains(body, "Now:") {
			t.Errorf("%s is a request:\n%s", ref, body)
		}
	}
	if gtt(t, root, "idea", "add", "--type", "solicitud", "--change", "x", "--yes").code == 0 {
		t.Error("a request without its reason is refused")
	}
	// the architect's draft
	gtt(t, root, "adr", "new", "Usar Redis para sesiones", "--yes").must(t, 0)
	if !exists(root, "gtt-domain/adr/drafts/IDEA-005-usar-redis-para-sesiones.md") {
		t.Error("the draft is in adr/drafts/")
	}
	if gtt(t, root, "adr", "new", "Usar Redis para sesiones", "--yes").code == 0 {
		t.Error("a title is unique")
	}
	// the list, by stage, with the origin filter
	all := entries(t, root)
	if len(all) != 5 || all["IDEA-005"]["stage"] != "new" || all["IDEA-001"]["now"] != true {
		t.Errorf("five new entries: %v", all)
	}
	out := gtt(t, root, "inbox").must(t, 0).out
	if !strings.Contains(out, "New (5)") || !strings.Contains(out, "IDEA-001  sincronizar parcelas") {
		t.Errorf("listed by stage:\n%s", out)
	}
	if n := len(entries(t, root, "--origin", "agent")); n != 0 {
		t.Errorf("no agent entry yet: %d", n)
	}
	if out := gtt(t, root, "inbox", "show", "idea-1").must(t, 0).out; !strings.Contains(out, "sincronizar parcelas") {
		t.Errorf("show:\n%s", out)
	}
	if gtt(t, root, "inbox", "show", "IDEA-099").code == 0 {
		t.Error("an unknown entry is an error")
	}
}

func TestParkReopenDropAndPriorityThroughTheInbox(t *testing.T) {
	root := inboxProject(t)
	gtt(t, root, "idea", "add", "una", "--yes").must(t, 0)
	gtt(t, root, "idea", "add", "otra", "--yes").must(t, 0)
	pre := treeHash(t, root)
	if r := gtt(t, root, "inbox", "park", "IDEA-001"); r.code == 0 || treeHash(t, root) != pre {
		t.Errorf("an act needs the human's confirmation, and nothing changed without it: %d", r.code)
	}
	gtt(t, root, "inbox", "park", "IDEA-001", "--yes").must(t, 0)
	if e := entries(t, root, "--all")["IDEA-001"]; e["stage"] != "parked" {
		t.Errorf("parked: %v", e)
	}
	gtt(t, root, "inbox", "reopen", "IDEA-001", "--yes").must(t, 0)
	if e := entries(t, root)["IDEA-001"]; e["stage"] != "new" {
		t.Errorf("reopened: %v", e)
	}
	if r := gtt(t, root, "inbox", "priority", "IDEA-001", "urgent", "--yes"); r.code == 0 {
		t.Errorf("a new idea has no priority to change (T22):\n%s%s", r.out, r.err)
	}
	if r := gtt(t, root, "inbox", "priority", "IDEA-001", "asap", "--yes"); r.code != 3 {
		t.Errorf("an unknown priority is a usage error: %d", r.code)
	}
	gtt(t, root, "idea", "drop", "IDEA-002", "--yes").must(t, 0)
	gtt(t, root, "inbox", "drop", "IDEA-001", "--yes", "--reason", "ya no").must(t, 0)
	if n := len(entries(t, root)); n != 0 {
		t.Errorf("the inbox is empty: %d", n)
	}
	if n := len(entries(t, root, "--all")); n != 2 {
		t.Errorf("--all keeps them: %d", n)
	}
	if out := gtt(t, root, "inbox").must(t, 0).out; !strings.Contains(out, "empty") {
		t.Errorf("an empty inbox says so:\n%s", out)
	}
}

func TestAProposalIsDecidedWithAnswerAndTheCLIRendersTheReview(t *testing.T) {
	root := inboxProject(t)
	gtt(t, root, "idea", "add", "Sesiones en Redis", "--yes").must(t, 0)
	agentProposes(t, root, "redis", "IDEA-001")
	agentProposes(t, root, "queue", "")
	out := gtt(t, root, "pending").must(t, 0).out
	for _, want := range []string{"REV", "1. ", "2. ", "answer: gtt answer 1 <number>"} {
		if !strings.Contains(out, want) {
			t.Errorf("a REV is shown with its options, missing %q:\n%s", want, out)
		}
	}
	agent := entries(t, root, "--origin", "agent")
	if len(agent) != 2 || agent["PROP-001"]["stage"] != "review" || agent["PROP-001"]["command"] != "gtt answer 1 1" {
		t.Fatalf("two proposals in review: %v", agent)
	}
	if out := gtt(t, root, "proposals").must(t, 0).out; !strings.Contains(out, "PROP-001") || !strings.Contains(out, "In review (2)") {
		t.Errorf("gtt proposals is the agent's side of the inbox:\n%s", out)
	}
	// the answer names the option and writes the decision; the entry is accepted
	r := gtt(t, root, "answer", "REV-001", "1").must(t, 0)
	if !strings.Contains(r.out, "PROP-001") {
		t.Errorf("it says what was decided:\n%s", r.out)
	}
	if e := entries(t, root)["PROP-001"]; e["stage"] != "accepted" {
		t.Errorf("accepted: %v", e)
	}
	if e := entries(t, root)["IDEA-001"]; e["stage"] != "accepted" {
		t.Errorf("its idea follows: %v", e)
	}
	gtt(t, root, "inbox", "priority", "PROP-001", "urgente", "--yes").must(t, 0)
	if e := entries(t, root)["PROP-001"]; e["priority"] != "urgent" {
		t.Errorf("urgent, in the entry's own trail: %v", e)
	}
	// answering twice is refused, not repeated
	if gtt(t, root, "answer", "1", "1").code == 0 {
		t.Error("a decided REV is not decided again")
	}
	// the other one is rejected without a REV, through the alias
	gtt(t, root, "proposals", "reject", "PROP-002", "--yes").must(t, 0)
	if e := entries(t, root, "--all")["PROP-002"]; e["stage"] != "rejected" {
		t.Errorf("rejected: %v", e)
	}
	if out := gtt(t, root, "pending", "--json").must(t, 0).out; strings.Contains(out, `"state": "open"`) {
		t.Errorf("no REV is left open:\n%s", out)
	}
	if out := gtt(t, root, "proposals", "show", "PROP-001").must(t, 0).out; !strings.Contains(out, "Suggested change") {
		t.Errorf("proposals show takes a ref:\n%s", out)
	}
	// what exists keeps working: withdraw by ref goes to reject
	agentProposes(t, root, "third", "")
	gtt(t, root, "proposals", "withdraw", "PROP-003", "--yes").must(t, 0)
	if e := entries(t, root, "--all")["PROP-003"]; e["stage"] != "rejected" {
		t.Errorf("withdrawn is rejected: %v", e)
	}
}

func TestTheMigrationOfA122ProjectShowsFirstAndRunsTwiceWithoutChange(t *testing.T) {
	root := inboxProject(t)
	// migration: a pending request on the old desk, and again (idempotent)
	write(t, root, "gtt-domain/change-request.md", "# Change Request\n\n> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v3.0.md\n\n## GTT Request\n\n```text\nId: CR-20261001-101500\nRequested: 2026-10-01T10:15:00Z by Ana\nChange: Move sessions to Redis\nReason: Horizontal scaling\nTrigger: latency\nScope:\nImpact:\nRisk:\nPriority: high\n```\n")
	if r := gtt(t, root, "inbox", "migrate"); r.code != 0 || !exists(root, "gtt-domain/change-request.md") || !strings.Contains(r.out, "would do") {
		t.Errorf("without a terminal and without --yes it only says what it would do, and writes nothing:\n%s%s", r.out, r.err)
	}
	gtt(t, root, "inbox", "migrate", "--yes").must(t, 0)
	if exists(root, "gtt-domain/change-request.md") {
		t.Error("the desk is gone")
	}
	var found string
	for ref := range entries(t, root) {
		if strings.Contains(read(t, root, "docs/ideas/"+ref+".md"), "Move sessions to Redis") {
			found = ref
		}
	}
	if found == "" {
		t.Error("the request became an idea")
	}
	again := treeHash(t, root)
	if out := gtt(t, root, "inbox", "migrate", "--yes").must(t, 0).out; !strings.Contains(out, "nothing to migrate") || treeHash(t, root) != again {
		t.Errorf("a second run changes nothing:\n%s", out)
	}
}

func TestEveryInboxCommandWithABootstrapWithoutItIsOneLineAndNothingChanges(t *testing.T) {
	root := inboxProject(t)
	gtt(t, root, "idea", "add", "una", "--yes").must(t, 0)
	downgrade(t, root)
	pre := treeHash(t, root)
	for _, args := range [][]string{
		{"inbox"}, {"inbox", "--origin", "agent"}, {"inbox", "show", "IDEA-001"}, {"inbox", "park", "IDEA-001"}, {"inbox", "reject", "PROP-001"},
		{"inbox", "reopen", "IDEA-001"}, {"inbox", "drop", "IDEA-001"}, {"inbox", "priority", "IDEA-001", "urgent"}, {"inbox", "migrate"},
		{"idea", "add", "x"}, {"idea", "add", "--type", "solicitud", "--change", "a", "--reason", "b"}, {"idea", "drop", "IDEA-001"},
		{"adr", "new", "Algo"}, {"adr", "adopt", "gtt-domain/adr/ADR-1.md"},
	} {
		r := gtt(t, root, append(args, "--yes")...)
		if r.code != 0 || strings.Count(strings.TrimSpace(r.out), "\n") != 0 || !strings.Contains(r.out, "gtt update") {
			t.Errorf("%v: one line, exit 0, got %d:\n%s%s", args, r.code, r.out, r.err)
		}
	}
	if treeHash(t, root) != pre {
		t.Error("nothing was touched")
	}
}

func TestTheSpanishProjectSpeaksSpanish(t *testing.T) {
	root := inboxProject(t)
	gtt(t, root, "config", "language", "es").must(t, 0)
	gtt(t, root, "idea", "add", "una", "--yes").must(t, 0)
	if out := gtt(t, root, "inbox").must(t, 0).out; !strings.Contains(out, "Nuevas (1)") {
		t.Errorf("the stages are in the language:\n%s", out)
	}
	downgrade(t, root)
	if out := gtt(t, root, "inbox").must(t, 0).out; strings.TrimSpace(out) != "este Bootstrap no ofrece la bandeja: gtt update" {
		t.Errorf("the one line, in Spanish:\n%s", out)
	}
}
