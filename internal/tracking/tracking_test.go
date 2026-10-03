package tracking

import (
	"os"
	"path/filepath"
	"testing"
)

func setup(t *testing.T) (string, *Ledger) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "INITIAL-001.md"), []byte("v1"), 0o644)
	l, err := Store{Root: root}.Load()
	if err != nil {
		t.Fatal(err)
	}
	return root, l
}

func TestReadAppliedValidatedOrder(t *testing.T) {
	root, l := setup(t)
	e, err := l.Observe(root, "INITIAL-001.md")
	if err != nil || e.State != Discovered {
		t.Fatalf("%v %v", e, err)
	}
	if e.MarkApplied() == nil {
		t.Error("APPLIED before READ must be refused")
	}
	if e.MarkValidated() == nil {
		t.Error("VALIDATED before APPLIED must be refused")
	}
	e.MarkRead()
	if e.State != Read || e.ReadAt == "" {
		t.Error("READ")
	}
	if err := e.MarkApplied(); err != nil || e.State != Applied {
		t.Error("APPLIED")
	}
	if e.Done() {
		t.Error("applied is not done until validated")
	}
	if err := e.MarkValidated(); err != nil || !e.Done() || e.ValidatedAt == "" {
		t.Error("VALIDATED")
	}
}

func TestFailureIsNeverApplied(t *testing.T) {
	root, l := setup(t)
	e, _ := l.Observe(root, "INITIAL-001.md")
	e.MarkRead()
	e.MarkFailed("boom")
	if e.State != Failed || e.AppliedAt != "" || e.Done() {
		t.Errorf("a failed document must not look applied: %+v", e)
	}
	if e.MarkValidated() == nil {
		t.Error("a failed document cannot be validated")
	}
}

func TestUnchangedIsNotReprocessedAndChangeIsDetected(t *testing.T) {
	root, l := setup(t)
	e, _ := l.Observe(root, "INITIAL-001.md")
	e.MarkRead()
	e.MarkApplied()
	e.MarkValidated()
	if err := (Store{Root: root}).Save(l); err != nil {
		t.Fatal(err)
	}
	l2, _ := Store{Root: root}.Load()
	again, _ := l2.Observe(root, "INITIAL-001.md")
	if !again.Done() {
		t.Fatal("same content must stay VALIDATED: no reprocessing")
	}
	// Same name, different content: the name alone does not identify a version.
	os.WriteFile(filepath.Join(root, "INITIAL-001.md"), []byte("v2"), 0o644)
	changed, _ := l2.Observe(root, "INITIAL-001.md")
	if changed.Done() || changed.State != Discovered || !changed.Changed {
		t.Fatalf("changed content must be pending again: %+v", changed)
	}
	if len(changed.Previous) != 1 || changed.Previous[0].State != Validated {
		t.Error("the processed version must be kept as history")
	}
}

func TestDeclaredRecordTakesOverFromTheLegacyLocation(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "INITIAL-001.md"), []byte("# directive\n"), 0o644)
	legacy := Store{Root: root}
	l, _ := legacy.Load()
	if _, err := l.Observe(root, "INITIAL-001.md"); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Save(l); err != nil {
		t.Fatal(err)
	}
	declared := Store{Root: root, Path: ".gtt/tracking/initial-sources.json"}
	got, err := declared.Load()
	if err != nil || got.Entries["INITIAL-001.md"] == nil {
		t.Fatalf("entries recorded before the Bootstrap declared a record are kept: %+v %v", got, err)
	}
	if err := declared.Save(got); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".gtt", "tracking", "initial-sources.json")); err != nil {
		t.Errorf("the next write goes to the declared record: %v", err)
	}
	// Once the declared record holds entries, it is the only one read.
	got.Entries["OTHER.md"] = &Entry{Path: "OTHER.md", State: Discovered}
	declared.Save(got)
	again, _ := declared.Load()
	if again.Entries["OTHER.md"] == nil {
		t.Error("the declared record is authoritative once it holds entries")
	}
}
