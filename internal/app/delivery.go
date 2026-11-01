package app

import (
	"context"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/lifecycle"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Delivery Planning: who builds, with how many parallel fronts and for when,
// the plan before starting and the real time while working. The questions,
// their options and every text come from the Bootstrap's delivery contract
// (operation delivery.questions); the plan, the forecast and the report are
// computed by the Bootstrap (delivery.plan, delivery.report). The CLI renders
// and records: it writes no question of its own, applies no default and never
// launches an agent. An unanswered question stays OPEN.

// DeliveryQuestion is one setup question, as the Bootstrap declares it.
type DeliveryQuestion struct {
	ID      string              `json:"id"`
	Key     string              `json:"key"`
	AskedBy string              `json:"asked_by"`
	OnlyIf  map[string][]string `json:"only_if"`
	Prompt  string              `json:"prompt"`
	Note    string              `json:"note"`
	Options []DeliveryOption    `json:"options"`
}

// DeliveryOption is one answer.
type DeliveryOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Recommended bool   `json:"recommended"`
	Days        int    `json:"days"`
}

// DeliveryQuestions is `delivery.questions`.
type DeliveryQuestions struct {
	Schema    int                `json:"schema"`
	Questions []DeliveryQuestion `json:"questions"`
}

// DeliveryState is `gtt delivery`: the setup answer and what is in force.
type DeliveryState struct {
	Schema     int            `json:"schema"`
	Supported  bool           `json:"supported"`
	Declared   bool           `json:"declared"`
	State      map[string]any `json:"state"`
	Thresholds map[string]any `json:"thresholds"`
	Profile    string         `json:"profile"`
}

// DeliveryPlan is `gtt plan` or `gtt plan --lanes`; only one of the two sets is filled.
type DeliveryPlan struct {
	Schema      int                 `json:"schema"`
	Supported   bool                `json:"supported"`
	Profile     string              `json:"profile"`
	Calibration DeliveryCalibration `json:"calibration"`
	Lanes       any                 `json:"lanes"`
	Chain       []string            `json:"chain"`
	ChainHours  float64             `json:"chain_hours"`
	Days        int                 `json:"days"`
	ETA         string              `json:"eta"`
	Deadline    string              `json:"deadline"`
	Fits        *bool               `json:"fits"`
	Late        *struct {
		Days int `json:"late_days"`
	} `json:"late"`
	Levers []struct {
		Epic     string  `json:"epic"`
		SavesH   float64 `json:"saves_h"`
		Lever    string  `json:"lever"`
		Critical bool    `json:"on_critical_chain"`
	} `json:"levers"`
	Unestimated []string `json:"unestimated"`
	OverCap     []string `json:"over_cap"`
	Cycles      []string `json:"cycles"`
	Waiting     []struct {
		Story string  `json:"story"`
		Hours float64 `json:"waiting_h"`
	} `json:"waiting_on_you"`
	// WorkTrees is the `--lanes` view: one entry per front.
	Fronts []DeliveryFront `json:"fronts,omitempty"`
}

// DeliveryFront is one front of `gtt plan --lanes`.
type DeliveryFront struct {
	Lane      string   `json:"lane"`
	Owners    []string `json:"owners"`
	Epics     []string `json:"epics"`
	Touches   []string `json:"touches"`
	DependsOn []string `json:"depends_on"`
	DoneWhen  []string `json:"done_when"`
	Worktree  string   `json:"worktree"`
}

// DeliveryCalibration says how far the estimates are measured.
type DeliveryCalibration struct {
	Measured     int      `json:"measured"`
	Needed       int      `json:"needed"`
	Uncalibrated bool     `json:"uncalibrated"`
	Ratio        *float64 `json:"ratio"`
}

// DeliveryReport is `gtt report`.
type DeliveryReport struct {
	Schema       int                 `json:"schema"`
	Supported    bool                `json:"supported"`
	Uncalibrated bool                `json:"uncalibrated"`
	Calibration  DeliveryCalibration `json:"calibration"`
	Epics        []struct {
		Epic        string   `json:"epic"`
		Status      string   `json:"status"`
		Lane        string   `json:"lane"`
		EstimateH   *float64 `json:"estimate_h"`
		Percent     int      `json:"percent"`
		PercentBy   string   `json:"percent_by"`
		Stories     int      `json:"stories"`
		StoriesDone int      `json:"stories_done"`
		RealH       float64  `json:"real_h"`
	} `json:"epics"`
	Forecast struct {
		Chain    []string `json:"chain"`
		Hours    float64  `json:"hours"`
		ETA      string   `json:"eta"`
		Deadline string   `json:"deadline"`
		Fits     *bool    `json:"fits"`
	} `json:"forecast"`
	PlannedH float64 `json:"planned_h"`
	RealH    float64 `json:"real_h"`
	Signals  []struct {
		Rule     string `json:"rule"`
		Artifact string `json:"artifact"`
		Detail   string `json:"detail"`
	} `json:"signals"`
}

