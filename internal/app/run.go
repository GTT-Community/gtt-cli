package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Simple execution: when the human has to run something, GTT hands over one
// line and one command (`gtt run <n>`). The plan, the sealed script, the
// review and the execution belong to the Bootstrap (operations run.*); the
// CLI lists, shows, asks once and calls run.execute. It reads no Bootstrap
// file and writes no script.

// RunFinding is one result of the Bootstrap's review (RUN-01 to RUN-09).
type RunFinding struct {
	Rule    string `json:"rule"`
	Level   string `json:"level"`
	Where   string `json:"where"`
	Message string `json:"message"`
}

// RunStep is one step of a plan.
type RunStep struct {
	Title   string `json:"title"`
	Command string `json:"command"`
}

// RunItem is one pending run, as `run.list` declares it.
type RunItem struct {
	ID        string       `json:"id"`
	Number    int          `json:"number"`
	Title     string       `json:"title"`
	What      string       `json:"what"`
	Why       string       `json:"why"`
	Changes   string       `json:"changes"`
	Checks    []string     `json:"checks"`
	Steps     []RunStep    `json:"steps"`
	Next      string       `json:"next"`
	Generated bool         `json:"generated"`
	State     string       `json:"state"`
	OK        bool         `json:"ok"`
	Findings  []RunFinding `json:"findings"`
}

// RunList is `gtt run`.
type RunList struct {
	Schema    int       `json:"schema"`
	Supported bool      `json:"supported"`
	Runs      []RunItem `json:"runs"`
}

// RunCheck is `gtt run check`.
type RunCheck struct {
	Schema int  `json:"schema"`
	OK     bool `json:"ok"`
	Runs   []struct {
		ID       string       `json:"id"`
		OK       bool         `json:"ok"`
		Findings []RunFinding `json:"findings"`
	} `json:"runs"`
}

// RunResult is `gtt run <n>`.
type RunResult struct {
	Schema  int    `json:"schema"`
	ID      string `json:"id"`
	Applied bool   `json:"applied"`
	Output  string `json:"output"`
}

func notRunSupported(version string) error {
	return &core.Error{Code: core.ExitIncompatible, Unmodified: true,
		What: "The installed GTT Bootstrap " + version + " does not offer simple execution (gtt run).", Next: "Run: gtt update"}
}

func (a *App) openRun(ctx context.Context) (*Project, error) {
	p, err := a.open(ctx)
	if err != nil {
		return nil, err
	}
	if !supports(ctx, p.B, "run.list") || !supports(ctx, p.B, "run.execute") {
		return nil, notRunSupported(p.Release.Bootstrap.Version)
	}
	return p, nil
}

// RunPending lists the runs waiting for the human.
func (a *App) RunPending(ctx context.Context) (RunList, error) {
	res := RunList{Schema: core.OutputSchema}
	p, err := a.openRun(ctx)
	if err != nil {
		return res, err
	}
	if _, err = a.query(ctx, p.B, "run.list", nil, &res); err != nil {
		return res, err
	}
	res.Schema, res.Supported = core.OutputSchema, true
	return res, nil
}

// runRef accepts `3`, `003` and `RUN-003`.
func runRef(ref string) (int, error) {
	s := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(ref)), "RUN-")
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, &core.Error{Code: core.ExitUsage, Unmodified: true, What: fmt.Sprintf("%q is not a run.", ref), Next: "Use the number `gtt run` lists: gtt run 3"}
	}
	return n, nil
}

// RunShow is one pending run in full.
func (a *App) RunShow(ctx context.Context, ref string) (RunItem, error) {
	n, err := runRef(ref)
	if err != nil {
		return RunItem{}, err
	}
	list, err := a.RunPending(ctx)
	if err != nil {
		return RunItem{}, err
	}
	for _, r := range list.Runs {
		if r.Number == n {
			return r, nil
		}
	}
	return RunItem{}, &core.Error{Code: core.ExitUsage, Unmodified: true, What: fmt.Sprintf("There is no pending run %d.", n), Next: "List them with: gtt run"}
}

