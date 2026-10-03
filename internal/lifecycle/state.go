// Package lifecycle keeps the CLI's own operational state: the resolution
// record, the init transaction journal and the operation log. All of it is
// derived and operational - never an authority - and lives under .gtt/cli/.
package lifecycle

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/fsx"
)

// Dir is the CLI state directory inside a project.
const Dir = ".gtt/cli"

// JournalFile records an init transaction in progress. It lives at the
// project root because .gtt/ may not exist yet.
const JournalFile = ".gtt-init-transaction.json"

// Resolution is the record of the Bootstrap a project was installed from.
type Resolution struct {
	ID            string `json:"id"`
	Version       string `json:"version"`
	SchemaVersion int    `json:"schema_version"`
	Channel       string `json:"channel"`
	Source        string `json:"source"`
	Origin        string `json:"origin"`
	Checksum      string `json:"checksum"`
	Integrity     string `json:"integrity"`
	Signature     string `json:"signature"`
	ResolvedAt    string `json:"resolved_at"`
}

// Event is one entry of the CLI history (updates, clean exports, snapshots).
type Event struct {
	At     string `json:"at"`
	Event  string `json:"event"`
	Detail string `json:"detail,omitempty"`
}

// State is .gtt/cli/state.json.
type State struct {
	Schema     int               `json:"schema"`
	Notice     string            `json:"notice"`
	CLI        map[string]string `json:"cli"`
	Bootstrap  *Resolution       `json:"bootstrap,omitempty"`
	CoreLedger map[string]string `json:"core_ledger,omitempty"`
	History    []Event           `json:"history,omitempty"`
}

const notice = "GTT CLI operational state: derived, reconstructible, never an authority. Written only by the gtt CLI."

// Store reads and writes CLI state for one project.
type Store struct{ Root string }

func (s Store) path(name string) string { return filepath.Join(s.Root, filepath.FromSlash(Dir), name) }

// Load returns the state; a project without one yields an empty state.
func (s Store) Load() (State, error) {
	st := State{Schema: 1}
	err := fsx.ReadJSON(s.path("state.json"), &st)
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	return st, err
}

// Save writes the state.
func (s Store) Save(st State) error {
	st.Schema, st.Notice = 1, notice
	st.CLI = map[string]string{"version": core.Version}
	return fsx.WriteJSON(s.path("state.json"), st)
}

// Record appends a history event and saves.
func (s Store) Record(event, detail string) error {
	st, err := s.Load()
	if err != nil {
		return err
	}
	st.History = append(st.History, Event{At: Timestamp(), Event: event, Detail: detail})
	return s.Save(st)
}

// Log appends one line to the operation log. Values must never be secrets;
// callers pass operation names and results only.
func (s Store) Log(fields map[string]any) {
	if !fsx.Exists(filepath.Join(s.Root, ".gtt")) {
		return
	}
	fields["timestamp"] = Timestamp()
	line, err := json.Marshal(fields)
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(s.path("operations.log")), 0o755) != nil {
		return
	}
	f, err := os.OpenFile(s.path("operations.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(line, '\n'))
}

// Timestamp is the UTC time in RFC 3339.
func Timestamp() string { return time.Now().UTC().Format(time.RFC3339) }

// ------------------------------------------------------------- transaction

// Init transaction states (operational only).
const (
	StateConfiguring = "CONFIGURING"
	StateStaging     = "STAGING"
	StateInstalling  = "INSTALLING"
	StateValidating  = "VALIDATING"
	StateInitialized = "INITIALIZED"
)

// Selections are the human's choices collected by init.
type Selections struct {
	Detected      []string `json:"detected"`
	Participating []string `json:"participating"`
	Primary       string   `json:"primary"`
	Language      string   `json:"language"`
	Profile       string   `json:"profile"`
	Sources       []Source `json:"sources"`
	Questionnaire bool     `json:"questionnaire"`
}

// Source is one selected initial source.
type Source struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// Journal is the recorded state of an init transaction.
type Journal struct {
	Schema     int               `json:"schema"`
	State      string            `json:"state"`
	StartedAt  string            `json:"started_at"`
	Bootstrap  Resolution        `json:"bootstrap"`
	Selections Selections        `json:"selections"`
	CorePaths  []string          `json:"core_paths"`
	CoreLedger map[string]string `json:"core_ledger,omitempty"`
	Completed  []string          `json:"completed"`
}

// Done reports whether a step was completed.
func (j Journal) Done(step string) bool {
	for _, s := range j.Completed {
		if s == step {
			return true
		}
	}
	return false
}

// LoadJournal returns the journal of an interrupted transaction, if any.
func (s Store) LoadJournal() (*Journal, error) {
	var j Journal
	err := fsx.ReadJSON(filepath.Join(s.Root, JournalFile), &j)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// SaveJournal persists the journal.
func (s Store) SaveJournal(j *Journal) error {
	j.Schema = 1
	return fsx.WriteJSON(filepath.Join(s.Root, JournalFile), j)
}

// ClearJournal removes the journal once the transaction ends.
func (s Store) ClearJournal() error {
	err := os.Remove(filepath.Join(s.Root, JournalFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
