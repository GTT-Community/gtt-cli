package app

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// The change flow during development. The human requests a change, follows
// the proposals the agent drafts, promotes one, withdraws one, or steps back
// one promotion. Every step is a Bootstrap operation (change.*, proposal.*,
// promotion.*); the CLI collects the human's words, asks for confirmation and
// reports. It never drafts, decides or applies anything on its own.

// ChangeOptions are the request fields given on the command line.
type ChangeOptions struct {
	Change, Reason, Trigger, Scope, Impact, Risk, Priority, By string
	Replace                                                    bool
}

// ChangeState is the change request as the Bootstrap reads it.
type ChangeState struct {
	Schema    int               `json:"schema"`
	State     string            `json:"state"` // absent | empty | pending | processed
	ID        string            `json:"id"`
	Fields    map[string]string `json:"fields"`
	Proposals []string          `json:"proposals"`
	// Primary is the ADE that will be told to process it (via the session's next action).
	Primary string `json:"primary,omitempty"`
}

// Proposal is one draft under gtt-domain/proposals/.
type Proposal struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Kind    string   `json:"kind"`
	Title   string   `json:"title"`
	Request string   `json:"request"`
	State   string   `json:"state"`
	Files   []string `json:"files,omitempty"`
}

// ProposalsResult is `gtt proposals`.
type ProposalsResult struct {
	Schema    int         `json:"schema"`
	Request   ChangeState `json:"request"`
	Proposals []Proposal  `json:"proposals"`
}

// ProposalView is `gtt proposals show`.
type ProposalView struct {
	Schema   int      `json:"schema"`
	Proposal Proposal `json:"proposal"`
	Content  string   `json:"content"`
}

// StepResult reports a human act carried out through the Bootstrap.
type StepResult struct {
	Schema  int    `json:"schema"`
	Action  string `json:"action"`
	Target  string `json:"target,omitempty"`
	Applied bool   `json:"applied"`
	Output  string `json:"output"`
}

// LastPromotion is what `gtt undo` would put back.
type LastPromotion struct {
	Promotion *struct {
		Name  string   `json:"name"`
		At    string   `json:"at"`
		Files []string `json:"files"`
		Froze bool     `json:"froze"`
	} `json:"promotion"`
	Undoable bool     `json:"undoable"`
	Problems []string `json:"problems"`
}

func supports(ctx context.Context, b ports.Bootstrap, op string) bool {
	ops, err := b.Operations(ctx)
	if err != nil {
		return false
	}
	_, ok := ops[op]
	return ok
}

func (a *App) openFlow(ctx context.Context) (*Project, error) {
	p, err := a.open(ctx)
	if err != nil {
		return nil, err
	}
	if !supports(ctx, p.B, "change.request") {
		return nil, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: "The installed GTT Bootstrap " + p.Release.Bootstrap.Version + " does not offer the change flow.", Next: "Run: gtt update"}
	}
	return p, nil
}

// human runs a human-authority operation: a dry run the human sees, their
// confirmation, then the applied run. A refused dry run stops everything.
func (a *App) human(ctx context.Context, b ports.Bootstrap, op string, args map[string]string, question string) (StepResult, error) {
	res := StepResult{Schema: core.OutputSchema, Action: op}
	dry, err := b.Execute(ctx, ports.OperationRequest{Operation: op, Args: args, ConfirmedByHuman: true})
	if err != nil {
		return res, err
	}
	if !dry.OK() {
		return res, &core.Error{Code: core.ExitConflict, Unmodified: true, What: "The Bootstrap refused " + op + ".",
			Why: strings.TrimSpace(dry.Stdout + "\n" + dry.Stderr)}
	}
	a.Report.Info("%s", strings.TrimRight(dry.Stdout, "\n"))
	ok, err := a.confirm(question, false)
	if err != nil {
		return res, err
	}
	if !ok {
		return res, core.ErrCancelled
	}
	done, err := b.Execute(ctx, ports.OperationRequest{Operation: op, Args: args, Apply: true, ConfirmedByHuman: true})
	if err != nil {
		return res, err
	}
	res.Output = strings.TrimSpace(done.Stdout + "\n" + done.Stderr)
	if !done.OK() {
		return res, &core.Error{Code: core.ExitFailure, What: "The Bootstrap could not apply " + op + ".", Why: res.Output}
	}
	res.Applied = true
	return res, nil
}

