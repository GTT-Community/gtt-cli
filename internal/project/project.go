// Package project detects the host project: its root, its Git repository,
// candidate source documents and GTT namespace collisions.
package project

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Git is the read-only Git adapter.
type Git struct{ Runner ports.Runner }

// Available reports whether git is on PATH.
func (g Git) Available() bool {
	_, err := g.Runner.LookPath("git")
	return err == nil
}

// Root returns the repository root containing dir.
func (g Git) Root(ctx context.Context, dir string) (string, bool) {
	git, err := g.Runner.LookPath("git")
	if err != nil {
		return "", false
	}
	res, err := g.Runner.Run(ctx, ports.Command{Path: git, Args: []string{"rev-parse", "--show-toplevel"}, Dir: dir})
	if err != nil || res.ExitCode != 0 {
		return "", false
	}
	return strings.TrimSpace(string(res.Stdout)), true
}

// Detection is what the CLI found about where it is running.
type Detection struct {
	Root      string `json:"root"`
	GitRoot   string `json:"git_root,omitempty"`
	Basis     string `json:"basis"` // gtt | git | cwd
	Confident bool   `json:"confident"`
}

// Detect finds the project root: the nearest ancestor that holds GTT, else
// the Git repository root, else the working directory (not confident).
func Detect(ctx context.Context, git ports.Git, installed func(string) bool, start string) Detection {
	abs, err := filepath.Abs(start)
	if err != nil {
		abs = start
	}
	gitRoot, hasGit := git.Root(ctx, abs)
	for dir := abs; ; dir = filepath.Dir(dir) {
		if installed(dir) {
			return Detection{Root: dir, GitRoot: gitRoot, Basis: "gtt", Confident: true}
		}
		if dir == filepath.Dir(dir) || (hasGit && dir == gitRoot) {
			break
		}
	}
	if hasGit {
		return Detection{Root: gitRoot, GitRoot: gitRoot, Basis: "git", Confident: true}
	}
	return Detection{Root: abs, Basis: "cwd", Confident: false}
}

// documentExts are the formats offered as candidate initial sources.
var documentExts = map[string]bool{".md": true, ".txt": true, ".pdf": true, ".docx": true, ".doc": true, ".rst": true}

// documentDirs are the conventional places a design source is kept.
var documentDirs = []string{"docs", "design", "architecture", "requirements", "specification"}

// scaffolding names are ordinary project files, never offered as sources.
var scaffolding = map[string]bool{"license": true, "license.md": true, "license.txt": true, "changelog.md": true,
	"contributing.md": true, "code_of_conduct.md": true, "security.md": true, "notice": true}

// Documents lists candidate source documents: files at the project root and
// one level inside the conventional directories. skip holds paths owned by
// GTT (from the Bootstrap). Candidates are observations; which one is
// authoritative is the human's decision.
func Documents(root string, skip func(rel string) bool) []string {
	var out []string
	add := func(dir string) {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			return
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || strings.HasPrefix(name, ".") || !documentExts[strings.ToLower(filepath.Ext(name))] {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(dir, name))
			if scaffolding[strings.ToLower(name)] || skip(rel) {
				continue
			}
			out = append(out, rel)
		}
	}
	add("")
	for _, d := range documentDirs {
		add(d)
	}
	sort.Strings(out)
	return out
}
