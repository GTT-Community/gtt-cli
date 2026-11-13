package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/output"
)

// upgradeCmd updates the CLI itself. `gtt update` is the project's Bootstrap; this is the binary.
func upgradeCmd(e *env) *cobra.Command {
	var check bool
	var to string
	var auto string
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Update this gtt to the latest release (the CLI itself; for a project's Bootstrap use: gtt update)",
		Long: "Replaces this gtt with the latest release, or with --to <version>, the way it was installed\n" +
			"(npm, uv, pipx, or a verified download). With --check it only reports. It is the only command\n" +
			"that replaces the binary, and a human act: an agent does not run it. gtt also looks for a newer release by itself\n" +
			"(at most every 6 hours; one request for the public release page, nothing about you or your projects) and, in a\n" +
			"terminal, upgrades itself: --auto off stops that, GTT_NO_UPDATE_CHECK=1 stops the look as well.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if auto != "" {
				if auto != "on" && auto != "off" {
					return usage("--auto takes on or off, not %q.", auto)
				}
				if err := e.app.SetAutoUpgrade(auto == "on"); err != nil {
					return err
				}
				return e.out.Emit(map[string]any{"schema": 1, "auto": auto == "on"}, func(p output.Printer) {
					p.Line("gtt upgrades itself in a terminal: %s", auto)
				})
			}
			res, err := e.app.Upgrade(cmd.Context(), check, to)
			return e.emit(res, err, func(p output.Printer) {
				if err != nil {
					return // the error says what happened and what to run
				}
				switch res.Status {
				case "up-to-date":
					p.Line("✓ gtt %s is the release asked for. Nothing to do.", res.Installed)
				case "ahead":
					p.Line("gtt %s is newer than the latest published (%s). Nothing to do; --to <version> goes to another one.", res.Installed, res.Latest)
				case "available":
					p.Line("gtt %s -> %s available.", res.Installed, res.Target)
					if pl := res.Plan; pl != nil {
						if pl.Manual {
							p.Line("This gtt cannot upgrade itself here: %s.", pl.Reason)
							p.Line("Run: %s", pl.Hint)
						} else if len(pl.Command) > 0 {
							p.Line("Upgrade with: gtt upgrade   (runs: %s)", strings.Join(pl.Command, " "))
						} else {
							p.Line("Upgrade with: gtt upgrade   (verified download, replaces %s)", pl.Exe)
						}
					}
				case "upgraded":
					p.Line("✓ gtt %s -> %s. In each project: gtt update.", res.Installed, res.Target)
					if res.Plan != nil && len(res.Plan.Command) == 0 {
						p.Line("The previous binary stays as %s.old.", res.Plan.Exe)
					}
				}
			})
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "only compare the installed release with the latest; change nothing")
	cmd.Flags().StringVar(&auto, "auto", "", "on|off: whether gtt upgrades itself when it finds a newer release (default on, only in a terminal)")
	cmd.Flags().StringVar(&to, "to", "", "a specific release (the only way to go to an older one)")
	return cmd
}
