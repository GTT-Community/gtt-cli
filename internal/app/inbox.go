package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// The inbox (capability inbox.v1): one pipeline for every change. A person writes an IDEA (an idea, a change
// request or an ADR draft), the agent writes a PROP; each entry advances by stages and keeps its own trail. The
// files, the numbering, the stages and the transitions belong to the Bootstrap (operations inbox.*, adr.*); the CLI
// collects the human's words, shows what the act will do, asks, and reports. It reads no Bootstrap file and writes
// none. A Bootstrap without the capability gets one line and the command ends well.

// inboxCapability is the capability this CLI negotiates for the inbox.
const inboxCapability = "inbox.v1"

// inboxOps are the operations the human's commands need; a Bootstrap offering the capability declares them all.
var inboxOps = []string{"inbox.add", "inbox.list", "inbox.show", "inbox.park", "inbox.reject", "inbox.reopen", "inbox.drop", "inbox.priority"}

// notInboxMessage is the single line a Bootstrap without the inbox gets.
const notInboxMessage = "this Bootstrap does not offer the inbox: gtt update"

// supportsInbox negotiates inbox.v1: the Bootstrap declares the capability and the operations it names.
func supportsInbox(ctx context.Context, b ports.Bootstrap) bool {
	caps, err := b.Capabilities(ctx)
	if err != nil {
		return false
	}
	found := false
	for _, c := range caps {
		found = found || c.ID == inboxCapability
	}
	if !found {
		return false
	}
	for _, op := range inboxOps {
		if !supports(ctx, b, op) {
			return false
		}
	}
	return true
}

// InboxTrail is one dated line of an entry's trail.
type InboxTrail struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	At    string `json:"at"`
	Actor string `json:"actor"`
}

// InboxEntry is one entry as `inbox.list` and `inbox.show` declare it.
type InboxEntry struct {
	Ref         string       `json:"ref"`
	Type        string       `json:"type"`
	Origin      string       `json:"origin"` // human | agent
	From        string       `json:"from"`
	Title       string       `json:"title"`
	Level       string       `json:"level"`
	Stage       string       `json:"stage"`
	Priority    string       `json:"priority"`
	DaysInStage int          `json:"days_in_stage"`
	Event       string       `json:"event"`
	Command     string       `json:"command"`
	Path        string       `json:"path"`
	Created     string       `json:"created"`
	Now         bool         `json:"now"`
	Trail       []InboxTrail `json:"trail,omitempty"`
	Text        string       `json:"text,omitempty"`
}

// InboxList is `gtt inbox`.
type InboxList struct {
	Schema    int            `json:"schema"`
	Supported bool           `json:"supported"`
	Message   string         `json:"message,omitempty"`
	Entries   []InboxEntry   `json:"entries"`
	Counts    map[string]int `json:"counts,omitempty"`
}

// InboxEntryView is `gtt inbox show`.
type InboxEntryView struct {
	Schema    int        `json:"schema"`
	Supported bool       `json:"supported"`
	Message   string     `json:"message,omitempty"`
	Entry     InboxEntry `json:"entry"`
}

// InboxResult reports a human act on the inbox carried out through the Bootstrap.
type InboxResult struct {
	Schema    int    `json:"schema"`
	Supported bool   `json:"supported"`
	Message   string `json:"message,omitempty"`
	Action    string `json:"action,omitempty"`
	Target    string `json:"target,omitempty"`
	Applied   bool   `json:"applied"`
	Output    string `json:"output,omitempty"`
}

func notInbox() InboxResult {
	return InboxResult{Schema: core.OutputSchema, Message: notInboxMessage}
}

// openInbox opens the project; ok is false (and nothing is asked of the Bootstrap) when it lacks the inbox.
func (a *App) openInbox(ctx context.Context) (p *Project, ok bool, err error) {
	if p, err = a.open(ctx); err != nil {
		return nil, false, err
	}
	return p, supportsInbox(ctx, p.B), nil
}

var refRE = regexp.MustCompile(`^(IDEA|PROP)-0*(\d+)$`)

