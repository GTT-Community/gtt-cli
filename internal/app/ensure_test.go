package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/fsx"
	"github.com/GTT-Community/gtt-cli/internal/lifecycle"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

type fakeInstaller struct{ layout ports.CoreLayout }

func (f fakeInstaller) Layout(ports.Package) (ports.CoreLayout, error)         { return f.layout, nil }
func (fakeInstaller) Conflicts(ports.Package, string) ([]string, error)        { return nil, nil }
func (fakeInstaller) Install(ports.Package, string) (map[string]string, error) { return nil, nil }
func (fakeInstaller) Remove(string, ports.CoreLayout) error                    { return nil }

// fakeResolver records what was asked of it and answers with one package.
type fakeResolver struct {
	asked   []ports.ResolveRequest
	version string
}

func (f *fakeResolver) Resolve(_ context.Context, req ports.ResolveRequest) (ports.Package, error) {
	f.asked = append(f.asked, req)
	var pkg ports.Package
	pkg.Root, pkg.Release.Bootstrap.ID, pkg.Release.Bootstrap.Version = req.Path, core.BootstrapID, f.version
	return pkg, nil
}
func (f *fakeResolver) Verify(context.Context, *ports.Package) error { return nil }

// installedProject is a project that already holds a Bootstrap of the given
// release, with an install ledger over the files in content.
func installedProject(t *testing.T, version string, content map[string]string) (*App, *fakeBootstrap, *prompter, *fakeResolver) {
	t.Helper()
	b := &fakeBootstrap{compatible: true, version: version}
	a, pr := newApp(t, b)
	res := &fakeResolver{version: version}
	a.Resolver = res
	a.Installer = fakeInstaller{ports.CoreLayout{Paths: []string{".gtt", "gtt-domain", "AGENTS.md"}, Package: []string{".gtt", "AGENTS.md"},
		Rewritten: []string{".gtt/local/", ".gtt/index/technical-index.json"}}}
	ledger := map[string]string{}
	for rel, text := range content {
		full := filepath.Join(a.Cwd, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		sum, err := fsx.HashFile(full)
		if err != nil {
			t.Fatal(err)
		}
		ledger[rel] = sum
	}
	if err := (lifecycle.Store{Root: a.Cwd}).Save(lifecycle.State{CoreLedger: ledger, Rewritten: []string{".gtt/index/artifacts.json"}}); err != nil {
		t.Fatal(err)
	}
	return a, b, pr, res
}

var packageContent = map[string]string{
	".gtt/scripts/gtt-status.sh":      "status\n",
	".gtt/docs/usage.md":              "usage\n",
	".gtt/local/last-validation.json": "{}\n",
	".gtt/index/technical-index.json": "{}\n",
	".gtt/index/artifacts.json":       "{}\n",
	"AGENTS.md":                       "contract\n",
	"gtt-domain/context/stack.md":     "template\n",
}

func change(t *testing.T, a *App, rel, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(a.Cwd, filepath.FromSlash(rel)), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInitReusesThePinnedBootstrapAndWritesNothing(t *testing.T) {
	a, b, pr, resolver := installedProject(t, core.BootstrapPin, packageContent)
	a.AssumeYes = true
	before, _ := os.ReadFile(filepath.Join(a.Cwd, ".gtt", "cli", "state.json"))

	res, err := a.Init(context.Background(), InitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	c := res.Installed
	if res.Outcome != "already-initialized" || c == nil || !c.Verified || c.Version != core.BootstrapPin || c.Required != core.BootstrapPin {
		t.Fatalf("the pinned release must be reused and reported as verified: %+v %+v", res, c)
	}
	if len(c.Modified)+len(c.Missing) != 0 {
		t.Errorf("an untouched install has no differences: %+v", c)
	}
	if len(pr.asked) != 0 || len(resolver.asked) != 0 || len(b.requests) != 0 {
		t.Errorf("reusing the release asks nothing, downloads nothing and runs no operation: %v %v %v", pr.asked, resolver.asked, b.requests)
	}
	if after, _ := os.ReadFile(filepath.Join(a.Cwd, ".gtt", "cli", "state.json")); string(after) != string(before) {
		t.Error("reusing the release must write nothing")
	}
}

func TestInitReportsCustomizedFilesAndKeepsThem(t *testing.T) {
	a, _, pr, _ := installedProject(t, core.BootstrapPin, packageContent)
	change(t, a, ".gtt/scripts/gtt-status.sh", "status\n# customized path globs\n")
	// What the Bootstrap rewrites by itself, and governed state, are not customizations.
	change(t, a, ".gtt/local/last-validation.json", `{"result":"pass"}`)
	change(t, a, ".gtt/index/technical-index.json", `{"rebuilt":true}`)
	change(t, a, ".gtt/index/artifacts.json", `{"reconciled":true}`)
	change(t, a, "gtt-domain/context/stack.md", "the project's governed stack\n")

	res, err := a.Init(context.Background(), InitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(res.Installed.Modified, ","); got != ".gtt/scripts/gtt-status.sh" {
		t.Errorf("only the customized package file is reported: %q", got)
	}
	if len(pr.asked) != 0 {
		t.Errorf("a customized file in the required release is reported, not put to a question: %v", pr.asked)
	}
	if data, _ := os.ReadFile(filepath.Join(a.Cwd, ".gtt", "scripts", "gtt-status.sh")); !strings.Contains(string(data), "customized") {
		t.Error("the customized file must be kept")
	}
}

func TestInitNeverChangesAnotherReleaseWithoutAHumanConfirmation(t *testing.T) {
	a, b, pr, _ := installedProject(t, "1.4.0", packageContent)
	a.AssumeYes = true // --yes is not that confirmation

	res, err := a.Init(context.Background(), InitOptions{Resolve: ResolveOptions{Version: "1.5.0"}})
	e, ok := err.(*core.Error)
	if !ok || e.Code != core.ExitConflict || !e.Unmodified {
		t.Fatalf("without a terminal a different release is a conflict that changes nothing: %v", err)
	}
	if !strings.Contains(e.What, "1.4.0") || !strings.Contains(e.What, "1.5.0") || !strings.Contains(e.Why, "--yes") {
		t.Errorf("the message must name the installed and the required release: %s / %s", e.What, e.Why)
	}
	if res.Installed == nil || res.Installed.Version != "1.4.0" || res.Installed.Required != "1.5.0" || res.Installed.Verified {
		t.Errorf("the mismatch must be reported: %+v", res.Installed)
	}
	if len(pr.asked) != 0 || len(b.requests) != 0 {
		t.Errorf("nothing may be asked or run: %v %v", pr.asked, b.requests)
	}

	// In a terminal the human is asked, and "no" changes nothing.
	pr.interactive, pr.answer = true, false
	if _, err := a.Init(context.Background(), InitOptions{Resolve: ResolveOptions{Version: "1.5.0"}}); err != core.ErrCancelled {
		t.Fatalf("declining must cancel: %v", err)
	}
	if len(pr.asked) != 1 || !strings.Contains(pr.asked[0], "from 1.4.0 to 1.5.0") || len(b.requests) != 0 {
		t.Errorf("exactly one explicit question, and no operation after a no: %v %v", pr.asked, b.requests)
	}
}

func TestInitRequiresThePinnedReleaseUnlessAnotherIsNamed(t *testing.T) {
	a, _, _, resolver := installedProject(t, "1.4.0", packageContent)
	res, err := a.Init(context.Background(), InitOptions{})
	if core.CodeOf(err) != core.ExitConflict || res.Installed.Required != core.BootstrapPin {
		t.Fatalf("with no flag the required release is the pinned one: %+v %v", res.Installed, err)
	}
	// An explicit package is the human's choice: its release is the required one.
	resolver.version = "1.4.0"
	res, err = a.Init(context.Background(), InitOptions{Resolve: ResolveOptions{Path: "/some/catalog"}})
	if err != nil || res.Outcome != "already-initialized" || res.Installed.Required != "1.4.0" || res.Installed.Pinned != core.BootstrapPin {
		t.Fatalf("an explicit package of the installed release is reused: %+v %v", res.Installed, err)
	}
}

func TestInitNeverDowngrades(t *testing.T) {
	a, _, pr, _ := installedProject(t, "9.0.0", packageContent)
	pr.interactive, pr.answer = true, true
	_, err := a.Init(context.Background(), InitOptions{})
	e, ok := err.(*core.Error)
	if !ok || e.Code != core.ExitIncompatible || !e.Unmodified || !strings.Contains(e.What, "9.0.0") {
		t.Fatalf("a project on a newer release is refused, never downgraded: %v", err)
	}
	if len(pr.asked) != 0 {
		t.Errorf("a downgrade is not even offered: %v", pr.asked)
	}
}

func TestInitFailsWhenTheInstalledBootstrapIsIncompleteOrBroken(t *testing.T) {
	a, b, _, _ := installedProject(t, core.BootstrapPin, packageContent)
	if err := os.Remove(filepath.Join(a.Cwd, ".gtt", "docs", "usage.md")); err != nil {
		t.Fatal(err)
	}
	res, err := a.Init(context.Background(), InitOptions{})
	e, ok := err.(*core.Error)
	if !ok || e.Code != core.ExitConflict || !strings.Contains(e.What, ".gtt/docs/usage.md") {
		t.Fatalf("a missing package file is not a verified Bootstrap: %v", err)
	}
	if res.Outcome == "already-initialized" || res.Installed.Verified {
		t.Errorf("success must not be reported: %+v", res)
	}

	a, b, _, _ = installedProject(t, core.BootstrapPin, packageContent)
	b.problems = []string{"operations.json names a script that does not exist"}
	_, err = a.Init(context.Background(), InitOptions{})
	if e, ok := err.(*core.Error); !ok || e.Code != core.ExitIntegrity || !strings.Contains(e.Why, "operations.json") {
		t.Fatalf("a Bootstrap that fails its own contract check is an integrity failure: %v", err)
	}
}

func TestResolutionDefaultsToThePinnedReleaseNeverABranch(t *testing.T) {
	if _, ok := core.ParseSemver(core.BootstrapPin); !ok {
		t.Fatalf("the pinned Bootstrap must be an exact release: %q", core.BootstrapPin)
	}
	if req := (ResolveOptions{}).request(); req.Version != core.BootstrapPin || req.Path != "" {
		t.Errorf("no flag means the pinned release: %+v", req)
	}
	if req := (ResolveOptions{Offline: true}).request(); req.Version != core.BootstrapPin || !req.Offline {
		t.Errorf("offline still means the pinned release: %+v", req)
	}
	if req := (ResolveOptions{Version: "1.4.2"}).request(); req.Version != "1.4.2" {
		t.Errorf("an explicit version is the human's choice: %+v", req)
	}
	if req := (ResolveOptions{Path: "/catalog"}).request(); req.Version != "" || req.Path != "/catalog" {
		t.Errorf("an explicit package is taken as it is: %+v", req)
	}
}

func TestVersionReportsTheCLIAndTheBootstrapItPins(t *testing.T) {
	a, _ := newApp(t, nil)
	v := a.Version(context.Background())
	if v.CLI != core.Version || v.Pinned.Version != core.BootstrapPin || v.Pinned.Tag != "v"+core.BootstrapPin || v.Bootstrap != nil {
		t.Errorf("outside a project: %+v", v)
	}
	a, _, _, _ = installedProject(t, "1.4.0", nil)
	v = a.Version(context.Background())
	if v.Bootstrap == nil || v.Bootstrap.Version != "1.4.0" || v.Bootstrap.MatchesPin || v.Pinned.Version != core.BootstrapPin {
		t.Errorf("in a project on another release: %+v %+v", v, v.Bootstrap)
	}
}
