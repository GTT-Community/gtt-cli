package export

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatches(t *testing.T) {
	cases := []struct {
		rel, pattern string
		want         bool
	}{
		{".gtt/scripts/x.sh", ".gtt/", true},
		{".gtt", ".gtt/", true},
		{".gttx/file", ".gtt/", false},
		{"AGENTS.md", "AGENTS.md", true},
		{"docs/AGENTS.md", "AGENTS.md", false},
		{"SOURCE-BRIEF.md", "SOURCE-BRIEF.*", true},
		{"docs/SOURCE-BRIEF.md", "SOURCE-BRIEF.*", false},
		{".claude/CLAUDE.md", ".claude/CLAUDE.md", true},
		{".claude/mine.md", ".claude/CLAUDE.md", false},
	}
	for _, c := range cases {
		if got := Matches(c.rel, c.pattern); got != c.want {
			t.Errorf("Matches(%q, %q) = %v", c.rel, c.pattern, got)
		}
	}
}

func TestCopyLeavesSourceIntactAndSkipsItself(t *testing.T) {
	src := t.TempDir()
	write := func(rel string) {
		p := filepath.Join(src, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(rel), 0o644)
	}
	for _, f := range []string{"src/main.go", "tests/a_test.go", ".gtt/x", "gtt-domain/context/stack.md", "AGENTS.md", ".claude/CLAUDE.md", ".claude/mine.md", ".git/HEAD"} {
		write(f)
	}
	dst := filepath.Join(src, "dist", "out")
	res, err := Copy(src, dst, []string{".gtt/", "gtt-domain/", "AGENTS.md", ".claude/CLAUDE.md"})
	if err != nil {
		t.Fatal(err)
	}
	for _, kept := range []string{"src/main.go", "tests/a_test.go", ".claude/mine.md"} {
		if _, err := os.Stat(filepath.Join(dst, kept)); err != nil {
			t.Errorf("%s must be delivered", kept)
		}
	}
	for _, gone := range []string{".gtt", "gtt-domain", "AGENTS.md", ".claude/CLAUDE.md", ".git", "dist"} {
		if _, err := os.Stat(filepath.Join(dst, gone)); err == nil {
			t.Errorf("%s must not be delivered", gone)
		}
	}
	if res.Copied != 3 || res.Excluded != 4 {
		t.Errorf("counts: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(src, ".gtt", "x")); err != nil {
		t.Error("the source project was modified")
	}
	if _, err := Copy(src, dst, nil); err == nil {
		t.Error("a non-empty destination must be refused")
	}
}