// inboxRef accepts `idea-7`, `IDEA-007` and `PROP-12`.
func inboxRef(ref string) (string, error) {
	m := refRE.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(ref)))
	if m == nil {
		return "", &core.Error{Code: core.ExitUsage, Unmodified: true, What: fmt.Sprintf("%q is not an entry of the inbox.", ref),
			Next: "Use the ref `gtt inbox` lists: IDEA-007 or PROP-012"}
	}
	n, _ := strconv.Atoi(m[2])
	return fmt.Sprintf("%s-%03d", m[1], n), nil
}

// InboxQuery are the filters of `gtt inbox`.
type InboxQuery struct {
	All    bool
	Origin string // human | agent
}

// Inbox lists the entries (all=true adds the closed, parked and rejected ones).
func (a *App) Inbox(ctx context.Context, q InboxQuery) (InboxList, error) {
	res := InboxList{Schema: core.OutputSchema}
	if q.Origin != "" && q.Origin != "human" && q.Origin != "agent" {
		return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "--origin is human or agent."}
	}
	p, ok, err := a.openInbox(ctx)
	if err != nil {
		return res, err
	}
	if !ok {
		res.Message = notInboxMessage
		return res, nil
	}
	args := map[string]string{}
	if q.All {
		args["all"] = "true"
	}
	if q.Origin != "" {
		args["origin"] = q.Origin
	}
	if _, err = a.query(ctx, p.B, "inbox.list", args, &res); err != nil {
		return res, err
	}
	res.Schema, res.Supported = core.OutputSchema, true
	return res, nil
}

// InboxShow is `gtt inbox show REF`: the entry with its trail.
func (a *App) InboxShow(ctx context.Context, ref string) (InboxEntryView, error) {
	res := InboxEntryView{Schema: core.OutputSchema}
	ref, err := inboxRef(ref)
	if err != nil {
		return res, err
	}
	p, ok, err := a.openInbox(ctx)
	if err != nil {
		return res, err
	}
	if !ok {
		res.Message = notInboxMessage
		return res, nil
	}
	out, err := p.B.Execute(ctx, ports.OperationRequest{Operation: "inbox.show", Args: map[string]string{"ref": ref}})
	if err != nil {
		return res, err
	}
	if !out.OK() {
		return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: strings.TrimSpace(out.Stderr), Next: "See them with: gtt inbox"}
	}
	if err := out.JSON(&res); err != nil {
		return res, &core.Error{Code: core.ExitFailure, What: "The Bootstrap result of inbox.show is unreadable.", Why: err.Error()}
	}
	res.Schema, res.Supported = core.OutputSchema, true
	return res, nil
}

// inboxAct runs one human act on an entry: park, reject, reopen, drop and priority. The Bootstrap shows what it
// would write (a dry run); the human confirms (--yes pre-authorizes); then it is applied.
func (a *App) inboxAct(ctx context.Context, op, ref, question string, extra map[string]string) (InboxResult, error) {
	res := notInbox()
	ref, err := inboxRef(ref)
	if err != nil {
		return res, err
	}
	p, ok, err := a.openInbox(ctx)
	if err != nil || !ok {
		return res, err
	}
	args := map[string]string{"ref": ref}
	for k, v := range extra {
		if strings.TrimSpace(v) != "" {
			args[k] = v
		}
	}
	step, err := a.human(ctx, p.B, op, args, question)
	res = InboxResult{Schema: core.OutputSchema, Supported: true, Action: op, Target: ref, Applied: step.Applied, Output: step.Output}
	if err != nil {
		return res, err
	}
	a.log(p.Root, op, "success", map[string]any{"ref": ref})
	return res, nil
}

// InboxPark sets an entry aside without going through its REV (T21).
func (a *App) InboxPark(ctx context.Context, ref, reason string) (InboxResult, error) {
	return a.inboxAct(ctx, "inbox.park", ref, "Park it? You can reopen it later", map[string]string{"reason": reason})
}

// InboxReject discards a proposal without going through its REV (T21).
func (a *App) InboxReject(ctx context.Context, ref, reason string) (InboxResult, error) {
	return a.inboxAct(ctx, "inbox.reject", ref, "Reject it? It is archived, not deleted", map[string]string{"reason": reason})
}

// InboxDrop discards an idea (T21).
func (a *App) InboxDrop(ctx context.Context, ref, reason string) (InboxResult, error) {
	return a.inboxAct(ctx, "inbox.drop", ref, "Discard it? It is archived, not deleted", map[string]string{"reason": reason})
}

