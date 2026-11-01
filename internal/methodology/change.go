package methodology

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Directions of a plan change.
const (
	Initial    = "selected"
	Relaxed    = "relaxed"
	Reinforced = "reinforced"
)

// Difference is one control whose requirement differs between two plans, in
// the Bootstrap's own words.
type Difference struct {
	Control string `json:"control"`
	From    string `json:"from"`
	To      string `json:"to"`
}

// Change describes what a plan change means. Every sentence about a plan comes
// from the Bootstrap's profile contract; the CLI only compares and points to
// the way back.
type Change struct {
	From        string       `json:"from,omitempty"`
	To          string       `json:"to"`
	Direction   string       `json:"direction"`
	Relaxes     []string     `json:"relaxes,omitempty"`
	Differences []Difference `json:"differences,omitempty"`
	// Opposite is the plan that moves the other way from this one, if any.
	Opposite string `json:"opposite,omitempty"`
	// Revert is the plan the project came from, if any.
	Revert string `json:"revert,omitempty"`
}

// Compare reads the profile contract and describes moving from one plan to
// another. from is empty when no plan was selected before.
func Compare(ctx context.Context, b ports.Bootstrap, from, to string) (Change, error) {
	raw, err := b.Show(ctx, "profiles")
	if err != nil {
		return Change{}, err
	}
	var doc struct {
		Order    []string `json:"strictness_order"`
		Profiles map[string]struct {
			Semantics map[string]string `json:"semantics"`
			Relaxes   []struct {
				Control string `json:"control"`
				How     string `json:"how"`
			} `json:"relaxes"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Change{}, &core.Error{Code: core.ExitIntegrity, What: "The Bootstrap profile contract is unreadable.", Why: err.Error()}
	}
	pos := func(id string) int {
		for i, o := range doc.Order {
			if o == id {
				return i
			}
		}
		return -1
	}
	c := Change{From: from, To: to, Direction: Initial}
	at := pos(to)
	switch {
	case from != "" && pos(from) >= 0 && at >= 0 && at < pos(from):
		c.Direction = Relaxed
	case from != "" && pos(from) >= 0 && at >= 0 && at > pos(from):
		c.Direction = Reinforced
	}
	if from != "" && from != to {
		c.Revert = from
	}
	// The suggestion points the other way: if the plan proves too loose,
	// the next stricter one; too heavy, the next lighter.
	switch {
	case c.Direction == Relaxed && at >= 0 && at+1 < len(doc.Order):
		c.Opposite = doc.Order[pos(from)]
	case c.Direction == Reinforced && at > 0:
		c.Opposite = doc.Order[at-1]
	case c.Direction == Initial && at >= 0 && at+1 < len(doc.Order):
		c.Opposite = doc.Order[at+1]
	}
	target := doc.Profiles[to]
	for _, r := range target.Relaxes {
		c.Relaxes = append(c.Relaxes, r.Control+": "+r.How)
	}
	if from != "" {
		before := doc.Profiles[from].Semantics
		keys := make([]string, 0, len(target.Semantics))
		for k := range target.Semantics {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if before[k] != target.Semantics[k] {
				c.Differences = append(c.Differences, Difference{Control: k, From: before[k], To: target.Semantics[k]})
			}
		}
	}
	return c, nil
}
