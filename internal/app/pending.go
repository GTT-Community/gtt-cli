package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Pending events: what waits for the human is one small file per question or
// decision in gtt-domain/pending/. The files, the limits, the key check and the
// numbering belong to the Bootstrap (operations pending.*); the CLI lists them
// and records the human's answer. It reads no Bootstrap file and writes none.

// PendingItem is one event, as `pending.list` declares it.
type PendingItem struct {
	ID          string            `json:"id"`
	Number      int               `json:"number"`
	Kind        string            `json:"kind"` // ASK | DEC
	Title       string            `json:"title"`
	Class       string            `json:"class"`
	Created     string            `json:"created"`
	State       string            `json:"state"` // open | answered | attended
	Answer      string            `json:"answer"`
	Blocks      string            `json:"blocks"`
	Recommended int               `json:"recommended"`
	Options     []string          `json:"options"`
	Notes       map[string]string `json:"notes"`
	Topic       string            `json:"topic"`
	Discuss     bool              `json:"discuss"`
	Closing     []string          `json:"closing"`
	Path        string            `json:"path"`
}

// PendingList is `gtt pending`.
type PendingList struct {
	Schema    int           `json:"schema"`
	Supported bool          `json:"supported"`
	Events    []PendingItem `json:"events"`
	Open      int           `json:"open"`
}

// PendingAnswer is `gtt answer`: what was written, per event.
type PendingAnswer struct {
	Schema   int      `json:"schema"`
	Answered []string `json:"answered"`
	Output   string   `json:"output"`
}

func notPendingSupported(version string) error {
	return &core.Error{Code: core.ExitIncompatible, Unmodified: true,
		What: "The installed GTT Bootstrap " + version + " does not offer pending events (gtt pending, gtt answer).", Next: "Run: gtt update"}
}

func (a *App) openPending(ctx context.Context) (*Project, error) {
	p, err := a.open(ctx)
	if err != nil {
		return nil, err
	}
	if !supports(ctx, p.B, "pending.list") || !supports(ctx, p.B, "pending.answer") {
		return nil, notPendingSupported(p.Release.Bootstrap.Version)
	}
	return p, nil
}

// PendingEvents lists what waits for the human (all=true adds the marked ones).
func (a *App) PendingEvents(ctx context.Context, all bool) (PendingList, error) {
	res := PendingList{Schema: core.OutputSchema}
	p, err := a.openPending(ctx)
	if err != nil {
		return res, err
	}
	args := map[string]string{}
	if all {
		args["all"] = "true"
	}
	if _, err = a.query(ctx, p.B, "pending.list", args, &res); err != nil {
		return res, err
	}
	res.Schema, res.Supported = core.OutputSchema, true
	return res, nil
}

// pendingRef accepts `14`, `014`, `ASK-14` and `DEC-015`.
func pendingRef(ref string) (int, error) {
	s := strings.ToUpper(strings.TrimSpace(ref))
	s = strings.TrimPrefix(strings.TrimPrefix(s, "ASK-"), "DEC-")
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, &core.Error{Code: core.ExitUsage, Unmodified: true, What: fmt.Sprintf("%q is not an event.", ref), Next: "Use the number `gtt pending` lists: gtt answer 14 1"}
	}
	return n, nil
}

// describe is the line the human sees before the answer is written.
func describe(it PendingItem, reply string) string {
	line := fmt.Sprintf("%s · %s  ←  %s", it.ID, it.Title, reply)
	if n, err := strconv.Atoi(strings.TrimSpace(reply)); err == nil && n >= 1 && n <= len(it.Options) {
		line = fmt.Sprintf("%s · %s  ←  %d. %s", it.ID, it.Title, n, it.Options[n-1])
	}
	return line
}

// PendingRespond is the human's act. It shows what it will write, then writes it
// through pending.answer with confirmed_by_human. `ok` alone approves every
// decision that is still pending; there is no --yes and an agent is denied it.
func (a *App) PendingRespond(ctx context.Context, ref, reply string) (PendingAnswer, error) {
	res := PendingAnswer{Schema: core.OutputSchema}
	list, err := a.PendingEvents(ctx, false)
	if err != nil {
		return res, err
	}
	args := map[string]string{}
	var targets []string
	if reply == "" {
		if !strings.EqualFold(strings.TrimSpace(ref), "ok") {
			return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "An answer is missing.", Next: "Use: gtt answer 14 1   (or: gtt answer ok, to approve every decision)"}
		}
		for _, it := range list.Events {
			if it.Kind == "DEC" && it.State != "attended" {
				targets = append(targets, describe(it, "ok"))
			}
		}
		if len(targets) == 0 {
			a.Report.Info("No decision is pending.\n")
			return res, nil
		}
		args["ref"] = "ok"
	} else {
		n, err := pendingRef(ref)
		if err != nil {
			return res, err
		}
		var found *PendingItem
		for i := range list.Events {
			if list.Events[i].Number == n {
				found = &list.Events[i]
			}
		}
		if found == nil {
			return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: fmt.Sprintf("There is no pending event %d.", n), Next: "List them with: gtt pending"}
		}
		if found.State == "attended" {
			return res, &core.Error{Code: core.ExitConflict, Unmodified: true, What: found.ID + " is already attended.", Next: "The next session deletes it."}
		}
		targets = append(targets, describe(*found, reply))
		args["ref"], args["reply"] = strconv.Itoa(n), reply
	}
	for _, t := range targets {
		a.Report.Info("%s\n", t)
	}
	p, err := a.openPending(ctx)
	if err != nil {
		return res, err
	}
	out, err := p.B.Execute(ctx, ports.OperationRequest{Operation: "pending.answer", Args: args, Apply: true, ConfirmedByHuman: true})
	if err != nil {
		return res, err
	}
	res.Output = strings.TrimSpace(out.Stdout + "\n" + out.Stderr)
	a.log(p.Root, "answer", outcome(out.OK()), map[string]any{"ref": args["ref"]})
	if !out.OK() {
		return res, &core.Error{Code: core.ExitFailure, What: "The answer was not written.", Why: res.Output, Unmodified: true}
	}
	res.Answered = targets
	return res, nil
}
