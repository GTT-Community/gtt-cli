// Package agents keeps the synchronisation state of the project's ADE
// integrations. The project agent registry itself is the Bootstrap's (its
// ADE registry and participation state); this package only records what the
// CLI observed and did, and that record can always be rebuilt.
package agents

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/GTT-Community/gtt-cli/internal/ade"
	"github.com/GTT-Community/gtt-cli/internal/fsx"
	"github.com/GTT-Community/gtt-cli/internal/lifecycle"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Adapter is the AgentAdapter built from one Bootstrap registry entry.
type Adapter struct{ ADE ports.ADE }

// ID returns the agent id.
func (a Adapter) ID() string { return a.ADE.ID }

// Name returns the display name.
func (a Adapter) Name() string { return a.ADE.Name }

// Version returns the version of the registry entry.
func (a Adapter) Version() int { return a.ADE.Version }

// Detect returns the signals the registry entry declares that are present.
func (a Adapter) Detect(projectRoot string, runner ports.Runner) []string {
	return ade.Detect([]ports.ADE{a.ADE}, projectRoot, runner)[0].Signals
}

// ContextLocations returns the project paths the Bootstrap declares as owned.
func (a Adapter) ContextLocations() []string { return a.ADE.OwnedPaths }

// InstructionEntry returns the file the agent reads first.
func (a Adapter) InstructionEntry() string { return a.ADE.Entry }

// Registry is the AgentRegistry over the Bootstrap ADE registry.
type Registry struct{ adapters []ports.AgentAdapter }

// NewRegistry builds one adapter per registry entry, in registry order.
func NewRegistry(entries []ports.ADE) Registry {
	r := Registry{adapters: make([]ports.AgentAdapter, 0, len(entries))}
	for _, e := range entries {
		r.adapters = append(r.adapters, Adapter{ADE: e})
	}
	return r
}

// Adapters returns every adapter.
func (r Registry) Adapters() []ports.AgentAdapter { return r.adapters }

// Adapter returns the adapter of one agent. An unknown id has none: GTT does
// not synchronise an environment it has no integration for.
func (r Registry) Adapter(id string) (ports.AgentAdapter, bool) {
	for _, a := range r.adapters {
		if a.ID() == id {
			return a, true
		}
	}
	return nil, false
}

// States of one managed artifact.
const (
	ArtifactSynchronized = "synchronized"
	ArtifactChanged      = "changed"
	ArtifactMissing      = "missing"
	ArtifactUnexpected   = "unexpected"
)

// Artifact is one file of an agent's managed context, as the Bootstrap
// ownership ledger reports it.
type Artifact struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Basis  string `json:"basis,omitempty"`
}

// ArtifactStatus maps a Bootstrap ownership status to an artifact state. An
// unknown status is passed through rather than guessed.
func ArtifactStatus(ledger string) string {
	switch ledger {
	case "unmodified":
		return ArtifactSynchronized
	case "modified":
		return ArtifactChanged
	case "missing":
		return ArtifactMissing
	case "unrecorded":
		return ArtifactUnexpected
	}
	return ledger
}

// Context states of one agent.
const (
	Synchronized = "synchronized"
	Missing      = "missing or changed"
	NotRequired  = "not required"
)

// Agent is one registry ADE seen from this project and this machine.
type Agent struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Enabled  bool     `json:"enabled"`  // participating: the human chose it
	Primary  bool     `json:"primary"`  // workflow identifier, no authority
	Excluded bool     `json:"excluded"` // the human declined it
	Detected bool     `json:"detected"`
	Signals  []string `json:"signals,omitempty"`
	Context  string   `json:"context"`
	Detail   string   `json:"detail,omitempty"`
	// From the agent's adapter and the Bootstrap ownership ledger.
	AdapterVersion int        `json:"adapter_version,omitempty"`
	Entry          string     `json:"instruction_entry,omitempty"`
	Locations      []string   `json:"context_locations,omitempty"`
	Artifacts      []Artifact `json:"artifacts,omitempty"`
}

// Count returns how many managed artifacts are in the given state.
func (a Agent) Count(status string) int {
	n := 0
	for _, x := range a.Artifacts {
		if x.Status == status {
			n++
		}
	}
	return n
}

// Environment is the result of an agent check.
type Environment struct {
	Schema     int      `json:"schema"`
	Configured bool     `json:"configured"`
	Agents     []Agent  `json:"agents"`
	Undeclared []string `json:"undeclared_detected"`
	Problems   []string `json:"problems"`
}

// OK reports whether every declared agent has its context.
func (e Environment) OK() bool { return len(e.Problems) == 0 }

// Record is the sync state of one agent.
type Record struct {
	Status   string `json:"status"`
	LastSync string `json:"last_sync,omitempty"`
}

// State is .gtt/cli/agents.json.
type State struct {
	Version int               `json:"version"`
	Notice  string            `json:"notice"`
	Agents  map[string]Record `json:"agents"`
}

// Store persists the sync state.
type Store struct{ Root string }

func (s Store) path() string {
	return filepath.Join(s.Root, filepath.FromSlash(lifecycle.Dir), "agents.json")
}

// Load returns the state; absent means empty.
func (s Store) Load() (State, error) {
	st := State{Version: 1, Agents: map[string]Record{}}
	err := fsx.ReadJSON(s.path(), &st)
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if st.Agents == nil {
		st.Agents = map[string]Record{}
	}
	return st, err
}

// Save rewrites the state from an environment. It is idempotent: the same
// environment produces the same file apart from the sync time of agents
// whose status changed.
func (s Store) Save(env Environment, synced map[string]bool) error {
	st, err := s.Load()
	if err != nil {
		return err
	}
	next := map[string]Record{}
	changed := len(st.Agents) == 0
	for _, a := range env.Agents {
		if !a.Enabled {
			continue
		}
		rec := st.Agents[a.ID]
		if rec.Status != a.Context || synced[a.ID] {
			rec.Status = a.Context
			if a.Context == Synchronized {
				rec.LastSync = lifecycle.Timestamp()
			}
			changed = true
		}
		next[a.ID] = rec
	}
	if !changed && len(next) == len(st.Agents) {
		return nil
	}
	st.Version = 1
	st.Notice = "Agent context synchronisation state. Reconstructible from the Bootstrap ADE state; never a source of truth."
	st.Agents = next
	return fsx.WriteJSON(s.path(), st)
}
