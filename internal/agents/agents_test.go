package agents

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

func env(context string) Environment {
	return Environment{Agents: []Agent{
		{ID: "claude", Enabled: true, Context: context},
		{ID: "kiro", Enabled: false, Detected: true, Context: NotRequired},
	}}
}

func TestSaveRecordsOnlyParticipatingAgents(t *testing.T) {
	s := Store{Root: t.TempDir()}
	if st, err := s.Load(); err != nil || len(st.Agents) != 0 {
		t.Fatalf("absent state is empty, not an error: %+v, %v", st, err)
	}
	if err := s.Save(env(Synchronized), nil); err != nil {
		t.Fatal(err)
	}
	st, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Agents) != 1 || st.Agents["claude"].Status != Synchronized || st.Agents["claude"].LastSync == "" {
		t.Errorf("only the participating agent is recorded, with its sync time: %+v", st.Agents)
	}
	if _, ok := st.Agents["kiro"]; ok {
		t.Error("a detected agent that does not participate must not be recorded")
	}
}

func TestSaveIsIdempotent(t *testing.T) {
	s := Store{Root: t.TempDir()}
	if err := s.Save(env(Synchronized), nil); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(s.path())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(s.path(), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(env(Synchronized), nil); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(s.path())
	if string(before) != string(after) {
		t.Errorf("an unchanged environment must not rewrite the state:\n%s\n%s", before, after)
	}
}

func TestSaveFollowsAStatusChange(t *testing.T) {
	s := Store{Root: t.TempDir()}
	if err := s.Save(env(Synchronized), nil); err != nil {
		t.Fatal(err)
	}
	synced, _ := s.Load()
	if err := s.Save(env(Missing), nil); err != nil {
		t.Fatal(err)
	}
	st, _ := s.Load()
	if st.Agents["claude"].Status != Missing || st.Agents["claude"].LastSync != synced.Agents["claude"].LastSync {
		t.Errorf("a lost context is recorded and the last sync time is kept: %+v", st.Agents)
	}
}

func TestEnvironmentOK(t *testing.T) {
	if !(Environment{}).OK() || (Environment{Problems: []string{"missing"}}).OK() {
		t.Error("an environment is OK exactly when it has no problems")
	}
}

type noBinaries struct{}

func (noBinaries) LookPath(string) (string, error) { return "", os.ErrNotExist }
func (noBinaries) Run(context.Context, ports.Command) (ports.Result, error) {
	return ports.Result{}, nil
}

func TestRegistryBuildsOneAdapterPerEntryFromData(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".future"), 0o755); err != nil {
		t.Fatal(err)
	}
	// An ADE the CLI has never heard of: a registry entry is all it takes.
	r := NewRegistry([]ports.ADE{
		{ID: "claude", Name: "Claude Code", DetectPaths: []string{".claude/"}, OwnedPaths: []string{".claude/"}, Entry: ".claude/CLAUDE.md", Version: 1},
		{ID: "future", Name: "Future ADE", DetectPaths: []string{".future/"}, OwnedPaths: []string{".future/"}, Entry: ".future/rules.md", Version: 3},
	})
	if len(r.Adapters()) != 2 {
		t.Fatalf("one adapter per registry entry: %d", len(r.Adapters()))
	}
	a, ok := r.Adapter("future")
	if !ok || a.Name() != "Future ADE" || a.Version() != 3 || a.InstructionEntry() != ".future/rules.md" ||
		!reflect.DeepEqual(a.ContextLocations(), []string{".future/"}) {
		t.Fatalf("adapter: %+v, %v", a, ok)
	}
	if got := a.Detect(root, noBinaries{}); !reflect.DeepEqual(got, []string{"path:.future/"}) {
		t.Errorf("signals: %v", got)
	}
	claude, _ := r.Adapter("claude")
	if got := claude.Detect(root, noBinaries{}); len(got) != 0 {
		t.Errorf("an absent agent has no signals: %v", got)
	}
	if _, ok := r.Adapter("unknown"); ok {
		t.Error("an agent without a registry entry has no adapter")
	}
}

func TestArtifactStatusAndCount(t *testing.T) {
	for ledger, want := range map[string]string{"unmodified": ArtifactSynchronized, "modified": ArtifactChanged,
		"missing": ArtifactMissing, "unrecorded": ArtifactUnexpected, "something-new": "something-new"} {
		if got := ArtifactStatus(ledger); got != want {
			t.Errorf("ArtifactStatus(%q) = %q, want %q", ledger, got, want)
		}
	}
	a := Agent{Artifacts: []Artifact{{Status: ArtifactChanged}, {Status: ArtifactSynchronized}, {Status: ArtifactChanged}}}
	if a.Count(ArtifactChanged) != 2 || a.Count(ArtifactMissing) != 0 {
		t.Errorf("counts: %d changed, %d missing", a.Count(ArtifactChanged), a.Count(ArtifactMissing))
	}
}
