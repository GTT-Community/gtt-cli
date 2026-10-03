package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/agents"
	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/output"
)

func names(m map[string]string, ids []string) string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id
		if n := m[id]; n != "" {
			out[i] = n
		}
	}
	return output.List(out)
}

func upper(s string) string { return strings.ToUpper(s) }

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func statusCmd(e *env) *cobra.Command {
	var fast bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show deterministic project state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.Status(cmd.Context(), !fast)
			return e.emit(res, err, func(p output.Printer) { renderStatus(p, res, e.opts.Verbose) })
		},
	}
	cmd.Flags().BoolVar(&fast, "fast", false, "skip running the Bootstrap validation")
	return cmd
}

func renderStatus(p output.Printer, r app.StatusResult, verbose bool) {
	v := r.View
	p.Line("GTT Project Status")
	p.Section("Project", r.Project)
	p.Section("CLI", r.CLI)
	p.Section("Bootstrap", r.Bootstrap.Version, fmt.Sprintf("Schema: %d", r.Bootstrap.Schema), "Compatibility: "+r.Bootstrap.Compatibility)
	p.Section("ADE", "Primary: "+names(r.ADENames, nonEmpty(v.ADE.Primary)), "Participating: "+names(r.ADENames, v.ADE.Participating),
		"Excluded: "+names(r.ADENames, v.ADE.Excluded))
	plan := title(v.Methodology.Profile)
	if v.Methodology.Source != "project" {
		plan = "not selected (run: gtt method set <plan>)"
	}
	p.Section("Methodology", plan)
	applied, pending := 0, 0
	for _, s := range r.Sources.Sources {
		if s.Outcome == "unchanged" {
			applied++
		} else {
			pending++
		}
	}
	src := []string{fmt.Sprintf("%d initial source(s) selected", v.Sources.Selected), fmt.Sprintf("Applied and validated: %d", applied)}
	if pending > 0 {
		src = append(src, fmt.Sprintf("Pending: %d (see: gtt sources)", pending))
	}
	p.Section("Sources", src...)
	freeze := "NOT FROZEN"
	if v.Freeze.Frozen {
		freeze = "ACTIVE"
	}
	p.Section("Governance", "Freeze: "+freeze, fmt.Sprintf("OPEN: %d", v.Governance.Gaps.Open), fmt.Sprintf("BLOCKING: %d", v.Governance.Gaps.Blocking),
		fmt.Sprintf("Conflicts: %d", v.Governance.UnresolvedConflicts), fmt.Sprintf("Proposals: %d", len(v.Governance.PendingProposals)),
		"Change Request: "+upper(v.Governance.ChangeRequest))
	val := "NOT RUN"
	if v.Validation.Result != "" {
		val = upper(v.Validation.Result)
	}
	p.Section("Validation", val)
	sync := "OK"
	if !r.Agents.OK() {
		sync = "INCONSISTENT (run: gtt agents sync)"
	}
	lines := []string{"Context: " + sync}
	if len(r.Agents.Undeclared) > 0 {
		lines = append(lines, "Detected, not participating: "+names(r.ADENames, r.Agents.Undeclared))
	}
	p.Section("Agents", lines...)
	session := "NONE"
	if v.Session.Exists {
		session = "CURRENT"
	}
	p.Section("Session", session)
	if verbose {
		p.Line("\nBootstrap status document:\n%s", string(r.Status))
	}
}

func nonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

func inspectCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect",
		Short: "Show topology and installation state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.Inspect(cmd.Context())
			return e.emit(res, err, func(p output.Printer) {
				p.Line("GTT Inspection")
				p.Section("Project root", res.ProjectRoot)
				p.Section("Git root", orNone(res.GitRoot))
				b := res.Bootstrap
				p.Section("Bootstrap", b.ID+" "+b.Version, fmt.Sprintf("Schema: %d", b.Schema), "Channel: "+b.Channel,
					fmt.Sprintf("Scaffold layout: %d", b.ScaffoldLayout), "Canon: "+b.Canon, "CLI compatibility: "+b.Compatibility)
				if r := res.Resolution; r != nil {
					p.Section("Resolved from", r.Source+": "+r.Origin, "Checksum: "+r.Checksum, "Integrity: "+r.Integrity, "Signature: "+r.Signature, "At: "+r.ResolvedAt)
				}
				var lines []string
				for _, ag := range res.Agents.Agents {
					lines = append(lines, agentLine(ag))
				}
				p.Section("ADEs", lines...)
				plan := res.Plan.Detail.Label
				if !res.Plan.Selected {
					plan = "not selected"
				}
				p.Section("Methodology profile", plan, "Language: "+orNone(res.Plan.Language))
				p.Section("GTT scaffold", res.Scaffold...)
				caps := make([]string, len(res.Capabilities))
				for i, c := range res.Capabilities {
					caps[i] = fmt.Sprintf("%s v%d", c.ID, c.Version)
				}
				p.Section("Bootstrap capabilities", caps...)
				p.Section("Bootstrap operations", fmt.Sprintf("%d declared", len(res.Operations)))
				p.Section("Freeze state", map[bool]string{true: "frozen", false: "not frozen"}[res.Frozen])
				p.Section("Session state", map[bool]string{true: "present", false: "none"}[res.Session])
				if e.opts.Verbose {
					p.Line("\nSources:\n%s\nGTT-owned paths:\n%s\nTemplates:\n%s", res.Sources, res.OwnedByGTT, res.Templates)
				}
			})
		},
	}
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func agentLine(a agents.Agent) string {
	role := "not participating"
	switch {
	case a.Primary:
		role = "primary"
	case a.Enabled:
		role = "participating"
	case a.Excluded:
		role = "excluded"
	}
	return fmt.Sprintf("%-16s %-18s detected: %-3s context: %s", a.Name, role, output.YesNo(a.Detected), a.Context)
}

func validateCmd(e *env) *cobra.Command {
	var ci bool
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Run the Bootstrap validation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.Validate(cmd.Context())
			return e.emit(res, err, func(p output.Printer) {
				if res.Result == "" {
					return
				}
				for _, c := range res.Report.Checks {
					if e.opts.Verbose || c.Result != "pass" {
						p.Line("[%s] %s", upper(c.Result), c.Check)
					}
				}
				for _, m := range res.Report.Messages {
					p.Line("  %s", m)
				}
				p.Line("GTT validation: %s (%d checks, Bootstrap %s)", upper(res.Result), len(res.Report.Checks), res.Bootstrap)
			})
		},
	}
	cmd.Flags().BoolVar(&ci, "ci", false, "CI mode: non-interactive, stable exit codes")
	return cmd
}

func resumeCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "resume",
		Short: "Reconstruct the working context from actual project state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.Resume(cmd.Context())
			return e.emit(res, err, func(p output.Printer) {
				if res.Text == "" && res.Context == nil {
					return
				}
				p.Line("GTT Session Context (derived from project state; operational only)\n")
				if res.Interrupted != "" {
					p.Line("Incomplete initialization: %s (gtt init --resume | --rollback)\n", res.Interrupted)
				}
				if res.Text != "" {
					p.Line("%s", strings.TrimRight(res.Text, "\n"))
				} else {
					p.Line("%s", string(res.Context))
				}
				if h := res.Handoff; h != nil && h.PrimaryName != "" {
					p.Line("\nNext:\nContinue in %s (instruction entry: %s).", h.PrimaryName, h.InstructionEntry)
				}
			})
		},
	}
}

func freezeCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "freeze",
		Short: "Ratify the governed context through the Bootstrap (a human act)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			text, err := e.app.Freeze(cmd.Context())
			return e.emit(map[string]any{"schema": 1, "frozen": err == nil, "bootstrap_output": text}, err, func(p output.Printer) {
				if err == nil {
					p.Line("%s", text)
				}
			})
		},
	}
}

