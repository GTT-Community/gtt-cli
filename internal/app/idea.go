package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Ideas: the human's loose ideas, one file each in docs/ideas/. The files, the numbering, the format and the
// rules belong to the Bootstrap (operations idea.*); the CLI hands the text over exactly as given and records
// nothing itself. Adding, parking and dropping are acts of the human: the command is the confirmation.

// IdeaItem is one idea, as `idea.list` declares it.
type IdeaItem struct {
	Ref     string `json:"ref"`
	Title   string `json:"title"`
	State   string `json:"state"`
	To      string `json:"to"`
	Created string `json:"created"`
	Path    string `json:"path"`
}

// IdeaList is `gtt idea list`.
type IdeaList struct {
	Schema int        `json:"schema"`
	Ideas  []IdeaItem `json:"ideas"`
	New    int        `json:"new"`
}

// IdeaResult is what an idea act wrote. A Bootstrap without the inbox (and without ideas) gets Message, one line.
type IdeaResult struct {
	Schema  int    `json:"schema"`
	Output  string `json:"output"`
	Message string `json:"message,omitempty"`
}

// IdeaAddOptions are the flags of `gtt idea add`.
type IdeaAddOptions struct {
	Text, File, By string
	Now            bool
	// Type is idea (the default) or solicitud; a request carries its fields instead of free text.
	Type                                                   string
	Change, Reason, Trigger, Scope, Impact, Risk, Priority string
}

func (a *App) openIdeas(ctx context.Context) (*Project, error) {
	p, err := a.open(ctx)
	if err != nil {
		return nil, err
	}
	for _, op := range []string{"idea.add", "idea.list", "idea.park", "idea.drop"} {
		if !supports(ctx, p.B, op) {
			return nil, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
				What: "The installed GTT Bootstrap " + p.Release.Bootstrap.Version + " does not offer ideas (gtt idea).", Next: "Run: gtt update"}
		}
	}
	return p, nil
}

func fromInbox(r InboxResult) IdeaResult {
	return IdeaResult{Schema: core.OutputSchema, Output: r.Output, Message: r.Message}
}

// IdeaAdd writes an idea (F1) or a change request (F2) into the inbox, exactly as given: the text travels in a
// file, so a newline, a quote or a dollar sign cannot change what is stored. The Bootstrap shows what it will
// create and the human confirms (--yes pre-authorizes). A Bootstrap from before the inbox only keeps plain ideas.
func (a *App) IdeaAdd(ctx context.Context, o IdeaAddOptions) (IdeaResult, error) {
	res := IdeaResult{Schema: core.OutputSchema}
	request := o.Type == "solicitud"
	if o.Type != "" && o.Type != "idea" && !request {
		return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "--type is idea or solicitud.", Next: "An ADR draft is: gtt adr new \"<title>\""}
	}
	if request {
		if strings.TrimSpace(o.Text) != "" || o.File != "" {
			return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "A request is written with --change and --reason, not as free text."}
		}
	} else {
		if strings.TrimSpace(o.Text) == "" && o.File == "" {
			return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "The idea is empty.", Next: "Use: gtt idea add 'your idea'"}
		}
		if strings.TrimSpace(o.Text) != "" && o.File != "" {
			return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Give the idea as text or with --file, not both."}
		}
	}
	p, ok, err := a.openInbox(ctx)
	if err != nil {
		return res, err
	}
	if !ok {
		if request || o.File != "" || o.Now || !supports(ctx, p.B, "idea.add") {
			res.Message = notInboxMessage
			return res, nil
		}
		return a.ideaAddLegacy(ctx, p, o.Text)
	}
	if request {
		st, err := a.ChangeRequest(ctx, ChangeOptions{Change: o.Change, Reason: o.Reason, Trigger: o.Trigger, Scope: o.Scope,
			Impact: o.Impact, Risk: o.Risk, Priority: o.Priority, By: o.By})
		res.Output = st.Output
		return res, err
	}
	args := map[string]string{}
	if o.File != "" {
		args["file"] = a.pathFor(p.Root, o.File)
	} else {
		name, err := tempText(o.Text)
		if err != nil {
			return res, &core.Error{Code: core.ExitFailure, What: "The idea could not be staged.", Why: err.Error(), Unmodified: true}
		}
		defer os.Remove(name)
		args["file"] = name
	}
	if o.Now {
		args["now"] = "true"
	}
	if o.By != "" {
		args["by"] = o.By
	}
	r, err := a.inboxWrite(ctx, p, "inbox.add", args, "Write this idea down?")
	return fromInbox(r), err
}

