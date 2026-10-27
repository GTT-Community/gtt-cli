package cli

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/output"
)

// changeCmd is `gtt change`: write a change request in the middle of the
// work. The agent is told to process it (the session's next action); the
// CLI offers to open the Primary ADE so it happens right away.
func changeCmd(e *env) *cobra.Command {
	var o app.ChangeOptions
	var launch bool
	cmd := &cobra.Command{
		Use:   "change [what needs to change]",
		Short: "Request a governed change; the agent is told to process it into a proposal",
		Long: "Write gtt-domain/change-request.md through the Bootstrap - for a change to the design, the architecture,\n" +
			"the stack, a constraint or an Epic's goal or scope. Ordinary work needs no request. The request gets an id\n" +
			"(CR-...); until a proposal cites it, the session's Next action tells every ADE to process it on its first\n" +
			"turn. The agent drafts; you decide (gtt proposals, gtt promote).",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				o.Change = args[0]
			}
			res, err := e.app.ChangeRequest(cmd.Context(), o)
			if err := e.emit(res, err, func(p output.Printer) { renderChange(p, res) }); err != nil {
				return err
			}
			if e.opts.JSON || res.Primary == "" {
				return nil
			}
			if !launch {
				if !e.app.Prompt.Interactive() || e.opts.Yes {
					return nil
				}
				ok, err := e.app.Prompt.Confirm("Open the Primary ADE now so it processes the request?", true)
				if err != nil || !ok {
					return nil
				}
			}
			return e.app.LaunchPrimary(cmd.Context())
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.Reason, "reason", "", "why the change is needed")
	f.StringVar(&o.Trigger, "trigger", "", "what caused the request (bug, cost, requirement, review...)")
	f.StringVar(&o.Scope, "scope", "", "what is included and what is out of scope")
	f.StringVar(&o.Impact, "impact", "", "systems, modules, teams affected")
	f.StringVar(&o.Risk, "risk", "", "technical, operational, delivery risk")
	f.StringVar(&o.Priority, "priority", "", "critical|high|medium|low")
	f.StringVar(&o.By, "by", "", "who requests it")
	f.BoolVar(&o.Replace, "replace", false, "replace a request that is still pending")
	f.BoolVar(&launch, "launch", false, "open the Primary ADE afterwards without asking")
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the change request and whether a proposal answers it",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.ChangeShow(cmd.Context())
			return e.emit(res, err, func(p output.Printer) { renderChange(p, res) })
		},
	})
	return cmd
}

func renderChange(p output.Printer, r app.ChangeState) {
	if r.State == "" {
		return
	}
	p.Line("Change request: %s %s", r.State, r.ID)
	keys := make([]string, 0, len(r.Fields))
	for k := range r.Fields {
		keys = append(keys, k)
	}
	order := map[string]int{"Id": 0, "Requested": 1, "Change": 2, "Reason": 3, "Trigger": 4, "Scope": 5, "Impact": 6, "Risk": 7, "Priority": 8}
	sort.Slice(keys, func(i, j int) bool { return order[keys[i]] < order[keys[j]] })
	for _, k := range keys {
		if k != "Id" && r.Fields[k] != "" {
			p.Line("  %-9s %s", k+":", r.Fields[k])
		}
	}
	switch r.State {
	case "pending":
		p.Line("\nThe agent is told to process it at the start of its next session; it drafts a proposal and decides nothing.")
		p.Line("Follow it with: gtt proposals")
	case "processed":
		p.Line("\nAnswered by:\n  %s", strings.Join(r.Proposals, "\n  "))
	}
}

// proposalsCmd is `gtt proposals`.
func proposalsCmd(e *env) *cobra.Command {
	list := func(cmd *cobra.Command, _ []string) error {
		res, err := e.app.Proposals(cmd.Context())
		return e.emit(res, err, func(p output.Printer) { renderProposals(p, res) })
	}
	cmd := &cobra.Command{Use: "proposals", Short: "List proposals, ADR drafts and promotion sets, with their state", Args: cobra.NoArgs, RunE: list}
	var by string
	withdraw := &cobra.Command{
		Use:   "withdraw <name>",
		Short: "Withdraw a proposal or promotion set nobody will promote (archived, not deleted)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.ProposalWithdraw(cmd.Context(), args[0], by)
			return e.emit(res, err, func(p output.Printer) { p.Line("%s", res.Output) })
		},
	}
	withdraw.Flags().StringVar(&by, "by", "", "who withdraws it")
	cmd.AddCommand(&cobra.Command{Use: "list", Short: "List them", Args: cobra.NoArgs, RunE: list},
		&cobra.Command{
			Use:   "show <name>",
			Short: "Show one proposal, or the files of a promotion set",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				res, err := e.app.ProposalShow(cmd.Context(), args[0])
				return e.emit(res, err, func(p output.Printer) { p.Line("%s", strings.TrimRight(res.Content, "\n")) })
			},
		}, withdraw)
	return cmd
}

func renderProposals(p output.Printer, r app.ProposalsResult) {
	if r.Request.State == "pending" || r.Request.State == "processed" {
		p.Line("Change request %s: %s\n", r.Request.ID, r.Request.State)
	}
	if len(r.Proposals) == 0 {
		p.Line("No proposals.")
		return
	}
	for _, x := range r.Proposals {
		line := x.State + "  " + x.Kind + "  " + x.Name
		if x.Request != "" {
			line += "  (" + x.Request + ")"
		}
		if x.Title != "" {
			line += " - " + x.Title
		}
		p.Line("%s", line)
	}
	for _, x := range r.Proposals {
		if x.Kind == "promotion-set" && x.State == "ready to promote" {
			p.Line("\nPromote with: gtt promote <set>   (review first: gtt proposals show <set>)")
			break
		}
	}
}

// promoteCmd is `gtt promote <set>`: the human applies a staged promotion set.
func promoteCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "promote <set>",
		Short: "Apply a staged promotion set after reviewing its diff (a human act)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.Promote(cmd.Context(), args[0])
			return e.emit(res, err, func(p output.Printer) { p.Line("%s\nStep back with: gtt undo", res.Output) })
		},
	}
}

// undoCmd is `gtt undo`: one step back, never an unfreeze.
func undoCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "undo [set]",
		Short: "Undo the last promotion (a frozen one is re-frozen over the earlier state; never an unfreeze)",
		Long: "Put back what the last promotion replaced, while its files are as it left them. If that promotion\n" +
			"recorded a freeze, a new freeze records the earlier state; every baseline stays in the history.\n" +
			"Going further back is a change request (gtt change).",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.Undo(cmd.Context(), first(args))
			return e.emit(res, err, func(p output.Printer) { p.Line("%s", res.Output) })
		},
	}
}