// DeliverySetOptions are the answers given on the command line.
type DeliverySetOptions struct {
	Who, Lanes, Horizon, Priority, ThirdParty, Restrictions, EpicCap, Team string
}

func (o DeliverySetOptions) empty() bool { return o == DeliverySetOptions{} }

func (o DeliverySetOptions) args() map[string]string {
	args := map[string]string{}
	for k, v := range map[string]string{"who": o.Who, "lanes": o.Lanes, "horizon": o.Horizon, "priority": o.Priority,
		"third_party": o.ThirdParty, "restrictions": o.Restrictions, "epic_cap": o.EpicCap, "team": o.Team} {
		if v != "" {
			args[k] = v
		}
	}
	return args
}

// deliverySupported says whether a Bootstrap offers Delivery Planning. An
// earlier Bootstrap does not: nothing is asked and nothing is refused.
func deliverySupported(ctx context.Context, b ports.Bootstrap) bool {
	ops, err := b.Operations(ctx)
	if err != nil {
		return false
	}
	_, ok := ops["delivery.questions"]
	return ok
}

func notSupported(version string) error {
	return &core.Error{Code: core.ExitIncompatible, Unmodified: true,
		What: "The installed GTT Bootstrap " + version + " does not offer Delivery Planning.", Next: "Run: gtt update"}
}

func (a *App) deliveryQuestions(ctx context.Context, b ports.Bootstrap, lang string) (DeliveryQuestions, error) {
	if lang != "es" && lang != "en" {
		lang = "en"
	}
	var q DeliveryQuestions
	_, err := a.query(ctx, b, "delivery.questions", map[string]string{"lang": lang}, &q)
	return q, err
}

func (q DeliveryQuestion) options() []ports.Option {
	out := make([]ports.Option, 0, len(q.Options))
	for _, o := range q.Options {
		label := o.Label
		if o.Recommended {
			label += " (recommended)"
		}
		out = append(out, ports.Option{Label: label, Detail: ""})
	}
	return out
}

// collectDelivery asks D1-D3 after the Autonomy Profile, or takes them from the
// flags. Without a terminal and without flags nothing is recorded: they stay
// OPEN. A Bootstrap without Delivery Planning is skipped with one line.
func (a *App) collectDelivery(ctx context.Context, catalog ports.Bootstrap, sel *lifecycle.Selections, o InitOptions) error {
	given := o.DeliveryWho != "" || o.DeliveryLanes != "" || o.DeliveryHorizon != "" || o.DeliveryTeam != ""
	if !deliverySupported(ctx, catalog) {
		if given {
			return &core.Error{Code: core.ExitIncompatible, Unmodified: true,
				What: "This GTT Bootstrap does not offer Delivery Planning (no delivery.* operations).",
				Next: "Drop --team/--lanes/--horizon, or use a Bootstrap that declares them."}
		}
		a.Report.Info("This Bootstrap has no delivery planning: who builds and for when is not asked.")
		return nil
	}
	if given || !a.Prompt.Interactive() {
		sel.DeliveryWho, sel.DeliveryLanes, sel.DeliveryHorizon, sel.DeliveryTeam = o.DeliveryWho, o.DeliveryLanes, o.DeliveryHorizon, o.DeliveryTeam
		return nil
	}
	cat, err := a.deliveryQuestions(ctx, catalog, sel.Language)
	if err != nil {
		return err
	}
	for _, q := range cat.Questions {
		if q.AskedBy != "init" {
			continue
		}
		if need, ok := q.OnlyIf["who"]; ok && !contains(need, sel.DeliveryWho) {
			continue
		}
		title := q.Prompt
		if q.Note != "" {
			title += "\n  " + q.Note
		}
		i, err := a.Prompt.Select(title, q.options())
		if err != nil {
			return err
		}
		answer := q.Options[i].ID
		switch q.Key {
		case "who":
			sel.DeliveryWho = answer
			if answer == "team" {
				names, err := a.Prompt.Ask("Team members (comma separated; they own the fronts)")
				if err != nil {
					return err
				}
				sel.DeliveryTeam = strings.TrimSpace(names)
			}
		case "lanes":
			sel.DeliveryLanes = answer
		case "horizon":
			sel.DeliveryHorizon = answer
		}
	}
	return nil
}

