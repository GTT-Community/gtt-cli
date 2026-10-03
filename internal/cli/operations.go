package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/agents"
	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/output"
)

func updateCmd(e *env) *cobra.Command {
	var o app.UpdateOptions
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the installed Bootstrap transactionally",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.Update(cmd.Context(), o)
			return e.emit(res, err, func(p output.Printer) {
				switch res.Outcome {
				case "already-current":
					p.Line("Bootstrap %s is already installed. Nothing to do.", res.To)
				case "rolled-back":
					p.Line("The interrupted update was rolled back.")
				case "updated":
					p.Line("PASS\n\nBootstrap updated successfully: %s -> %s", res.From, res.To)
					p.Line("  replaced %d, added %d, removed %d, preserved (changed locally) %d", len(res.Replaced), len(res.Added), len(res.Removed), len(res.Preserved))
					if e.opts.Verbose {
						p.Line("\nPreserved:\n  %s", strings.Join(res.Preserved, "\n  "))
					}
				}
			})
		},
	}
	addResolveFlags(cmd, &o.Resolve)
	cmd.Flags().StringVar(&o.Base, "base", "", "catalog of the installed Bootstrap version (projects not installed by the gtt CLI)")
	cmd.Flags().BoolVar(&o.Rollback, "rollback", false, "roll back an interrupted update")
	return cmd
}

func exportCmd(e *env) *cobra.Command {
	var clean bool
	cmd := &cobra.Command{
		Use:   "export --clean [destination]",
		Short: "Create a clean delivery artifact; the project is not modified",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !clean {
				return usage("gtt export requires --clean (the only export defined in v1.0)")
			}
			dest := ""
			if len(args) == 1 {
				dest = args[0]
			}
			res, err := e.app.ExportClean(cmd.Context(), dest)
			return e.emit(res, err, func(p output.Printer) {
				if err != nil {
					return
				}
				p.Line("✓ Clean export created: %s", res.Destination)
				p.Line("  %d file(s) copied, %d GTT file(s) excluded. The project was not modified.", res.Copied, res.Excluded)
				if len(res.KeptForReview) > 0 {
					p.Line("⚠ Not excluded (changed or adopted; review them):\n  %s", strings.Join(res.KeptForReview, "\n  "))
				}
			})
		},
	}
	cmd.Flags().BoolVar(&clean, "clean", false, "exclude everything GTT owns, per the Bootstrap export policy")
	return cmd
}

func cleanCmd(e *env) *cobra.Command {
	var noSnapshot bool
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove GTT from this project (destructive; asks for confirmation)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.Clean(cmd.Context(), noSnapshot)
			return e.emit(res, err, func(p output.Printer) {
				if err != nil {
					return
				}
				p.Line("✓ GTT removed from this project (%d path(s)).", len(res.Removed))
				if len(res.Kept) > 0 {
					p.Line("⚠ Kept (changed since GTT installed them):\n  %s", strings.Join(res.Kept, "\n  "))
				}
				if res.Snapshot != "" {
					p.Line("  Recovery snapshot: %s\n→ `gtt init` in this project offers to restore it.", res.Snapshot)
				}
			})
		},
	}
	cmd.Flags().BoolVar(&noSnapshot, "no-snapshot", false, "do not create a recovery snapshot first")
	return cmd
}

func versionCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the CLI version and the installed Bootstrap",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res := e.app.Version(cmd.Context())
			return e.emit(res, nil, func(p output.Printer) {
				p.Line("GTT CLI       %s", res.CLI)
				if b := res.Bootstrap; b != nil {
					p.Line("Bootstrap     %s\nSchema        %d\nCompatibility %s", b.Version, b.Schema, b.Compatibility)
				}
			})
		},
	}
}

func configCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Operational configuration (validated by the Bootstrap)"}
	cmd.AddCommand(&cobra.Command{
		Use:   "primary-ade [id]",
		Short: "Change the Primary ADE explicitly",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := e.app.ConfigPrimary(cmd.Context(), first(args))
			return e.emit(map[string]any{"schema": 1, "primary_ade": name}, err, func(p output.Printer) {
				if err == nil {
					p.Line("✓ Primary ADE: %s (secondary ADEs unchanged)", name)
				}
			})
		},
	}, &cobra.Command{
		Use:   "methodology [plan]",
		Short: "Change the Method Plan (same as: gtt method set)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.MethodSet(cmd.Context(), first(args))
			return e.emit(res, err, func(p output.Printer) { renderMethodSet(p, res, err) })
		},
	}, &cobra.Command{
		Use:   "language <lang>",
		Short: "Change the interaction language",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := e.app.ConfigLanguage(cmd.Context(), args[0])
			return e.emit(map[string]any{"schema": 1, "language": args[0]}, err, func(p output.Printer) {
				if err == nil {
					p.Line("✓ Language: %s", args[0])
				}
			})
		},
	})
	return cmd
}

func first(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

func renderMethodSet(p output.Printer, r app.MethodResult, err error) {
	if err == nil {
		p.Line("✓ Method Plan: %s", r.Plan.Detail.Label)
	}
}

func methodCmd(e *env) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		res, err := e.app.MethodShow(cmd.Context())
		return e.emit(res, err, func(p output.Printer) {
			if err != nil {
				return
			}
			plan := res.Plan
			p.Line("GTT Method Plan")
			if !plan.Selected {
				p.Section("Plan", "not selected", "Choose one: gtt method set <"+choiceIDs(res)+">")
				return
			}
			pol := plan.Detail.Policy
			p.Section("Plan", plan.Detail.Label, plan.Detail.Summary)
			p.Section("Automation", title(pol.Automation["level"]))
			p.Section("Without asking", output.List(plan.Interaction.WithoutAsking))
			asks := make([]string, 0, len(plan.Interaction.AsksFor))
			for k := range plan.Interaction.AsksFor {
				asks = append(asks, k)
			}
			p.Section("Asks the human for", output.List(sorted(asks)))
			p.Section("Governance", "Full (no plan weakens an invariant)")
			p.Section("Agent synchronization", strings.ReplaceAll(plan.Value("agent_context_sync"), "_", " "))
			if len(plan.Team) > 0 {
				declared := make([]string, 0, len(plan.Team))
				for k, v := range plan.Team {
					declared = append(declared, k+": "+strings.ReplaceAll(v, "_", " "))
				}
				p.Section("Declared by the team (working agreements)", sorted(declared)...)
			}
			p.Section("Collaboration", "CI: "+pol.Collaboration["ci"], "Multi-user: "+pol.Collaboration["multi_user"], "Traceability: "+pol.Collaboration["traceability"])
			if e.opts.Verbose {
				p.Section("Delegates", plan.Detail.Delegates)
			}
		})
	}
	cmd := &cobra.Command{Use: "method", Short: "Method Plan: show, set, check", Args: cobra.NoArgs, RunE: show}
	cmd.AddCommand(&cobra.Command{Use: "show", Short: "Show the Method Plan in force", Args: cobra.NoArgs, RunE: show},
		&cobra.Command{
			Use:   "set [plan]",
			Short: "Select or change the Method Plan",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				res, err := e.app.MethodSet(cmd.Context(), first(args))
				return e.emit(res, err, func(p output.Printer) { renderMethodSet(p, res, err) })
			},
		},
		&cobra.Command{
			Use:   "check",
			Short: "Verify that the plan configuration is coherent",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				res, err := e.app.MethodCheck(cmd.Context())
				return e.emit(res, err, func(p output.Printer) { renderChecks(p, res) })
			},
		})
	return cmd
}

func choiceIDs(r app.MethodResult) string {
	ids := make([]string, len(r.Choices))
	for i, c := range r.Choices {
		ids[i] = c.ID
	}
	return strings.Join(ids, "|")
}

func renderAgents(p output.Printer, env agents.Environment) {
	p.Line("GTT Agent Environment\n\nProject agents\n────────────────────────────────")
	for _, a := range env.Agents {
		p.Line("\n%s", a.Name)
		enabled := output.YesNo(a.Enabled)
		if a.Primary {
			enabled += " (primary)"
		}
		p.Line("  enabled: %s\n  detected: %s\n  context: %s", enabled, output.YesNo(a.Detected), a.Context)
		if a.Enabled {
			p.Line("  files: %d\n  changed: %d\n  missing: %d", len(a.Artifacts), a.Count(agents.ArtifactChanged), a.Count(agents.ArtifactMissing))
		}
	}
}

