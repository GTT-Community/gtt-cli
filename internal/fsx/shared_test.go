package fsx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const contract = "# AGENTS.md — GTT Bootstrap Agent Contract\n\nIt names its own markers in prose: `<!-- gtt:end -->`.\n\n## Rules\n"

func TestSharedMergeKeepsHostAndStripsBack(t *testing.T) {
	host := "# Team rules\n\n- Use pnpm.\n"
	merged := Merge(host, contract)
	if !strings.HasPrefix(merged, host) {
		t.Fatalf("host lines must come first, got:\n%s", merged)
	}
	if again := Merge(merged, contract); again != merged {
		t.Fatalf("merging twice must replace the block, not add one:\n%s", again)
	}
	path := filepath.Join(t.TempDir(), SharedFile)
	if err := os.WriteFile(path, []byte(merged), 0o644); err != nil {
		t.Fatal(err)
	}
	if text, keep := HostText(path); !keep || text != host {
		t.Fatalf("HostText = %q, %v; want the host's lines only", text, keep)
	}
	if err := StripOrRemove(path); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != host {
		t.Fatalf("after strip the file must be the host's again, got:\n%s", got)
	}
}

func TestSharedBlockHashMatchesThePackageFile(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "pkg.md")
	path := filepath.Join(dir, SharedFile)
	os.WriteFile(pkg, []byte(contract), 0o644)
	os.WriteFile(path, []byte(Merge("# Host\n", contract)), 0o644)
	want, _ := HashFile(pkg)
	if got, ok := BlockHash(path); !ok || got != want {
		t.Fatalf("BlockHash = %s, %v; want the package file's hash %s", got, ok, want)
	}
}

func TestSharedRemoveNeverTakesAHostFileWithoutBlock(t *testing.T) {
	dir := t.TempDir()
	host := filepath.Join(dir, "host", SharedFile)
	os.MkdirAll(filepath.Dir(host), 0o755)
	os.WriteFile(host, []byte("# Team rules\n"), 0o644)
	if err := StripOrRemove(host); err != nil || !Exists(host) {
		t.Fatalf("a host AGENTS.md GTT never joined must stay (err %v)", err)
	}
	only := filepath.Join(dir, "only", SharedFile)
	os.MkdirAll(filepath.Dir(only), 0o755)
	os.WriteFile(only, []byte(contract), 0o644)
	if err := StripOrRemove(only); err != nil || Exists(only) {
		t.Fatalf("a file that is only GTT's contract is removed (err %v)", err)
	}
	joined := filepath.Join(dir, "joined", SharedFile)
	os.MkdirAll(filepath.Dir(joined), 0o755)
	os.WriteFile(joined, []byte(Merge("", contract)), 0o644)
	if err := StripOrRemove(joined); err != nil || Exists(joined) {
		t.Fatalf("a joined file with no host lines is removed (err %v)", err)
	}
}