// ideaAddLegacy is the idea of a Bootstrap from before the inbox: no preview, the command is the confirmation.
func (a *App) ideaAddLegacy(ctx context.Context, p *Project, text string) (IdeaResult, error) {
	name, err := tempText(text)
	if err != nil {
		return IdeaResult{Schema: core.OutputSchema}, &core.Error{Code: core.ExitFailure, What: "The idea could not be staged.", Why: err.Error(), Unmodified: true}
	}
	defer os.Remove(name)
	return a.ideaAct(ctx, p, "idea.add", map[string]string{"file": name})
}

// IdeaPark and IdeaDrop are the human's: set an idea aside, or discard it.
func (a *App) IdeaPark(ctx context.Context, ref string) (IdeaResult, error) {
	return a.ideaMove(ctx, "idea.park", ref, func() (InboxResult, error) { return a.InboxPark(ctx, ref, "") })
}

func (a *App) IdeaDrop(ctx context.Context, ref string) (IdeaResult, error) {
	return a.ideaMove(ctx, "idea.drop", ref, func() (InboxResult, error) { return a.InboxDrop(ctx, ref, "") })
}

func (a *App) ideaMove(ctx context.Context, op, ref string, viaInbox func() (InboxResult, error)) (IdeaResult, error) {
	p, ok, err := a.openInbox(ctx)
	if err != nil {
		return IdeaResult{Schema: core.OutputSchema}, err
	}
	if ok {
		r, err := viaInbox()
		return fromInbox(r), err
	}
	if !supports(ctx, p.B, op) {
		return IdeaResult{Schema: core.OutputSchema, Message: notInboxMessage}, nil
	}
	return a.ideaRef(ctx, op, ref)
}

func (a *App) ideaAct(ctx context.Context, p *Project, op string, args map[string]string) (IdeaResult, error) {
	res := IdeaResult{Schema: core.OutputSchema}
	out, err := p.B.Execute(ctx, ports.OperationRequest{Operation: op, Args: args, Apply: true, ConfirmedByHuman: true})
	if err != nil {
		return res, err
	}
	res.Output = strings.TrimSpace(out.Stdout + "\n" + out.Stderr)
	a.log(p.Root, op, outcome(out.OK()), nil)
	if !out.OK() {
		return res, &core.Error{Code: core.ExitConflict, What: fmt.Sprintf("The Bootstrap refused %s.", op), Why: res.Output, Unmodified: true}
	}
	return res, nil
}

// Ideas lists the ideas (all=true adds the dropped ones).
func (a *App) Ideas(ctx context.Context, all bool) (IdeaList, error) {
	res := IdeaList{Schema: core.OutputSchema}
	p, err := a.openIdeas(ctx)
	if err != nil {
		return res, err
	}
	args := map[string]string{}
	if all {
		args["all"] = "true"
	}
	if _, err = a.query(ctx, p.B, "idea.list", args, &res); err != nil {
		return res, err
	}
	res.Schema = core.OutputSchema
	return res, nil
}

func (a *App) ideaRef(ctx context.Context, op, ref string) (IdeaResult, error) {
	p, err := a.openIdeas(ctx)
	if err != nil {
		return IdeaResult{Schema: core.OutputSchema}, err
	}
	return a.ideaAct(ctx, p, op, map[string]string{"idea": strings.ToUpper(strings.TrimSpace(ref))})
}
