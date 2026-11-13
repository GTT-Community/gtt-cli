package app

import (
	"context"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

func opsOf(names ...string) map[string]ports.Operation {
	m := map[string]ports.Operation{}
	for _, n := range names {
		m[n] = ports.Operation{Version: 1}
	}
	return m
}

var withInbox = append(append([]string{}, inboxOps...), "inbox.reconcile", "inbox.migrate", "adr.new", "adr.adopt", "change.request", "pending.list", "pending.answer")

// inboxBootstrap offers inbox.v1 and every operation the commands use.
func inboxBootstrap() *fakeBootstrap {
	return &fakeBootstrap{compatible: true, ops: opsOf(withInbox...), caps: []ports.Capability{{ID: inboxCapability, Version: 1}},
		results: map[string]ports.OperationResult{
			"inbox.reconcile":  ok(`{"actions":[]}`),
			"inbox.add":        ok("gtt-inbox: dry run - would create docs/ideas/IDEA-004.md\n(nothing written; re-run with --apply)"),
			"inbox.add:apply":  ok("gtt-inbox: created docs/ideas/IDEA-004.md"),
			"inbox.park":       ok("gtt-inbox: dry run - would add a line\n(nothing written; re-run with --apply)"),
			"inbox.park:apply": ok("gtt-inbox: IDEA-002 parked"),
		}}
}

func ops(b *fakeBootstrap) []string {
	var out []string
	for _, r := range b.requests {
		n := r.Operation
		if r.Apply {
			n += ":apply"
		}
		out = append(out, n)
	}
	return out
}

func TestInboxIsNegotiatedByCapabilityAndOperations(t *testing.T) {
	ctx := context.Background()
	b := inboxBootstrap()
	if !supportsInbox(ctx, b) {
		t.Fatal("the capability and its operations are the inbox")
	}
	b.caps = nil
	if supportsInbox(ctx, b) {
		t.Error("operations without the capability are not the inbox")
	}
	b = inboxBootstrap()
	delete(b.ops, "inbox.priority")
	if supportsInbox(ctx, b) {
		t.Error("a capability missing one of its operations is not the inbox")
	}
}

func TestABootstrapWithoutTheInboxAnswersOneLineAndTouchesNothing(t *testing.T) {
	b := &fakeBootstrap{compatible: true, ops: opsOf("idea.list")}
	a, pr := newApp(t, b)
	pr.interactive, pr.answer = true, true
	ctx := context.Background()
	if l, err := a.Inbox(ctx, InboxQuery{}); err != nil || l.Supported || l.Message != notInboxMessage {
		t.Errorf("list: %+v %v", l, err)
	}
	if v, err := a.InboxShow(ctx, "IDEA-1"); err != nil || v.Message != notInboxMessage {
		t.Errorf("show: %+v %v", v, err)
	}
	for name, run := range map[string]func() (InboxResult, error){
		"park":     func() (InboxResult, error) { return a.InboxPark(ctx, "IDEA-1", "") },
		"reject":   func() (InboxResult, error) { return a.InboxReject(ctx, "PROP-1", "") },
		"reopen":   func() (InboxResult, error) { return a.InboxReopen(ctx, "IDEA-1") },
		"drop":     func() (InboxResult, error) { return a.InboxDrop(ctx, "IDEA-1", "") },
		"priority": func() (InboxResult, error) { return a.InboxPriority(ctx, "IDEA-1", "urgent") },
		"adr new":  func() (InboxResult, error) { return a.AdrNew(ctx, "Use Redis", "", "") },
		"adopt":    func() (InboxResult, error) { return a.AdrAdopt(ctx, "gtt-domain/adr/ADR-1.md", "") },
	} {
		if r, err := run(); err != nil || r.Message != notInboxMessage || r.Applied {
			t.Errorf("%s: %+v %v", name, r, err)
		}
	}
	if r, err := a.IdeaAdd(ctx, IdeaAddOptions{Type: "solicitud", Change: "x", Reason: "y"}); err != nil || r.Message != notInboxMessage {
		t.Errorf("a request needs the inbox: %+v %v", r, err)
	}
	if r, err := a.IdeaAdd(ctx, IdeaAddOptions{Text: "x"}); err != nil || r.Message != notInboxMessage {
		t.Errorf("without ideas either: %+v %v", r, err)
	}
	if len(b.requests) != 0 {
		t.Errorf("nothing was asked of the Bootstrap: %v", ops(b))
	}
	if m, err := a.InboxMigrate(ctx); err != nil || m.Offered {
		t.Errorf("no migration to offer: %+v %v", m, err)
	}
}

func TestAnIdeaFromABootstrapWithIdeasButNoInboxKeepsWorking(t *testing.T) {
	b := &fakeBootstrap{compatible: true, ops: opsOf("idea.add", "idea.list", "idea.park", "idea.drop"),
		results: map[string]ports.OperationResult{"idea.add": ok("gtt-idea: created docs/ideas/IDEA-001.md")}}
	a, _ := newApp(t, b)
	r, err := a.IdeaAdd(context.Background(), IdeaAddOptions{Text: "una idea"})
	if err != nil || !strings.Contains(r.Output, "IDEA-001") {
		t.Fatalf("%+v %v", r, err)
	}
	if got := ops(b); len(got) != 1 || got[0] != "idea.add:apply" {
		t.Errorf("the 1.20 way, no preview: %v", got)
	}
	if r, err := a.IdeaAdd(context.Background(), IdeaAddOptions{Text: "x", Now: true}); err != nil || r.Message != notInboxMessage {
		t.Errorf("--now needs the inbox: %+v %v", r, err)
	}
}

func TestIdeaAddShowsThePreviewAsksAndStoresTheTextInAFile(t *testing.T) {
	b := inboxBootstrap()
	a, pr := newApp(t, b)
	pr.interactive, pr.answer = true, true
	r, err := a.IdeaAdd(context.Background(), IdeaAddOptions{Text: "sincronizar\ncon 'comillas' y $HOME", Now: true})
	if err != nil || r.Output == "" {
		t.Fatalf("%+v %v", r, err)
	}
	got := ops(b)
	if len(got) < 3 || got[0] != "inbox.add" || got[1] != "inbox.add:apply" || got[2] != "inbox.reconcile:apply" {
		t.Fatalf("dry run, apply, then reconcile in silence: %v", got)
	}
	args := b.requests[1].Args
	if args["file"] == "" || args["now"] != "true" || args["text"] != "" {
		t.Errorf("the text travels in a file, never as an argument: %v", args)
	}
	if !b.requests[1].ConfirmedByHuman {
		t.Error("a human act is confirmed by the human")
	}
	if len(pr.asked) != 1 {
		t.Errorf("one question: %v", pr.asked)
	}
}

func TestAnActTheHumanRefusesWritesNothing(t *testing.T) {
	b := inboxBootstrap()
	a, pr := newApp(t, b)
	pr.interactive, pr.answer = true, false
	if _, err := a.InboxPark(context.Background(), "idea-2", ""); err == nil {
		t.Fatal("a refusal is a cancellation")
	}
	for _, o := range ops(b) {
		if strings.HasSuffix(o, ":apply") {
			t.Errorf("nothing was applied: %v", ops(b))
		}
	}
	if b.requests[0].Args["ref"] != "IDEA-002" {
		t.Errorf("the ref is normalized: %v", b.requests[0].Args)
	}
}

func TestWithoutATerminalAndWithoutYesAnActIsRefusedAndYesPreAuthorizes(t *testing.T) {
	b := inboxBootstrap()
	a, _ := newApp(t, b)
	if _, err := a.InboxPark(context.Background(), "IDEA-2", ""); err == nil {
		t.Fatal("no terminal, no --yes: refused")
	}
	a.AssumeYes = true
	if r, err := a.InboxPark(context.Background(), "IDEA-2", "later"); err != nil || !r.Applied {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestPriorityAndRefsAreValidatedBeforeAskingTheBootstrap(t *testing.T) {
	b := inboxBootstrap()
	a, _ := newApp(t, b)
	a.AssumeYes = true
	if _, err := a.InboxPriority(context.Background(), "IDEA-1", "asap"); err == nil {
		t.Error("an unknown priority is refused")
	}
	if _, err := a.InboxPark(context.Background(), "ADR-1", ""); err == nil {
		t.Error("an unknown ref is refused")
	}
	if len(b.requests) != 0 {
		t.Errorf("nothing was asked: %v", ops(b))
	}
	if ref, _ := inboxRef(" prop-12 "); ref != "PROP-012" {
		t.Errorf("normalized: %s", ref)
	}
}

func TestInboxListPassesTheFilters(t *testing.T) {
	b := inboxBootstrap()
	b.results["inbox.list"] = ok(`{"schema":1,"entries":[{"ref":"PROP-001","origin":"agent","stage":"review","command":"gtt answer 1 1"}],"counts":{"review":1}}`)
	a, _ := newApp(t, b)
	l, err := a.Inbox(context.Background(), InboxQuery{All: true, Origin: "agent"})
	if err != nil || !l.Supported || len(l.Entries) != 1 || l.Entries[0].Command != "gtt answer 1 1" {
		t.Fatalf("%+v %v", l, err)
	}
	if a := b.requests[0].Args; a["all"] != "true" || a["origin"] != "agent" {
		t.Errorf("%v", a)
	}
	if _, err := a.Inbox(context.Background(), InboxQuery{Origin: "robot"}); err == nil {
		t.Error("origin is human or agent")
	}
}

func TestAdrNewAndAdoptGoThroughTheirOperations(t *testing.T) {
	b := inboxBootstrap()
	b.results["adr.new"] = ok("gtt-inbox: dry run - would create gtt-domain/adr/drafts/IDEA-005-x.md")
	b.results["adr.new:apply"] = ok("gtt-inbox: created gtt-domain/adr/drafts/IDEA-005-x.md")
	b.results["adr.adopt"] = ok("dry")
	b.results["adr.adopt:apply"] = ok("adopted IDEA-006")
	a, _ := newApp(t, b)
	a.AssumeYes = true
	if r, err := a.AdrNew(context.Background(), "Use Redis", "", ""); err != nil || r.Target != "IDEA-005" {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := a.AdrAdopt(context.Background(), a.Cwd+"/gtt-domain/adr/ADR-9.md", ""); err != nil {
		t.Fatal(err)
	}
	for _, r := range b.requests {
		if r.Operation == "adr.adopt" && r.Args["file"] != "gtt-domain/adr/ADR-9.md" {
			t.Errorf("a path inside the project is given relative to it: %v", r.Args)
		}
	}
	if _, err := a.AdrNew(context.Background(), "  ", "", ""); err == nil {
		t.Error("a title is needed")
	}
}

func TestChangeRequestIsAnIdeaInTheInbox(t *testing.T) {
	b := inboxBootstrap()
	b.results["change.request"] = ok("gtt-inbox: dry run - would create docs/ideas/IDEA-007.md")
	b.results["change.request:apply"] = ok("gtt-inbox: created docs/ideas/IDEA-007.md")
	b.results["change.state"] = ok(`{"state":"absent","id":"","fields":{},"proposals":[]}`)
	b.ops["change.state"] = ports.Operation{}
	b.ops["ade.state"] = ports.Operation{}
	b.results["ade.state"] = ok(`{"primary":"claude"}`)
	a, _ := newApp(t, b)
	a.AssumeYes = true
	st, err := a.ChangeRequest(context.Background(), ChangeOptions{Change: "Redis", Reason: "scale", Replace: true})
	if err != nil || st.ID != "IDEA-007" || st.State != "pending" {
		t.Fatalf("%+v %v", st, err)
	}
	for _, r := range b.requests {
		if _, has := r.Args["replace"]; has {
			t.Error("the inbox has no desk to replace and does not declare the argument")
		}
	}
	// through idea add --type solicitud it is the same act
	if r, err := a.IdeaAdd(context.Background(), IdeaAddOptions{Type: "solicitud", Change: "Redis", Reason: "scale"}); err != nil || !strings.Contains(r.Output, "IDEA-007") {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := a.IdeaAdd(context.Background(), IdeaAddOptions{Type: "solicitud", Text: "free text", Change: "x", Reason: "y"}); err == nil {
		t.Error("a request is not free text")
	}
}

func TestMigrationShowsFirstAsksAndAppliesAndNeverFailsWithoutAnAnswer(t *testing.T) {
	ctx := context.Background()
	b := inboxBootstrap()
	b.results["inbox.migrate"] = ok("gtt-inbox: migrate - would do:\n  CR -> IDEA-001\n(dry run - nothing written; re-run with --apply)")
	b.results["inbox.migrate:apply"] = ok("gtt-inbox: migrated. Report:\n  CR -> IDEA-001")
	a, pr := newApp(t, b)
	pr.interactive, pr.answer = true, true
	m, err := a.migrateInbox(ctx, b, a.Cwd)
	if err != nil || !m.Applied || !strings.Contains(m.Report, "migrated") {
		t.Fatalf("%+v %v", m, err)
	}
	if got := ops(b); got[0] != "inbox.migrate" || got[1] != "inbox.migrate:apply" {
		t.Errorf("dry run first, then apply: %v", got)
	}
	b.requests, pr.answer = nil, false
	if m, err := a.migrateInbox(ctx, b, a.Cwd); err != nil || m.Applied || !m.Pending {
		t.Errorf("not confirmed: pending, applied nothing: %+v %v", m, err)
	}
	b.requests = nil
	b.results["inbox.migrate"] = ok("gtt-inbox: nothing to migrate: the inbox is already the 1.23 one.")
	if m, err := a.migrateInbox(ctx, b, a.Cwd); err != nil || m.Applied || m.Pending || len(b.requests) != 1 {
		t.Errorf("nothing to do: one dry run and no question: %+v %v %v", m, err, ops(b))
	}
}

func TestAnswerAcceptsAReviewByItsNumberOrItsName(t *testing.T) {
	for _, in := range []string{"1", "REV-001", "rev-1"} {
		if n, err := pendingRef(in); err != nil || n != 1 {
			t.Errorf("%s: %d %v", in, n, err)
		}
	}
	b := inboxBootstrap()
	b.results["pending.list"] = ok(`{"events":[{"id":"REV-001","number":1,"kind":"REV","title":"PROP-001 · Redis","state":"open","options":["Aceptar","Rechazar"]}]}`)
	b.results["pending.answer"] = ok("REV-001 · PROP-001  <-  option 1: PROP-001: aceptada")
	a, _ := newApp(t, b)
	r, err := a.PendingRespond(context.Background(), "REV-001", "1")
	if err != nil || !r.Review || len(r.Answered) != 1 || !strings.Contains(r.Answered[0], "1. Aceptar") {
		t.Fatalf("%+v %v", r, err)
	}
	got := ops(b)
	if got[len(got)-2] != "pending.answer:apply" || got[len(got)-1] != "inbox.reconcile:apply" {
		t.Errorf("the answer, then the inbox is reconciled in silence: %v", got)
	}
}

func TestReconcileNeverFailsAnActAndIsSkippedWithoutTheInbox(t *testing.T) {
	b := &fakeBootstrap{compatible: true, ops: opsOf("idea.list")}
	a, _ := newApp(t, b)
	a.reconcileInbox(a.Cwd)
	if len(b.requests) != 0 {
		t.Errorf("%v", ops(b))
	}
	b = inboxBootstrap()
	delete(b.results, "inbox.reconcile") // the fake errors: the act must not
	a, _ = newApp(t, b)
	a.reconcileInbox(a.Cwd)
}

func TestProposalsByRefUseTheInbox(t *testing.T) {
	b := inboxBootstrap()
	b.ops["change.request"] = ports.Operation{}
	b.results["inbox.reject"] = ok("gtt-inbox: dry run - would add")
	b.results["inbox.reject:apply"] = ok("gtt-inbox: PROP-001 rejected")
	b.results["inbox.show"] = ok(`{"entry":{"ref":"PROP-001","type":"proposal","title":"t","stage":"review","text":"the text"}}`)
	a, _ := newApp(t, b)
	a.AssumeYes = true
	if r, err := a.ProposalWithdraw(context.Background(), "prop-1", ""); err != nil || !r.Applied || !strings.Contains(r.Output, "rejected") {
		t.Fatalf("%+v %v", r, err)
	}
	if v, err := a.ProposalShow(context.Background(), "PROP-001"); err != nil || v.Content != "the text" {
		t.Fatalf("%+v %v", v, err)
	}
}
