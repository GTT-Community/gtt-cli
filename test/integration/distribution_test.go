package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/core"
)

func TestCleanExportLeavesDevelopmentIntact(t *testing.T) {
	root := installed(t, map[string]string{"deploy/Dockerfile": "FROM scratch\n", ".claude/keep-me.txt": "the host's own claude file\n",
		"docs/spec.md": "# spec\n"}, "--ade", "claude", "--language", "en", "--method", "medium", "--sources", "docs/spec.md")
	before := treeHash(t, root)
	dest := filepath.Join(t.TempDir(), "delivery")
	r := gtt(t, root, "export", "--clean", dest).must(t, 0)
	if treeHash(t, root) != before {
		t.Fatal("export --clean must not modify the development project")
	}
	for _, kept := range []string{"src/main.go", "tests/main_test.go", "deploy/Dockerfile", ".claude/keep-me.txt", "docs/spec.md"} {
		if !exists(dest, kept) {
			t.Errorf("%s must be in the delivery", kept)
		}
	}
	for _, gone := range []string{".gtt", "gtt-domain", "AGENTS.md", "readme-gtt.md", "readme-gtt.es.md", ".claude/CLAUDE.md", ".claude/hooks", ".claude/settings.json"} {
		if exists(dest, gone) {
			t.Errorf("%s is GTT-owned and must not be delivered", gone)
		}
	}
	if !strings.Contains(r.out, "The project was not modified") {
		t.Errorf("got:\n%s", r.out)
	}
	gtt(t, root, "export", "--clean", dest).must(t, 1) // a non-empty destination is refused
	gtt(t, root, "validate").must(t, 0)
}

func TestExportKeepsLocallyChangedOverlayFilesVisible(t *testing.T) {
	root := installed(t, nil)
	write(t, root, ".claude/CLAUDE.md", read(t, root, ".claude/CLAUDE.md")+"\nmy local addition\n")
	dest := filepath.Join(t.TempDir(), "delivery")
	r := gtt(t, root, "export", "--clean", dest).must(t, 0)
	if !exists(dest, ".claude/CLAUDE.md") || !strings.Contains(r.out, ".claude/CLAUDE.md") {
		t.Errorf("a ledger file changed since install is not excluded silently:\n%s", r.out)
	}
}

func TestCleanNeedsConfirmationAndSnapshotRestores(t *testing.T) {
	root := installed(t, map[string]string{".claude/keep-me.txt": "mine\n", "docs/spec.md": "# spec\n"},
		"--ade", "claude,copilot", "--primary", "copilot", "--language", "es", "--method", "hard", "--sources", "docs/spec.md")
	host := map[string]string{}
	for _, f := range []string{"src/main.go", "tests/main_test.go", ".claude/keep-me.txt", "docs/spec.md"} {
		host[f] = read(t, root, f)
	}
	before := treeHash(t, root)
	gtt(t, root, "clean").must(t, 3) // no terminal, no --yes: refused
	gttIn(t, root, "y\nn\n", "clean").must(t, 4)
	if treeHash(t, root) != before {
		t.Fatal("a declined clean must remove nothing")
	}

	r := gtt(t, root, "clean", "--yes").must(t, 0)
	for _, gone := range []string{".gtt", "gtt-domain", "AGENTS.md", "readme-gtt.md", ".claude/CLAUDE.md", overlayEntry(t, "copilot")} {
		if exists(root, gone) {
			t.Errorf("%s must be removed", gone)
		}
	}
	for f, content := range host {
		if !exists(root, f) || read(t, root, f) != content {
			t.Errorf("application file %s must be untouched", f)
		}
	}
	if !strings.Contains(r.out, "Snapshot de recuperación") { // the project speaks Spanish
		t.Fatalf("a snapshot is created before a destructive clean:\n%s", r.out)
	}
	if snaps := gtt(t, root, "snapshot", "list").must(t, 0).out; !strings.Contains(snaps, "Primary copilot") {
		t.Fatalf("the snapshot must survive clean:\n%s", snaps)
	}

	// A later init detects the snapshot; without a human it does not restore.
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	r = gtt(t, root, "init", "--bootstrap", pristine).must(t, 3)
	// Nothing records the language any more: the detection is in English.
	if !strings.Contains(r.out, "GTT recovery snapshot detected") || exists(root, ".gtt") {
		t.Fatalf("detection without restore:\n%s\n%s", r.out, r.err)
	}
	r = gtt(t, root, "init", "--bootstrap", pristine, "--yes").must(t, 0)
	if !strings.Contains(r.out, "restaurada") { // the restored project speaks Spanish again
		t.Fatalf("got:\n%s", r.out)
	}
	st := gtt(t, root, "status", "--fast", "--json").must(t, 0).json(t)["status"].(map[string]any)
	if a := st["ade"].(map[string]any); a["primary"] != "copilot" || len(a["participating"].([]any)) != 2 {
		t.Errorf("ADE state must be restored: %v", a)
	}
	plan := methodology(t, root)
	if plan["profile"] != "hard" || plan["language"] != "es" {
		t.Errorf("plan and language must be restored: %v %v", plan["profile"], plan["language"])
	}
	if s := gtt(t, root, "sources").must(t, 0).out; !strings.Contains(s, "docs/spec.md") {
		t.Errorf("selected sources must be restored:\n%s", s)
	}
	if exists(root, "gtt-domain/.frozen") {
		t.Error("a restore never freezes")
	}
	for f, content := range host {
		if read(t, root, f) != content {
			t.Errorf("recovery must not touch %s", f)
		}
	}
	gtt(t, root, "validate").must(t, 0)
}

