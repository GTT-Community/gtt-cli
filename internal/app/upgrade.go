package app

import (
	"context"
	"strings"

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
