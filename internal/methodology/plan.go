// Package methodology reads the selected Method Plan from the Bootstrap and
// answers one operational question: may the CLI do this without asking?
// What a plan means is defined by the Bootstrap; this package only executes
// the operating policy the Bootstrap declares, using its policy vocabulary.
package methodology

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Decision is how the CLI must treat one kind of operation.
type Decision int

const (
	// Automatic: do it and report.
	Automatic Decision = iota
	// AutomaticSafe: do it without asking only when nothing human-authored
	// would be overwritten; otherwise ask.
	AutomaticSafe
	// Confirm: prepare, show, apply after the human confirms.
	Confirm
	// Propose: propose with reasoning; nothing is written until confirmed.
	Propose
)

// Operation kinds the Bootstrap's operating policy names.
const (
	IdentityResolution = "identity_resolution"
	ReferenceUpdates   = "reference_updates"
	IndexRebuild       = "index_rebuild"
	Validation         = "validation"
	AgentContextSync   = "agent_context_sync"
	RelevantChange     = "relevant_change"
	Destructive        = "destructive"
	GovernedDecision   = "governed_decision"
)

// Plan is the selected Method Plan as the Bootstrap reports it.
type Plan struct {
	Profile   string   `json:"profile"`
	Selected  bool     `json:"selected"`
	Language  string   `json:"language"`
	Frozen    bool     `json:"frozen"`
	Supported []string `json:"supported"`
	Detail    struct {
		Label     string `json:"label"`
		Summary   string `json:"summary"`
		Delegates string `json:"delegates"`
		Policy    struct {
			Automation map[string]string `json:"automation"`
			Human      struct {
				Confirmation map[string]string `json:"confirmation"`
			} `json:"human"`
			Collaboration map[string]string `json:"collaboration"`
		} `json:"policy"`
	} `json:"plan"`
	Interaction struct {
		WithoutAsking []string          `json:"without_asking"`
		AsksFor       map[string]string `json:"asks_for"`
	} `json:"interaction"`
	// Team holds the values the team declared for the kinds the plan leaves
	// to team policy, read from the team's working agreements.
	Team map[string]string `json:"team_policy_declared,omitempty"`
}

// TeamPolicy is the Bootstrap's value for "as the team declares".
const TeamPolicy = "team_policy"

// teamValues are the values a team may declare. Only these: a declaration
// can choose how much the CLI asks, never remove a required confirmation.
var teamValues = map[string]bool{"automatic": true, "automatic_safe": true, "confirm": true, "propose_confirm": true}

// agreementLine is one line of the Bootstrap's agreements report:
// "<id>  [<scope>]  applies to: <applies>  - <text>  (<file>)".
var agreementLine = regexp.MustCompile(`^(\S+)\s+\[(\w+)\]\s+applies to:\s+(.*?)\s+-\s+(.*)\s+\(([^()]*)\)\s*$`)

// TeamDeclarations extracts, from the Bootstrap's agreements report, the
// policy values a team declared. A team agreement declares one when it
// applies to "policy.<kind>" and its text starts with a policy value, for
// example: TA-01 | team | policy.agent_context_sync | automatic_safe.
// Personal preferences (scope user) never set team policy.
func TeamDeclarations(report string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(report, "\n") {
		m := agreementLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil || m[2] != "team" || !strings.HasPrefix(m[3], "policy.") {
			continue
		}
		kind, words := strings.TrimPrefix(m[3], "policy."), strings.Fields(m[4])
		if len(words) == 0 {
			continue
		}
		if value := strings.TrimRight(words[0], ":;,."); teamValues[value] {
			out[kind] = value
		}
	}
	return out
}

// usesTeamPolicy reports whether any value of the plan is left to the team.
func (p Plan) usesTeamPolicy() bool {
	for _, v := range p.Detail.Policy.Automation {
		if v == TeamPolicy {
			return true
		}
	}
	for _, v := range p.Detail.Policy.Human.Confirmation {
		if v == TeamPolicy {
			return true
		}
	}
	return false
}

// Load asks the Bootstrap for the plan in force.
func Load(ctx context.Context, b ports.Bootstrap) (Plan, error) {
	var p Plan
	res, err := b.Execute(ctx, ports.OperationRequest{Operation: "methodology.profile.get"})
	if err != nil {
		return p, err
	}
	if !res.OK() {
		return p, &core.Error{Code: core.ExitFailure, What: "The Bootstrap could not report the Method Plan.", Why: res.Stderr}
	}
	if err := res.JSON(&p); err != nil {
		return p, &core.Error{Code: core.ExitFailure, What: "The Method Plan report is unreadable.", Why: err.Error()}
	}
	if !p.usesTeamPolicy() {
		return p, nil
	}
	agreements, err := b.Execute(ctx, ports.OperationRequest{Operation: "query.governance", Args: map[string]string{"kind": "agreements"}})
	if err != nil {
		return p, err
	}
	if !agreements.OK() {
		return p, &core.Error{Code: core.ExitFailure, What: "The Bootstrap could not report the team's working agreements.", Why: agreements.Stderr}
	}
	p.Team = TeamDeclarations(agreements.Stdout)
	return p, nil
}

// value returns the policy value in force for an operation kind: the
// Bootstrap's, or the team's declaration where the Bootstrap leaves it to
// team policy. With no declaration, team policy stays propose and confirm.
func (p Plan) value(kind string) string {
	v, ok := p.Detail.Policy.Automation[kind]
	if !ok {
		v = p.Detail.Policy.Human.Confirmation[kind]
	}
	if v == TeamPolicy {
		if declared := p.Team[kind]; declared != "" {
			return declared
		}
		return "propose_confirm"
	}
	return v
}

// Value is the policy value in force for an operation kind.
func (p Plan) Value(kind string) string { return p.value(kind) }

// Decide maps the Bootstrap's policy value to a Decision. A destructive
// operation and a governed decision are confirmed in every plan, and any
// value this CLI does not know is treated as Confirm: never guess.
func (p Plan) Decide(kind string) Decision {
	if kind == Destructive || kind == GovernedDecision {
		return Confirm
	}
	switch p.value(kind) {
	case "automatic", "not_required":
		return Automatic
	case "automatic_safe":
		return AutomaticSafe
	case "propose_confirm":
		return Propose
	default:
		return Confirm
	}
}

// Choice is one selectable plan, with the Bootstrap's own wording.
type Choice struct {
	ID      string
	Label   string
	Summary string
}

// Choices reads the selectable plans and their descriptions from the
// Bootstrap, in the Bootstrap's order.
func Choices(ctx context.Context, b ports.Bootstrap) ([]Choice, error) {
	raw, err := b.Show(ctx, "profiles")
	if err != nil {
		return nil, err
	}
	var doc struct {
		Supported []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"supported"`
		Profiles map[string]struct {
			Plan struct {
				Summary string `json:"summary"`
			} `json:"plan"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, &core.Error{Code: core.ExitIntegrity, What: "The Bootstrap profile contract is unreadable.", Why: err.Error()}
	}
	out := make([]Choice, 0, len(doc.Supported))
	for _, s := range doc.Supported {
		out = append(out, Choice{ID: s.ID, Label: s.Label, Summary: doc.Profiles[s.ID].Plan.Summary})
	}
	return out, nil
}
