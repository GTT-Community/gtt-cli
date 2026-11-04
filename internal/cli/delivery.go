package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/output"
)

// deliveryCmd is `gtt delivery`: the setup answer of Delivery Planning (who
// builds, fronts, deadline). `set` records an operating answer at any time: it
// is not architecture, so it needs no change request.
func deliveryCmd(e *env) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		res, err := e.app.DeliveryShow(cmd.Context())
		return e.emit(res, err, func(p output.Printer) { renderDelivery(p, res) })
	}
	cmd := &cobra.Command{
		Use:   "delivery",
		Short: "Show who builds, with how many fronts and for when (Delivery Planning)",
		Long: "Show the delivery answers: who builds, how many parallel fronts, the deadline, the priority and the cap per\n" +
			"Epic. They are an operating answer, not architecture: change them any time with `gtt delivery set`.",
		Args: cobra.NoArgs,
		RunE: show,
	}
	var o app.DeliverySetOptions
	set := &cobra.Command{
		Use:   "set",
		Short: "Change the delivery answers (allowed at any time)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.DeliverySet(cmd.Context(), o)
			return e.emit(res, err, func(p output.Printer) { renderDelivery(p, res) })
		},
	}
	f := set.Flags()
	f.StringVar(&o.Who, "who", "", "solo, solo-multi or team")
	f.StringVar(&o.Lanes, "lanes", "", "parallel fronts: a number, or propose")
	f.StringVar(&o.Horizon, "horizon", "", "days, weeks, month or none")
	f.StringVar(&o.Pace, "pace", "", "days or flow: the rhythm every plan document must speak in")
	f.StringVar(&o.Priority, "priority", "", "speed, balance or own")
	f.StringVar(&o.ThirdParty, "third-party", "", "third-party services: yes, oss, restricted or OPEN")
	f.StringVar(&o.Restrictions, "restrictions", "", "which restrictions apply to third-party services")
	f.StringVar(&o.EpicCap, "epic-cap", "", "agent hours per Epic before GTT warns (default 8)")
	f.StringVar(&o.Team, "team", "", "team member names, comma separated")
	cmd.AddCommand(&cobra.Command{Use: "show", Short: "Show the delivery answers", Args: cobra.NoArgs, RunE: show}, set)
	return cmd
}

// planCmd is `gtt plan`: the delivery plan, computed by the Bootstrap.
func planCmd(e *env) *cobra.Command {
	var lanes bool
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show the delivery plan: fronts, longest chain, estimated finish and levers",
		Long: "Show the plan per front, the longest dependency chain, the estimated finish against the deadline and, when it does\n" +
			"not fit, the levers with the hours each saves. With --lanes, one summary per front and the `git worktree add`\n" +
			"commands to copy. GTT creates no agent, no session and no worktree.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.DeliveryPlanFor(cmd.Context(), lanes)
			return e.emit(res, err, func(p output.Printer) { renderPlan(p, res, lanes) })
		},
	}
	cmd.Flags().BoolVar(&lanes, "lanes", false, "one summary per front and the worktree commands")
	return cmd
}

// reportCmd is `gtt report`: progress and real time against the estimate.
func reportCmd(e *env) *cobra.Command {
	var epic string
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Show progress per Epic, real against estimated time, calibration and open DLV signals",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.DeliveryReportFor(cmd.Context(), epic)
			return e.emit(res, err, func(p output.Printer) { renderReport(p, res) })
		},
	}
	cmd.Flags().StringVar(&epic, "epic", "", "only this Epic (EPIC-NNN)")
	return cmd
}

func renderDelivery(p output.Printer, r app.DeliveryState) {
	if !r.Declared {
		p.Line("Delivery: not declared yet (gtt delivery set --who ... --horizon ...)")
		return
	}
	get := func(k string) string {
		if v, ok := r.State[k]; ok && v != nil {
			return fmt.Sprint(v)
		}
		return "OPEN"
	}
	p.Line("Delivery (profile %s)", r.Profile)
	for _, k := range []string{"who", "lanes", "horizon", "deadline", "pace", "priority", "third_party"} {
		if v, ok := r.State[k]; ok && v != nil {
			p.Line("  %-12s %s", k+":", fmt.Sprint(v))
		}
	}
	if _, ok := r.State["who"]; !ok {
		p.Line("  who:         %s", get("who"))
	}
	p.Line("  Epic cap:    %v h", r.Thresholds["epic_cap_h"])
	if team, ok := r.State["team"].([]any); ok && len(team) > 0 {
		names := make([]string, 0, len(team))
		for _, n := range team {
			names = append(names, fmt.Sprint(n))
		}
		p.Line("  team:        %s", strings.Join(names, ", "))
	}
}

