// Package tracking records which initial Markdown sources the CLI has read,
// applied and validated. Identity is by content hash, never by file name.
// The record is derived operational state under .gtt/cli/: it can be rebuilt
// and is never a source of truth about the project.
package tracking

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/GTT-Community/gtt-cli/internal/fsx"
	"github.com/GTT-Community/gtt-cli/internal/lifecycle"
)

// States of an initial MD.
const (
	Discovered = "DISCOVERED"
	Read       = "READ"
	Applied    = "APPLIED"
	Validated  = "VALIDATED"
	Failed     = "FAILED"
)

// Version is one processed content of a document.
type Version struct {
	SHA256      string `json:"sha256"`
	State       string `json:"state"`
	ValidatedAt string `json:"validated_at,omitempty"`
}

// Entry is the tracking record of one document.
type Entry struct {
	Path         string    `json:"path"`
	SHA256       string    `json:"sha256"`
	State        string    `json:"state"`
	DiscoveredAt string    `json:"discovered_at"`
	ReadAt       string    `json:"read_at,omitempty"`
	AppliedAt    string    `json:"applied_at,omitempty"`
	ValidatedAt  string    `json:"validated_at,omitempty"`
	Error        string    `json:"error,omitempty"`
	Changed      bool      `json:"changed,omitempty"` // content differs from the last processed version
	Previous     []Version `json:"previous,omitempty"`
}

// Done reports whether this exact content was applied and validated.
func (e Entry) Done() bool { return e.State == Validated }

// Ledger is .gtt/cli/tracking.json.
type Ledger struct {
	Schema  int               `json:"schema"`
	Notice  string            `json:"notice"`
	Entries map[string]*Entry `json:"entries"`
}

const notice = "Initial MD tracking: which content the gtt CLI read, applied and validated. Derived state, never an authority."

// Store persists the ledger of one project.
type Store struct {
	Root string
	// Path is the record the Bootstrap declares, relative to Root. Empty
	// means the CLI's own location under .gtt/cli/, which is also read as a
	// fallback until the declared record holds entries.
	Path string
}

// legacy is the CLI's own location, used before the Bootstrap declared one.
func (s Store) legacy() string {
	return filepath.Join(s.Root, filepath.FromSlash(lifecycle.Dir), "tracking.json")
}

func (s Store) path() string {
	if s.Path == "" {
		return s.legacy()
	}
	return filepath.Join(s.Root, filepath.FromSlash(s.Path))
}

// Load returns the ledger; a project without one yields an empty ledger.
// Entries recorded at the legacy location are carried over to a declared
// record that has none yet, and saved there on the next write.
func (s Store) Load() (*Ledger, error) {
	l := &Ledger{Schema: 1, Entries: map[string]*Entry{}}
	err := fsx.ReadJSON(s.path(), l)
	if s.Path != "" && (errors.Is(err, os.ErrNotExist) || (err == nil && len(l.Entries) == 0)) {
		legacy := &Ledger{}
		if lerr := fsx.ReadJSON(s.legacy(), legacy); lerr == nil && len(legacy.Entries) > 0 {
			l, err = legacy, nil
		}
	}
	if errors.Is(err, os.ErrNotExist) {
		return l, nil
	}
	if l.Entries == nil {
		l.Entries = map[string]*Entry{}
	}
	return l, err
}

// Save writes the ledger.
func (s Store) Save(l *Ledger) error {
	l.Schema, l.Notice = 1, notice
	return fsx.WriteJSON(s.path(), l)
}

// Observe registers the current content of a document. A new document is
// DISCOVERED. A document whose content changed since it was processed goes
// back to DISCOVERED, keeping the processed version as history. Unchanged
// content keeps its state.
func (l *Ledger) Observe(root, rel string) (*Entry, error) {
	sum, err := fsx.HashFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, err
	}
	now := lifecycle.Timestamp()
	e, ok := l.Entries[rel]
	if !ok {
		e = &Entry{Path: rel, SHA256: sum, State: Discovered, DiscoveredAt: now}
		l.Entries[rel] = e
		return e, nil
	}
	if e.SHA256 != sum {
		e.Previous = append(e.Previous, Version{SHA256: e.SHA256, State: e.State, ValidatedAt: e.ValidatedAt})
		*e = Entry{Path: rel, SHA256: sum, State: Discovered, DiscoveredAt: now, Changed: true, Previous: e.Previous}
	}
	return e, nil
}

// MarkRead records that the content was read.
func (e *Entry) MarkRead() {
	e.State, e.ReadAt, e.Error = Read, lifecycle.Timestamp(), ""
}

// MarkApplied records a successful application. It is refused unless the
// content was read first.
func (e *Entry) MarkApplied() error {
	if e.State != Read {
		return fmt.Errorf("tracking: %s cannot be APPLIED from %s", e.Path, e.State)
	}
	e.State, e.AppliedAt = Applied, lifecycle.Timestamp()
	return nil
}

// MarkValidated records a successful validation. It is refused unless the
// content was applied: READ + APPLIED is recorded only after both happened.
func (e *Entry) MarkValidated() error {
	if e.State != Applied {
		return fmt.Errorf("tracking: %s cannot be VALIDATED from %s", e.Path, e.State)
	}
	e.State, e.ValidatedAt, e.Changed = Validated, lifecycle.Timestamp(), false
	return nil
}

// MarkFailed records a failure. A failed document is never APPLIED.
func (e *Entry) MarkFailed(reason string) {
	e.State, e.Error, e.AppliedAt, e.ValidatedAt = Failed, reason, "", ""
}

// Sorted returns the entries ordered by path.
func (l *Ledger) Sorted() []*Entry {
	out := make([]*Entry, 0, len(l.Entries))
	for _, e := range l.Entries {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
