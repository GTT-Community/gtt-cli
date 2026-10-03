// Package compose wires adapters to use cases. It is the composition root
// shared by the gtt command and by the integration tests.
package compose

import (
	"io"
	"os"
	"path/filepath"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/bootstrap"
	"github.com/GTT-Community/gtt-cli/internal/cli"
	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/execution"
	"github.com/GTT-Community/gtt-cli/internal/output"
	"github.com/GTT-Community/gtt-cli/internal/ports"
	"github.com/GTT-Community/gtt-cli/internal/project"
	"github.com/GTT-Community/gtt-cli/internal/ui"
)

// Home is the user-level GTT home: cache and snapshots, outside any project.
func Home() string {
	if h := os.Getenv("GTT_HOME"); h != "" {
		return h
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, ".gtt")
}

// Streams are the process streams of one invocation.
type Streams struct {
	In       io.Reader
	Out, Err io.Writer
	// Prompter overrides the terminal prompter (tests).
	Prompter ports.Prompter
}

// Wire returns the cli.Wire for the given streams.
func Wire(s Streams) cli.Wire {
	return func(o cli.Options) (*app.App, output.Renderer, error) {
		out := output.Renderer{Out: s.Out, Err: s.Err, JSON: o.JSON, Verbose: o.Verbose, LogLevel: o.LogLevel}
		cwd, err := os.Getwd()
		if err != nil {
			return nil, out, core.Errorf(core.ExitFailure, "cannot determine the working directory: %v", err)
		}
		if o.Dir != "" {
			if cwd, err = filepath.Abs(o.Dir); err != nil {
				return nil, out, core.Errorf(core.ExitUsage, "invalid directory %q", o.Dir)
			}
			if info, serr := os.Stat(cwd); serr != nil || !info.IsDir() {
				return nil, out, core.Errorf(core.ExitUsage, "not a directory: %s", cwd)
			}
		}
		prompter := s.Prompter
		if prompter == nil || o.NoInput || o.JSON {
			// Prompts go to stderr so that stdout stays clean for --json.
			prompter = ui.New(s.In, s.Err, o.NoInput || o.JSON || s.Prompter != nil)
		}
		runner := &execution.Runner{Log: out}
		factory := bootstrap.Factory{Runner: runner}
		return &app.App{
			Factory: factory,
			Resolver: bootstrap.Resolver{Runner: runner, Factory: factory, Home: Home(), Remote: os.Getenv("GTT_BOOTSTRAP_SOURCE"),
				KeysDir: filepath.Join(Home(), "trusted-keys")},
			Signer:    bootstrap.Signer{Factory: factory},
			Installer: bootstrap.Installer{},
			Runner:    runner,
			Git:       project.Git{Runner: runner},
			Prompt:    prompter,
			Report:    out,
			Home:      Home(),
			Cwd:       cwd,
			AssumeYes: o.Yes,
		}, out, nil
	}
}
