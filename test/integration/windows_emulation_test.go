package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Windows process creation, emulated.
//
// On Windows a program started by name (CreateProcess, which is what Python's
// subprocess uses) is looked up in the system directory BEFORE the PATH, and
// that directory holds bash.exe, the WSL launcher. A shell resolves names
// through the PATH, so Git Bash starting "bash" gets itself; Python starting
// "bash" gets WSL, which cannot run a script by its Windows path.
//
// The emulation reproduces exactly that asymmetry on any system: a
// sitecustomize module makes every Python process resolve a bare program
// name in a fake system directory first, and that directory holds a "bash"
// that fails the way the WSL launcher does. Shells are left alone.

const wslFailure = "Windows Subsystem for Linux has no installed distributions."

// emulateWindowsProcessCreation arranges the environment and returns the
// bash the CLI must resolve and hand over (GTT_BASH), as it does on Windows.
func emulateWindowsProcessCreation(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("this is the emulation; on Windows the real behaviour is under test")
	}
	realBash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not available")
	}
	dir := t.TempDir()
	system32, site := filepath.Join(dir, "System32"), filepath.Join(dir, "site")
	write(t, system32, "bash", "#!/bin/sh\necho '"+wslFailure+"' >&2\nexit 1\n")
	if err := os.Chmod(filepath.Join(system32, "bash"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, site, "sitecustomize.py", `import os, subprocess

_system = os.environ["GTT_TEST_SYSTEM32"]
_init = subprocess.Popen.__init__


def _create_process(self, args, *rest, **kwargs):
    # CreateProcess: a bare name is found in the system directory first.
    if isinstance(args, (list, tuple)) and args and isinstance(args[0], str) and os.sep not in args[0]:
        candidate = os.path.join(_system, args[0])
        if os.path.isfile(candidate):
            args = [candidate] + list(args[1:])
    _init(self, args, *rest, **kwargs)


subprocess.Popen.__init__ = _create_process
`)
	t.Setenv("GTT_TEST_SYSTEM32", system32)
	t.Setenv("PYTHONPATH", site)
	// What the CLI finds on Windows: Git Bash, by its full path.
	t.Setenv("GTT_BASH", realBash)
}

// bashFix is the correction proposed to the Bootstrap for this defect. It
// lives with the development material and is absent from a published tree.
func bashFix(t *testing.T) string {
	t.Helper()
	patch, err := filepath.Abs(filepath.Join("..", "..", "reports", "bootstrap-1.5.0-windows-bash.patch"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(patch); err != nil {
		t.Skip("the proposed Bootstrap correction is not in this tree (reports/ is development material)")
	}
	return patch
}

// corrected returns a copy of the catalog that carries the correction: the
// catalog itself when it already does, the patched copy otherwise.
func corrected(t *testing.T, patch string) string {
	t.Helper()
	catalog := catalogCopy(t)
	apply := func(args ...string) error {
		cmd := exec.Command("git", append(append([]string{"apply", "--whitespace=nowarn"}, args...), patch)...)
		cmd.Dir = catalog
		return cmd.Run()
	}
	if apply("--check", "--reverse") == nil {
		return catalog // already corrected
	}
	if err := apply(); err != nil {
		t.Skipf("the proposed correction neither applies to nor is part of this Bootstrap catalog (%s)", catalogVersion(t))
	}
	return catalog
}

// The CLI's part: it resolves a bash that is not the WSL launcher and hands
// it to the Bootstrap as GTT_BASH on every contract call. With a Bootstrap
// that runs its scripts with that bash, a whole project lifecycle works
// under Windows process creation.
func TestWindowsEmulationTheCLIHandsItsBashToTheBootstrap(t *testing.T) {
	patch := bashFix(t)
	catalog := corrected(t, patch)
	root := project(t, map[string]string{"docs/spec.md": "# spec\n"})
	emulateWindowsProcessCreation(t)

	initProject(t, root, catalog, "--ade", "claude,copilot", "--primary", "claude", "--language", "es", "--method", "light", "--sources", "docs/spec.md")
	gtt(t, root, "validate", "--ci").must(t, 0)
	gtt(t, root, "status", "--json").must(t, 0)
	gtt(t, root, "doctor").must(t, 0)
	gtt(t, root, "agents", "check", "--ci").must(t, 0)
	gtt(t, root, "sync").must(t, 0)
	if r := gtt(t, root, "init", "--bootstrap", catalog, "--yes"); r.code != 0 || !strings.Contains(r.out, "presente y verificado") {
		t.Errorf("a second init must reuse the verified Bootstrap:\n%s\n%s", r.out, r.err)
	}
	// The same lifecycle through an update to a later release of the corrected Bootstrap.
	later := corrected(t, patch)
	want := next(t, catalogVersion(t), 1)
	edit(t, filepath.Join(later, ".gtt/contract/release.json"), func(d map[string]any) {
		d["bootstrap"].(map[string]any)["version"] = want
	})
	gtt(t, root, "update", "--bootstrap", later).must(t, 0)
	gtt(t, root, "validate", "--ci").must(t, 0)
}

// The Bootstrap's part: it must run its scripts with the bash it was handed.
// A Bootstrap that starts "bash" by name fails here exactly as it does on
// Windows; that is the Bootstrap's defect to correct, and the CLI must not
// patch around it. The test passes for real once the Bootstrap under test
// carries the correction.
func TestWindowsEmulationTheBootstrapUnderTest(t *testing.T) {
	root := project(t, nil)
	emulateWindowsProcessCreation(t)
	before := treeHash(t, root)
	r := gtt(t, root, append([]string{"init", "--bootstrap", pristine, "--yes"}, defaults...)...)
	if r.code == 0 {
		gtt(t, root, "validate", "--ci").must(t, 0)
		return
	}
	if !strings.Contains(r.err, wslFailure) {
		t.Fatalf("unexpected failure under Windows process creation (exit %d):\n%s", r.code, r.err)
	}
	// The failure is the known one, and it must be a clean one.
	if !strings.Contains(r.err, "rolled back") || treeHash(t, root) != before {
		t.Errorf("a Bootstrap that cannot run on this system must leave the project untouched:\n%s", r.err)
	}
	t.Skipf("blocked by the Bootstrap: GTT Bootstrap %s starts \"bash\" by name and ignores GTT_BASH, so on Windows it runs the WSL launcher "+
		"(reports/BUG-bootstrap-windows-wsl-bash.md; correction: reports/bootstrap-1.5.0-windows-bash.patch)", catalogVersion(t))
}