// ChangeRequest records the human's change request through the Bootstrap.
// What it does not get from flags it asks; without a terminal, the change
// and its reason are required. The agent is told to process it through the
// session's next action - it drafts a proposal and decides nothing.
func (a *App) ChangeRequest(ctx context.Context, o ChangeOptions) (ChangeState, error) {
	p, err := a.openFlow(ctx)
	if err != nil {
		return ChangeState{}, err
	}
	if o.Change == "" || o.Reason == "" {
		if !a.Prompt.Interactive() {
			return ChangeState{}, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "A change request needs what to change and why.",
				Next: "Run in a terminal, or: gtt change \"<what>\" --reason \"<why>\""}
		}
		a.Report.Info("GTT change request - a governed change: design, architecture, stack, constraints or an Epic's goal or scope.\n" +
			"Ordinary work (code, refactors, tests, Stories) needs none.\n")
		for _, f := range []struct {
			v        *string
			q        string
			required bool
		}{{&o.Change, "What needs to change?", true}, {&o.Reason, "Why?", true}, {&o.Trigger, "What triggered it? (optional)", false},
			{&o.Scope, "Scope - what is in and out? (optional)", false}, {&o.Impact, "Impact? (optional)", false},
			{&o.Risk, "Risk? (optional)", false}} {
			for *f.v == "" {
				answer, err := a.Prompt.Ask(f.q)
				if err != nil {
					return ChangeState{}, err
				}
				*f.v = answer
				if !f.required {
					break
				}
			}
		}
		if o.Priority == "" {
			opts := []ports.Option{{Label: "critical"}, {Label: "high"}, {Label: "medium"}, {Label: "low"}}
			i, err := a.Prompt.Select("Priority:", opts)
			if err != nil {
				return ChangeState{}, err
			}
			o.Priority = opts[i].Label
		}
	}
	args := map[string]string{"change": o.Change, "reason": o.Reason}
	for k, v := range map[string]string{"trigger": o.Trigger, "scope": o.Scope, "impact": o.Impact, "risk": o.Risk, "priority": o.Priority, "by": o.By} {
		if strings.TrimSpace(v) != "" {
			args[k] = v
		}
	}
	if o.Replace {
		args["replace"] = "true"
	}
	if _, err := a.human(ctx, p.B, "change.request", args, "Record this change request?"); err != nil {
		return ChangeState{}, err
	}
	st, err := a.changeState(ctx, p.B)
	if err != nil {
		return st, err
	}
	var ade adeState
	if _, err := a.query(ctx, p.B, "ade.state", nil, &ade); err == nil {
		st.Primary = ade.Primary
	}
	_ = p.Store.Record("change-request", st.ID)
	a.log(p.Root, "change.request", "success", map[string]any{"id": st.ID})
	return st, nil
}

func (a *App) changeState(ctx context.Context, b ports.Bootstrap) (ChangeState, error) {
	st := ChangeState{Schema: core.OutputSchema}
	_, err := a.query(ctx, b, "change.state", nil, &st)
	st.Schema = core.OutputSchema
	return st, err
}

// ChangeShow is `gtt change show`.
func (a *App) ChangeShow(ctx context.Context) (ChangeState, error) {
	p, err := a.openFlow(ctx)
	if err != nil {
		return ChangeState{}, err
	}
	return a.changeState(ctx, p.B)
}

