package app

import (
	"context"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// The consultation thread: the questions that need thinking go to ONE conversation per project in Claude web. The thread, the first
// message, the batch of open questions and the reading of the answer block belong to the Bootstrap (operations consult.*); the CLI
// only runs them and says what they said. Nothing here is sent anywhere: Sonar copies the text and the person pastes it.

// ConsultResult is what a consult operation answered.
type ConsultResult struct {
	Schema    int    `json:"schema"`
	Supported bool   `json:"supported"`
	Operation string `json:"operation"`
	Output    string `json:"output"`
}

func notConsultSupported(version string) error {
	return &core.Error{Code: core.ExitIncompatible, Unmodified: true,
		What: "The installed GTT Bootstrap " + version + " does not offer the consultation thread (gtt consult).", Next: "Run: gtt update"}
}

// Consult runs one consult operation. Reads (status, seed, batch) only read; set, clear, seen and answers act on the person's own
// state or on the answers they pasted, so they run with apply and nothing is asked twice. The text is the Bootstrap's own, in the
// project's language.
func (a *App) Consult(ctx context.Context, op string, args map[string]string, apply bool) (ConsultResult, error) {
	res := ConsultResult{Schema: core.OutputSchema, Operation: op}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	if !supports(ctx, p.B, op) {
		return res, notConsultSupported(p.Release.Bootstrap.Version)
	}
	out, err := p.B.Execute(ctx, ports.OperationRequest{Operation: op, Args: args, Apply: apply})
	if err != nil {
		return res, err
	}
	res.Supported = true
	res.Output = strings.TrimRight(out.Stdout+out.Stderr, "\n")
	if apply {
		a.log(p.Root, op, outcome(out.OK()), nil)
	}
	if !out.OK() {
		return res, &core.Error{Code: core.ExitFailure, What: "The Bootstrap refused " + op + ".", Why: res.Output, Unmodified: true}
	}
	return res, nil
}

// ConsultNote is the first line of `consult status`: whether the project's consultation thread is registered, or how to create it. The report of
// an update says it, so the person finds out the thread exists. Empty when the installed Bootstrap has no consultation thread or says nothing.
func (a *App) ConsultNote(ctx context.Context) string {
	res, err := a.Consult(ctx, "consult.status", nil, false)
	if err != nil || !res.Supported {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(res.Output, "\n", 2)[0])
}