// RunReview is the Bootstrap's review of one run, or of all of them.
func (a *App) RunReview(ctx context.Context, ref string) (RunCheck, error) {
	res := RunCheck{Schema: core.OutputSchema}
	p, err := a.openRun(ctx)
	if err != nil {
		return res, err
	}
	args := map[string]string{}
	if ref != "" {
		n, err := runRef(ref)
		if err != nil {
			return res, err
		}
		args["run"] = strconv.Itoa(n)
	}
	out, err := p.B.Execute(ctx, ports.OperationRequest{Operation: "run.check", Args: args})
	if err != nil {
		return res, err
	}
	if jerr := out.JSON(&res); jerr != nil {
		return res, &core.Error{Code: core.ExitFailure, What: "The Bootstrap result of run.check is unreadable.", Why: jerr.Error()}
	}
	res.Schema = core.OutputSchema
	if !res.OK {
		var why []string
		for _, r := range res.Runs {
			for _, f := range r.Findings {
				if f.Level == "fail" {
					why = append(why, r.ID+" "+f.Rule+": "+f.Message)
				}
			}
		}
		return res, &core.Error{Code: core.ExitFailure, What: "The review of a run failed.", Why: strings.Join(why, "\n"), Next: "Fix the plan and generate it again."}
	}
	return res, nil
}

// RunExecute is the human's act. It shows the steps, asks once when the run
// changes something (never pre-answered: there is no --yes here), and
// executes through run.execute with confirmed_by_human. A run that changes
// nothing needs no question and runs without a terminal.
func (a *App) RunExecute(ctx context.Context, ref string) (RunResult, error) {
	res := RunResult{Schema: core.OutputSchema}
	item, err := a.RunShow(ctx, ref)
	if err != nil {
		return res, err
	}
	res.ID = item.ID
	if !item.OK {
		var why []string
		for _, f := range item.Findings {
			if f.Level == "fail" {
				why = append(why, f.Rule+": "+f.Message)
			}
		}
		next := "Ask the agent to fix the plan and generate it again."
		if !item.Generated {
			next = "Ask the agent to generate it: gtt-run.sh generate " + item.ID
		}
		return res, &core.Error{Code: core.ExitConflict, Unmodified: true, What: item.ID + " did not pass its review; nothing was run.", Why: strings.Join(why, "\n"), Next: next}
	}
	a.Report.Info("%s - %s\nWhy: %s\n", item.ID, item.What, item.Why)
	for _, c := range item.Checks {
		a.Report.Info("  check  %s\n", c)
	}
	for i, s := range item.Steps {
		a.Report.Info("  %d. %s\n", i+1, s.Title)
	}
	a.Report.Info("Changes: %s\n", item.Changes)
	for _, f := range item.Findings {
		a.Report.Info("  %s %s: %s\n", strings.ToUpper(f.Level), f.Rule, f.Message)
	}
	if item.Changes != "none" {
		if !a.Prompt.Interactive() {
			return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: item.ID + " changes " + item.Changes + "; running it is a human confirmation.",
				Why: "It must be confirmed by a person in a terminal; it is never run unattended or by an agent.", Next: "Run `gtt run " + strconv.Itoa(item.Number) + "` yourself."}
		}
		ok, err := a.Prompt.Confirm("Run it now?", false)
		if err != nil {
			return res, err
		}
		if !ok {
			return res, core.ErrCancelled
		}
	}
	p, err := a.openRun(ctx)
	if err != nil {
		return res, err
	}
	out, err := p.B.Execute(ctx, ports.OperationRequest{Operation: "run.execute", Args: map[string]string{"run": strconv.Itoa(item.Number)}, Apply: true, ConfirmedByHuman: true})
	if err != nil {
		return res, err
	}
	res.Output = strings.TrimSpace(out.Stdout + "\n" + out.Stderr)
	a.log(p.Root, "run", outcome(out.OK()), map[string]any{"run": item.ID})
	if !out.OK() {
		return res, &core.Error{Code: core.ExitFailure, What: item.ID + " stopped; it stays pending so it can be run again once fixed.", Why: res.Output}
	}
	res.Applied = true
	return res, nil
}
