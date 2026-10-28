package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/output"
)

// devMethodCmd is `gtt dev-method`: the development method (Method
// Governance), read-only. It has no subcommand that changes the method: a
// ratified method changes only through a change request.
func devMethodCmd(e *env) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		res, err := e.app.DevMethodShow(cmd.Context())
		return e.emit(res, err, func(p output.Printer) { renderDevMethod(p, res) })
	}
	var lang string
	list := &cobra.Command{
		Use:   "list",
		Short: "List the development methods (Method Packs) with their short text",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.DevMethodList(cmd.Context(), lang)
			return e.emit(res, err, func(p output.Printer) { renderDevMethodList(p, res) })
		},
	}
	list.Flags().StringVar(&lang, "lang", "", "language of the texts (en|es); default: the project's")
	cmd := &cobra.Command{
		Use:   "dev-method",
		Short: "Show the development method in force (Vertical Slice, DDD...): read-only",
		Long: "Show the development method in force: its status (ratified / open), the ADR that locks it and its\n" +
			"verification level. It is chosen at gtt init and ratified by the freeze; afterwards it changes only\n" +
			"through a change request. Not to be confused with the Autonomy Profile (gtt autonomy).",
		Args: cobra.NoArgs,
		RunE: show,
	}
	cmd.AddCommand(&cobra.Command{Use: "show", Short: "Show the development method in force", Args: cobra.NoArgs, RunE: show}, list)
	return cmd
}

func joinOpen(ids []string) string {
	if len(ids) == 0 {
		return "OPEN"
	}
	return strings.Join(ids, ", ")
}

func renderDevMethod(p output.Printer, r app.DevMethodState) {
	f, l := r.InForce, r.Lock
	p.Line("Development method: %s", f.Status)
	p.Line("  style:        %s", joinOpen(f.Style))
	start := f.Start
	if start == "" {
		start = "OPEN"
	}
	p.Line("  start:        %s", start)
	if l.Present {
		p.Line("  locked by:    %s", l.LockedBy)
		p.Line("  verification: %s (gtt check verifies the code where an adapter exists)", l.Verification)
		if l.Gap != "" {
			p.Line("  gap:          %s", l.Gap)
		}
	}
	switch f.Source {
	case "setup":
		p.Line("  source:       the setup answer - it reaches the governed context as the Method Lock at the freeze")
	case "none":
		p.Line("  source:       none declared")
	}
	if r.Frozen {
		p.Line("\nThe method is ratified with the design: it changes only through a change request.")
	}
}

func renderDevMethodList(p output.Printer, c app.DevMethodCatalog) {
	recommended, fits := "(recommended)", "Fits when"
	if c.Lang == "es" {
		recommended, fits = "(recomendado)", "Conviene si"
	}
	for _, q := range c.Questions {
		p.Line("%d. %s", q.Order, q.Prompt)
		for _, o := range q.Options {
			mark := ""
			if o.Recommended {
				mark = " " + recommended
			}
			p.Line("   - %s%s [%s] - verification: %s", o.Name, mark, o.ID, o.Verification)
			p.Line("       %s", o.WhatItIs)
			p.Line("       %s: %s", fits, o.FitsWhen)
		}
		for _, x := range q.Extra {
			p.Line("   - %s [%s]", x.Label, x.ID)
		}
		p.Line("")
	}
}

// checkCmd is `gtt check`: Method Conformance of the code.
func checkCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Verify that the code respects the development method's dependency rules",
		Long: "Method Conformance. Reads the ratified Method Lock and its dependency rules, as stack.md section 5\n" +
			"writes them with the project's real paths, and verifies them with the language's own tool through an\n" +
			"adapter (Go: go list; Java/Kotlin: jdeps; PHP: deptrac; Python: import-linter; JavaScript/TypeScript:\n" +
			"dependency-cruiser). A forbidden dependency is reported as a conflict -\n" +
			"rule, source file, dependency - and never corrected.\n\n" +
			"Exit codes: 0 pass (or nothing to check, or declared-only: no adapter for the language),\n" +
			"1 conflict, 7 cannot be determined (the language's tool is not installed).",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.Check(cmd.Context())
			return e.emit(res, err, func(p output.Printer) { renderConformance(p, res) })
		},
	}
}

func renderConformance(p output.Printer, r app.CheckResult) {
	if r.Status == "" {
		return
	}
	p.Line("Method Conformance: %s", strings.ToUpper(r.Status))
	if len(r.Packs) > 0 {
		p.Line("  method:       %s", strings.Join(r.Packs, ", "))
	}
	if r.Adapter != "" {
		p.Line("  adapter:      %s (%s)", r.Adapter, r.Language)
	}
	p.Line("  verification: %s", r.Verification)
	if len(r.Checked) > 0 {
		p.Line("  rules:        %s", strings.Join(r.Checked, ", "))
	}
	if len(r.NotInMap) > 0 {
		p.Line("  not in stack.md section 5 (not verified): %s", strings.Join(r.NotInMap, ", "))
	}
	if len(r.AuditOnly) > 0 {
		p.Line("  left to audit (human judgment): %s", strings.Join(r.AuditOnly, ", "))
	}
	for _, c := range r.Conflicts {
		p.Line("  CONFLICT %s: %s -> %s", c.Rule, c.From, c.To)
	}
	if r.Note != "" {
		p.Line("%s", r.Note)
	}
}