func renderChecks(p output.Printer, r app.DoctorResult) {
	for _, c := range r.Checks {
		line := fmt.Sprintf("[%s] %s", c.Level, c.Name)
		if c.Detail != "" && c.Level != app.Pass {
			line += ": " + c.Detail
		}
		p.Line("%s", line)
	}
	p.Line("\nResult:")
	for _, level := range []string{app.Pass, app.Warn, app.Fail, app.CannotDetermine} {
		if n := r.Totals[level]; n > 0 || level != app.CannotDetermine {
			p.Line("%d %s", n, level)
		}
	}
}

func doctorCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose the installation (reports, never repairs)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.Doctor(cmd.Context())
			return e.emit(res, err, func(p output.Printer) { renderChecks(p, res) })
		},
	}
}

func auditCmd(e *env) *cobra.Command {
	var md bool
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Operational traceability report",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.Audit(cmd.Context())
			return e.emit(res, err, func(p output.Printer) { renderAudit(p, res, md) })
		},
	}
	cmd.Flags().BoolVar(&md, "md", false, "Markdown output")
	return cmd
}

func renderAudit(p output.Printer, r app.AuditResult, md bool) {
	if r.Project == "" {
		return
	}
	head := func(s string) {
		if md {
			p.Line("\n## %s\n", s)
		} else {
			p.Line("\n%s:", s)
		}
	}
	item := func(k, v string) {
		if md {
			p.Line("- **%s:** %s", k, v)
		} else {
			p.Line("  %s: %s", k, v)
		}
	}
	if md {
		p.Line("# GTT Audit\n\n- **Generated:** %s\n- **Project:** %s", r.Generated, r.Project)
	} else {
		p.Line("GTT Audit\n\n  Generated: %s\n  Project: %s", r.Generated, r.Project)
	}
	v := r.View
	head("Releases")
	item("CLI", r.CLI)
	item("Bootstrap", fmt.Sprintf("%s %s (schema %d, %s)", r.Bootstrap.ID, r.Bootstrap.Version, r.Bootstrap.Schema, r.Bootstrap.Channel))
	item("Compatibility", r.Bootstrap.Compatibility)
	if res := r.Resolution; res != nil {
		item("Resolved from", res.Source+" "+res.Origin)
		item("Checksum", res.Checksum)
		item("Integrity", res.Integrity)
	}
	head("ADE")
	item("Primary", names(r.ADENames, nonEmpty(v.ADE.Primary)))
	item("Participating", names(r.ADENames, v.ADE.Participating))
	item("Excluded", names(r.ADENames, v.ADE.Excluded))
	head("Methodology")
	item("Profile", orNone(v.Methodology.Profile)+" (source: "+v.Methodology.Source+")")
	item("Language", orNone(v.Methodology.Language))
	head("Initial sources")
	for _, s := range r.Tracking.Sources {
		item(s.Path, s.State+" / "+s.Outcome)
	}
	if len(r.Tracking.Sources) == 0 {
		item("Selected", "none")
	}
	head("Governance state (as reported by Bootstrap)")
	item("Validation", upper(r.Validation.Result))
	item("Frozen", output.YesNo(v.Freeze.Frozen))
	item("Proposals", output.List(v.Governance.PendingProposals))
	item("Change request", v.Governance.ChangeRequest)
	item("OPEN / BLOCKING", fmt.Sprintf("%d / %d", v.Governance.Gaps.Open, v.Governance.Gaps.Blocking))
	head("Recovery snapshots")
	for _, s := range r.Snapshots {
		item(s.Meta.CreatedAt, s.Snapshot+" ("+s.Meta.Reason+")")
	}
	if len(r.Snapshots) == 0 {
		item("Stored", "none")
	}
	head("CLI history (init, updates, clean exports, snapshots)")
	for _, h := range r.History {
		item(h.At, h.Event+" "+h.Detail)
	}
	if len(r.History) == 0 {
		item("Events", "none")
	}
}
