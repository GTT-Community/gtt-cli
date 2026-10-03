package cli

import (
	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/output"
)

// releaseCmd holds the publisher's tools for signing a Bootstrap release.
func releaseCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{Use: "release", Short: "Sign GTT Bootstrap releases (for whoever publishes them)"}
	var dir, id string
	keygen := &cobra.Command{
		Use:   "keygen",
		Short: "Create an ed25519 release key pair (<id>.key, <id>.pub)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if id == "" {
				return usage("--id is required")
			}
			res, err := e.app.ReleaseKeygen(dir, id)
			return e.emit(res, err, func(p output.Printer) {
				p.Line("✓ Release key created.\n  private: %s (keep it secret)\n  public:  %s", res.Private, res.Public)
				p.Line("→ Users trust it by copying %s.pub into ~/.gtt/trusted-keys/", res.ID)
			})
		},
	}
	keygen.Flags().StringVar(&dir, "out", ".", "directory for the key files")
	keygen.Flags().StringVar(&id, "id", "", "key id (letters, digits, '.', '_', '-')")
	var key string
	sign := &cobra.Command{
		Use:   "sign <bootstrap-dir>",
		Short: "Write the release signature into a Bootstrap package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if key == "" {
				return usage("--key is required")
			}
			res, err := e.app.ReleaseSign(cmd.Context(), args[0], key)
			return e.emit(res, err, func(p output.Printer) {
				p.Line("✓ Bootstrap %s signed with key %s.\n  %s", res.Version, res.KeyID, res.Path)
			})
		},
	}
	sign.Flags().StringVar(&key, "key", "", "private key file written by: gtt release keygen")
	cmd.AddCommand(keygen, sign)
	return cmd
}
