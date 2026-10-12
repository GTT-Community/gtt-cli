package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/execution"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// These tests fetch from a Git repository on the local disk: no network.

// treeFactory treats a directory holding a VERSION file as a Bootstrap whose
// release version is the content of that file.
type treeFactory struct{}

func (treeFactory) Installed(root string) bool {
	_, err := os.Stat(filepath.Join(root, "VERSION"))
	return err == nil
}
func (treeFactory) At(root string) ports.Bootstrap { return treeBootstrap(root) }

type treeBootstrap string

func (b treeBootstrap) Root() string { return string(b) }
func (b treeBootstrap) Release(context.Context) (ports.Release, error) {
	var r ports.Release
	data, err := os.ReadFile(filepath.Join(string(b), "VERSION"))
	r.Bootstrap.ID, r.Bootstrap.Version = core.BootstrapID, strings.TrimSpace(string(data))
	return r, err
}
func (treeBootstrap) Negotiate(context.Context) (ports.Negotiation, error) {
	return ports.Negotiation{Compatible: true}, nil
}
func (treeBootstrap) Capabilities(context.Context) ([]ports.Capability, error) { return nil, nil }
func (treeBootstrap) Operations(context.Context) (map[string]ports.Operation, error) {
	return nil, nil
}
func (treeBootstrap) Show(context.Context, string) (json.RawMessage, error) { return nil, nil }
func (treeBootstrap) CheckContracts(context.Context) ([]string, error)      { return nil, nil }
func (treeBootstrap) Execute(context.Context, ports.OperationRequest) (ports.OperationResult, error) {
	return ports.OperationResult{}, nil
}

// repo is a Bootstrap distribution source on disk.
type repo struct {
	t   *testing.T
	dir string
}

func newRepo(t *testing.T) repo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	r := repo{t, filepath.Join(t.TempDir(), "source")}
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		t.Fatal(err)
	}
	r.git("init", "--quiet", "--initial-branch", "main")
	return r
}

func (r repo) git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", r.dir, "-c", "user.name=test", "-c", "user.email=test@example.invalid",
		"-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// commit records a tree whose release version is version and whose content
// is marked, so a test can tell which commit was fetched.
func (r repo) commit(version, marker string) {
	r.t.Helper()
	for name, content := range map[string]string{"VERSION": version + "\n", "MARKER": marker + "\n"} {
		if err := os.WriteFile(filepath.Join(r.dir, name), []byte(content), 0o644); err != nil {
			r.t.Fatal(err)
		}
	}
	r.git("add", "-A")
	r.git("commit", "--quiet", "-m", marker)
}

func resolver(t *testing.T, source string) Resolver {
	t.Helper()
	return Resolver{Runner: execution.New(), Factory: treeFactory{}, Home: filepath.Join(t.TempDir(), "home"), Remote: source}
}

// cacheEntries lists everything in the cache directory.
func cacheEntries(t *testing.T, r Resolver) []string {
	t.Helper()
	entries, err := os.ReadDir(r.CacheDir())
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func marker(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "MARKER"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(data))
}

func TestRemoteFetchesTheReleaseTagNeverTheBranchHead(t *testing.T) {
	src := newRepo(t)
	src.commit("1.5.0", "release")
	src.git("tag", "-a", "v1.5.0", "-m", "release 1.5.0")
	src.commit("1.6.0", "branch head, not released")
	r := resolver(t, src.dir)

	pkg, err := r.Resolve(context.Background(), ports.ResolveRequest{Version: "1.5.0"})
	if err != nil {
		t.Fatal(err)
	}
	if marker(t, pkg.Root) != "release" || pkg.Release.Bootstrap.Version != "1.5.0" {
		t.Errorf("fetched %q (%s): the release tag must be fetched, never the branch head", marker(t, pkg.Root), pkg.Release.Bootstrap.Version)
	}
	if pkg.Source != "remote" || !strings.HasSuffix(pkg.Origin, "@v1.5.0") {
		t.Errorf("the record must name the tag: %s %s", pkg.Source, pkg.Origin)
	}
	if got := strings.Join(cacheEntries(t, r), ","); got != "1.5.0,1.5.0.sha256" {
		t.Errorf("the cache must hold exactly the release and its checksum, no staging left: %s", got)
	}
	if _, err := os.Stat(filepath.Join(pkg.Root, ".git")); err == nil {
		t.Error("the cached package must not carry the Git metadata")
	}

	// The cached release is reused: the source is no longer needed.
	if err := os.RemoveAll(src.dir); err != nil {
		t.Fatal(err)
	}
	again, err := r.Resolve(context.Background(), ports.ResolveRequest{Version: "1.5.0"})
	if err != nil || again.Source != "cache" || again.Root != pkg.Root {
		t.Fatalf("the verified cache must be reused: %+v %v", again, err)
	}
	recorded, _ := os.ReadFile(pkg.Root + ".sha256")
	if again.Checksum == "" || again.Checksum != strings.TrimSpace(string(recorded)) {
		t.Errorf("a cached release carries its recorded checksum as the expected value: %q", again.Checksum)
	}
}

