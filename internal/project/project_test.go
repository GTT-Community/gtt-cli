package project

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type fakeGit struct{ root string }

func (g fakeGit) Available() bool { return g.root != "" }
func (g fakeGit) Root(context.Context, string) (string, bool) {
	return g.root, g.root != ""
}

func TestDetectPrefersGTTThenGitThenAsks(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	os.MkdirAll(sub, 0o755)
	none := func(string) bool { return false }
	if d := Detect(context.Background(), fakeGit{}, none, sub); d.Confident || d.Root != sub {
		t.Errorf("without markers the directory is not assumed to be a project: %+v", d)
	}
	if d := Detect(context.Background(), fakeGit{root: root}, none, sub); !d.Confident || d.Root != root || d.Basis != "git" {
		t.Errorf("git root: %+v", d)
	}
	installed := func(dir string) bool { return dir == filepath.Join(root, "a") }
	if d := Detect(context.Background(), fakeGit{root: root}, installed, sub); d.Root != filepath.Join(root, "a") || d.Basis != "gtt" {
		t.Errorf("nearest GTT project wins: %+v", d)
	}
}

func TestDocumentsAreCandidatesOnly(t *testing.T) {
	root := t.TempDir()
	for _, f := range []string{"README.md", "LICENSE", "design.pdf", "AGENTS.md", "notes.go", ".hidden.md", "docs/requirements.md", "docs/deep/x.md", "src/a.md"} {
		p := filepath.Join(root, f)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("x"), 0o644)
	}
	got := Documents(root, func(rel string) bool { return rel == "AGENTS.md" })
	want := []string{"README.md", "design.pdf", "docs/requirements.md"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}