// renderManifest lists what each agent's adapter manages in this project.
func renderManifest(p output.Printer, env agents.Environment) {
	p.Line("GTT Agent Context Manifest")
	for _, a := range env.Agents {
		if !a.Enabled {
			continue
		}
		p.Line("\n%s (adapter version %d)\n  instruction entry: %s\n  context locations: %s", a.Name, a.AdapterVersion, a.Entry, output.List(a.Locations))
		for _, x := range a.Artifacts {
			p.Line("  %-13s %s", x.Status, x.Path)
		}
		if len(a.Artifacts) == 0 {
			p.Line("  no managed files")
		}
	}
}

func agentsCmd(e *env) *cobra.Command {
	list := func(cmd *cobra.Command, _ []string) error {
		res, err := e.app.AgentsCheck(cmd.Context())
		return e.emit(res, err, func(p output.Printer) {
			if len(res.Agents) > 0 {
				renderAgents(p, res)
			}
		})
	}
	var ci bool
	var add string
	var resolve app.ResolveOptions
	cmd := &cobra.Command{Use: "agents", Short: "ADE/agent context discovery and synchronisation", Args: cobra.NoArgs, RunE: list}
	check := &cobra.Command{
		Use:   "check",
		Short: "Compare declared and detected agents (read-only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.AgentsCheck(cmd.Context())
			return e.emit(res, err, func(p output.Printer) {
				if len(res.Agents) == 0 {
					return
				}
				var declared, detected []string
				p.Line("Project configuration: %s\n", map[bool]string{true: "OK", false: "NOT CONFIGURED"}[res.Configured])
				for _, a := range res.Agents {
					if a.Enabled {
						declared = append(declared, a.Name)
					}
					if a.Detected {
						detected = append(detected, a.Name)
					}
				}
				p.Line("Declared agents:\n  %s\n\nDetected:\n  %s\n\nSynchronization:", output.List(declared), output.List(detected))
				for _, a := range res.Agents {
					if a.Enabled {
						p.Line("  %-16s %s", a.Name, map[bool]string{true: "OK", false: "INCONSISTENT - " + a.Detail}[a.Context == agents.Synchronized])
					}
				}
				if len(res.Undeclared) > 0 {
					p.Line("\nUndeclared detected agent:\n  %s", strings.Join(res.Undeclared, "\n  "))
				}
				p.Line("\nNo changes were made.")
			})
		},
	}
	check.Flags().BoolVar(&ci, "ci", false, "CI mode: fail only on a governed inconsistency")
	sync := &cobra.Command{
		Use:   "sync",
		Short: "Synchronise the managed context of declared agents",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.AgentsSync(cmd.Context(), split(add), resolve)
			return e.emit(res, err, func(p output.Printer) {
				if err != nil {
					return
				}
				if !res.Changed {
					p.Line("✓ Agent context is synchronized. Nothing to do.")
				} else {
					p.Line("✓ Synchronization completed: %s (source: %s)", output.List(res.Synced), res.Source)
				}
				if un := res.Environment.Undeclared; len(un) > 0 {
					p.Line("⚠ Detected but not declared: %s. Not modified. To incorporate: gtt agents sync --add <id>", strings.Join(un, ", "))
				}
			})
		},
	}
	sync.Flags().StringVar(&add, "add", "", "incorporate these ADEs as participating (asks for confirmation)")
	addResolveFlags(sync, &resolve)
	manifest := &cobra.Command{
		Use:   "manifest",
		Short: "List the context each declared agent's adapter manages (read-only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.AgentsCheck(cmd.Context())
			return e.emit(res, err, func(p output.Printer) {
				if len(res.Agents) > 0 {
					renderManifest(p, res)
				}
			})
		},
	}
	cmd.AddCommand(&cobra.Command{Use: "list", Short: "List project agents", Args: cobra.NoArgs, RunE: list}, check, sync, manifest)
	return cmd
}

