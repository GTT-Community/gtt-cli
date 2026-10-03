package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// fakeBootstrap answers declared operations from a table and records every
// request, so a test can tell what the use case asked for and in what order.
type fakeBootstrap struct {
	root       string
	compatible bool
	results    map[string]ports.OperationResult // by operation, or "op:apply" for the applied run
	shows      map[string]string
	requests   []ports.OperationRequest
}

func (f *fakeBootstrap) Root() string { return f.root }
func (f *fakeBootstrap) Release(context.Context) (ports.Release, error) {
	var r ports.Release
	r.Bootstrap.ID, r.Bootstrap.Version = core.BootstrapID, "1.2.0"
	return r, nil
}
func (f *fakeBootstrap) Negotiate(context.Context) (ports.Negotiation, error) {
	if f.compatible {
		return ports.Negotiation{Compatible: true}, nil
	}
	return ports.Negotiation{Reasons: []string{"requires CLI 9.0.0"}}, nil
}
func (f *fakeBootstrap) Capabilities(context.Context) ([]ports.Capability, error) { return nil, nil }
func (f *fakeBootstrap) Operations(context.Context) (map[string]ports.Operation, error) {
	return nil, nil
}
func (f *fakeBootstrap) Show(_ context.Context, contract string) (json.RawMessage, error) {
	return json.RawMessage(f.shows[contract]), nil
}
func (f *fakeBootstrap) CheckContracts(context.Context) ([]string, error) { return nil, nil }
func (f *fakeBootstrap) Execute(_ context.Context, req ports.OperationRequest) (ports.OperationResult, error) {
	f.requests = append(f.requests, req)
	key := req.Operation
	if req.Apply {
		if r, ok := f.results[key+":apply"]; ok {
			return r, nil
		}
	}
	r, ok := f.results[key]
	if !ok {
		return ports.OperationResult{}, errors.New("unexpected operation " + key)
	}
	return r, nil
}

type fakeFactory struct{ b *fakeBootstrap }

func (f fakeFactory) At(string) ports.Bootstrap  { return f.b }
func (f fakeFactory) Installed(root string) bool { return f.b != nil && root == f.b.root }

type noGit struct{}

func (noGit) Available() bool                             { return false }
func (noGit) Root(context.Context, string) (string, bool) { return "", false }

type silent struct{}

func (silent) Info(string, ...any) {}
func (silent) Detail(string)       {}

type prompter struct {
	interactive bool
	answer      bool
	asked       []string
}

func (p *prompter) Interactive() bool { return p.interactive }
func (p *prompter) Confirm(q string, _ bool) (bool, error) {
	p.asked = append(p.asked, q)
	return p.answer, nil
}
func (p *prompter) Select(string, []ports.Option) (int, error)        { return 0, nil }
func (p *prompter) MultiSelect(string, []ports.Option) ([]int, error) { return nil, nil }

func ok(stdout string) ports.OperationResult { return ports.OperationResult{Stdout: stdout} }

func newApp(t *testing.T, b *fakeBootstrap) (*App, *prompter) {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".gtt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if b != nil {
		b.root = root
	}
	pr := &prompter{}
	return &App{Factory: fakeFactory{b}, Git: noGit{}, Prompt: pr, Report: silent{}, Cwd: root, Home: t.TempDir()}, pr
}

func TestOpenRefusesAProjectWithoutGTT(t *testing.T) {
	a, _ := newApp(t, nil)
	if _, err := a.open(context.Background()); core.CodeOf(err) != core.ExitConflict {
		t.Errorf("got %v, want a project state conflict", err)
	}
}

func TestOpenRefusesAnIncompatibleBootstrapBeforeAnyOperation(t *testing.T) {
	b := &fakeBootstrap{}
	a, _ := newApp(t, b)
	_, err := a.open(context.Background())
	if core.CodeOf(err) != core.ExitIncompatible || !strings.Contains(err.Error(), "requires CLI 9.0.0") {
		t.Errorf("got %v, want an incompatibility with its reason", err)
	}
	if len(b.requests) != 0 {
		t.Errorf("no operation may run against an incompatible Bootstrap: %v", b.requests)
	}
}

