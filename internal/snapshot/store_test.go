package snapshot

import (
	"os"
	"path/filepath"
	"testing"
)

const document = `{"bootstrap":{"version":"1.2.0"},"ade":{"primary":"claude","participating":["claude","copilot"]},` +
	`"methodology":{"profile":"light","language":"es"}}`

func TestSaveKeepsTheDocumentVerbatimOutsideTheProject(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	s := Store{Home: home}
	info, err := s.Save(project, []byte(document), Meta{CLIVersion: "1.0.0", Reason: "clean"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(info.Snapshot)
	if err != nil || string(got) != document {
		t.Errorf("the Bootstrap document is stored unaltered: %q, %v", got, err)
	}
	if rel, err := filepath.Rel(project, info.Dir); err != nil || !filepath.IsAbs(info.Dir) || rel[:2] != ".." {
		t.Errorf("a snapshot must survive gtt clean: stored at %s, inside project %s", info.Dir, project)
	}
	if info.Meta.Schema != 1 || info.Meta.ProjectRoot != project || info.Meta.Reason != "clean" {
		t.Errorf("metadata: %+v", info.Meta)
	}
	if info.BootstrapVersion != "1.2.0" || info.Primary != "claude" || len(info.Participating) != 2 ||
		info.Profile != "light" || info.Language != "es" {
		t.Errorf("summary read from the document: %+v", info)
	}
}

func TestListIsPerProjectAndNewestFirst(t *testing.T) {
	s := Store{Home: t.TempDir()}
	project, other := t.TempDir(), t.TempDir()
	if got := s.List(project); len(got) != 0 {
		t.Fatalf("no snapshots yet: %v", got)
	}
	first, err := s.Save(project, []byte(document), Meta{Reason: "first"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Save(project, []byte(document), Meta{Reason: "second"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(other, []byte(document), Meta{Reason: "other"}); err != nil {
		t.Fatal(err)
	}
	got := s.List(project)
	if len(got) != 2 || got[0].Dir != second.Dir || got[1].Dir != first.Dir {
		t.Errorf("want the two snapshots of this project, newest first: %+v", got)
	}
}

func TestKeyDistinguishesProjectsWithTheSameName(t *testing.T) {
	a, b := Key("/work/a/app"), Key("/work/b/app")
	if a == b || Key("/work/a/app") != a {
		t.Errorf("keys must be stable and distinct per path: %q, %q", a, b)
	}
}

func TestUnreadableSnapshotIsAnErrorNotAnEmptyOne(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	if _, err := ReadFile(path); err == nil {
		t.Error("a missing snapshot file must be an error")
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFile(path); err == nil {
		t.Error("a malformed snapshot must be an error")
	}
	if err := os.WriteFile(path, []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	if info, err := ReadFile(path); err != nil || info.Primary != "claude" {
		t.Errorf("a portable snapshot file is readable: %+v, %v", info, err)
	}
}
