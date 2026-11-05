package app

import (
	"context"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Delivery review: how big the backlog is and the way back to the answers. GTT
// measures and reports; what is too much is the human's to say. The numbers, the
// choices and their texts come from the Bootstrap (delivery.size); the CLI shows
// them and records the human's answer through delivery.select.

// DeliverySize is `delivery.size`.
type DeliverySize struct {
	Schema       int               `json:"schema"`
	Supported    bool              `json:"supported"`
	Declared     bool              `json:"declared"`
	Pace         string            `json:"pace"`
	Deadline     string            `json:"deadline"`
	OpenStories  int               `json:"open_stories"`
	DoneStories  int               `json:"done_stories"`
	LiveEpics    int               `json:"live_epics"`
	OpenEpics    int               `json:"open_epics"`
	CoveragePct  int               `json:"coverage_pct"`
	Measurable   bool              `json:"measurable"`
	RemainingH   *float64          `json:"remaining_h"`
	DaysOfWork   *int              `json:"days_of_work"`
	Fronts       int               `json:"fronts"`
	CapacityH    *float64          `json:"capacity_h"`
	OfCapacity   *int              `json:"of_capacity_pct"`
	GrowthLimit  float64           `json:"growth_pct_limit"`
	Target       DeliveryTarget    `json:"target"`
	Baseline     *DeliveryBaseline `json:"baseline"`
	Growth       *DeliveryGrowth   `json:"growth"`
	Choices      []DeliveryChoice  `json:"choices"`
	SlimNote     string            `json:"slim_note"`
	ExpandNote   string            `json:"expand_note"`
	TotalH       *float64          `json:"total_h"`
	Floor        DeliveryFloor     `json:"floor"`
	PaceValues   map[string]string `json:"pace_values"`
	Prompts      map[string]string `json:"prompts"`
	Action       string            `json:"action,omitempty"`
	Levers       []DeliveryLever   `json:"levers,omitempty"`
	ChangedState *DeliveryState    `json:"state,omitempty"`
}

// DeliveryTarget is the cap the human declared, if any.
type DeliveryTarget struct {
	Declared bool     `json:"declared"`
	CapH     *float64 `json:"cap_h"`
	State    string   `json:"state"`
	OverPct  *int     `json:"over_pct"`
}

// DeliveryFloor is the minimum the human declared, if any.
type DeliveryFloor struct {
	Declared bool     `json:"declared"`
	MinH     *float64 `json:"min_h"`
	State    string   `json:"state"`
}

// DeliveryBaseline is the size when the human declared the plan.
type DeliveryBaseline struct {
	At          string   `json:"at"`
	OpenStories int      `json:"open_stories"`
	RemainingH  *float64 `json:"remaining_h"`
}

// DeliveryGrowth is the change since the baseline.
type DeliveryGrowth struct {
	StoriesAdded int  `json:"stories_added"`
	StoriesPct   *int `json:"stories_pct"`
	HoursPct     *int `json:"hours_pct"`
}

// DeliveryChoice is one way forward the Bootstrap offers in the review.
type DeliveryChoice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// DeliveryLever is one thing the plan says could shorten the work.
type DeliveryLever struct {
	Epic   string  `json:"epic"`
	SavesH float64 `json:"saves_h"`
	Lever  string  `json:"lever"`
}

// ReviewOptions are the choices given on the command line; without them a terminal is asked.
type ReviewOptions struct {
	Keep, SinglePass bool
	ScopeCap         string
	ScopeMin         string
	GrowthPct        string
}

// DeliverySizeFor is the read-only measurement.
func (a *App) DeliverySizeFor(ctx context.Context, lang string) (DeliverySize, error) {
	res := DeliverySize{Schema: core.OutputSchema}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	if !deliverySupported(ctx, p.B) {
		return res, notSupported(p.Release.Bootstrap.Version)
	}
	ops, err := p.B.Operations(ctx)
	if err != nil {
		return res, err
	}
	if _, ok := ops["delivery.size"]; !ok {
		return res, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: "The installed GTT Bootstrap " + p.Release.Bootstrap.Version + " cannot measure the size of the backlog (no delivery.size).",
			Next: "Run: gtt update"}
	}
	if lang != "es" {
		lang = "en"
	}
	_, err = a.query(ctx, p.B, "delivery.size", map[string]string{"lang": lang}, &res)
	res.Schema, res.Supported = core.OutputSchema, true
	return res, err
}

