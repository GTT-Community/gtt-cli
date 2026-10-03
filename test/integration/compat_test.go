package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func refused(t *testing.T, catalog string, code int, wants ...string) {
	t.Helper()
	root := project(t, nil)
	before := treeHash(t, root)
	r := gtt(t, root, append([]string{"init", "--bootstrap", catalog, "--yes"}, defaults...)...).must(t, code)
	for _, w := range wants {
		if !strings.Contains(r.err, w) {
			t.Errorf("the error must say %q:\n%s", w, r.err)
		}
	}
	if !strings.Contains(r.err, "No project files were modified") || treeHash(t, root) != before {
		t.Errorf("a refusal must leave the project untouched:\n%s", r.err)
	}
}

func TestBootstrapRequiringANewerCLIIsRefused(t *testing.T) {
	c := catalogCopy(t)
	edit(t, filepath.Join(c, ".gtt/contract/release.json"), func(d map[string]any) {
		d["compatibility"].(map[string]any)["cli"].(map[string]any)["min_version"] = "9.0.0"
	})
	refused(t, c, 2, "not compatible", "9.0.0")
}

func TestUnsupportedCapabilityIsRefusedNotGuessed(t *testing.T) {
	c := catalogCopy(t)
	edit(t, filepath.Join(c, ".gtt/contract/release.json"), func(d map[string]any) {
		d["requires_cli_capabilities"] = append(d["requires_cli_capabilities"].([]any), "new-runtime-capability-X")
	})
	refused(t, c, 2, "new-runtime-capability-X")
}

func TestNewerCapabilityContractIsRefused(t *testing.T) {
	c := catalogCopy(t)
	edit(t, filepath.Join(c, ".gtt/contract/capabilities.json"), func(d map[string]any) {
		for _, cap := range d["capabilities"].([]any) {
			if m := cap.(map[string]any); m["id"] == "methodology.profile" {
				m["version"] = 2
			}
		}
	})
	refused(t, c, 2, "methodology.profile.v2")
}

func TestUnknownSchemaIsRefused(t *testing.T) {
	c := catalogCopy(t)
	edit(t, filepath.Join(c, ".gtt/contract/release.json"), func(d map[string]any) {
		d["bootstrap"].(map[string]any)["schema_version"] = 2
		d["compatibility"].(map[string]any)["schema"].(map[string]any)["version"] = 2
	})
	refused(t, c, 2, "schema")
}

func TestInconsistentContractsAreAnIntegrityFailure(t *testing.T) {
	c := catalogCopy(t)
	// A registry that smuggles in an unfreeze operation is not a valid Bootstrap.
	edit(t, filepath.Join(c, ".gtt/contract/operations.json"), func(d map[string]any) {
		ops := d["operations"].(map[string]any)
		ops["unfreeze"] = ops["freeze"]
	})
	refused(t, c, 6, "integrity")
}

func TestPackageWithEscapingSymlinkIsRefused(t *testing.T) {
	c := catalogCopy(t)
	if err := os.Symlink("/etc", filepath.Join(c, ".gtt", "docs", "etc")); err != nil {
		t.Skip("symlinks unavailable")
	}
	refused(t, c, 6, "integrity")
}

func TestNotABootstrapPackage(t *testing.T) {
	refused(t, t.TempDir(), 6, "Not a GTT Bootstrap package")
}

func TestOfflineWithoutACacheFailsClearly(t *testing.T) {
	root := project(t, nil)
	r := gtt(t, root, append([]string{"init", "--offline", "--yes"}, defaults...)...).must(t, 7)
	if !strings.Contains(r.err, "offline") || exists(root, ".gtt") {
		t.Errorf("got:\n%s", r.err)
	}
}

func TestTamperedCacheIsAnIntegrityFailure(t *testing.T) {
	root := project(t, nil)
	cache := filepath.Join(os.Getenv("GTT_HOME"), "cache", "bootstrap")
	if err := copyTree(pristine, filepath.Join(cache, "1.2.0")); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(cache, "1.2.0.sha256"), []byte("0000\n"), 0o644)
	r := gtt(t, root, append([]string{"init", "--offline", "--yes"}, defaults...)...).must(t, 6)
	if !strings.Contains(r.err, "checksum mismatch") || exists(root, ".gtt") {
		t.Errorf("got:\n%s", r.err)
	}
}

