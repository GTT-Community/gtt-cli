package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/output"
)

// runCmd is `gtt run`: simple execution. With no argument it lists what is
// pending; `gtt run <n>` shows the steps, asks once when the run changes
// something, executes it and archives it. Executing is the human's act.
func runCmd(e *env) *cobra.Command {
	list := func(cmd *cobra.Command) error {
		res, err := e.app.RunPending(cmd.Context())
		return e.emit(res, err, func(p output.Printer) { renderRuns(p, res) })
	}
	cmd := &cobra.Command{
		Use:   "run [n|RUN-NNN]",
		Short: "List what GTT hands you to run, or run one (a human act)",
		Long: "GTT gives you one line saying what it does and one command: `gtt run 3`. With no argument, list the pending\n" +
			"runs. With a number, show the steps, ask once if it changes something, run the sealed script the Bootstrap\n" +
			"generated, log it and archive it. A run that changes nothing needs no question. There is no --yes: the\n" +
			"confirmation is yours, and an agent is denied this command.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return list(cmd)
			}
			res, err := e.app.RunExecute(cmd.Context(), args[0])
			return e.emit(res, err, func(p output.Printer) {
				if res.Output != "" {
					p.Line("%s", res.Output)
				}
				p.Line("\n%s done and archived.", res.ID)
			})
		},
	}
	show := &cobra.Command{
		Use:   "show <n>",
		Short: "Show the steps of a pending run without running it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.RunShow(cmd.Context(), args[0])
			return e.emit(res, err, func(p output.Printer) { renderRun(p, res) })
		},
	}
	check := &cobra.Command{
		Use:   "check [n]",
		Short: "Review the pending runs (RUN-01 to RUN-09) without running them",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.RunReview(cmd.Context(), first(args))
			return e.emit(res, err, func(p output.Printer) {
				if len(res.Runs) == 0 {
					p.Line("No pending runs.")
				}
				for _, r := range res.Runs {
					state := "ok"
					if !r.OK {
						state = "FAILED"
					}
					p.Line("%s: %s", r.ID, state)
					for _, f := range r.Findings {
						p.Line("  %s %s (%s): %s", strings.ToUpper(f.Level), f.Rule, f.Where, f.Message)
					}
				}
			})
		},
	}
	cmd.AddCommand(show, check)
	return cmd
}

func renderRuns(p output.Printer, r app.RunList) {
	if len(r.Runs) == 0 {
		p.Line("No pending runs.")
		return
	}
	for _, x := range r.Runs {
		what := x.What
		if what == "" {
			what = x.Title
		}
		p.Line("gtt run %-3d %s  [changes: %s; %s]", x.Number, what, x.Changes, x.State)
	}
}

func renderRun(p output.Printer, x app.RunItem) {
	p.Line("%s - %s", x.ID, x.What)
	p.Line("Why: %s", x.Why)
	for _, c := range x.Checks {
		p.Line("  check  %s", c)
	}
	for i, s := range x.Steps {
		p.Line("  %d. %s   $ %s", i+1, s.Title, s.Command)
	}
	p.Line("Changes: %s   State: %s", x.Changes, x.State)
	for _, f := range x.Findings {
		p.Line("  %s %s: %s", strings.ToUpper(f.Level), f.Rule, f.Message)
	}
	if x.Next != "" {
		p.Line("Next: %s", x.Next)
	}
}
