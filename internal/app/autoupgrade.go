package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GTT-Community/gtt-cli/internal/core"
)

// Automatic upgrade: gtt looks for a newer release on its own, at most once every checkEvery, and says so (or,
// in a terminal, replaces itself). What leaves the machine is one HEAD request for the public "latest release"
// page of the project, with the user agent "gtt-cli": no project, no path, no identity. The answer is kept in
// <home>/update-check.json, so every other command costs nothing. It never runs in CI or in --json mode, and
// GTT_NO_UPDATE_CHECK=1 turns the whole thing off.

const (
	checkEvery     = 6 * time.Hour
	retryAfterFail = time.Hour
	nagEvery       = 12 * time.Hour
	nagBootstrap   = 24 * time.Hour
	checkTimeout   = 2 * time.Second
)

type upgradeState struct {
	Schema         int               `json:"schema"`
	CheckedAt      string            `json:"checked_at,omitempty"`
	FailedAt       string            `json:"failed_at,omitempty"`
	Latest         string            `json:"latest,omitempty"`
	NaggedAt       string            `json:"nagged_at,omitempty"`
	Auto           *bool             `json:"auto,omitempty"`
	BootstrapNagAt map[string]string `json:"bootstrap_nag_at,omitempty"`
}

func (a *App) upgradeStatePath() string { return filepath.Join(a.Home, "update-check.json") }

func (a *App) readUpgradeState() upgradeState {
	var st upgradeState
	if data, err := os.ReadFile(a.upgradeStatePath()); err == nil {
		_ = json.Unmarshal(data, &st)
	}
	st.Schema = 1
	return st
}

func (a *App) writeUpgradeState(st upgradeState) {
	if a.Home == "" {
		return
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil || os.MkdirAll(a.Home, 0o755) != nil {
		return
	}
	tmp := a.upgradeStatePath() + ".tmp"
	if os.WriteFile(tmp, append(data, '\n'), 0o644) == nil {
		_ = os.Rename(tmp, a.upgradeStatePath())
	}
}

// SetAutoUpgrade records the human's choice: whether gtt replaces itself when it finds a newer release.
func (a *App) SetAutoUpgrade(on bool) error {
	if a.Home == "" {
		return &core.Error{Code: core.ExitFailure, What: "There is no GTT home to keep the choice in.", Unmodified: true}
	}
	st := a.readUpgradeState()
	st.Auto = &on
	a.writeUpgradeState(st)
	return nil
}

// AutoUpgradeOn is true unless the human turned it off (upgrade --auto off, or GTT_NO_AUTO_UPGRADE).
func (a *App) AutoUpgradeOn() bool {
	if os.Getenv("GTT_NO_AUTO_UPGRADE") != "" {
		return false
	}
	st := a.readUpgradeState()
	return st.Auto == nil || *st.Auto
}

func due(stamp string, now time.Time, every time.Duration) bool {
	t, err := time.Parse(time.RFC3339, stamp)
	return err != nil || now.Sub(t) >= every
}

func stamp(now time.Time) string { return now.UTC().Format(time.RFC3339) }

// AutoUpgrade looks for a newer release (cached) and returns the one line to say, or "". In an interactive
// terminal, with the choice on, it also replaces the binary by the same verified path as `gtt upgrade`.
func (a *App) AutoUpgrade(ctx context.Context, now time.Time) string {
	if a.Home == "" || a.Upgrader == nil || os.Getenv("GTT_NO_UPDATE_CHECK") != "" || os.Getenv("CI") != "" {
		return ""
	}
	st := a.readUpgradeState()
	if due(st.CheckedAt, now, checkEvery) && due(st.FailedAt, now, retryAfterFail) {
		cctx, cancel := context.WithTimeout(ctx, checkTimeout)
		latest, err := a.Upgrader.Latest(cctx)
		cancel()
		if err != nil {
			st.FailedAt = stamp(now)
		} else {
			st.Latest, st.CheckedAt, st.FailedAt = latest, stamp(now), ""
		}
		a.writeUpgradeState(st)
	}
	cur, okCur := core.ParseSemver(core.Version)
	lv, okLatest := core.ParseSemver(st.Latest)
	if !okCur || !okLatest || core.CompareSemver(cur, lv) >= 0 {
		return ""
	}
	if a.Prompt != nil && a.Prompt.Interactive() && a.AutoUpgradeOn() {
		if plan := a.Upgrader.Plan(st.Latest); !plan.Manual {
			if err := a.Upgrader.Apply(ctx, plan); err != nil {
				return fmt.Sprintf("⚠ gtt %s -> %s: could not upgrade by itself (%v). Run: gtt upgrade  |  no se pudo actualizar solo: gtt upgrade", core.Version, st.Latest, err)
			}
			return fmt.Sprintf("✓ gtt %s -> %s (upgraded by itself; gtt upgrade --auto off to stop). In each project: gtt update  |  actualizado solo; en cada proyecto: gtt update", core.Version, st.Latest)
		}
	}
	if !due(st.NaggedAt, now, nagEvery) {
		return ""
	}
	st.NaggedAt = stamp(now)
	a.writeUpgradeState(st)
	return fmt.Sprintf("⚠ gtt %s -> %s is available: gtt upgrade  |  hay una versión nueva: gtt upgrade", core.Version, st.Latest)
}

// ProjectBootstrapNotice says, once a day per project and without network, that the Bootstrap of the project
// you are in is older than the one this gtt ships (`gtt update` is the human's act that brings it up).
func (a *App) ProjectBootstrapNotice(ctx context.Context, now time.Time) string {
	if a.Home == "" || os.Getenv("CI") != "" {
		return ""
	}
	det := a.detect(ctx)
	if det.Root == "" || !a.Factory.Installed(det.Root) {
		return ""
	}
	st := a.readUpgradeState()
	if !due(st.BootstrapNagAt[det.Root], now, nagBootstrap) {
		return "" // said today already: no need even to ask the project
	}
	rel, err := a.gate(ctx, a.Factory.At(det.Root))
	if err != nil {
		return ""
	}
	have, ok1 := core.ParseSemver(rel.Bootstrap.Version)
	ship, ok2 := core.ParseSemver(core.BootstrapPin)
	if !ok1 || !ok2 || core.CompareSemver(have, ship) >= 0 {
		return ""
	}
	if st.BootstrapNagAt == nil {
		st.BootstrapNagAt = map[string]string{}
	}
	st.BootstrapNagAt[det.Root] = stamp(now)
	a.writeUpgradeState(st)
	return fmt.Sprintf("⚠ This project runs Bootstrap %s; this gtt ships %s. Update it: gtt update  |  el Bootstrap de este proyecto es el %s; este gtt trae el %s: gtt update",
		rel.Bootstrap.Version, core.BootstrapPin, rel.Bootstrap.Version, core.BootstrapPin)
}
