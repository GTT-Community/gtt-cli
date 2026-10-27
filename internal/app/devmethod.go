package app

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/lifecycle"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Method Governance: the development method is declared at setup, ratified
// by the Bootstrap as the Method Lock at freeze, and verified by `gtt check`.
// The questions, the options and every text come from the Bootstrap's
// Method Packs (operation method.list). The CLI renders and records them; it
// never writes a text of its own, never recommends and never applies a
// default: an unanswered question stays OPEN.

// DevMethodQuestion is one setup question, as the Bootstrap declares it.
type DevMethodQuestion struct {
	ID       string            `json:"id"`
	Order    int               `json:"order"`
	Multiple bool              `json:"multiple"`
	Prompt   string            `json:"prompt"`
	Options  []DevMethodOption `json:"options"`
	Extra    []DevMethodExtra  `json:"extra_options"`
}

// DevMethodOption is one Method Pack.
type DevMethodOption struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Kind             string   `json:"kind"`
	Recommended      bool     `json:"recommended"`
	RequiresADR      bool     `json:"requires_adr"`
	Verification     string   `json:"verification"`
	WhatItIs         string   `json:"what_it_is"`
	FitsWhen         string   `json:"fits_when"`
	IncompatibleWith []string `json:"incompatible_with"`
}

// DevMethodExtra is an option that is not a pack (help me choose, direct).
type DevMethodExtra struct {
	ID      string `json:"id"`
	Records string `json:"records"`
	Label   string `json:"label"`
	Summary string `json:"summary"`
}

// DevMethodCatalog is `gtt dev-method list`.
type DevMethodCatalog struct {
	Schema    int                 `json:"schema"`
	Lang      string              `json:"lang"`
	Questions []DevMethodQuestion `json:"questions"`
}

// DevMethodState is `gtt dev-method`: the method in force, read-only.
type DevMethodState struct {
	Schema    int            `json:"schema"`
	Supported bool           `json:"supported"`
	Frozen    bool           `json:"frozen"`
	InForce   DevMethodForce `json:"in_force"`
	Lock      DevMethodLock  `json:"lock"`
	Change    string         `json:"change"`
}

// IDList is a list of pack ids. The Bootstrap writes an undecided list as the
// single word OPEN; it is read as ["OPEN"].
type IDList []string

