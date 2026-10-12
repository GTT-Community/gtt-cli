package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/core"
)

// The CLI and the Bootstrap go together: `gtt init` always leaves the
// Bootstrap release this CLI pins installed and verified in the project.
// Every release here is served from a Git repository on the local disk.

// releaseSource publishes the catalog as a Bootstrap distribution source: a
// Git repository whose release tag names the given version. With tag set to
// false the release exists only on the branch.
func releaseSource(t *testing.T, version string, tag bool) string {
	t.Helper()
	dir := catalogCopy(t)
	edit(t, filepath.Join(dir, ".gtt/contract/release.json"), func(d map[string]any) {
		d["bootstrap"].(map[string]any)["version"] = version
	})
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=test", "-c", "user.email=test@example.invalid",
			"-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false", "-c", "core.autocrlf=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "--quiet", "--initial-branch", "main")
	git("add", "-A")
	git("commit", "--quiet", "-m", "GTT Bootstrap "+version)
	if tag {
		git("tag", "-a", core.BootstrapTag(version), "-m", "GTT Bootstrap "+version)
	}
	return dir
}

// catalogVersion is the release of the catalog under test.
func catalogVersion(t *testing.T) string {
	t.Helper()
	var doc struct {
		Bootstrap struct {
			Version string `json:"version"`
		} `json:"bootstrap"`
	}
	if err := json.Unmarshal([]byte(read(t, pristine, ".gtt/contract/release.json")), &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Bootstrap.Version
}

// overlayEntry is the instruction entry the catalog under test declares for
// an ADE (scaffold manifest, `overlays`): where it lives is the Bootstrap's
// to decide, and it has moved between releases.
func overlayEntry(t *testing.T, id string) string {
	t.Helper()
	for _, line := range strings.Split(read(t, pristine, ".gtt/scaffold/manifest.yaml"), "\n") {
		if !strings.Contains(line, "{ id: "+id+",") {
			continue
		}
		if i := strings.Index(line, "entry: "); i >= 0 {
			return strings.TrimSpace(strings.SplitN(line[i+len("entry: "):], ",", 2)[0])
		}
	}
	t.Fatalf("the catalog declares no overlay entry for %s", id)
	return ""
}

// next returns version with its minor number raised by n.
func next(t *testing.T, version string, n int) string {
	t.Helper()
	v, ok := core.ParseSemver(version)
	if !ok {
		t.Fatalf("not a release version: %q", version)
	}
	return strconv.Itoa(v[0]) + "." + strconv.Itoa(v[1]+n) + ".0"
}

func cacheNames(t *testing.T) []string {
	t.Helper()
	entries, _ := os.ReadDir(filepath.Join(os.Getenv("GTT_HOME"), "cache", "bootstrap"))
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func leftovers(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	for _, pattern := range []string{".gtt-staging-*", ".gtt-init-transaction.json", ".gtt-update-backup"} {
		m, _ := filepath.Glob(filepath.Join(root, pattern))
		out = append(out, m...)
	}
	return out
}

func TestInitInstallsThePinnedReleaseAndReusesIt(t *testing.T) {
	root := project(t, nil)
	source := releaseSource(t, core.BootstrapPin, true)
	t.Setenv("GTT_BOOTSTRAP_SOURCE", source)

	// Bootstrap missing: the pinned release is fetched from its tag and installed.
	r := gtt(t, root, append([]string{"init", "--yes"}, defaults...)...).must(t, 0)
	if !strings.Contains(r.out+r.err, "pinned by GTT CLI") {
		t.Errorf("the resolution must say the release is the pinned one:\n%s", r.err)
	}
	st := gtt(t, root, "inspect", "--json").must(t, 0).json(t)
	if b := st["bootstrap"].(map[string]any); b["version"] != core.BootstrapPin {
		t.Fatalf("installed %v, want the pinned %s", b["version"], core.BootstrapPin)
	}
	if res := st["resolution"].(map[string]any); res["source"] != "remote" || !strings.HasSuffix(res["origin"].(string), "@"+core.BootstrapTag(core.BootstrapPin)) {
		t.Errorf("the record must name the release tag: %v", res)
	}
	if got := strings.Join(cacheNames(t), ","); got != core.BootstrapPin+","+core.BootstrapPin+".sha256" {
		t.Errorf("the cache holds the release and its checksum only: %s", got)
	}
	if left := leftovers(t, root); len(left) != 0 {
		t.Errorf("nothing of the transaction may remain: %v", left)
	}

	// Already present and matching: reused, verified, nothing rewritten -
	// however many times it runs, and without the source.
	if err := os.Rename(source, source+".gone"); err != nil {
		t.Fatal(err)
	}
	before := treeHash(t, root)
	for i := 0; i < 3; i++ {
		r = gtt(t, root, append([]string{"init", "--yes"}, defaults...)...).must(t, 0)
		if !strings.Contains(r.out, "already initialized") || !strings.Contains(r.out, "present and verified") {
			t.Fatalf("run %d must reuse the installed release:\n%s", i+2, r.out)
		}
		if treeHash(t, root) != before {
			t.Fatalf("run %d rewrote the project", i+2)
		}
	}
	doc := gtt(t, root, "init", "--json").must(t, 0).json(t)
	check := doc["installed_bootstrap"].(map[string]any)
	if doc["outcome"] != "already-initialized" || check["verified"] != true || check["version"] != core.BootstrapPin || check["pinned"] != core.BootstrapPin {
		t.Errorf("got %v", doc)
	}
	if _, has := check["modified_locally"]; has {
		t.Errorf("a fresh install has no customized file; what the Bootstrap rewrites itself is not one: %v", check["modified_locally"])
	}

	// A second project on this machine is served by the verified cache.
	other := filepath.Join(t.TempDir(), "billing-api")
	write(t, other, "src/main.go", "package main\n")
	gtt(t, other, append([]string{"init", "--offline", "--yes"}, defaults...)...).must(t, 0)
	if v := gtt(t, other, "version", "--json").must(t, 0).json(t)["bootstrap"].(map[string]any); v["version"] != core.BootstrapPin || v["matches_pin"] != true {
		t.Errorf("the second project must hold the pinned release: %v", v)
	}
}

func TestBootstrapFetchFailureFailsInitAndLeavesNothing(t *testing.T) {
	cases := map[string]struct {
		source func(t *testing.T) string
		want   string
	}{
		"the release was never published": {func(t *testing.T) string { return releaseSource(t, core.BootstrapPin, false) }, "is not published"},
		"the source cannot be reached":    {func(t *testing.T) string { return filepath.Join(t.TempDir(), "nowhere") }, "Could not reach"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root := project(t, nil)
			t.Setenv("GTT_BOOTSTRAP_SOURCE", c.source(t))
			before := treeHash(t, root)
			r := gtt(t, root, append([]string{"init", "--yes"}, defaults...)...).must(t, 7)
			if !strings.Contains(r.err, c.want) || !strings.Contains(r.err, "No project files were modified") {
				t.Errorf("the failure must be clear and distinct:\n%s", r.err)
			}
			if strings.Contains(r.out, "initialization complete") {
				t.Errorf("success must not be reported:\n%s", r.out)
			}
			if treeHash(t, root) != before || exists(root, ".gtt") {
				t.Error("a failed fetch must leave the project untouched")
			}
			if left := cacheNames(t); len(left) != 0 {
				t.Errorf("a failed fetch must leave no partial download: %v", left)
			}
		})
	}
}

func TestABranchHeadIsNeverInstalledInPlaceOfTheRelease(t *testing.T) {
	root := project(t, nil)
	source := releaseSource(t, core.BootstrapPin, true)
	t.Setenv("GTT_BOOTSTRAP_SOURCE", source)
	// The branch moves on after the release.
	write(t, source, ".gtt/docs/usage.md", read(t, source, ".gtt/docs/usage.md")+"\nUnreleased work on the branch head.\n")
	for _, args := range [][]string{{"add", "-A"}, {"commit", "--quiet", "-m", "unreleased"}} {
		cmd := exec.Command("git", append([]string{"-C", source, "-c", "user.name=test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	gtt(t, root, append([]string{"init", "--yes"}, defaults...)...).must(t, 0)
	if strings.Contains(read(t, root, ".gtt/docs/usage.md"), "Unreleased work") {
		t.Error("the branch head was installed instead of the release tag")
	}
}

// governed are the files an installation, reinstallation or update of the
// Bootstrap must never create, modify or delete.
var governed = []string{"gtt-domain/context/stack.md", "gtt-domain/context/constraints.md", "gtt-domain/adr/ADR-900-project-decision.md",
	"gtt-domain/change-request.md", "gtt-domain/.frozen", "SOURCE-BRIEF.md"}

func TestAnotherReleaseIsReportedAndChangedOnlyOnConfirmation(t *testing.T) {
	root := installed(t, nil)
	have := catalogVersion(t)
	// A project with a populated governed context, an ADR, a preserved
	// source, an open change request and a freeze marker.
	write(t, root, "gtt-domain/context/stack.md", read(t, root, "gtt-domain/context/stack.md")+"\nProject note: the governed stack of this project.\n")
	write(t, root, "gtt-domain/context/constraints.md", read(t, root, "gtt-domain/context/constraints.md")+"\nProject note: a hard limit of this project.\n")
	write(t, root, "gtt-domain/adr/ADR-900-project-decision.md", read(t, root, "gtt-domain/adr/ADR-TEMPLATE.md"))
	write(t, root, "gtt-domain/change-request.md", read(t, root, "gtt-domain/change-request.md")+"\nProject note: a request the team is drafting.\n")
	write(t, root, "gtt-domain/.frozen", "frozen: 2026-01-01T00:00:00Z\n")
	write(t, root, "SOURCE-BRIEF.md", "# The design this project was bootstrapped from\n")
	state := func() string {
		var parts []string
		for _, rel := range governed {
			parts = append(parts, rel+"\x00"+read(t, root, rel))
		}
		return strings.Join(parts, "\x01")
	}
	governedBefore := state()
	// A Bootstrap artifact the team customized.
	write(t, root, ".gtt/docs/docs.md", read(t, root, ".gtt/docs/docs.md")+"\nlocal note of this team\n")

	// Without a terminal: the mismatch is reported and nothing is replaced. --yes is no bypass.
	want := next(t, have, 1)
	target := newer(t, want)
	before := treeHash(t, root)
	r := gtt(t, root, "init", "--bootstrap", target, "--yes").must(t, 5)
	if !strings.Contains(r.err, "runs GTT Bootstrap "+have) || !strings.Contains(r.err, "requires "+want) || !strings.Contains(r.err, "--yes does not give it") {
		t.Errorf("installed and required release must both be reported:\n%s", r.err)
	}
	if treeHash(t, root) != before {
		t.Fatal("a different release was changed without a human confirmation")
	}
	// In a terminal, declined: still nothing.
	gttIn(t, root, "n\n", "init", "--bootstrap", target).must(t, 4)
	if treeHash(t, root) != before {
		t.Fatal("a declined update changed the project")
	}

	// Confirmed; the customized file is shown and, not confirmed, kept.
	r = gttIn(t, root, "y\nn\n", "init", "--bootstrap", target).must(t, 0)
	if !strings.Contains(r.out, "GTT Bootstrap updated: "+have+" -> "+want) {
		t.Errorf("the update must be reported:\n%s", r.out)
	}
	if shown := r.out + r.err; !strings.Contains(shown, ".gtt/docs/docs.md (this project)") || !strings.Contains(shown, "-local note of this team") {
		t.Errorf("the difference of the customized file must be shown:\n%s", shown)
	}
	if !strings.Contains(read(t, root, ".gtt/docs/docs.md"), "local note of this team") {
		t.Error("the customized file was replaced without its own confirmation")
	}
	if !strings.Contains(read(t, root, ".gtt/docs/usage.md"), "added in "+want) {
		t.Error("the release content must be installed")
	}
	if v := gtt(t, root, "version", "--json").must(t, 0).json(t)["bootstrap"].(map[string]any); v["version"] != want {
		t.Errorf("installed version: %v", v)
	}
	if state() != governedBefore {
		t.Fatal("the update touched governed state")
	}
	if left := leftovers(t, root); len(left) != 0 {
		t.Errorf("nothing of the transaction may remain: %v", left)
	}

	// The same release again: reused, and the customized file is reported.
	doc := gtt(t, root, "init", "--bootstrap", target, "--json").must(t, 0).json(t)
	check := doc["installed_bootstrap"].(map[string]any)
	if doc["outcome"] != "already-initialized" || strings.Join(toStrings(check["modified_locally"]), ",") != ".gtt/docs/docs.md" {
		t.Errorf("only the customized file is a local modification: %v", check)
	}

	// A later release, with the replacement confirmed this time.
	want2 := next(t, have, 2)
	r = gttIn(t, root, "y\ny\n", "init", "--bootstrap", newer(t, want2)).must(t, 0)
	if strings.Contains(read(t, root, ".gtt/docs/docs.md"), "local note of this team") {
		t.Errorf("a confirmed replacement must install the release content:\n%s", r.err)
	}
	if state() != governedBefore {
		t.Fatal("the update touched governed state")
	}
	// A project ahead of what the CLI requires is never downgraded.
	before = treeHash(t, root)
	r = gttIn(t, root, "y\n", "init", "--bootstrap", target).must(t, 2)
	if !strings.Contains(r.err, "never downgrades") || treeHash(t, root) != before {
		t.Errorf("a downgrade must be refused untouched:\n%s", r.err)
	}
}

func TestUpdateKeepsCustomizedFilesWithoutATerminal(t *testing.T) {
	root := installed(t, nil)
	write(t, root, ".gtt/docs/docs.md", read(t, root, ".gtt/docs/docs.md")+"\nlocal note of this team\n")
	// The same release: there is nothing to install, and the difference is shown.
	r := gtt(t, root, "update", "--bootstrap", pristine, "--yes").must(t, 0)
	if !strings.Contains(r.out, "already installed") || !strings.Contains(r.out+r.err, "-local note of this team") {
		t.Errorf("got:\n%s\n%s", r.out, r.err)
	}
	if !strings.Contains(read(t, root, ".gtt/docs/docs.md"), "local note of this team") {
		t.Error("--yes must not replace a customized file")
	}
	// A deleted package file is restored by the same release.
	os.Remove(filepath.Join(root, ".gtt", "docs", "usage.md"))
	r = gtt(t, root, "init", "--bootstrap", pristine, "--yes").must(t, 5)
	if !strings.Contains(r.err, "incomplete") || !strings.Contains(r.err, ".gtt/docs/usage.md") {
		t.Errorf("an incomplete Bootstrap is not a success:\n%s", r.err)
	}
	gttIn(t, root, "y\nn\n", "init", "--bootstrap", pristine).must(t, 0)
	if !exists(root, ".gtt/docs/usage.md") || !strings.Contains(read(t, root, ".gtt/docs/docs.md"), "local note of this team") {
		t.Error("the missing file must be restored and the customized one kept")
	}
}

func TestInterruptedInstallLeavesNoHalfInstalledBootstrap(t *testing.T) {
	// A process killed while the core was being committed: the journal says
	// STAGING, one path is in place, the staging directory is still there.
	root := project(t, nil)
	write(t, root, ".gtt-init-transaction.json", `{"schema":1,"state":"STAGING","core_paths":[".gtt","gtt-domain","AGENTS.md","readme-gtt.md","readme-gtt.es.md"],"completed":[]}`)
	write(t, root, ".gtt/contract/release.json", "{}\n")
	write(t, root, ".gtt-staging-1234/gtt-domain/backlog.md", "staged\n")
	write(t, root, ".gtt-staging-1234/AGENTS.md", "staged\n")
	before := treeHash(t, root)

	// It is never mistaken for an installed project, nor silently completed.
	r := gtt(t, root, append([]string{"init", "--bootstrap", pristine, "--yes"}, defaults...)...).must(t, 5)
	if !strings.Contains(r.err, "Incomplete GTT initialization") || treeHash(t, root) != before {
		t.Errorf("got:\n%s", r.err)
	}
	gtt(t, root, "init", "--rollback").must(t, 0)
	for _, rel := range []string{".gtt", "gtt-domain", "AGENTS.md", ".gtt-staging-1234", ".gtt-init-transaction.json"} {
		if exists(root, rel) {
			t.Errorf("%s was left behind", rel)
		}
	}
	if !exists(root, "src/main.go") {
		t.Error("the rollback removed a host file")
	}
	// And the project installs cleanly afterwards.
	initProject(t, root, pristine, defaults...)
}

func TestVersionReportsBothVersions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GTT_HOME", filepath.Join(t.TempDir(), "gtt-home"))
	doc := gtt(t, dir, "version", "--json").must(t, 0).json(t)
	pin := doc["pinned_bootstrap"].(map[string]any)
	if doc["cli_version"] != core.Version || pin["version"] != core.BootstrapPin || pin["tag"] != core.BootstrapTag(core.BootstrapPin) {
		t.Errorf("got %v", doc)
	}
	text := gtt(t, dir, "version").must(t, 0).out
	if !strings.Contains(text, "GTT CLI       "+core.Version) || !strings.Contains(text, "GTT Bootstrap "+core.BootstrapPin) {
		t.Errorf("got:\n%s", text)
	}
}
