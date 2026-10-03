// Package execution runs external processes through structured argv. It
// never builds or interprets a shell string.
package execution

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Runner is the process adapter.
type Runner struct {
	// Log receives one debug line per process; nil means no logging.
	Log ports.Logger
}

// New returns a Runner.
func New() *Runner { return &Runner{} }

// describe names a command for the log without its values: the executable
// base name and each argument up to its "=", so that no path, credential or
// other value an argument carries is ever written.
func describe(c ports.Command) string {
	// The base name without ".exe", so a log reads the same on every platform.
	parts := []string{strings.TrimSuffix(filepath.Base(c.Path), ".exe")}
	for _, a := range c.Args {
		if i := strings.IndexByte(a, '='); i >= 0 {
			a = a[:i+1] + "..."
		}
		parts = append(parts, a)
	}
	return strings.Join(parts, " ")
}

func (r Runner) trace(c ports.Command, res ports.Result, err error) {
	if r.Log == nil {
		return
	}
	if err != nil {
		r.Log.Log("debug", "exec %s: not run", describe(c))
		return
	}
	r.Log.Log("debug", "exec %s: exit %d", describe(c), res.ExitCode)
}

// LookPath resolves an executable name on PATH. bash is resolved with the
// platform rules of lookBash (Git Bash on Windows, never the WSL launcher).
func (Runner) LookPath(name string) (string, error) {
	if name == "bash" {
		return systemBash(exec.LookPath)
	}
	return exec.LookPath(name)
}

// Run executes one command. A non-zero exit is reported in the result, not
// as an error; an error means the process could not be run at all.
func (r Runner) Run(ctx context.Context, c ports.Command) (ports.Result, error) {
	cmd := exec.CommandContext(ctx, c.Path, c.Args...)
	cmd.Dir = c.Dir
	if c.Env != nil {
		cmd.Env = append(os.Environ(), c.Env...)
	}
	if c.Interactive {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		res, err := result(nil, nil, cmd.Run())
		r.trace(c, res, err)
		return res, err
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	res, err := result(stdout.Bytes(), stderr.Bytes(), runErr)
	r.trace(c, res, err)
	return res, err
}

func result(stdout, stderr []byte, err error) (ports.Result, error) {
	res := ports.Result{Stdout: stdout, Stderr: stderr}
	if err == nil {
		return res, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		res.ExitCode = exit.ExitCode()
		return res, nil
	}
	return res, err
}
