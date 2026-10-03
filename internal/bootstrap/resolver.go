package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/fsx"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// CanonicalSource is the canonical distribution source of GTT Bootstrap.
const CanonicalSource = "https://github.com/GTT-Community/gtt-bootstrap.git"

// Resolver finds a Bootstrap catalog in this order: explicit path, the
// source the project recorded, the verified user cache, the configured
// source, the canonical source. It never picks an unknown version silently.
type Resolver struct {
	Runner  ports.Runner
	Factory ports.BootstrapFactory
	Home    string // user-level GTT home (cache lives under it)
	Remote  string // configured distribution source; empty = canonical
	// KeysDir holds the trusted release keys. When it holds any, every
	// package must carry a signature that verifies against one of them.
	KeysDir string
}

// CacheDir is where verified catalogs are kept, outside any project.
func (r Resolver) CacheDir() string { return filepath.Join(r.Home, "cache", "bootstrap") }

// Resolve returns a catalog on disk. The result is not yet verified.
func (r Resolver) Resolve(ctx context.Context, req ports.ResolveRequest) (ports.Package, error) {
	if req.Path != "" {
		return r.local(ctx, req.Path, "local", req)
	}
	if req.Recorded != "" && isDir(req.Recorded) {
		if pkg, err := r.local(ctx, req.Recorded, "recorded", req); err == nil {
			return pkg, nil
		}
	}
	if !req.Refresh || req.Offline || req.Version != "" {
		if pkg, ok, err := r.cached(ctx, req); err != nil || ok {
			return pkg, err
		}
	}
	if req.Offline {
		want := "any version"
		if req.Version != "" {
			want = "version " + req.Version
		}
		return ports.Package{}, &core.Error{Code: core.ExitUnavailable, Unmodified: true,
			What: "No verified GTT Bootstrap is available offline (" + want + ").",
			Why:  "Offline mode uses only --bootstrap or the local cache at " + r.CacheDir() + ".",
			Next: "Run once online, or pass --bootstrap /path/to/gtt-bootstrap."}
	}
	return r.remote(ctx, req)
}

func (r Resolver) local(ctx context.Context, path, source string, req ports.ResolveRequest) (ports.Package, error) {
	abs, err := filepath.Abs(path)
	if err != nil || !isDir(abs) {
		return ports.Package{}, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Bootstrap path is not a directory: " + path}
	}
	if !r.Factory.Installed(abs) {
		return ports.Package{}, &core.Error{Code: core.ExitIntegrity, Unmodified: true,
			What: "Not a GTT Bootstrap package: " + abs, Why: "The contract entry point is missing."}
	}
	rel, err := r.Factory.At(abs).Release(ctx)
	if err != nil {
		return ports.Package{}, err
	}
	if req.Version != "" && rel.Bootstrap.Version != req.Version {
		return ports.Package{}, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: fmt.Sprintf("Requested Bootstrap %s, but %s holds %s.", req.Version, abs, rel.Bootstrap.Version)}
	}
	return ports.Package{Root: abs, Source: source, Origin: abs, Release: rel, Checksum: req.Checksum}, nil
}

// cached returns the requested version from the cache, or the newest cached
// one when no version was requested. A cache entry whose content no longer
// matches its recorded checksum is an integrity failure.
func (r Resolver) cached(ctx context.Context, req ports.ResolveRequest) (ports.Package, bool, error) {
	entries, err := os.ReadDir(r.CacheDir())
	if err != nil {
		return ports.Package{}, false, nil
	}
	var versions [][3]int
	names := map[[3]int]string{}
	for _, e := range entries {
		if v, ok := core.ParseSemver(e.Name()); ok && e.IsDir() {
			versions = append(versions, v)
			names[v] = e.Name()
		}
	}
	if len(versions) == 0 {
		return ports.Package{}, false, nil
	}
	sort.Slice(versions, func(i, j int) bool { return core.CompareSemver(versions[i], versions[j]) > 0 })
	pick := names[versions[0]]
	if req.Version != "" {
		if _, ok := core.ParseSemver(req.Version); !ok {
			return ports.Package{}, false, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Bootstrap version must be MAJOR.MINOR.PATCH: " + req.Version}
		}
		pick = ""
		for _, n := range names {
			if n == req.Version {
				pick = n
			}
		}
		if pick == "" {
			return ports.Package{}, false, nil
		}
	}
	root := filepath.Join(r.CacheDir(), pick)
	recorded, err := os.ReadFile(root + ".sha256")
	if err != nil {
		return ports.Package{}, false, &core.Error{Code: core.ExitIntegrity, Unmodified: true,
			What: "Cached Bootstrap " + pick + " has no recorded checksum.", Next: "Remove " + root + " and resolve again."}
	}
	pkg, err := r.local(ctx, root, "cache", ports.ResolveRequest{Version: req.Version})
	if err != nil {
		return ports.Package{}, false, err
	}
	pkg.Checksum = strings.TrimSpace(string(recorded)) // expected; Verify recomputes and compares
	return pkg, true, nil
}

