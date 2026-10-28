package cli

import (
	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/output"
)

// designCmd restarts or feeds the design before the first freeze (Canon 3.0 §19).
func designCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{Use: "design", Short: "Start the design over, or add design documents, before the first freeze"}
	var resolve app.ResolveOptions
	reset := &cobra.Command{
		Use:   "reset",
		Short: "Return the design to the Bootstrap's templates (with a backup); refused after a freeze",
		Long: "Returns the governed context, Epic designs, ADRs and backlog to the Bootstrap's templates and removes the\n" +
			"SOURCE-BRIEF and the questionnaire or assessment working copies. Setup choices and recorded sources stay.\n" +
			"Everything changed is copied to .gtt/local/design-reset/ first. After a freeze there is no reset: write a\n" +
			"change request instead.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := e.app.DesignReset(cmd.Context(), resolve)
			return e.emit(res, err, func(p output.Printer) {
				if err != nil || res.Backup == "" {
					return
				}
				p.Line("✓ Design reset: %d file(s) back to the templates, %d removed.", len(res.Restored), len(res.Removed))
				p.Section("Backup", res.Backup)
				p.Section("Next", "Add your design documents (gtt design add <path>...), or open the project in the Primary ADE\n"+
					"and write anything: it starts the design from your documents or the guided questionnaire.")
			})
		},
	}
	addResolveFlags(reset, &resolve)
	add := &cobra.Command{
		Use:   "add <path[:type]>...",
		Short: "Add one or more design documents as sources; the Primary ADE then assesses them",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := e.app.SourcesApply(cmd.Context(), args)
			return e.emit(res, err, func(p output.Printer) {
				for _, s := range res.Sources {
					renderSource(p, s)
				}
				if err == nil {
					p.Section("Next", "Open the project in the Primary ADE and write anything: it assesses the documents\n"+
						"and, with several, asks you whether to consolidate them or keep them as declared sources.")
				}
			})
		},
	}
	cmd.AddCommand(reset, add)
	return cmd
}