// newer builds a catalog that claims a later release with changed content.
func newer(t *testing.T, version string) string {
	c := catalogCopy(t)
	edit(t, filepath.Join(c, ".gtt/contract/release.json"), func(d map[string]any) {
		d["bootstrap"].(map[string]any)["version"] = version
	})
	usage := filepath.Join(c, ".gtt/docs/usage.md")
	data, _ := os.ReadFile(usage)
	os.WriteFile(usage, append(data, []byte("\nA paragraph added in "+version+".\n")...), 0o644)
	return c
}

func TestUpdateIsTransactionalAndPreservesProjectState(t *testing.T) {
	root := installed(t, map[string]string{"docs/spec.md": "# spec\n"},
		"--ade", "claude,copilot", "--primary", "copilot", "--language", "es", "--method", "hard", "--sources", "docs/spec.md")
	// Same version: nothing to do.
	r := gtt(t, root, "update", "--bootstrap", pristine).must(t, 0)
	if !strings.Contains(r.out, "ya está instalado") {
		t.Fatalf("got:\n%s", r.out)
	}
	// A local change to a package file must survive an update.
	write(t, root, ".gtt/docs/docs.md", read(t, root, ".gtt/docs/docs.md")+"\nlocal note\n")

	from, to := catalogVersion(t), next(t, catalogVersion(t), 1)
	target := newer(t, to)
	doc := gtt(t, root, "update", "--bootstrap", target, "--json").must(t, 0).json(t)
	if doc["outcome"] != "updated" || doc["from"] != from || doc["to"] != to {
		t.Fatalf("got %v", doc)
	}
	if !strings.Contains(read(t, root, ".gtt/docs/usage.md"), "added in "+to) {
		t.Error("changed package content must be replaced")
	}
	if !strings.Contains(read(t, root, ".gtt/docs/docs.md"), "local note") {
		t.Error("a package file changed locally must be preserved")
	}
	if !strings.Contains(strings.Join(toStrings(doc["preserved_modified"]), ","), ".gtt/docs/docs.md") {
		t.Errorf("preserved files must be reported: %v", doc["preserved_modified"])
	}
	v := gtt(t, root, "version", "--json").must(t, 0).json(t)["bootstrap"].(map[string]any)
	if v["version"] != to {
		t.Errorf("installed version: %v", v)
	}
	st := gtt(t, root, "status", "--fast", "--json").must(t, 0).json(t)["status"].(map[string]any)
	if a := st["ade"].(map[string]any); a["primary"] != "copilot" || len(a["participating"].([]any)) != 2 {
		t.Errorf("ADE state must survive: %v", a)
	}
	if plan := methodology(t, root); plan["profile"] != "hard" || plan["language"] != "es" {
		t.Errorf("plan and language must survive: %v", plan)
	}
	if s := gtt(t, root, "sources").must(t, 0).out; !strings.Contains(s, "Sin cambios") {
		t.Errorf("selected sources and their tracking must survive:\n%s", s)
	}
	if exists(root, ".gtt-update-backup") {
		t.Error("the backup must be removed after a committed update")
	}
	if a := gtt(t, root, "audit").must(t, 0).out; !strings.Contains(a, "update "+from+" -> "+to) {
		t.Errorf("the update must be traceable:\n%s", a)
	}
	// Downgrade is refused.
	gtt(t, root, "update", "--bootstrap", pristine).must(t, 2)
}