// InboxReopen brings a parked entry back (T20).
func (a *App) InboxReopen(ctx context.Context, ref string) (InboxResult, error) {
	return a.inboxAct(ctx, "inbox.reopen", ref, "Reopen it?", nil)
}

// InboxPriority changes the priority of an accepted entry (T22): urgent, normal or low (or urgente, baja).
func (a *App) InboxPriority(ctx context.Context, ref, priority string) (InboxResult, error) {
	priority = strings.ToLower(strings.TrimSpace(priority))
	switch priority {
	case "urgent", "normal", "low", "urgente", "baja":
	default:
		return notInbox(), &core.Error{Code: core.ExitUsage, Unmodified: true, What: fmt.Sprintf("%q is not a priority.", priority), Next: "Use: gtt inbox priority IDEA-007 urgent|normal|low"}
	}
	return a.inboxAct(ctx, "inbox.priority", ref, "Change the priority?", map[string]string{"priority": priority})
}

// ------------------------------------------------------------------------------------------ ADR drafts

// AdrNew starts a draft of an ADR (F4): the architect's own text, in the inbox as an IDEA.
func (a *App) AdrNew(ctx context.Context, title, file, by string) (InboxResult, error) {
	res := notInbox()
	if strings.TrimSpace(title) == "" {
		return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "The ADR needs a title.", Next: "Use: gtt adr new \"Sessions live in Redis\""}
	}
	p, ok, err := a.openInbox(ctx)
	if err != nil || !ok {
		return res, err
	}
	if !supports(ctx, p.B, "adr.new") {
		return res, nil
	}
	args := map[string]string{"title": strings.TrimSpace(title)}
	if file != "" {
		args["file"] = a.pathFor(p.Root, file)
	}
	if by != "" {
		args["by"] = by
	}
	return a.inboxWrite(ctx, p, "adr.new", args, "Create this ADR draft?")
}

// AdrAdopt takes an ADR written by hand in gtt-domain/adr/ into the inbox (F5).
func (a *App) AdrAdopt(ctx context.Context, file, by string) (InboxResult, error) {
	res := notInbox()
	if strings.TrimSpace(file) == "" {
		return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Name the ADR file.", Next: "Use: gtt adr adopt gtt-domain/adr/ADR-007-sessions.md"}
	}
	p, ok, err := a.openInbox(ctx)
	if err != nil || !ok {
		return res, err
	}
	if !supports(ctx, p.B, "adr.adopt") {
		return res, nil
	}
	args := map[string]string{"file": a.pathFor(p.Root, file)}
	if by != "" {
		args["by"] = by
	}
	return a.inboxWrite(ctx, p, "adr.adopt", args, "Take this ADR into the inbox?")
}

// inboxWrite is a human act that writes a new entry (add, adr.new, adr.adopt).
func (a *App) inboxWrite(ctx context.Context, p *Project, op string, args map[string]string, question string) (InboxResult, error) {
	step, err := a.human(ctx, p.B, op, args, question)
	res := InboxResult{Schema: core.OutputSchema, Supported: true, Action: op, Applied: step.Applied, Output: step.Output}
	if err != nil {
		return res, err
	}
	res.Target = firstRef(step.Output)
	a.log(p.Root, op, "success", map[string]any{"ref": res.Target})
	return res, nil
}

var anyRefRE = regexp.MustCompile(`\b(?:IDEA|PROP)-\d+\b`)

// firstRef is the first entry ref a Bootstrap line mentions, or "".
func firstRef(text string) string { return anyRefRE.FindString(text) }

// pathFor gives the Bootstrap a path it can open from the project root: the one the human typed, resolved from
// where they typed it, and relative to the project when it lies inside.
func (a *App) pathFor(root, given string) string {
	abs := given
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(a.Cwd, given)
	}
	abs = filepath.Clean(abs)
	if rel, err := filepath.Rel(root, abs); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return abs
}

// ------------------------------------------------------------------------------------------ keeping it current

