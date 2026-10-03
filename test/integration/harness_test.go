// Package integration drives the real CLI against a real GTT Bootstrap
// catalog. Set GTT_TEST_BOOTSTRAP to a catalog; by default the sibling
// checkout ../gtt-bootstrap is used. Without one the tests are skipped.
package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/cli"
	"github.com/GTT-Community/gtt-cli/internal/compose"
	"github.com/GTT-Community/gtt-cli/internal/ui"
)

var (
	pristine  string                // a private copy of the catalog, never mutated
	templates string                // initialized projects, reused by tests that only need "an installed project"
	prepared  = map[string]string{} // init flags + files -> template directory
)

func TestMain(m *testing.M) {
	src := os.Getenv("GTT_TEST_BOOTSTRAP")
	if src == "" {
		src = filepath.Join("..", "..", "..", "gtt-bootstrap")
	}
	if _, err := os.Stat(filepath.Join(src, ".gtt", "contract", "release.json")); err != nil {
		fmt.Println("integration tests skipped: no GTT Bootstrap catalog (set GTT_TEST_BOOTSTRAP)")
		os.Exit(0)
	}
	tmp, err := os.MkdirTemp("", "gtt-it-")
	if err != nil {
		panic(err)
	}
	pristine, templates = filepath.Join(tmp, "catalog"), filepath.Join(tmp, "templates")
	if err := copyTree(src, pristine); err != nil {
		panic(err)
	}
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "__pycache__" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		info, _ := d.Info()
		return os.WriteFile(filepath.Join(dst, rel), data, info.Mode().Perm())
	})
}

// catalogCopy returns a private, mutable copy of the catalog.
func catalogCopy(t *testing.T) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "catalog")
	if err := copyTree(pristine, dst); err != nil {
		t.Fatal(err)
	}
	return dst
}

// project creates a host project with the given files and an isolated GTT home.
func project(t *testing.T, files map[string]string) string {
	t.Helper()
	t.Setenv("GTT_HOME", filepath.Join(t.TempDir(), "gtt-home"))
	// Keep ADE detection independent of what is installed on this machine.
	t.Setenv("PATH", minimalPath(t))
	root := filepath.Join(t.TempDir(), "payments-api")
	write(t, root, "src/main.go", "package main\n")
	write(t, root, "tests/main_test.go", "package main\n")
	for rel, content := range files {
		write(t, root, rel, content)
	}
	return root
}

// minimalPath exposes only the tools the Bootstrap needs.
func minimalPath(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "bin")
	os.MkdirAll(bin, 0o755)
	for _, tool := range []string{"bash", "python3", "git", "date", "grep", "cat", "head", "ls", "rm", "dirname", "sed", "awk", "sort",
		"wc", "tr", "find", "mkdir", "cp", "mv", "basename", "env", "tail", "cut", "uniq", "diff", "sha256sum", "xargs", "mktemp", "touch", "printf", "tee", "test", "id"} {
		for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
			if _, err := os.Stat(filepath.Join(dir, tool)); err == nil {
				os.Symlink(filepath.Join(dir, tool), filepath.Join(bin, tool))
				break
			}
		}
	}
	return bin
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

func exists(root, rel string) bool {
	_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

type result struct {
	code     int
	out, err string
}

func (r result) json(t *testing.T) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(r.out), &v); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, r.out)
	}
	return v
}

// gtt runs the CLI in dir without a terminal.
func gtt(t *testing.T, dir string, args ...string) result { return gttIn(t, dir, "", args...) }

// gttIn runs the CLI with a human answering on stdin.
func gttIn(t *testing.T, dir, stdin string, args ...string) result {
	t.Helper()
	var out, errb bytes.Buffer
	streams := compose.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}
	if stdin != "" {
		streams.Prompter = ui.Scripted(strings.NewReader(stdin), &errb)
	}
	code := cli.Execute(context.Background(), compose.Wire(streams), append([]string{"-C", dir}, args...), &out, &errb)
	return result{code: code, out: out.String(), err: errb.String()}
}

func (r result) must(t *testing.T, code int) result {
	t.Helper()
	if r.code != code {
		t.Fatalf("exit %d, want %d\nstdout:\n%s\nstderr:\n%s", r.code, code, r.out, r.err)
	}
	return r
}

// treeHash fingerprints every file under root (path, mode kind, content).
func treeHash(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		data, _ := os.ReadFile(p)
		sum := sha256.Sum256(data)
		lines = append(lines, rel+" "+hex.EncodeToString(sum[:]))
		return nil
	})
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

// initProject runs a complete non-interactive init.
func initProject(t *testing.T, root, catalog string, extra ...string) result {
	t.Helper()
	args := append([]string{"init", "--bootstrap", catalog, "--yes"}, extra...)
	return gtt(t, root, args...).must(t, 0)
}

var defaults = []string{"--ade", "claude", "--language", "en", "--method", "medium", "--no-sources", "--no-questionnaire"}

// edit rewrites a JSON file of a catalog.
func edit(t *testing.T, path string, fn func(doc map[string]any)) {
	t.Helper()
	var doc map[string]any
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	fn(doc)
	data, _ = json.MarshalIndent(doc, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
