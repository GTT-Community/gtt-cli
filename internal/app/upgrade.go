package app

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// UpgradeResult is what `gtt upgrade` found and did.
type UpgradeResult struct {
	Schema    int                `json:"schema"`
	Installed string             `json:"installed"`
	Latest    string             `json:"latest,omitempty"`
	Target    string             `json:"target,omitempty"`
	Status    string             `json:"status"` // up-to-date, ahead, available, upgraded
	Changed   bool               `json:"changed"`
	Plan      *ports.UpgradePlan `json:"plan,omitempty"`
}

// Upgrade replaces this binary with a newer release (or, with an explicit target, any release). It touches
// no project. check only reports. The human confirms unless --yes was given.
func (a *App) Upgrade(ctx context.Context, check bool, to string) (UpgradeResult, error) {
	res := UpgradeResult{Schema: 1, Installed: core.Version}
	to = strings.TrimPrefix(strings.TrimSpace(to), "v")
	if to != "" {
		if _, ok := core.ParseSemver(to); !ok {
			return res, &core.Error{Code: core.ExitUsage, Unmodified: true,
				What: "--to needs a release like 1.21.0, not " + to + "."}
		}
	}
	latest := ""
	if to == "" { // an explicit release needs no question to the network
		var err error
		if latest, err = a.Upgrader.Latest(ctx); err != nil {
			return res, err
		}
	}
	res.Latest = latest
	cur, curOK := core.ParseSemver(core.Version)
	if to == "" {
		res.Target = latest
		if lv, ok := core.ParseSemver(latest); ok && curOK {
			switch core.CompareSemver(cur, lv) {
			case 0:
				res.Status = "up-to-date"
				return res, nil
			case 1:
				res.Status = "ahead" // never goes down without --to
				return res, nil
			}
		}
	} else {
		res.Target = to
		if curOK && core.CompareSemver(cur, mustSemver(to)) == 0 {
			res.Status = "up-to-date"
			return res, nil
		}
	}
	plan := a.Upgrader.Plan(res.Target)
	res.Plan = &plan
	res.Status = "available"
	if check {
		return res, nil
	}
	if plan.Manual {
		return res, &core.Error{Code: core.ExitUnavailable, Unmodified: true,
			What: "gtt cannot upgrade itself here: " + plan.Reason + ".", Why: "Nothing was changed.",
			Next: "Run: " + plan.Hint}
	}
	how := plan.Channel
	if len(plan.Command) > 0 {
		how = strings.Join(plan.Command, " ")
	}
	ok, err := a.confirm("Upgrade gtt "+core.Version+" -> "+res.Target+" ("+how+")?", true)
	if err != nil {
		return res, err
	}
	if !ok {
		return res, &core.Error{Code: core.ExitCancelled, Unmodified: true, What: "Upgrade cancelled. Nothing was changed."}
	}
	if err := a.Upgrader.Apply(ctx, plan); err != nil {
		return res, err
	}
	res.Status, res.Changed = "upgraded", true
	return res, nil
}

func mustSemver(s string) [3]int { v, _ := core.ParseSemver(s); return v }

// selfUpgradeLook bounds the look for a newer release made by `gtt update`.
const selfUpgradeLook = 5 * time.Second

// SelfUpgrade is the first half of `gtt update`: the Bootstrap a gtt installs is the one it ships with, so one
// command that brings the whole suite up starts by bringing gtt itself up. It returns the release gtt was
// replaced by ("" when it did not change) and one line to say, "" when there is nothing to say. It never fails
// an update: not reaching the network, or a channel gtt cannot replace itself in, is said in a line and the
// update goes on with the Bootstrap this gtt ships. It runs the same verified path as `gtt upgrade`.
func (a *App) SelfUpgrade(ctx context.Context) (upgraded, note string) {
	if a.Upgrader == nil || os.Getenv("GTT_NO_UPDATE_CHECK") != "" || os.Getenv("GTT_NO_AUTO_UPGRADE") != "" || os.Getenv("CI") != "" {
		return "", ""
	}
	// One short question: with no network `gtt update` must not wait on it.
	lctx, cancel := context.WithTimeout(ctx, selfUpgradeLook)
	latest, err := a.Upgrader.Latest(lctx)
	cancel()
	if err != nil {
		return "", "gtt could not look for a newer release (" + err.Error() + "); updating the Bootstrap with this gtt " + core.Version +
			"  |  no se pudo buscar un gtt más nuevo; se actualiza el Bootstrap con este gtt"
	}
	cur, ok1 := core.ParseSemver(core.Version)
	lv, ok2 := core.ParseSemver(latest)
	if !ok1 || !ok2 || core.CompareSemver(cur, lv) >= 0 {
		return "", ""
	}
	plan := a.Upgrader.Plan(latest)
	if plan.Manual {
		return "", "gtt " + latest + " is available but gtt cannot replace itself here (" + plan.Reason + "). Run: " + plan.Hint +
			" and then gtt update again. Meanwhile the Bootstrap is updated with this gtt  |  hay un gtt más nuevo; no puede cambiarse solo aquí"
	}
	a.selfExe, _ = os.Executable() // before Apply: afterwards it names gtt.old
	if err := a.Upgrader.Apply(ctx, plan); err != nil {
		return "", "gtt " + core.Version + " -> " + latest + " could not be applied (" + err.Error() + "); the previous gtt stays. Run: gtt upgrade"
	}
	return latest, "✓ gtt " + core.Version + " -> " + latest
}

// ContinueAs runs the same command with the gtt that has just replaced this one, on the same terminal. GTT_SELF_UPDATED
// keeps it from looking again. ok is false when the new binary could not be started: the caller then says to run it again.
func (a *App) ContinueAs(ctx context.Context, args []string) (code int, ok bool) {
	exe := a.selfExe // the installed path, taken before the replacement renamed this binary to gtt.old
	if exe == "" {
		var err error
		if exe, err = os.Executable(); err != nil {
			return 0, false
		}
	}
	if a.Runner == nil {
		return 0, false
	}
	res, err := a.Runner.Run(ctx, ports.Command{Path: exe, Args: args, Env: []string{"GTT_SELF_UPDATED=1"}, Interactive: true})
	if err != nil {
		return 0, false
	}
	return res.ExitCode, true
}