// UnmarshalJSON accepts a list, a single string or null.
func (l *IDList) UnmarshalJSON(data []byte) error {
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*l = IDList{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*l = many
	return nil
}

// DevMethodForce is what applies now: the lock, else the setup answer, else nothing.
type DevMethodForce struct {
	Source string `json:"source"` // lock | setup | none
	Status string `json:"status"` // ratified | open | selected | not declared
	Style  IDList `json:"style"`
	Start  string `json:"start"`
	Packs  IDList `json:"packs"`
}

// DevMethodLock is the Method Lock as the Bootstrap reads it.
type DevMethodLock struct {
	Present      bool   `json:"present"`
	Style        IDList `json:"style"`
	Start        string `json:"start"`
	Status       string `json:"status"`
	LockedBy     string `json:"locked_by"`
	Verification string `json:"verification"`
	Gap          string `json:"gap"`
}

// devMethodSupported says whether a Bootstrap offers Method Governance.
// An earlier Bootstrap does not: nothing is asked and nothing is refused.
func devMethodSupported(ctx context.Context, b ports.Bootstrap) bool {
	ops, err := b.Operations(ctx)
	if err != nil {
		return false
	}
	_, ok := ops["method.list"]
	return ok
}

// devMethodCatalog reads the questions in a language.
func (a *App) devMethodCatalog(ctx context.Context, b ports.Bootstrap, lang string) (DevMethodCatalog, error) {
	if lang != "es" && lang != "en" {
		lang = "en"
	}
	cat := DevMethodCatalog{Schema: core.OutputSchema, Lang: lang}
	_, err := a.query(ctx, b, "method.list", map[string]string{"lang": lang}, &cat)
	cat.Schema, cat.Lang = core.OutputSchema, lang
	return cat, err
}

func (q DevMethodQuestion) options(lang string) []ports.Option {
	recommended, fits := " (recommended)", "Fits when: "
	if lang == "es" {
		recommended, fits = " (recomendado)", "Conviene si: "
	}
	out := make([]ports.Option, 0, len(q.Options)+len(q.Extra))
	for _, o := range q.Options {
		label := o.Name
		if o.Recommended {
			label += recommended
		}
		out = append(out, ports.Option{Label: label, Detail: strings.TrimSpace(o.WhatItIs + " " + fits + o.FitsWhen)})
	}
	for _, e := range q.Extra {
		out = append(out, ports.Option{Label: e.Label, Detail: e.Summary})
	}
	return out
}

// choice maps picked indexes back to pack ids or an extra option id.
func (q DevMethodQuestion) choice(picked []int) string {
	var ids []string
	for _, i := range picked {
		if i < len(q.Options) {
			ids = append(ids, q.Options[i].ID)
			continue
		}
		if extra := q.Extra[i-len(q.Options)]; extra.ID == "help" {
			return "help" // help me choose decides nothing, whatever else was ticked
		} else {
			ids = append(ids, extra.ID)
		}
	}
	return strings.Join(ids, ",")
}

// collectDevMethod asks the two questions, or takes them from the flags.
// Without a terminal and without flags nothing is recorded: the method stays
// OPEN. A combination the Bootstrap refuses is shown and asked again.
func (a *App) collectDevMethod(ctx context.Context, catalog ports.Bootstrap, sel *lifecycle.Selections, o InitOptions) error {
	if !devMethodSupported(ctx, catalog) {
		if o.DevStyle != "" || o.DevStart != "" {
			return &core.Error{Code: core.ExitIncompatible, Unmodified: true,
				What: "This GTT Bootstrap does not offer Method Governance (no method.* operations).",
				Next: "Drop --dev-method/--start, or use a Bootstrap that declares them."}
		}
		return nil
	}
	check := func(style, start string) error {
		args := map[string]string{}
		if style != "" {
			args["style"] = style
		}
		if start != "" {
			args["start"] = start
		}
		if len(args) == 0 {
			return nil
		}
		res, err := catalog.Execute(ctx, ports.OperationRequest{Operation: "method.select", Args: args})
		if err != nil {
			return err
		}
		if !res.OK() {
			return &core.Error{Code: core.ExitUsage, Unmodified: true, What: "The development method was refused by the Bootstrap.",
				Why: strings.TrimSpace(res.Stdout + "\n" + res.Stderr)}
		}
		return nil
	}
	if o.DevStyle != "" || o.DevStart != "" || !a.Prompt.Interactive() {
		if err := check(o.DevStyle, o.DevStart); err != nil {
			return err
		}
		sel.DevStyle, sel.DevStart = o.DevStyle, o.DevStart
		return nil
	}
	cat, err := a.devMethodCatalog(ctx, catalog, sel.Language)
	if err != nil {
		return err
	}
	for _, q := range cat.Questions {
		for attempt := 0; ; attempt++ {
			var answer string
			if q.Multiple {
				picked, err := a.Prompt.MultiSelect(q.Prompt, q.options(cat.Lang))
				if err != nil {
					return err
				}
				answer = q.choice(picked)
			} else {
				i, err := a.Prompt.Select(q.Prompt, q.options(cat.Lang))
				if err != nil {
					return err
				}
				answer = q.choice([]int{i})
			}
			style, start := sel.DevStyle, sel.DevStart
			if q.ID == "development" {
				style = answer
			} else {
				start = answer
			}
			err := check(style, start)
			if err == nil {
				sel.DevStyle, sel.DevStart = style, start
				break
			}
			if attempt >= 2 {
				return err
			}
			a.Report.Info("%s\nChoose again.", err.Error())
		}
	}
	return nil
}

// recordDevMethod writes the setup answer through the Bootstrap.
func (a *App) recordDevMethod(ctx context.Context, b ports.Bootstrap, sel lifecycle.Selections) error {
	if sel.DevStyle == "" && sel.DevStart == "" {
		return nil
	}
	args := map[string]string{}
	if sel.DevStyle != "" {
		args["style"] = sel.DevStyle
	}
	if sel.DevStart != "" {
		args["start"] = sel.DevStart
	}
	_, err := a.mutate(ctx, b, "method.select", args)
	return err
}

// devMethodSummary is the one line init shows before it proceeds.
func devMethodSummary(sel lifecycle.Selections) string {
	show := func(v string) string {
		switch v {
		case "", "help":
			return "OPEN"
		}
		return v
	}
	return "style " + show(sel.DevStyle) + ", start " + show(sel.DevStart)
}

// DevMethodShow is `gtt dev-method`: read-only.
func (a *App) DevMethodShow(ctx context.Context) (DevMethodState, error) {
	res := DevMethodState{Schema: core.OutputSchema}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	if res.Supported = devMethodSupported(ctx, p.B); !res.Supported {
		return res, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: "The installed GTT Bootstrap " + p.Release.Bootstrap.Version + " does not offer Method Governance.", Next: "Run: gtt update"}
	}
	var got struct {
		InForce DevMethodForce `json:"in_force"`
		Lock    DevMethodLock  `json:"lock"`
		Frozen  bool           `json:"frozen"`
		Change  string         `json:"change"`
	}
	if _, err := a.query(ctx, p.B, "method.get", nil, &got); err != nil {
		return res, err
	}
	res.InForce, res.Lock, res.Frozen, res.Change = got.InForce, got.Lock, got.Frozen, got.Change
	return res, nil
}

// DevMethodList is `gtt dev-method list`.
func (a *App) DevMethodList(ctx context.Context, lang string) (DevMethodCatalog, error) {
	p, err := a.open(ctx)
	if err != nil {
		return DevMethodCatalog{}, err
	}
	if !devMethodSupported(ctx, p.B) {
		return DevMethodCatalog{}, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: "The installed GTT Bootstrap " + p.Release.Bootstrap.Version + " does not offer Method Governance.", Next: "Run: gtt update"}
	}
	if lang == "" {
		var plan struct {
			Language string `json:"language"`
		}
		_, _ = a.query(ctx, p.B, "methodology.profile.get", nil, &plan)
		lang = plan.Language
	}
	return a.devMethodCatalog(ctx, p.B, lang)
}

// devMethodNote explains, on an initialized project, why the development
// method is not asked again. Asking for it on the command line is refused
// in a frozen project: the ratified method changes by change request only.
func (a *App) devMethodNote(ctx context.Context, root string, o InitOptions) string {
	b := a.Factory.At(root)
	if !devMethodSupported(ctx, b) {
		return ""
	}
	var got struct {
		Frozen bool `json:"frozen"`
	}
	if _, err := a.query(ctx, b, "method.get", nil, &got); err != nil || !got.Frozen {
		return ""
	}
	return "The project is frozen: the development method is not asked again. It changes only through a change request " +
		"(gtt-domain/change-request.md); see it with: gtt dev-method"
}