func TestUpdateRefusalsAndRollbackLeaveTheProjectUnchanged(t *testing.T) {
	root := installed(t, nil)
	log := filepath.Join(root, ".gtt/cli/operations.log")
	os.Remove(log)
	before := treeHash(t, root)
	same := func(name string) {
		t.Helper()
		os.Remove(log)
		if h := treeHash(t, root); h != before {
			t.Errorf("%s: the project changed", name)
		}
	}

	to := next(t, catalogVersion(t), 1)
	incompatible := newer(t, to)
	// A Bootstrap that needs a CLI newer than the one under test, whatever its version.
	cli, _ := core.ParseSemver(core.Version)
	required := fmt.Sprintf("%d.%d.0", cli[0], cli[1]+1)
	edit(t, filepath.Join(incompatible, ".gtt/contract/release.json"), func(d map[string]any) {
		d["compatibility"].(map[string]any)["cli"].(map[string]any)["min_version"] = required
	})
	r := gtt(t, root, "update", "--bootstrap", incompatible).must(t, 2)
	if !strings.Contains(r.err, required) {
		t.Errorf("got:\n%s", r.err)
	}
	same("incompatible CLI")

	profile := newer(t, to)
	edit(t, filepath.Join(profile, ".gtt/contract/capabilities.json"), func(d map[string]any) {
		for _, c := range d["capabilities"].([]any) {
			if m := c.(map[string]any); m["id"] == "methodology.profile" {
				m["version"] = 2
			}
		}
	})
	gtt(t, root, "update", "--bootstrap", profile).must(t, 2)
	same("incompatible profile contract")

	broken := newer(t, to)
	os.Remove(filepath.Join(broken, ".gtt/scaffold/manifest.yaml"))
	gtt(t, root, "update", "--bootstrap", broken).must(t, 6)
	same("failed integrity")

	// A target whose mechanics fail after being applied (here the index cannot be rebuilt) is rolled back.
	failing := newer(t, to)
	os.WriteFile(filepath.Join(failing, ".gtt/scripts/gtt-index.sh"), []byte("#!/usr/bin/env bash\necho 'FAIL  injected' >&2\nexit 1\n"), 0o755)
	r = gtt(t, root, "update", "--bootstrap", failing).must(t, 1)
	if !strings.Contains(r.err, "rolled back") {
		t.Errorf("got:\n%s", r.err)
	}
	if exists(root, ".gtt-update-backup") {
		t.Error("no backup may be left behind after a rollback")
	}
	if v := gtt(t, root, "version", "--json").must(t, 0).json(t)["bootstrap"].(map[string]any); v["version"] != catalogVersion(t) {
		t.Errorf("still on the previous release: %v", v)
	}
	if read(t, root, ".gtt/scripts/gtt-index.sh") != read(t, pristine, ".gtt/scripts/gtt-index.sh") {
		t.Error("the replaced script must be restored")
	}
	gtt(t, root, "validate").must(t, 0)
}