// recordDelivery writes the setup answer through the Bootstrap.
func (a *App) recordDelivery(ctx context.Context, b ports.Bootstrap, sel lifecycle.Selections) error {
	if sel.DeliveryWho == "" && sel.DeliveryLanes == "" && sel.DeliveryHorizon == "" && sel.DeliveryTeam == "" {
		return nil
	}
	if !deliverySupported(ctx, b) {
		return nil
	}
	o := DeliverySetOptions{Who: sel.DeliveryWho, Lanes: sel.DeliveryLanes, Horizon: sel.DeliveryHorizon, Team: sel.DeliveryTeam}
	_, err := a.mutate(ctx, b, "delivery.select", o.args())
	return err
}

// deliverySummary is the one line init shows before it proceeds.
func deliverySummary(sel lifecycle.Selections) string {
	show := func(v string) string {
		if v == "" {
			return "OPEN"
		}
		return v
	}
	return "who " + show(sel.DeliveryWho) + ", fronts " + show(sel.DeliveryLanes) + ", horizon " + show(sel.DeliveryHorizon)
}

// DeliveryShow is `gtt delivery`: read-only.
func (a *App) DeliveryShow(ctx context.Context) (DeliveryState, error) {
	res := DeliveryState{Schema: core.OutputSchema}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	if res.Supported = deliverySupported(ctx, p.B); !res.Supported {
		return res, notSupported(p.Release.Bootstrap.Version)
	}
	_, err = a.query(ctx, p.B, "delivery.get", nil, &res)
	res.Schema, res.Supported = core.OutputSchema, true
	return res, err
}

// DeliverySet is `gtt delivery set`: an operating answer, allowed at any time.
func (a *App) DeliverySet(ctx context.Context, o DeliverySetOptions) (DeliveryState, error) {
	p, err := a.open(ctx)
	if err != nil {
		return DeliveryState{}, err
	}
	if !deliverySupported(ctx, p.B) {
		return DeliveryState{}, notSupported(p.Release.Bootstrap.Version)
	}
	if o.empty() {
		return DeliveryState{}, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Nothing to set.",
			Next: "Give --who, --lanes, --horizon, --priority, --third-party, --restrictions, --epic-cap or --team."}
	}
	if _, err := a.mutate(ctx, p.B, "delivery.select", o.args()); err != nil {
		return DeliveryState{}, err
	}
	return a.DeliveryShow(ctx)
}

// DeliveryPlanFor is `gtt plan [--lanes]`.
func (a *App) DeliveryPlanFor(ctx context.Context, lanes bool) (DeliveryPlan, error) {
	res := DeliveryPlan{Schema: core.OutputSchema}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	if res.Supported = deliverySupported(ctx, p.B); !res.Supported {
		return res, notSupported(p.Release.Bootstrap.Version)
	}
	args := map[string]string{}
	if lanes {
		args["lanes"] = "true"
		var raw struct {
			Lanes []DeliveryFront `json:"lanes"`
		}
		if _, err := a.query(ctx, p.B, "delivery.plan", args, &raw); err != nil {
			return res, err
		}
		res.Fronts = raw.Lanes
		return res, nil
	}
	_, err = a.query(ctx, p.B, "delivery.plan", args, &res)
	res.Schema, res.Supported = core.OutputSchema, true
	return res, err
}

// DeliveryReportFor is `gtt report [--epic]`.
func (a *App) DeliveryReportFor(ctx context.Context, epic string) (DeliveryReport, error) {
	res := DeliveryReport{Schema: core.OutputSchema}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	if res.Supported = deliverySupported(ctx, p.B); !res.Supported {
		return res, notSupported(p.Release.Bootstrap.Version)
	}
	args := map[string]string{}
	if epic != "" {
		args["epic"] = epic
	}
	_, err = a.query(ctx, p.B, "delivery.report", args, &res)
	res.Schema, res.Supported = core.OutputSchema, true
	return res, err
}

// deliveryNote explains, on an initialized project, why init does not ask again.
func (a *App) deliveryNote(ctx context.Context, root string) string {
	b := a.Factory.At(root)
	if !deliverySupported(ctx, b) {
		return ""
	}
	return "Delivery planning is not asked again; change it any time with: gtt delivery set"
}

// deliveryLine is the summary `gtt status` shows ("" when the Bootstrap has none).
func (a *App) deliveryLine(ctx context.Context, b ports.Bootstrap) string {
	if !deliverySupported(ctx, b) {
		return ""
	}
	res, err := b.Execute(ctx, ports.OperationRequest{Operation: "delivery.summary"})
	if err != nil || !res.OK() {
		return ""
	}
	return strings.TrimSpace(res.Stdout)
}