func renderSource(p output.Printer, s app.SourceReport) {
	switch s.Outcome {
	case "applied":
		p.Line("✓ Directiva aplicada y validada: %s", s.Path)
	case "unchanged":
		p.Line("✓ Sin cambios (ya aplicada y validada): %s", s.Path)
	case "failed":
		p.Line("✗ Directiva no aplicada: %s (%s)", s.Path, s.Reason)
	default:
		p.Line("⚠ Pendiente: %s (%s)", s.Path, s.Reason)
	}
}

func sourcesCmd(e *env) *cobra.Command {
	render := func(res app.SourcesResult) func(output.Printer) {
		return func(p output.Printer) {
			for _, s := range res.Sources {
				renderSource(p, s)
			}
			if len(res.Sources) == 0 {
				p.Line("No initial sources.")
			}
		}
	}
	list := func(cmd *cobra.Command, _ []string) error {
		res, err := e.app.SourcesList(cmd.Context())
		return e.emit(res, err, render(res))
	}
	cmd := &cobra.Command{Use: "sources", Short: "Initial MD tracking: what was read, applied and validated", Args: cobra.NoArgs, RunE: list}
	cmd.AddCommand(&cobra.Command{Use: "list", Short: "Show the tracking state (read-only)", Args: cobra.NoArgs, RunE: list},
		&cobra.Command{
			Use:   "apply [path[:type]...]",
			Short: "Read, apply and validate initial sources; named paths are thereby selected",
			RunE: func(cmd *cobra.Command, args []string) error {
				res, err := e.app.SourcesApply(cmd.Context(), args)
				return e.emit(res, err, render(res))
			},
		})
	return cmd
}

func syncCmd(e *env) *cobra.Command {
	var resolve app.ResolveOptions
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Deterministic upkeep: agent context, source tracking, index and validation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.Sync(cmd.Context(), resolve)
			return e.emit(res, err, func(p output.Printer) {
				if res.Agents.Changed {
					p.Line("✓ Agent context synchronized: %s", output.List(res.Agents.Synced))
				}
				for _, s := range res.Sources.Sources {
					if s.Outcome != "unchanged" {
						renderSource(p, s)
					}
				}
				if res.Maintain != "" {
					p.Line("%s", res.Maintain)
				}
			})
		},
	}
	addResolveFlags(cmd, &resolve)
	return cmd
}

func snapshotCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{Use: "snapshot", Short: "Recovery snapshots of GTT configuration"}
	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create a recovery snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.SnapshotCreate(cmd.Context())
			return e.emit(res, err, func(p output.Printer) {
				if err == nil {
					p.Line("✓ Recovery snapshot: %s", res.Snapshot)
				}
			})
		},
	}, &cobra.Command{
		Use:   "list",
		Short: "List the recovery snapshots of this project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res := e.app.SnapshotList(cmd.Context())
			return e.emit(map[string]any{"schema": 1, "snapshots": res}, nil, func(p output.Printer) {
				for _, s := range res {
					p.Line("%s  Bootstrap %s  Primary %s  Plan %s\n  %s", s.Meta.CreatedAt, s.BootstrapVersion, s.Primary, s.Profile, s.Snapshot)
				}
				if len(res) == 0 {
					p.Line("No recovery snapshots.")
				}
			})
		},
	})
	return cmd
}

func artifactCmd(e *env) *cobra.Command {
	var kind, requested string
	next := &cobra.Command{
		Use:   "next-id",
		Short: "Allocate the next free artifact id (resolved by the Bootstrap)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if kind == "" {
				return usage("--kind is required")
			}
			res, err := e.app.ArtifactNextID(cmd.Context(), kind, requested)
			return e.emit(res, err, func(p output.Printer) {
				if err != nil {
					return
				}
				if res.Automatic {
					p.Line("%s", res.ID)
				} else {
					p.Line("%s (proposed: needs your confirmation under the selected plan)", res.ID)
				}
			})
		},
	}
	next.Flags().StringVar(&kind, "kind", "", "artifact kind, as the Bootstrap declares it")
	next.Flags().StringVar(&requested, "requested", "", "the id you wanted")
	cmd := &cobra.Command{Use: "artifact", Short: "Artifact identity"}
	cmd.AddCommand(next)
	return cmd
}

func sorted(s []string) []string {
	out := append([]string{}, s...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

var _ = fmt.Sprintf
