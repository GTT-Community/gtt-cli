package snapshot

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func put(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSaveDomainZipsTheDomainOutsideTheProject(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	put(t, root, "gtt-domain/backlog.md", "# backlog\n")
	put(t, root, "gtt-domain/context/stack.md", "# stack\n")
	put(t, root, "docs/ideas/IDEA-001.md", "idea\n")
	put(t, root, "src/main.go", "package main\n")
	if err := os.Symlink("/etc/passwd", filepath.Join(root, "gtt-domain", "link")); err != nil {
		t.Log("no symlinks here:", err)
	}
	out, err := Store{Home: home}.SaveDomain(root)
	if err != nil || out == "" {
		t.Fatalf("got %q %v", out, err)
	}
	if rel, _ := filepath.Rel(root, out); !filepath.IsAbs(out) || rel == filepath.Base(out) {
		t.Errorf("the zip must live outside the project: %s", out)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	got := map[string]bool{}
	for _, f := range zr.File {
		got[f.Name] = true
	}
	for _, want := range []string{"gtt-domain/backlog.md", "gtt-domain/context/stack.md", "docs/ideas/IDEA-001.md"} {
		if !got[want] {
			t.Errorf("missing %s in %v", want, got)
		}
	}
	if got["src/main.go"] || got["gtt-domain/link"] {
		t.Errorf("only the domain and the ideas, never a link: %v", got)
	}
}

func TestSaveDomainWithNothingToKeepWritesNothing(t *testing.T) {
	out, err := Store{Home: t.TempDir()}.SaveDomain(t.TempDir())
	if out != "" || err != nil {
		t.Fatalf("got %q %v", out, err)
	}
}

func TestSaveDomainKeepsTheLastFew(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	put(t, root, "gtt-domain/backlog.md", "x\n")
	s := Store{Home: home}
	dir := filepath.Join(home, "backups", Key(root))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"20200101T000000Z", "20200102T000000Z", "20200103T000000Z", "20200104T000000Z", "20200105T000000Z", "20200106T000000Z"} {
		os.WriteFile(filepath.Join(dir, n+"-gtt-domain.zip"), []byte("z"), 0o644)
	}
	if _, err := s.SaveDomain(root); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != keepDomainBackups {
		t.Errorf("kept %d, want %d", len(entries), keepDomainBackups)
	}
}

func TestSaveDomainInTheSameSecondNeverReplacesABackup(t *testing.T) {
	root, instant := t.TempDir(), time.Date(2026, 1, 2, 3, 4, 59, 0, time.UTC)
	s := Store{Home: t.TempDir(), clock: func() time.Time { return instant }}
	bodies := []string{"first\n", "second\n", "third\n"}
	var zips []string
	for _, body := range bodies {
		put(t, root, "gtt-domain/backlog.md", body)
		out, err := s.SaveDomain(root)
		if err != nil {
			t.Fatal(err)
		}
		zips = append(zips, out)
	}
	for i, path := range zips {
		if got := zipped(t, path, "gtt-domain/backlog.md"); got != bodies[i] {
			t.Errorf("backup %d holds %q, want %q", i, got, bodies[i])
		}
		if i > 0 && filepath.Base(path) <= filepath.Base(zips[i-1]) {
			t.Errorf("a later backup must sort later, pruning relies on it: %s after %s", path, zips[i-1])
		}
	}
}

func zipped(t *testing.T, path, name string) string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	f, err := zr.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	body, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