func TestRemoteFailuresAreDistinctAndLeaveNoPartialCache(t *testing.T) {
	tagged := newRepo(t)
	tagged.commit("1.5.0", "release")
	tagged.git("tag", "v1.5.0")

	untagged := newRepo(t)
	untagged.commit("1.5.0", "only on a branch")

	// A branch that takes the tag's name would be cloned instead of the tag.
	shadowed := newRepo(t)
	shadowed.commit("1.5.0", "release")
	shadowed.git("tag", "v1.5.0")
	shadowed.commit("1.5.0", "impostor")
	shadowed.git("branch", "v1.5.0")

	// A tag whose release descriptor names another version.
	mislabeled := newRepo(t)
	mislabeled.commit("1.4.0", "release")
	mislabeled.git("tag", "v1.5.0")

	cases := []struct {
		name, source, version string
		code                  core.ExitCode
		what                  string
	}{
		{"missing release", untagged.dir, "1.5.0", core.ExitUnavailable, "is not published"},
		{"unreachable source", filepath.Join(t.TempDir(), "nowhere"), "1.5.0", core.ExitUnavailable, "Could not reach"},
		{"branch named like the tag", shadowed.dir, "1.5.0", core.ExitIntegrity, "is not GTT Bootstrap release v1.5.0"},
		{"version mismatch", mislabeled.dir, "1.5.0", core.ExitIncompatible, "Requested Bootstrap 1.5.0"},
		{"no release named", tagged.dir, "", core.ExitUsage, "release tag"},
		{"a branch is not a release", tagged.dir, "main", core.ExitUsage, "MAJOR.MINOR.PATCH"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := resolver(t, c.source)
			_, err := r.Resolve(context.Background(), ports.ResolveRequest{Version: c.version})
			if err == nil {
				t.Fatal("must fail")
			}
			e, ok := err.(*core.Error)
			if !ok || e.Code != c.code || !strings.Contains(e.What+" "+e.Why, c.what) {
				t.Errorf("want exit %d mentioning %q, got: %v (%T)", c.code, c.what, err, err)
			}
			if ok && !e.Unmodified {
				t.Error("a failed resolution must state that the project was not modified")
			}
			if left := cacheEntries(t, r); len(left) != 0 {
				t.Errorf("a failed download must leave nothing in the cache: %v", left)
			}
		})
	}
}

func TestInterruptedDownloadIsNotACacheEntry(t *testing.T) {
	r := resolver(t, filepath.Join(t.TempDir(), "nowhere"))
	// What a download killed at its worst moments leaves behind: a staging
	// directory, and a checksum whose package was never moved into place.
	os.MkdirAll(filepath.Join(r.CacheDir(), ".download-123", "pkg"), 0o755)
	os.WriteFile(filepath.Join(r.CacheDir(), ".download-123", "pkg", "VERSION"), []byte("1.5.0\n"), 0o644)
	os.WriteFile(filepath.Join(r.CacheDir(), "1.5.0.sha256"), []byte("0000\n"), 0o644)

	_, err := r.Resolve(context.Background(), ports.ResolveRequest{Version: "1.5.0", Offline: true})
	if core.CodeOf(err) != core.ExitUnavailable {
		t.Fatalf("leftovers of an interrupted download must never be used as a release: %v", err)
	}
}

func TestUnwritableCacheIsAPermissionError(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("needs a file system that enforces directory permissions")
	}
	src := newRepo(t)
	src.commit("1.5.0", "release")
	src.git("tag", "v1.5.0")
	locked := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(locked, 0o555); err != nil {
		t.Fatal(err)
	}
	r := Resolver{Runner: execution.New(), Factory: treeFactory{}, Home: filepath.Join(locked, "home"), Remote: src.dir}
	_, err := r.Resolve(context.Background(), ports.ResolveRequest{Version: "1.5.0"})
	e, ok := err.(*core.Error)
	if !ok || e.Code != core.ExitFailure || !strings.Contains(e.What, "Cannot write the GTT Bootstrap cache") {
		t.Errorf("want a permission error naming the cache, got: %v", err)
	}
}

func TestTaggedCommitPrefersThePeeledCommit(t *testing.T) {
	out := "aaa\trefs/tags/v1.5.0\nbbb\trefs/tags/v1.5.0^{}\nccc\trefs/tags/v1.5.00\n"
	if got := taggedCommit(out, "v1.5.0"); got != "bbb" {
		t.Errorf("annotated tag: got %q", got)
	}
	if got := taggedCommit("aaa\trefs/tags/v1.5.0\n", "v1.5.0"); got != "aaa" {
		t.Errorf("lightweight tag: got %q", got)
	}
	if got := taggedCommit("ccc\trefs/tags/v1.5.00\n", "v1.5.0"); got != "" {
		t.Errorf("another tag must not match: got %q", got)
	}
}

func TestLayoutReadsWhatTheBootstrapRewritesItself(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".gtt", "scaffold"), 0o755)
	manifest := `layers:
  engine: .gtt/
  domain: gtt-domain/
engine:
  - { path: .gtt/scripts/, kind: dir, required: true, role: services }
  - { path: .gtt/index/technical-index.json, kind: file, required: true, derived: true, role: derived, never a source of truth }
  - { path: .gtt/local/, kind: dir, required: false, state: true, role: local state }
  - { path: .gtt/contract/release.json, kind: file, required: true, role: not derived: true in prose only }
domain:
  - { path: gtt-domain/session.md, kind: file, required: false, derived: true, role: session }
`
	os.WriteFile(filepath.Join(root, ".gtt", "scaffold", "manifest.yaml"), []byte(manifest), 0o644)
	pkg := ports.Package{Root: root}
	pkg.Release.Scaffold.Manifest = ".gtt/scaffold/manifest.yaml"
	l, err := Layout(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(l.Rewritten, ","); got != ".gtt/index/technical-index.json,.gtt/local/" {
		t.Errorf("derived and state engine paths: %s", got)
	}
}