// reconcileInbox lets the Bootstrap bring the trail of every entry up to date after an act of the human (a
// promotion, an answer, a start). It is quiet and it never fails the act.
func (a *App) reconcileInbox(root string) {
	defer func() { _ = recover() }()
	if !a.Factory.Installed(root) {
		return
	}
	ctx := context.Background()
	b := a.Factory.At(root)
	if !supportsInbox(ctx, b) || !supports(ctx, b, "inbox.reconcile") {
		return
	}
	_, _ = b.Execute(ctx, ports.OperationRequest{Operation: "inbox.reconcile", Apply: true})
}

// -------------------------------------------------------------------------------------------- migration

// InboxMigration is what the migration of a 1.22 project said and did.
type InboxMigration struct {
	Offered bool   `json:"offered"`           // the Bootstrap has inbox.migrate
	Report  string `json:"report,omitempty"`  // what it would do, or did
	Applied bool   `json:"applied"`           // it was done
	Pending bool   `json:"pending,omitempty"` // there is something to migrate and it was not confirmed
}

// migrateInbox is the migration a 1.22 project needs (section 14): the Bootstrap shows what it will do (a dry run),
// the human confirms (--yes pre-authorizes) and it is applied. Without a terminal or --yes it only says it is
// pending. It never fails an update: the Bootstrap leaves the project as it was when it refuses.
func (a *App) migrateInbox(ctx context.Context, b ports.Bootstrap, root string) (InboxMigration, error) {
	var m InboxMigration
	if !supportsInbox(ctx, b) || !supports(ctx, b, "inbox.migrate") {
		return m, nil
	}
	m.Offered = true
	dry, err := b.Execute(ctx, ports.OperationRequest{Operation: "inbox.migrate", ConfirmedByHuman: true})
	if err != nil {
		return m, err
	}
	m.Report = strings.TrimSpace(dry.Stdout + "\n" + dry.Stderr)
	if !dry.OK() {
		return m, &core.Error{Code: core.ExitConflict, Unmodified: true, What: "The Bootstrap refused inbox.migrate.", Why: m.Report}
	}
	if !strings.Contains(dry.Stdout, "--apply") {
		return m, nil // nothing to migrate: the Bootstrap says so and the dry run is the whole report
	}
	a.Report.Info("%s", strings.TrimRight(dry.Stdout, "\n"))
	if ok, cerr := a.confirm("Migrate the inbox as shown?", true); cerr != nil || !ok {
		m.Pending = true
		return m, nil
	}
	done, err := b.Execute(ctx, ports.OperationRequest{Operation: "inbox.migrate", Apply: true, ConfirmedByHuman: true})
	if err != nil {
		return m, err
	}
	m.Report = strings.TrimSpace(done.Stdout + "\n" + done.Stderr)
	if !done.OK() {
		return m, &core.Error{Code: core.ExitFailure, What: "The Bootstrap could not apply inbox.migrate.", Why: m.Report}
	}
	m.Applied = true
	a.log(root, "inbox.migrate", "success", nil)
	return m, nil
}

// InboxMigrate is `gtt inbox migrate`: the migration of a 1.22 project, on its own.
func (a *App) InboxMigrate(ctx context.Context) (InboxMigration, error) {
	p, ok, err := a.openInbox(ctx)
	if err != nil || !ok {
		return InboxMigration{}, err
	}
	return a.migrateInbox(ctx, p.B, p.Root)
}

// stageOrder keeps the inbox in the order the pipeline runs.
var stageOrder = []string{"new", "review", "accepted", "ratify", "work", "parked", "closed", "rejected"}

// StageHeading is the heading of a stage with its count as a format, so the translator can give it whole.
func StageHeading(stage string) string {
	switch stage {
	case "new":
		return "New (%d)"
	case "review":
		return "In review (%d)"
	case "accepted":
		return "Accepted (%d)"
	case "ratify":
		return "To ratify (%d)"
	case "work":
		return "In progress (%d)"
	case "parked":
		return "Parked (%d)"
	case "closed":
		return "Closed (%d)"
	case "rejected":
		return "Rejected (%d)"
	}
	return stage + " (%d)"
}

// StageOrder lists the stages in pipeline order.
func StageOrder() []string { return append([]string(nil), stageOrder...) }

// tempText writes the human's text to a file the Bootstrap reads, so a newline, a quote or a dollar sign cannot
// change what is stored. The caller removes it.
func tempText(text string) (string, error) {
	f, err := os.CreateTemp("", "gtt-idea-*.txt")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(text); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}