func TestConfirm(t *testing.T) {
	a, pr := newApp(t, nil)
	if _, err := a.confirm("Remove GTT?", false); core.CodeOf(err) != core.ExitUsage {
		t.Errorf("without a terminal and without --yes the operation is refused, not assumed: %v", err)
	}
	pr.interactive, pr.answer = true, true
	if yes, err := a.confirm("Remove GTT?", false); !yes || err != nil || len(pr.asked) != 1 {
		t.Errorf("a terminal asks the human: %v %v %v", yes, err, pr.asked)
	}
	a.AssumeYes = true
	if yes, err := a.confirm("Remove GTT?", false); !yes || err != nil || len(pr.asked) != 1 {
		t.Errorf("--yes pre-answers without asking: %v %v %v", yes, err, pr.asked)
	}
}

func TestMutateStopsWhenTheDryRunReportsAProblem(t *testing.T) {
	b := &fakeBootstrap{compatible: true, results: map[string]ports.OperationResult{
		"ade.install": {ExitCode: 1, Stdout: "conflict: settings.json exists"}}}
	a, _ := newApp(t, b)
	_, err := a.mutate(context.Background(), b, "ade.install", nil)
	if core.CodeOf(err) != core.ExitConflict || !strings.Contains(err.Error(), "settings.json") {
		t.Errorf("got %v", err)
	}
	if len(b.requests) != 1 || b.requests[0].Apply {
		t.Errorf("only the dry run may have run: %+v", b.requests)
	}
}

func TestMutateRunsTheDryRunThenApplies(t *testing.T) {
	b := &fakeBootstrap{compatible: true, results: map[string]ports.OperationResult{
		"index": ok("would rebuild"), "index:apply": ok("rebuilt")}}
	a, _ := newApp(t, b)
	res, err := a.mutate(context.Background(), b, "index", map[string]string{"x": "1"})
	if err != nil || res.Stdout != "rebuilt" {
		t.Fatalf("%v %v", res, err)
	}
	if len(b.requests) != 2 || b.requests[0].Apply || !b.requests[1].Apply || b.requests[1].Args["x"] != "1" {
		t.Errorf("dry run, then apply with the same arguments: %+v", b.requests)
	}
	b.results["index:apply"] = ports.OperationResult{ExitCode: 1, Stderr: "disk full"}
	if _, err := a.mutate(context.Background(), b, "index", nil); core.CodeOf(err) != core.ExitFailure {
		t.Errorf("a failed apply is an operational failure: %v", err)
	}
}

func TestQueryReportsAnUnreadableResult(t *testing.T) {
	b := &fakeBootstrap{compatible: true, results: map[string]ports.OperationResult{"status": ok("{not json")}}
	a, _ := newApp(t, b)
	var v map[string]any
	if _, err := a.query(context.Background(), b, "status", nil, &v); err == nil || !strings.Contains(err.Error(), "unreadable") {
		t.Errorf("got %v", err)
	}
}

// The registry and ledger of a project where Claude Code and Copilot
// participate and Kiro is only detected.
const registry = `{"ades":[
 {"id":"claude","name":"Claude Code","version":1,"owned_paths":[".claude/"],"handoff":{"instruction_entry":".claude/CLAUDE.md"},"detect":{"paths":[".claude/"]}},
 {"id":"copilot","name":"GitHub Copilot","version":2,"owned_paths":[".copilot/copilot-instructions.md"],"handoff":{"instruction_entry":".copilot/copilot-instructions.md"}},
 {"id":"kiro","name":"Kiro","version":1,"owned_paths":[".kiro/"]}]}`