// Proposals is `gtt proposals`.
func (a *App) Proposals(ctx context.Context) (ProposalsResult, error) {
	res := ProposalsResult{Schema: core.OutputSchema}
	p, err := a.openFlow(ctx)
	if err != nil {
		return res, err
	}
	_, err = a.query(ctx, p.B, "proposal.list", nil, &res)
	res.Schema = core.OutputSchema
	return res, err
}

// ProposalShow is `gtt proposals show <name>`.
func (a *App) ProposalShow(ctx context.Context, name string) (ProposalView, error) {
	res := ProposalView{Schema: core.OutputSchema}
	p, err := a.openFlow(ctx)
	if err != nil {
		return res, err
	}
	out, err := p.B.Execute(ctx, ports.OperationRequest{Operation: "proposal.show", Args: map[string]string{"proposal": name}})
	if err != nil {
		return res, err
	}
	if !out.OK() {
		return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: strings.TrimSpace(out.Stderr), Next: "See them with: gtt proposals"}
	}
	if err := json.Unmarshal([]byte(out.Stdout), &res); err != nil {
		return res, &core.Error{Code: core.ExitFailure, What: "The Bootstrap result of proposal.show is unreadable.", Why: err.Error()}
	}
	res.Schema = core.OutputSchema
	return res, nil
}

// ProposalWithdraw archives a draft nobody will promote (a human act).
func (a *App) ProposalWithdraw(ctx context.Context, name, by string) (StepResult, error) {
	p, err := a.openFlow(ctx)
	if err != nil {
		return StepResult{}, err
	}
	args := map[string]string{"proposal": name}
	if by != "" {
		args["by"] = by
	}
	res, err := a.human(ctx, p.B, "proposal.withdraw", args, "Withdraw it? It is archived, not deleted")
	res.Target = name
	if err == nil {
		a.log(p.Root, "proposal.withdraw", "success", map[string]any{"proposal": name})
	}
	return res, err
}

// Promote applies a staged promotion set: the human's act, after seeing it.
func (a *App) Promote(ctx context.Context, set string) (StepResult, error) {
	p, err := a.openFlow(ctx)
	if err != nil {
		return StepResult{}, err
	}
	res, err := a.human(ctx, p.B, "promotion.apply", map[string]string{"set": set}, "Apply this promotion set?")
	res.Target = set
	if err == nil {
		_ = p.Store.Record("promotion", set)
		a.log(p.Root, "promote", "success", map[string]any{"set": set})
	}
	return res, err
}

// Undo puts back what the last promotion replaced. It is never an unfreeze:
// when that promotion froze, the Bootstrap records a new freeze over the
// earlier state and keeps the history.
func (a *App) Undo(ctx context.Context, set string) (StepResult, error) {
	p, err := a.openFlow(ctx)
	if err != nil {
		return StepResult{}, err
	}
	var last LastPromotion
	if _, err := a.query(ctx, p.B, "promotion.last", nil, &last); err != nil {
		return StepResult{}, err
	}
	if last.Promotion == nil {
		return StepResult{}, &core.Error{Code: core.ExitConflict, Unmodified: true, What: "There is no promotion to undo.",
			Why: "Going back further than the last promotion is a change request: gtt change"}
	}
	if !last.Undoable {
		return StepResult{}, &core.Error{Code: core.ExitConflict, Unmodified: true, What: "The last promotion (" + last.Promotion.Name + ") cannot be undone.",
			Why: strings.Join(last.Problems, "\n"), Next: "Going back is then a change request: gtt change"}
	}
	args := map[string]string{}
	if set != "" {
		args["set"] = set
	}
	res, err := a.human(ctx, p.B, "promotion.undo", args, "Undo the promotion "+last.Promotion.Name+"?")
	res.Target = last.Promotion.Name
	if err == nil {
		_ = p.Store.Record("undo", last.Promotion.Name)
		a.log(p.Root, "undo", "success", map[string]any{"set": last.Promotion.Name})
	}
	return res, err
}