// What the new validators say about the project's own content is work to do, not a reason to stay on the old release.
func TestUpdateWithValidatorFindingsStillUpdatesAndReportsThem(t *testing.T) {
	root := installed(t, nil)
	to := next(t, catalogVersion(t), 1)
	failing := newer(t, to)
	os.WriteFile(filepath.Join(failing, ".gtt/scripts/gtt-check-backlog.sh"), []byte("#!/usr/bin/env bash\necho 'FAIL  injected finding' >&2\nexit 1\n"), 0o755)
	doc := gtt(t, root, "update", "--bootstrap", failing, "--json").must(t, 0).json(t)
	if doc["outcome"] != "updated" || doc["to"] != to {
		t.Fatalf("the update must stand: %v", doc)
	}
	if f, _ := doc["findings"].([]any); len(f) == 0 {
		t.Errorf("the findings must be reported: %v", doc)
	}
	if v := gtt(t, root, "version", "--json").must(t, 0).json(t)["bootstrap"].(map[string]any); v["version"] != to {
		t.Errorf("must be on the new release: %v", v)
	}
	if exists(root, ".gtt-update-backup") {
		t.Error("no backup may be left behind")
	}
}

// A project that was cloned, moved or shared has no install ledger and no copy of the catalog it came from.
// `gtt update` rebuilds the ledger from the published release tag, with no --base, and brings back what is gone.
func TestUpdateWithoutALedgerRebuildsItFromThePublishedRelease(t *testing.T) {
	root := installed(t, nil)
	from, to := catalogVersion(t), next(t, catalogVersion(t), 1)

	// The "published" source: a git repository holding the installed release under its tag.
	source := filepath.Join(t.TempDir(), "published")
	if err := copyTree(pristine, source); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.org", "-c", "commit.gpgsign=false", "-C", source}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "--quiet")
	git("add", "-A")
	git("commit", "--quiet", "-m", "release")
	git("tag", core.BootstrapTag(from))
	t.Setenv("GTT_BOOTSTRAP_SOURCE", source)

	os.Remove(filepath.Join(root, ".gtt/cli/state.json")) // the ledger is gone
	os.Remove(filepath.Join(root, ".gtt/docs/usage.md"))  // and so is a file of the package
	target := newer(t, to)
	doc := gtt(t, root, "update", "--bootstrap", target, "--json").must(t, 0).json(t)
	if doc["outcome"] != "updated" || doc["from"] != from || doc["to"] != to {
		t.Fatalf("got %v", doc)
	}
	if !strings.Contains(read(t, root, ".gtt/docs/usage.md"), "added in "+to) {
		t.Error("the missing file must come back, in the new release")
	}
	gtt(t, root, "validate").must(t, 0)
}

// `gtt update` never touches gtt-domain/: it keeps a zip of it outside the project and says when the new Bootstrap asks more of it.
func TestUpdateKeepsAZipOfTheDomainAndSaysWhenItMustBeMigrated(t *testing.T) {
	root := installed(t, nil)
	legacy := "# Project Backlog\n\n## Epics\n\n### EPIC-001 — Pagos\n\n**Status:** Planned\n**Goal:** Cobrar.\n**Approved:** ana - 2026-01-02\n\n" +
		"#### Stories\n\n##### STORY-001 — Cobro\n\n- **Status:** In Progress\n- **Implements:** (none)\n"
	write(t, root, "gtt-domain/backlog.md", legacy)
	to := next(t, catalogVersion(t), 1)
	doc := gtt(t, root, "update", "--bootstrap", newer(t, to), "--json").must(t, 0).json(t)
	if doc["outcome"] != "updated" {
		t.Fatalf("got %v", doc)
	}
	dom, _ := doc["domain"].(map[string]any)
	if dom == nil || dom["needed"] != true {
		t.Fatalf("the update must say the domain needs a migration: %v", doc)
	}
	zipped, _ := doc["domain_backup"].(string)
	if zipped == "" || !exists("", zipped) {
		t.Fatalf("a zip of the domain must be kept: %v", doc["domain_backup"])
	}
	if strings.HasPrefix(zipped, root) {
		t.Errorf("the zip must live outside the project: %s", zipped)
	}
	if read(t, root, "gtt-domain/backlog.md") != legacy {
		t.Error("an update never touches gtt-domain/")
	}
	// A project whose backlog already satisfies the Bootstrap is not told anything.
	clean := installed(t, nil)
	doc = gtt(t, clean, "update", "--bootstrap", newer(t, next(t, catalogVersion(t), 1)), "--json").must(t, 0).json(t)
	if doc["domain"] != nil {
		t.Errorf("nothing to migrate must say nothing: %v", doc["domain"])
	}
}
