// Package app holds the use cases of the CLI. It reaches the Bootstrap, the
// terminal, Git and processes only through ports, and never decides what GTT
// means: it resolves, selects, invokes and reports.
package app

import (
	"context"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/lifecycle"
	"github.com/GTT-Community/gtt-cli/internal/ports"
	"github.com/GTT-Community/gtt-cli/internal/project"
)

// App is the set of use cases, wired by the composition root.
type App struct {
	Factory   ports.BootstrapFactory
	Resolver  ports.Resolver
	Installer ports.Installer
	Signer    ports.ReleaseSigner
	Runner    ports.Runner
	Git       ports.Git
	Prompt    ports.Prompter
	Report    ports.Reporter

	Home      string // user-level GTT home
	Cwd       string
	AssumeYes bool // the human pre-answered confirmations with --yes
}

// Project is an installed project bound to its Bootstrap runtime.
type Project struct {
	Root      string
	Detection project.Detection
	B         ports.Bootstrap
	Release   ports.Release
	Store     lifecycle.Store
}

// detect finds where the CLI is running.
func (a *App) detect(ctx context.Context) project.Detection {
	return project.Detect(ctx, a.Git, a.Factory.Installed, a.Cwd)
}

// open binds the installed project and enforces compatibility. Every use
// case that operates the Bootstrap starts here: an incompatible pair is
// refused before anything else happens.
func (a *App) open(ctx context.Context) (*Project, error) {
	det := a.detect(ctx)
	if !a.Factory.Installed(det.Root) {
		return nil, &core.Error{Code: core.ExitConflict, Unmodified: true,
			What: "GTT is not initialized in this project (" + det.Root + ").", Next: "Run: gtt init"}
	}
	p := &Project{Root: det.Root, Detection: det, B: a.Factory.At(det.Root), Store: lifecycle.Store{Root: det.Root}}
	rel, err := a.gate(ctx, p.B)
	if err != nil {
		return nil, err
	}
	p.Release = rel
	return p, nil
}

// gate negotiates compatibility with a Bootstrap tree and refuses on any doubt.
func (a *App) gate(ctx context.Context, b ports.Bootstrap) (ports.Release, error) {
	rel, err := b.Release(ctx)
	if err != nil {
		return rel, err
	}
	n, err := b.Negotiate(ctx)
	if err != nil {
		return rel, err
	}
	if !n.Compatible {
		return rel, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: "GTT Bootstrap " + rel.Bootstrap.Version + " and GTT CLI " + core.Version + " are not compatible.",
			Why:  strings.Join(n.Reasons, "\n"),
			Next: "Use a compatible GTT CLI / Bootstrap pair."}
	}
	return rel, nil
}

// query runs a read-only operation and decodes its JSON result.
func (a *App) query(ctx context.Context, b ports.Bootstrap, op string, args map[string]string, v any) (ports.OperationResult, error) {
	res, err := b.Execute(ctx, ports.OperationRequest{Operation: op, Args: args})
	if err != nil {
		return res, err
	}
	if v != nil && res.Stdout != "" {
		if jerr := res.JSON(v); jerr != nil {
			return res, &core.Error{Code: core.ExitFailure, What: "The Bootstrap result of " + op + " is unreadable.", Why: jerr.Error()}
		}
	}
	return res, nil
}

// mutate runs a mutating operation as the Bootstrap intends: dry run first,
// then apply. A dry run that reports a problem stops before any change.
func (a *App) mutate(ctx context.Context, b ports.Bootstrap, op string, args map[string]string) (ports.OperationResult, error) {
	dry, err := b.Execute(ctx, ports.OperationRequest{Operation: op, Args: args})
	if err != nil {
		return dry, err
	}
	if !dry.OK() {
		return dry, &core.Error{Code: core.ExitConflict, What: "The Bootstrap refused " + op + ".",
			Why: strings.TrimSpace(dry.Stdout + "\n" + dry.Stderr)}
	}
	a.Report.Detail(dry.Stdout)
	res, err := b.Execute(ctx, ports.OperationRequest{Operation: op, Args: args, Apply: true})
	if err != nil {
		return res, err
	}
	if !res.OK() {
		return res, &core.Error{Code: core.ExitFailure, What: "The Bootstrap could not apply " + op + ".",
			Why: strings.TrimSpace(res.Stdout + "\n" + res.Stderr)}
	}
	return res, nil
}

// confirm asks the human. --yes pre-answers it; without a terminal and
// without --yes the operation is refused rather than assumed.
func (a *App) confirm(question string, def bool) (bool, error) {
	if a.AssumeYes {
		return true, nil
	}
	if !a.Prompt.Interactive() {
		return false, &core.Error{Code: core.ExitUsage, Unmodified: true,
			What: "This operation needs a human confirmation: " + question,
			Next: "Run it in a terminal, or pass --yes to confirm explicitly."}
	}
	return a.Prompt.Confirm(question, def)
}

// log records an operation in the project's operation log (no secrets).
func (a *App) log(root, operation, result string, extra map[string]any) {
	fields := map[string]any{"operation": operation, "result": result, "cli": core.Version}
	for k, v := range extra {
		fields[k] = v
	}
	lifecycle.Store{Root: root}.Log(fields)
}

// validation is the Bootstrap's validation result.
type validation struct {
	Result string `json:"result"`
	Checks []struct {
		Check  string `json:"check"`
		Result string `json:"result"`
	} `json:"checks"`
	Failing  []string `json:"failing"`
	Messages []string `json:"messages"`
}

// validateProject invokes the Bootstrap's single validation entry point.
func (a *App) validateProject(ctx context.Context, b ports.Bootstrap) (validation, error) {
	var v validation
	res, err := a.query(ctx, b, "validation.run", nil, &v)
	if err != nil {
		return v, err
	}
	if v.Result == "" {
		v.Result = "fail"
		v.Messages = append(v.Messages, strings.TrimSpace(res.Stderr))
	}
	return v, nil
}

// reindex reconciles the identity manifest with what is installed and
// rebuilds the derived index: the deterministic step after an installation.
func (a *App) reindex(ctx context.Context, b ports.Bootstrap) error {
	if _, err := a.mutate(ctx, b, "reconcile", map[string]string{"retire_missing": "true"}); err != nil {
		return err
	}
	res, err := b.Execute(ctx, ports.OperationRequest{Operation: "index", Apply: true})
	if err != nil {
		return err
	}
	if !res.OK() {
		return &core.Error{Code: core.ExitFailure, What: "The Bootstrap could not rebuild the index.", Why: strings.TrimSpace(res.Stdout + "\n" + res.Stderr)}
	}
	return nil
}