// DeliveryReview measures, then records the human's answer: from the flags, or by asking on a terminal.
// With neither it only reports, and says how to answer.
func (a *App) DeliveryReview(ctx context.Context, lang string, o ReviewOptions) (DeliverySize, error) {
	res, err := a.DeliverySizeFor(ctx, lang)
	if err != nil {
		return res, err
	}
	choice := ""
	switch {
	case o.Keep:
		choice = "keep"
	case o.SinglePass:
		choice = "single-pass"
	case o.ScopeCap != "" || o.ScopeMin != "" || o.GrowthPct != "":
		choice = "target"
	case a.Prompt.Interactive():
		labels := make([]ports.Option, 0, len(res.Choices))
		for _, c := range res.Choices {
			labels = append(labels, ports.Option{Label: c.Label})
		}
		i, err := a.Prompt.Select(res.Prompts["choose"], labels)
		if err != nil {
			return res, err
		}
		choice = res.Choices[i].ID
	}
	if choice == "" {
		return res, nil
	}
	set := DeliverySetOptions{}
	switch choice {
	case "keep":
		set.Rebase = true
	case "single-pass":
		set.Preset, set.Rebase = "single-pass", true
	case "target":
		set.ScopeCap, set.ScopeMin, set.GrowthPct = o.ScopeCap, o.ScopeMin, o.GrowthPct
		if set.ScopeCap == "" && set.ScopeMin == "" && set.GrowthPct == "" {
			answer, err := a.Prompt.Ask(res.Prompts["cap"])
			if err != nil {
				return res, err
			}
			set.ScopeCap = strings.TrimSpace(answer)
			answer, err = a.Prompt.Ask(res.Prompts["floor"])
			if err != nil {
				return res, err
			}
			set.ScopeMin = strings.TrimSpace(answer)
		}
	case "expand":
		res.Action = "expand"
		return res, nil
	case "slim":
		res.Action = "slim"
		if plan, err := a.DeliveryPlanFor(ctx, false); err == nil {
			for _, l := range plan.Levers {
				res.Levers = append(res.Levers, DeliveryLever{Epic: l.Epic, SavesH: l.SavesH, Lever: l.Lever})
			}
		}
		return res, nil
	case "answers":
		var err error
		if set, err = a.askDeliveryAnswers(ctx, lang, res); err != nil {
			return res, err
		}
	}
	if set.empty() {
		return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Nothing to record.",
			Next: "Give --keep, --single-pass or --scope-cap <hours|none>."}
	}
	if _, err := a.DeliverySet(ctx, set); err != nil {
		return res, err
	}
	res.Action = choice
	changed, err := a.DeliverySizeFor(ctx, lang)
	if err != nil {
		return res, err
	}
	changed.Action, changed.Levers = choice, nil
	return changed, nil
}

// askDeliveryAnswers asks again the questions of init that can change (who, fronts, date) and the pace.
func (a *App) askDeliveryAnswers(ctx context.Context, lang string, size DeliverySize) (DeliverySetOptions, error) {
	var set DeliverySetOptions
	p, err := a.open(ctx)
	if err != nil {
		return set, err
	}
	cat, err := a.deliveryQuestions(ctx, p.B, lang)
	if err != nil {
		return set, err
	}
	who := ""
	for _, q := range cat.Questions {
		if q.AskedBy != "init" {
			continue
		}
		if need, ok := q.OnlyIf["who"]; ok && !contains(need, who) {
			continue
		}
		i, err := a.Prompt.Select(q.Prompt, q.options())
		if err != nil {
			return set, err
		}
		switch q.Key {
		case "who":
			set.Who, who = q.Options[i].ID, q.Options[i].ID
		case "lanes":
			set.Lanes = q.Options[i].ID
		case "horizon":
			set.Horizon = q.Options[i].ID
		}
	}
	opts := []ports.Option{{Label: "days - " + size.PaceValues["days"]}, {Label: "flow - " + size.PaceValues["flow"]}}
	i, err := a.Prompt.Select(size.Prompts["pace"], opts)
	if err != nil {
		return set, err
	}
	set.Pace = []string{"days", "flow"}[i]
	return set, nil
}