// remote clones the distribution source into the cache.
func (r Resolver) remote(ctx context.Context, req ports.ResolveRequest) (ports.Package, error) {
	git, err := r.Runner.LookPath("git")
	if err != nil {
		return ports.Package{}, &core.Error{Code: core.ExitUnavailable, Unmodified: true,
			What: "git is required to download GTT Bootstrap.", Next: "Install git, or pass --bootstrap /path/to/gtt-bootstrap."}
	}
	source := r.Remote
	if source == "" {
		source = CanonicalSource
	}
	if err := os.MkdirAll(r.CacheDir(), 0o755); err != nil {
		return ports.Package{}, err
	}
	tmp, err := os.MkdirTemp(r.CacheDir(), ".download-")
	if err != nil {
		return ports.Package{}, err
	}
	defer os.RemoveAll(tmp)
	clone := func(extra ...string) (ports.Result, error) {
		os.RemoveAll(filepath.Join(tmp, "pkg"))
		args := append([]string{"clone", "--quiet", "--depth", "1"}, extra...)
		args = append(args, "--", source, filepath.Join(tmp, "pkg"))
		return r.Runner.Run(ctx, ports.Command{Path: git, Args: args, Env: []string{"GIT_TERMINAL_PROMPT=0"}})
	}
	res, err := clone()
	if req.Version != "" {
		// Prefer the release tag; when the source publishes none, the default
		// branch is accepted only if its release descriptor is that version.
		if tagged, terr := clone("--branch", "v"+req.Version); terr == nil && tagged.ExitCode == 0 {
			res, err = tagged, nil
		} else {
			res, err = clone()
		}
	}
	if err != nil || res.ExitCode != 0 {
		return ports.Package{}, &core.Error{Code: core.ExitUnavailable, Unmodified: true,
			What: "Could not download GTT Bootstrap from " + source + ".", Why: strings.TrimSpace(string(res.Stderr)),
			Next: "Check the network or the requested version, or pass --bootstrap /path/to/gtt-bootstrap."}
	}
	os.RemoveAll(filepath.Join(tmp, "pkg", ".git"))
	pkg, err := r.local(ctx, filepath.Join(tmp, "pkg"), "remote", req)
	if err != nil {
		return ports.Package{}, err
	}
	files, err := fsx.ListFiles(pkg.Root)
	if err != nil {
		return ports.Package{}, &core.Error{Code: core.ExitIntegrity, Unmodified: true, What: "The downloaded Bootstrap is unsafe.", Why: err.Error()}
	}
	sum, err := fsx.HashTree(pkg.Root, contentFiles(files))
	if err != nil {
		return ports.Package{}, err
	}
	final := filepath.Join(r.CacheDir(), pkg.Release.Bootstrap.Version)
	os.RemoveAll(final)
	if err := os.Rename(pkg.Root, final); err != nil {
		return ports.Package{}, err
	}
	if err := os.WriteFile(final+".sha256", []byte(sum+"\n"), 0o644); err != nil {
		return ports.Package{}, err
	}
	// The checksum just computed is recorded for the cache; it is not an
	// expected value, so Verify must not report it as an independent check.
	pkg.Root, pkg.Origin, pkg.Checksum = final, source, ""
	return pkg, nil
}

// Verify checks structure, path safety, release metadata, the content
// checksum, the release signature and the Bootstrap's own contract check. Any failure is fatal.
func (r Resolver) Verify(ctx context.Context, pkg *ports.Package) error {
	fail := func(what, why string) error {
		return &core.Error{Code: core.ExitIntegrity, Unmodified: true, What: "GTT Bootstrap integrity check failed: " + what, Why: why,
			Next: "Use a trusted Bootstrap package."}
	}
	files, err := fsx.ListFiles(pkg.Root)
	if err != nil {
		return fail("unsafe package content", err.Error())
	}
	if pkg.Release.Bootstrap.ID != core.BootstrapID {
		return fail("unexpected release identity", "id is "+pkg.Release.Bootstrap.ID)
	}
	if _, err := Layout(*pkg); err != nil {
		return err
	}
	sum, err := fsx.HashTree(pkg.Root, contentFiles(files))
	if err != nil {
		return fail("unreadable package content", err.Error())
	}
	if pkg.Checksum != "" && pkg.Checksum != sum {
		return fail("checksum mismatch", "expected "+pkg.Checksum+"\ncomputed "+sum)
	}
	keys, err := TrustedKeys(r.KeysDir)
	if err != nil {
		return err
	}
	signature, err := checkSignature(pkg.Root, pkg.Release.Bootstrap.Version, sum, keys, r.KeysDir)
	if err != nil {
		return fail("release signature", err.Error())
	}
	problems, err := r.Factory.At(pkg.Root).CheckContracts(ctx)
	if err != nil {
		return err
	}
	if len(problems) > 0 {
		return fail("the Bootstrap contracts are inconsistent", strings.Join(problems, "\n"))
	}
	if pkg.Checksum == "" {
		pkg.Integrity = "structure and contracts verified; checksum recorded (no expected value supplied)"
	} else {
		pkg.Integrity = "structure, contracts and checksum verified"
	}
	pkg.Checksum = sum
	pkg.Signature = signature
	return nil
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// Now is the clock used for resolution records.
var Now = func() time.Time { return time.Now().UTC() }
