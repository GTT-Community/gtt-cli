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

// IdeaResult is what an idea act wrote.
type IdeaResult struct {
	Schema int    `json:"schema"`
	Output string `json:"output"`
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

// IdeaAdd stores the text as a new idea, exactly as written. The text travels in a file, so a newline, a quote
// or a dollar sign cannot change what is stored.
func (a *App) IdeaAdd(ctx context.Context, text string) (IdeaResult, error) {
	res := IdeaResult{Schema: core.OutputSchema}
	if strings.TrimSpace(text) == "" {
		return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "The idea is empty.", Next: "Use: gtt idea add 'your idea'"}
	}
	p, err := a.openIdeas(ctx)
	if err != nil {
		return res, err
	}
	f, err := os.CreateTemp("", "gtt-idea-*.txt")
	if err != nil {
		return res, &core.Error{Code: core.ExitFailure, What: "The idea could not be staged.", Why: err.Error(), Unmodified: true}
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return res, &core.Error{Code: core.ExitFailure, What: "The idea could not be staged.", Why: err.Error(), Unmodified: true}
	}
	f.Close()
	return a.ideaAct(ctx, p, "idea.add", map[string]string{"file": f.Name()})
}

// IdeaPark and IdeaDrop are the human's: set an idea aside, or discard it.
func (a *App) IdeaPark(ctx context.Context, ref string) (IdeaResult, error) {
	return a.ideaRef(ctx, "idea.park", ref)
}

func (a *App) IdeaDrop(ctx context.Context, ref string) (IdeaResult, error) {
	return a.ideaRef(ctx, "idea.drop", ref)
}

func (a *App) ideaRef(ctx context.Context, op, ref string) (IdeaResult, error) {
	p, err := a.openIdeas(ctx)
	if err != nil {
		return IdeaResult{Schema: core.OutputSchema}, err
	}
	return a.ideaAct(ctx, p, op, map[string]string{"idea": strings.ToUpper(strings.TrimSpace(ref))})
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
