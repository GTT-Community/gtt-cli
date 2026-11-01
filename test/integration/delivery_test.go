package integration

import (
	"strings"
	"testing"
)

// Delivery Planning: D1-D3 are asked at init from the Bootstrap's own contract,
// recorded as an operating answer, and the plan and the report are computed by
// the Bootstrap and rendered by the CLI. An older Bootstrap is skipped.

const deliveryBacklog = `# Project Backlog

### EPIC-001 Identity
**Status:** Planned
**Goal:** login
**Approved:** ana - 2026-10-30
**Estimate:** 14h + 2 decisions
**Lane:** a
**Touches:** src/auth

##### STORY-001 Login form
- **Status:** Done
- **Estimate:** 1h

### EPIC-002 Billing
**Status:** Planned
**Goal:** pay
**Approved:** ana - 2026-10-30
**Estimate:** 6h
**Lane:** b
**Touches:** src/billing
`

func TestInitAsksDeliveryFromFlagsAndRecordsIt(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, append(defaults, "--builders", "team", "--team", "ana,luis", "--lanes", "3", "--horizon", "days")...)
	got := gtt(t, root, "delivery", "--json").must(t, 0).json(t)
	state := got["state"].(map[string]any)
	if state["who"] != "team" || state["horizon"] != "days" || state["lanes"] != float64(3) || state["deadline"] == nil {
		t.Errorf("the answers are recorded as given, with the deadline computed: %v", state)
	}
	if team := toStrings(state["team"]); strings.Join(team, ",") != "ana,luis" {
		t.Errorf("the team owns the fronts: %v", team)
	}
}

func TestWithoutAnswerDeliveryStaysOpenAndNothingIsInvented(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	if exists(root, ".gtt/delivery.json") {
		t.Error("no answer, nothing recorded")
	}
	if got := gtt(t, root, "delivery", "--json").must(t, 0).json(t); got["declared"] != false {
		t.Errorf("not declared: %v", got)
	}
}

func TestDeliverySetWorksAtAnyTimeAndRefusesNonsense(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	gtt(t, root, "delivery", "set", "--who", "solo-multi", "--lanes", "2", "--horizon", "weeks").must(t, 0)
	state := gtt(t, root, "delivery", "--json").must(t, 0).json(t)["state"].(map[string]any)
	if state["who"] != "solo-multi" || state["horizon"] != "weeks" {
		t.Errorf("recorded: %v", state)
	}
	gtt(t, root, "delivery", "set", "--who", "robots").must(t, 5)
	gtt(t, root, "delivery", "set").must(t, 3)
}

func TestPlanAndReportAreComputedByTheBootstrap(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, append(defaults, "--builders", "solo-multi", "--lanes", "2", "--horizon", "days")...)
	write(t, root, "gtt-domain/backlog.md", deliveryBacklog)
	report := gtt(t, root, "report", "--json").must(t, 0).json(t)
	if report["uncalibrated"] != true {
		t.Errorf("fewer than three Stories measured must say uncalibrated: %v", report["calibration"])
	}
	epics := report["epics"].([]any)
	if len(epics) != 2 || epics[0].(map[string]any)["epic"] != "EPIC-001" {
		t.Errorf("percent per Epic: %v", epics)
	}
	if out := gtt(t, root, "plan").must(t, 0).out; !strings.Contains(out, "front a") || !strings.Contains(out, "UNCALIBRATED") {
		t.Errorf("the plan shows the fronts and says it is uncalibrated:\n%s", out)
	}
}

func TestPlanLanesPrintsOneSummaryPerFrontAndCreatesNothing(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, append(defaults, "--builders", "solo-multi", "--lanes", "2", "--horizon", "days")...)
	write(t, root, "gtt-domain/backlog.md", deliveryBacklog)
	before := treeHash(t, root)
	out := gtt(t, root, "plan", "--lanes").must(t, 0).out
	if strings.Count(out, "git worktree add") != 2 || !strings.Contains(out, "front a") || !strings.Contains(out, "front b") {
		t.Errorf("one summary and one worktree command per front:\n%s", out)
	}
	if treeHash(t, root) != before {
		t.Error("`plan --lanes` created something")
	}
}

func TestACLIAheadOfTheBootstrapSkipsDelivery(t *testing.T) {
	catalog := catalogCopy(t)
	edit(t, catalog+"/.gtt/contract/operations.json", func(doc map[string]any) {
		ops := doc["operations"].(map[string]any)
		for name := range ops {
			if strings.HasPrefix(name, "delivery.") {
				delete(ops, name)
			}
		}
	})
	root := project(t, nil)
	r := initProject(t, root, catalog, defaults...)
	if !strings.Contains(r.out+r.err, "no delivery planning") {
		t.Errorf("init says in one line that this Bootstrap has no delivery planning:\n%s", r.out+r.err)
	}
	gtt(t, root, "plan").must(t, 2)
	gtt(t, root, "status").must(t, 0)
}