// The mandatory evolution test: the Bootstrap changes content that stays
// inside the existing capability surface, and the unchanged CLI consumes it.
func TestBootstrapEvolvesWithoutACLIChange(t *testing.T) {
	c := catalogCopy(t)
	tpl := filepath.Join(c, ".gtt/scaffold/templates/gtt-initial-design-questionnaire.md")
	data, _ := os.ReadFile(tpl)
	evolved := string(data) + "\n## 99. A question added by a later Bootstrap\n\nAnswer here.\n"
	os.WriteFile(tpl, []byte(evolved), 0o644)
	edit(t, filepath.Join(c, ".gtt/contract/profiles.json"), func(d map[string]any) {
		for _, s := range d["supported"].([]any) {
			if m := s.(map[string]any); m["id"] == "light" {
				m["label"] = "Featherweight Method"
			}
		}
		plan := d["profiles"].(map[string]any)["light"].(map[string]any)["plan"].(map[string]any)
		plan["label"], plan["summary"] = "Featherweight Method", "A summary rewritten by the Bootstrap."
	})
	root := project(t, nil)
	// The plan prompt shows the Bootstrap's wording, whatever it is.
	answers := "y\n1\n1\ny\n1\ny\n2\n" // project? ADE 1; language 1; questionnaire yes; plan 1; proceed; exit menu
	r := gttIn(t, root, answers, "init", "--bootstrap", c).must(t, 0)
	if !strings.Contains(r.err, "Featherweight Method") || !strings.Contains(r.err, "A summary rewritten by the Bootstrap.") {
		t.Errorf("plan names and descriptions come from the Bootstrap:\n%s", r.err)
	}
	if read(t, root, "gtt-domain/proposals/bootstrap/initial-design-questionnaire.md") != evolved {
		t.Error("the evolved questionnaire must be materialized as is")
	}
	if got := gtt(t, root, "method").must(t, 0).out; !strings.Contains(got, "Featherweight Method") {
		t.Errorf("got:\n%s", got)
	}
}

// The CLI source must not carry the questionnaire, the ADE catalog, plan
// semantics or the exclusion list.
func TestCLISourceHoldsNoMethodology(t *testing.T) {
	forbidden := []string{"Operating Contract for the ADE", "Minimum Viable Governed Design", "You are an architect",
		"copilot-instructions", "warnings_block_freeze", "provenance_policy", "\"Kiro\"", "\"Codex\"", "Claude Code"}
	root := filepath.Join("..", "..")
	for _, dir := range []string{"internal", "cmd"} {
		filepath.Walk(filepath.Join(root, dir), func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			data, _ := os.ReadFile(p)
			for _, f := range forbidden {
				if strings.Contains(string(data), f) {
					t.Errorf("%s contains %q", p, f)
				}
			}
			return nil
		})
	}
	for _, pkg := range []string{"governance", "architecture", "adr", "reasoning", "think", "gtt-rules"} {
		if _, err := os.Stat(filepath.Join(root, "internal", pkg)); err == nil {
			t.Errorf("internal/%s must not exist", pkg)
		}
	}
}

// trust installs a release key in the user's GTT home: from then on every
// Bootstrap package must carry a signature that verifies against it.
func trust(t *testing.T, keys string) {
	t.Helper()
	dir := filepath.Join(os.Getenv("GTT_HOME"), "trusted-keys")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(keys, "release.pub"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "release.pub"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSignedBootstrapIsVerifiedWhenAKeyIsTrusted(t *testing.T) {
	root, keys, c := project(t, nil), t.TempDir(), catalogCopy(t)
	gtt(t, root, "release", "keygen", "--id", "release", "--out", keys).must(t, 0)
	gtt(t, root, "release", "sign", c, "--key", filepath.Join(keys, "release.key")).must(t, 0)
	trust(t, keys)
	initProject(t, root, c, defaults...)
	if out := gtt(t, root, "inspect").must(t, 0).out; !strings.Contains(out, "Signature: verified (ed25519, key release)") {
		t.Errorf("inspect must report the verified signature:\n%s", out)
	}
}

func TestSignatureFailuresAreIntegrityFailures(t *testing.T) {
	keys := t.TempDir()
	for name, prepare := range map[string]func(t *testing.T, c string){
		"unsigned package": func(t *testing.T, c string) {},
		"content changed after signing": func(t *testing.T, c string) {
			gtt(t, c, "release", "sign", c, "--key", filepath.Join(keys, "release.key")).must(t, 0)
			write(t, c, ".gtt/scaffold/templates/gtt-initial-design-questionnaire.md", "# replaced\n")
		},
		"signed with an untrusted key": func(t *testing.T, c string) {
			other := t.TempDir()
			gtt(t, c, "release", "keygen", "--id", "release", "--out", other).must(t, 0)
			gtt(t, c, "release", "sign", c, "--key", filepath.Join(other, "release.key")).must(t, 0)
		},
	} {
		t.Run(name, func(t *testing.T) {
			root, c := project(t, nil), catalogCopy(t)
			if !exists(keys, "release.key") {
				gtt(t, root, "release", "keygen", "--id", "release", "--out", keys).must(t, 0)
			}
			prepare(t, c)
			trust(t, keys)
			r := gtt(t, root, append([]string{"init", "--bootstrap", c, "--yes"}, defaults...)...).must(t, 6)
			if !strings.Contains(r.err, "release signature") || exists(root, ".gtt") {
				t.Errorf("refused before any change:\n%s", r.err)
			}
		})
	}
}

// Archives are not a distribution format: a package is a directory tree. An
// archive handed over as a Bootstrap is refused before anything is read
// from it, so its paths are never extracted.
func TestArchiveIsRefusedNotExtracted(t *testing.T) {
	root := project(t, nil)
	archive := filepath.Join(t.TempDir(), "gtt-bootstrap.tar.gz")
	if err := os.WriteFile(archive, []byte("\x1f\x8b\x08\x00 ../../etc/passwd"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := treeHash(t, root)
	gtt(t, root, append([]string{"init", "--bootstrap", archive, "--yes"}, defaults...)...).must(t, 3)
	if treeHash(t, root) != before {
		t.Error("the project must be left unmodified")
	}
}
