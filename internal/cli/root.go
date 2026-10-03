// Package cli is the Cobra command tree: it parses flags, calls one use
// case and renders its result. It holds no logic of its own.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/output"
)

// Options are the global flags the composition root needs to wire an App.
type Options struct {
	JSON    bool
	Verbose bool
	Yes     bool
	NoInput bool
	Dir     string
	// LogLevel is the least severe internal log level written to stderr.
	LogLevel string
}

// Wire builds the use cases and the renderer for one invocation.
type Wire func(Options) (*app.App, output.Renderer, error)

type env struct {
	opts Options
	wire Wire
	app  *app.App
	out  output.Renderer
}

func (e *env) setup() error {
	if !slices.Contains(output.LogLevels, e.opts.LogLevel) {
		return usage("unknown log level %q: use %s", e.opts.LogLevel, strings.Join(output.LogLevels, ", "))
	}
	a, r, err := e.wire(e.opts)
	e.app, e.out = a, r
	return err
}

// emit renders a result; when the use case also failed, the result is still
// shown (a failed validation carries its report) and the error is returned.
func (e *env) emit(v any, err error, human func(p output.Printer)) error {
	if err == nil || e.opts.JSON {
		if rerr := e.out.Emit(v, human); rerr != nil && err == nil {
			return rerr
		}
	} else if showOnError(err) {
		_ = e.out.Emit(v, human)
	}
	return err
}

func showOnError(err error) bool {
	c := core.CodeOf(err)
	return c == core.ExitFailure
}

// Execute runs the CLI and returns the process exit code.
func Execute(ctx context.Context, wire Wire, args []string, stdout, stderr io.Writer) int {
	e := &env{wire: wire}
	root := newRoot(e)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	err := root.ExecuteContext(ctx)
	if err == nil {
		return int(core.ExitOK)
	}
	var ce *core.Error
	if !errors.As(err, &ce) {
		// Anything Cobra reports is an invalid invocation.
		ce = &core.Error{Code: core.ExitUsage, What: err.Error(), Next: "Run: gtt --help"}
	}
	if e.opts.JSON {
		doc := map[string]any{"schema": core.OutputSchema, "error": map[string]any{
			"exit_code": ce.Code, "what": ce.What, "why": ce.Why, "next": ce.Next, "project_unmodified": ce.Unmodified}}
		data, _ := json.MarshalIndent(doc, "", "  ")
		fmt.Fprintln(stderr, string(data))
	} else {
		fmt.Fprintln(stderr, ce.Error())
	}
	return int(ce.Code)
}

func newRoot(e *env) *cobra.Command {
	root := &cobra.Command{
		Use:   "gtt",
		Short: "GTT CLI - the operational doorway into GTT Bootstrap",
		Long: "gtt resolves, installs, selects, invokes, validates, updates, exports and recovers GTT Bootstrap.\n" +
			"It contains no GTT methodology: what GTT means and how it governs belongs to the Bootstrap.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return e.setup()
		},
	}
	pf := root.PersistentFlags()
	pf.BoolVar(&e.opts.JSON, "json", false, "machine-readable JSON output")
	pf.BoolVarP(&e.opts.Verbose, "verbose", "v", false, "show detail")
	pf.BoolVarP(&e.opts.Yes, "yes", "y", false, "answer confirmations with yes (an explicit human pre-authorisation)")
	pf.BoolVar(&e.opts.NoInput, "no-input", false, "never prompt; fail when a human decision is missing")
	pf.StringVar(&e.opts.LogLevel, "log-level", "warn", "internal log level on stderr: error, warn, info, debug")
	pf.StringVarP(&e.opts.Dir, "dir", "C", "", "run as if started in this directory")
	root.AddCommand(
		initCmd(e), statusCmd(e), inspectCmd(e), validateCmd(e), resumeCmd(e), freezeCmd(e), doctorCmd(e), auditCmd(e),
		updateCmd(e), exportCmd(e), cleanCmd(e), versionCmd(e),
		configCmd(e), methodCmd(e), agentsCmd(e), sourcesCmd(e), syncCmd(e), snapshotCmd(e), artifactCmd(e), releaseCmd(e),
	)
	return root
}

// usage builds an invalid-invocation error.
func usage(format string, args ...any) error {
	return &core.Error{Code: core.ExitUsage, Unmodified: true, What: fmt.Sprintf(format, args...)}
}

func addResolveFlags(cmd *cobra.Command, r *app.ResolveOptions) {
	f := cmd.Flags()
	f.StringVar(&r.Path, "bootstrap", "", "path to a local GTT Bootstrap package")
	f.StringVar(&r.Version, "bootstrap-version", "", "exact Bootstrap version to use")
	f.BoolVar(&r.Offline, "offline", false, "use only a local path or the verified cache")
}
