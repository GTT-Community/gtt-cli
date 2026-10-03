package methodology

import (
	"encoding/json"
	"testing"
)

func plan(t *testing.T, policy string) Plan {
	var p Plan
	if err := json.Unmarshal([]byte(`{"profile":"x","selected":true,"plan":{"policy":`+policy+`}}`), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

// The CLI executes the Bootstrap's policy values; it defines none per plan.
func TestDecideFollowsBootstrapValues(t *testing.T) {
	p := plan(t, `{"automation":{"identity_resolution":"automatic","agent_context_sync":"automatic_safe","reference_updates":"confirm","index_rebuild":"team_policy"},
		"human":{"confirmation":{"relevant_change":"not_required","destructive":"not_required","governed_decision":"not_required"}}}`)
	want := map[string]Decision{IdentityResolution: Automatic, AgentContextSync: AutomaticSafe, ReferenceUpdates: Confirm,
		IndexRebuild: Propose, RelevantChange: Automatic}
	for kind, d := range want {
		if got := p.Decide(kind); got != d {
			t.Errorf("%s: got %v want %v", kind, got, d)
		}
	}
}

func TestDestructiveAndGovernedAreAlwaysConfirmed(t *testing.T) {
	p := plan(t, `{"human":{"confirmation":{"destructive":"not_required","governed_decision":"automatic"}}}`)
	if p.Decide(Destructive) != Confirm || p.Decide(GovernedDecision) != Confirm {
		t.Error("no policy value can make a destructive operation or a governed decision automatic")
	}
}

func TestUnknownValueIsNeverGuessed(t *testing.T) {
	p := plan(t, `{"automation":{"validation":"something-new"}}`)
	if p.Decide(Validation) != Confirm || p.Decide("unknown-kind") != Confirm {
		t.Error("an unknown policy value or kind must fall back to asking")
	}
}

const report = `TA-01  [team]  applies to: policy.agent_context_sync  - automatic_safe: sync without asking unless a local file would be overwritten  (gtt-domain/working-agreements.md)
TA-02  [team]  applies to: policy.identity_resolution  - automatic  (gtt-domain/working-agreements.md)
TA-03  [team]  applies to: policy.relevant_change  - whenever possible  (gtt-domain/working-agreements.md)
TA-04  [team]  applies to: code reviews  - automatic formatting  (gtt-domain/working-agreements.md)
UP-01  [user]  applies to: policy.reference_updates  - automatic  (.gtt/local/preferences.md)
TA-05  [team]  applies to: policy.destructive  - automatic  (gtt-domain/working-agreements.md)`

func TestTeamDeclarationsReadOnlyTeamPolicyValues(t *testing.T) {
	got := TeamDeclarations(report)
	want := map[string]string{"agent_context_sync": "automatic_safe", "identity_resolution": "automatic", "destructive": "automatic"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: got %q, want %q", k, got[k], v)
		}
	}
	if len(TeamDeclarations("no working agreements")) != 0 {
		t.Error("no agreements, no declarations")
	}
}

func TestTeamPolicyFollowsTheTeamDeclaration(t *testing.T) {
	p := plan(t, `{"automation":{"identity_resolution":"team_policy","agent_context_sync":"team_policy","reference_updates":"team_policy","validation":"automatic"},
		"human":{"confirmation":{"relevant_change":"team_policy","destructive":"required","governed_decision":"required"}}}`)
	if !p.usesTeamPolicy() {
		t.Fatal("the plan leaves values to the team")
	}
	for _, kind := range []string{IdentityResolution, AgentContextSync, ReferenceUpdates, RelevantChange} {
		if p.Decide(kind) != Propose {
			t.Errorf("%s: with nothing declared, team policy is propose and confirm", kind)
		}
	}
	p.Team = TeamDeclarations(report)
	want := map[string]Decision{IdentityResolution: Automatic, AgentContextSync: AutomaticSafe, ReferenceUpdates: Propose,
		RelevantChange: Propose, Validation: Automatic, Destructive: Confirm, GovernedDecision: Confirm}
	for kind, d := range want {
		if got := p.Decide(kind); got != d {
			t.Errorf("%s: got %v, want %v", kind, got, d)
		}
	}
	if p.Value(AgentContextSync) != "automatic_safe" || p.Value(ReferenceUpdates) != "propose_confirm" {
		t.Errorf("values in force: %q, %q", p.Value(AgentContextSync), p.Value(ReferenceUpdates))
	}
}

func TestADeclarationNeverOverridesANonTeamValue(t *testing.T) {
	p := plan(t, `{"automation":{"agent_context_sync":"confirm"}}`)
	p.Team = map[string]string{AgentContextSync: "automatic"}
	if p.Decide(AgentContextSync) != Confirm {
		t.Error("a team declaration applies only where the Bootstrap leaves the value to team policy")
	}
}
