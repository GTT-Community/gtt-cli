package lifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogAppendsOneJSONLinePerOperation(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root}
	s.Log(map[string]any{"operation": "init"})
	if _, err := os.Stat(filepath.Join(root, ".gtt")); err == nil {
		t.Fatal("logging must never create .gtt in a project that has no GTT")
	}
	if err := os.Mkdir(filepath.Join(root, ".gtt"), 0o755); err != nil {
		t.Fatal(err)
	}
	s.Log(map[string]any{"operation": "init", "result": "ok"})
	s.Log(map[string]any{"operation": "validate", "result": "failed"})
	data, err := os.ReadFile(s.path("operations.log"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("want two lines, got %q", data)
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["operation"] != "validate" || entry["result"] != "failed" || entry["timestamp"] == nil {
		t.Errorf("entry: %v", entry)
	}
}

func TestRecordAppendsToHistory(t *testing.T) {
	s := Store{Root: t.TempDir()}
	if err := s.Record("update", "1.1.0 -> 1.2.0"); err != nil {
		t.Fatal(err)
	}
	if err := s.Record("snapshot", "before clean"); err != nil {
		t.Fatal(err)
	}
	st, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.History) != 2 || st.History[0].Event != "update" || st.History[1].Detail != "before clean" || st.History[1].At == "" {
		t.Errorf("history: %+v", st.History)
	}
}

func TestJournalLifecycle(t *testing.T) {
	s := Store{Root: t.TempDir()}
	if j, err := s.LoadJournal(); j != nil || err != nil {
		t.Fatalf("no interrupted transaction: %+v, %v", j, err)
	}
	j := &Journal{State: StateInstalling, Completed: []string{"core"},
		Selections: Selections{Participating: []string{"claude"}, Primary: "claude", Profile: "light"}}
	if err := s.SaveJournal(j); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadJournal()
	if err != nil || got == nil {
		t.Fatalf("an interrupted transaction is detected from recorded state: %+v, %v", got, err)
	}
	if got.State != StateInstalling || got.Selections.Primary != "claude" || !got.Done("core") || got.Done("ade") {
		t.Errorf("journal: %+v", got)
	}
	if err := s.ClearJournal(); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearJournal(); err != nil {
		t.Errorf("clearing an absent journal is not an error: %v", err)
	}
	if j, _ := s.LoadJournal(); j != nil {
		t.Error("the journal must be gone once the transaction ends")
	}
}