func renderPlan(p output.Printer, r app.DeliveryPlan, lanes bool) {
	if lanes {
		if len(r.Fronts) == 0 {
			p.Line("No live Epic: nothing to split into fronts.")
			return
		}
		for _, f := range r.Fronts {
			owner := ""
			if len(f.Owners) > 0 {
				owner = " - owner " + strings.Join(f.Owners, ", ")
			}
			p.Line("front %s%s", f.Lane, owner)
			p.Line("  Epics:     %s", strings.Join(f.Epics, ", "))
			p.Line("  may touch: %s", listOr(f.Touches, "(no Touches declared)"))
			p.Line("  depends on (through an interface): %s", listOr(f.DependsOn, "nothing outside the front"))
			for _, d := range f.DoneWhen {
				p.Line("  done when: %s", d)
			}
			p.Line("  %s", f.Worktree)
		}
		p.Line("GTT creates no agent, no session and no worktree: copy the commands you want.")
		return
	}
	cal := "UNCALIBRATED (fewer than the Stories needed are measured; estimates taken as given)"
	if !r.Calibration.Uncalibrated && r.Calibration.Ratio != nil {
		cal = fmt.Sprintf("calibrated on %d Stories (real/estimated x%.2f)", r.Calibration.Measured, *r.Calibration.Ratio)
	}
	p.Line("Delivery plan - profile %s, %s", r.Profile, cal)
	if m, ok := r.Lanes.(map[string]any); ok {
		for lane, ids := range m {
			var list []string
			if arr, ok := ids.([]any); ok {
				for _, v := range arr {
					list = append(list, fmt.Sprint(v))
				}
			}
			p.Line("  front %s: %s", lane, strings.Join(list, " -> "))
		}
	}
	if len(r.Chain) > 0 {
		p.Line("  longest chain: %s = %g agent h (~%d working day(s)), estimated finish %s", strings.Join(r.Chain, " -> "), r.ChainHours, r.Days, r.ETA)
	}
	switch {
	case r.Deadline == "":
		p.Line("  no deadline declared (gtt delivery set --horizon ...)")
	case r.Fits != nil && *r.Fits:
		p.Line("  deadline %s: fits", r.Deadline)
	case r.Late != nil:
		p.Line("  deadline %s: DOES NOT FIT - %d day(s) late", r.Deadline, r.Late.Days)
		for _, l := range r.Levers {
			mark := ""
			if l.Critical {
				mark = " [critical chain]"
			}
			p.Line("    lever: %s (%g h)%s - %s", l.Epic, l.SavesH, mark, l.Lever)
		}
	}
	if len(r.Unestimated) > 0 {
		p.Line("  without Estimate: %s", strings.Join(r.Unestimated, ", "))
	}
	if len(r.OverCap) > 0 {
		p.Line("  over the Epic cap: %s", strings.Join(r.OverCap, ", "))
	}
	for _, w := range r.Waiting {
		p.Line("  waiting on you: %s (%g h)", w.Story, w.Hours)
	}
	p.Line("  every lever is a proposal: you decide (Budget, Build, Lane).")
}

func listOr(list []string, none string) string {
	if len(list) == 0 {
		return none
	}
	return strings.Join(list, ", ")
}

func renderReport(p output.Printer, r app.DeliveryReport) {
	if r.Uncalibrated {
		p.Line("Delivery report - UNCALIBRATED (%d of %d Stories measured)", r.Calibration.Measured, r.Calibration.Needed)
	} else if r.Calibration.Ratio != nil {
		p.Line("Delivery report - calibrated, real/estimated x%.2f over %d Stories", *r.Calibration.Ratio, r.Calibration.Measured)
	}
	for _, e := range r.Epics {
		est := "no estimate"
		if e.EstimateH != nil {
			est = fmt.Sprintf("%gh", *e.EstimateH)
		}
		p.Line("  %-10s %-12s %3d%% (by %s)  %d/%d Stories  est %s, real %gh  front %s",
			e.Epic, e.Status, e.Percent, e.PercentBy, e.StoriesDone, e.Stories, est, e.RealH, e.Lane)
	}
	if r.Forecast.ETA != "" {
		tail := ""
		if r.Forecast.Deadline != "" {
			verdict := "DOES NOT FIT"
			if r.Forecast.Fits != nil && *r.Forecast.Fits {
				verdict = "fits"
			}
			tail = fmt.Sprintf(" (deadline %s: %s)", r.Forecast.Deadline, verdict)
		}
		p.Line("  forecast: %g agent h on the longest chain, finish %s%s", r.Forecast.Hours, r.Forecast.ETA, tail)
	}
	for _, s := range r.Signals {
		p.Line("  %s %s: %s", s.Rule, s.Artifact, s.Detail)
	}
}