func agentsBootstrap(copilotLevel string) *fakeBootstrap {
	return &fakeBootstrap{compatible: true, shows: map[string]string{"ade-registry": registry}, results: map[string]ports.OperationResult{
		"ade.state": ok(`{"configured":true,"participating":["claude","copilot"],"primary":"claude","excluded":[],
			"integrations":[{"ade":"claude","level":"PASS","message":"ok"},{"ade":"copilot","level":"` + copilotLevel + `","message":"overlay file missing"}]}`),
		"ade.detect": ok(`{"candidates":[{"id":"claude","name":"Claude Code","signals":["path:.claude/"]},
			{"id":"copilot","name":"GitHub Copilot","signals":[]},{"id":"kiro","name":"Kiro","signals":["binary:kiro"]}]}`),
		"ade.owned": ok(`{"surfaces":[{"ade":"claude","path":".claude/CLAUDE.md","status":"unmodified","basis":"ledger"},
			{"ade":"claude","path":".claude/rules/a.md","status":"modified","basis":"ledger"},
			{"ade":"copilot","path":".copilot/copilot-instructions.md","status":"missing","basis":"ledger"}]}`),
	}}
}

func TestAgentsCheckComparesDeclaredDetectedAndContext(t *testing.T) {
	a, _ := newApp(t, agentsBootstrap("FAIL"))
	env, err := a.AgentsCheck(context.Background())
	if core.CodeOf(err) != core.ExitFailure || len(env.Problems) != 1 || !strings.Contains(env.Problems[0], "GitHub Copilot") {
		t.Fatalf("a declared agent without its context fails the check: %v %v", err, env.Problems)
	}
	if len(env.Undeclared) != 1 || env.Undeclared[0] != "kiro" {
		t.Errorf("detected but not declared: %v", env.Undeclared)
	}
	byID := map[string]int{}
	for i, ag := range env.Agents {
		byID[ag.ID] = i
	}
	claude, copilot := env.Agents[byID["claude"]], env.Agents[byID["copilot"]]
	if !claude.Primary || claude.Context != "synchronized" || claude.Entry != ".claude/CLAUDE.md" || claude.Count("changed") != 1 {
		t.Errorf("claude: %+v", claude)
	}
	if copilot.AdapterVersion != 2 || copilot.Context != "missing or changed" || copilot.Count("missing") != 1 {
		t.Errorf("copilot: %+v", copilot)
	}
	if kiro := env.Agents[byID["kiro"]]; kiro.Enabled || kiro.Context != "not required" || len(kiro.Artifacts) != 0 {
		t.Errorf("kiro participates in nothing: %+v", kiro)
	}

	b := agentsBootstrap("PASS")
	a, _ = newApp(t, b)
	if _, err := a.AgentsCheck(context.Background()); err != nil {
		t.Errorf("consistent context: %v", err)
	}
	for _, r := range b.requests {
		if r.Apply {
			t.Errorf("check is read-only, yet %s was applied", r.Operation)
		}
	}
}

func TestArtifactNextIDIsRecordedAndFollowsTheTeam(t *testing.T) {
	b := &fakeBootstrap{compatible: true, results: map[string]ports.OperationResult{
		"artifact.next-id":        ok(`{"id":"ADR-003","resolution":"propose"}`),
		"methodology.profile.get": ok(`{"profile":"team","selected":true,"plan":{"policy":{"automation":{"identity_resolution":"team_policy"}}}}`),
		"query.governance":        ok("TA-01  [team]  applies to: policy.identity_resolution  - automatic  (gtt-domain/working-agreements.md)\n"),
	}}
	a, _ := newApp(t, b)
	res, err := a.ArtifactNextID(context.Background(), "adr", "ADR-001")
	if err != nil || res.ID != "ADR-003" || !res.Automatic {
		t.Fatalf("the team declared automatic id resolution: %+v %v", res, err)
	}
	log, err := os.ReadFile(filepath.Join(a.Cwd, ".gtt", "cli", "operations.log"))
	if err != nil || !strings.Contains(string(log), `"requested":"ADR-001"`) || !strings.Contains(string(log), `"id":"ADR-003"`) {
		t.Errorf("the allocation is recorded: %s %v", log, err)
	}
}
