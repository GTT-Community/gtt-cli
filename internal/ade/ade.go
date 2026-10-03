// Package ade detects ADE candidates from the Bootstrap registry and starts
// an ADE through a trusted executor. It contains no ADE catalog of its own
// and no methodology.
package ade

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Registry reads the supported ADEs from the Bootstrap.
func Registry(ctx context.Context, b ports.Bootstrap) ([]ports.ADE, error) {
	raw, err := b.Show(ctx, "ade-registry")
	if err != nil {
		return nil, err
	}
	var doc struct {
		ADEs []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Detect struct {
				Paths  []string `json:"paths"`
				Binary string   `json:"binary"`
			} `json:"detect"`
			Handoff struct {
				Entry string `json:"instruction_entry"`
			} `json:"handoff"`
			Invoke struct {
				Binary string `json:"binary"`
			} `json:"invoke"`
			Enforcement string   `json:"enforcement"`
			OwnedPaths  []string `json:"owned_paths"`
			Version     int      `json:"version"`
		} `json:"ades"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, &core.Error{Code: core.ExitIntegrity, What: "The Bootstrap ADE registry is unreadable.", Why: err.Error()}
	}
	out := make([]ports.ADE, 0, len(doc.ADEs))
	for _, a := range doc.ADEs {
		bin := a.Invoke.Binary
		if bin == "" {
			bin = a.Detect.Binary
		}
		out = append(out, ports.ADE{ID: a.ID, Name: a.Name, DetectPaths: a.Detect.Paths, Binary: bin,
			Entry: a.Handoff.Entry, Enforcement: a.Enforcement, OwnedPaths: a.OwnedPaths, Version: a.Version})
	}
	return out, nil
}

// Detect observes which registry ADEs show signals in the project or on the
// machine. A signal makes an ADE a candidate and nothing more.
func Detect(registry []ports.ADE, projectRoot string, runner ports.Runner) []ports.Candidate {
	out := make([]ports.Candidate, 0, len(registry))
	for _, a := range registry {
		c := ports.Candidate{ADE: a}
		for _, p := range a.DetectPaths {
			if _, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(p))); err == nil {
				c.Signals = append(c.Signals, "path:"+p)
			}
		}
		if a.Binary != "" {
			if _, err := runner.LookPath(a.Binary); err == nil {
				c.Signals = append(c.Signals, "binary:"+a.Binary)
			}
		}
		out = append(out, c)
	}
	return out
}

// Executor starts one ADE in the project. It passes no instructions: the ADE
// reads the Bootstrap-installed entry on its own.
type Executor struct {
	ADE    ports.ADE
	Runner ports.Runner
}

// ID returns the ADE id.
func (e Executor) ID() string { return e.ADE.ID }

// Available reports whether the ADE binary can be started.
func (e Executor) Available() bool {
	if e.ADE.Binary == "" {
		return false
	}
	_, err := e.Runner.LookPath(e.ADE.Binary)
	return err == nil
}

// BuildCommand returns the invocation: the declared binary, no arguments,
// in the project root, attached to the terminal.
func (e Executor) BuildCommand(c ports.HandoffContract) ports.Command {
	path, _ := e.Runner.LookPath(e.ADE.Binary)
	return ports.Command{Path: path, Dir: c.ProjectRoot, Interactive: true}
}

// Execute starts the ADE and waits for it.
func (e Executor) Execute(ctx context.Context, c ports.HandoffContract) error {
	if !e.Available() {
		return &core.Error{Code: core.ExitUnavailable, What: "The ADE " + e.ADE.Name + " cannot be launched from this CLI.",
			Why: "No executable for it was found on PATH."}
	}
	res, err := e.Runner.Run(ctx, e.BuildCommand(c))
	if err != nil {
		return &core.Error{Code: core.ExitUnavailable, What: "Could not start " + e.ADE.Name + ".", Why: err.Error()}
	}
	if res.ExitCode != 0 {
		return core.Errorf(core.ExitFailure, "%s exited with status %d", e.ADE.Name, res.ExitCode)
	}
	return nil
}

// Names maps ids to display names for rendering.
func Names(registry []ports.ADE, ids []string) string {
	byID := map[string]string{}
	for _, a := range registry {
		byID[a.ID] = a.Name
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id
		if n := byID[id]; n != "" {
			out[i] = n
		}
	}
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, ", ")
}
