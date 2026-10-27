package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Method Governance: the development method is asked at init with the
// Bootstrap's own options and texts, recorded through the Bootstrap, shown
// read-only by `gtt dev-method`, and verified in the code by `gtt check`.

func devMethod(t *testing.T, root string) map[string]any {
	t.Helper()
	return gtt(t, root, "dev-method", "--json").must(t, 0).json(t)
}

func inForce(t *testing.T, root string) map[string]any {
	t.Helper()
	return devMethod(t, root)["in_force"].(map[string]any)
}

func TestDevMethodIsAskedWithThePacksTextsAndRecorded(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, append(defaults, "--dev-method", "ddd,vertical-slice", "--start", "walking-skeleton")...)
	f := inForce(t, root)
	if f["status"] != "selected" || f["source"] != "setup" || strings.Join(toStrings(f["style"]), ",") != "ddd,vertical-slice" || f["start"] != "walking-skeleton" {
		t.Errorf("the answer is recorded as given, and it is a setup answer, not the lock: %v", f)
	}
	list := gtt(t, root, "dev-method", "list", "--lang", "es", "--json").must(t, 0).json(t)
	pack := read(t, pristine, ".gtt/methods/vertical-slice.md")
	text := strings.SplitN(strings.SplitN(pack, "## Qué es\n\n", 2)[1], "\n", 2)[0]
	var found map[string]any
	questions := list["questions"].([]any)
	for _, o := range questions[0].(map[string]any)["options"].([]any) {
		if o.(map[string]any)["id"] == "vertical-slice" {
			found = o.(map[string]any)
		}
	}
	if len(questions) != 2 || found == nil || !strings.HasPrefix(found["what_it_is"].(string), text) || found["recommended"] != true {
		t.Errorf("the options and their texts are the Method Packs' own: %v", found)
	}
}

func TestANewMethodPackNeedsNoCLIChange(t *testing.T) {
	catalog := catalogCopy(t)
	text := read(t, catalog, ".gtt/methods/event-first.md")
	text = strings.NewReplacer("id: event-first", "id: cell-based", "name: Event-First", "name: Cell-Based").Replace(text)
	write(t, catalog, ".gtt/methods/cell-based.md", regexp.MustCompile(`EF-(\d)`).ReplaceAllString(text, "CB-$1"))
	root := project(t, nil)
	initProject(t, root, catalog, append(defaults, "--dev-method", "cell-based")...)
	if out := gtt(t, root, "dev-method", "list").must(t, 0).out; !strings.Contains(out, "Cell-Based [cell-based]") {
		t.Errorf("a pack added to the Bootstrap is an option of the questionnaire:\n%s", out)
	}
	if got := toStrings(inForce(t, root)["style"]); len(got) != 1 || got[0] != "cell-based" {
		t.Errorf("and it can be chosen: %v", got)
	}
}

func TestAnIncompatibleDevMethodIsRefusedAndNothingIsInstalled(t *testing.T) {
	root := project(t, nil)
	before := treeHash(t, root)
	r := gtt(t, root, append([]string{"init", "--bootstrap", pristine, "--yes"}, append(defaults, "--dev-method", "vertical-slice,layered")...)...).must(t, 3)
	if !strings.Contains(r.err, "incompatible") || !strings.Contains(r.err, "No project files were modified") {
		t.Errorf("the refusal names the incompatibility and says nothing changed:\n%s", r.err)
	}
	if treeHash(t, root) != before {
		t.Error("the project was modified")
	}
}

func TestWithoutAnAnswerTheDevMethodStaysOpen(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	if exists(root, ".gtt/development-method.json") {
		t.Error("no answer, nothing recorded: the recommended method is never applied by default")
	}
	if f := inForce(t, root); f["status"] != "not declared" || strings.Join(toStrings(f["style"]), ",") != "OPEN" {
		t.Errorf("the method is OPEN: %v", f)
	}
	help := project(t, nil)
	initProject(t, help, pristine, append(defaults, "--dev-method", "help")...)
	if f := inForce(t, help); f["status"] != "open" || strings.Join(toStrings(f["style"]), ",") != "OPEN" {
		t.Errorf("help me choose decides nothing: %v", f)
	}
}

func TestDevMethodHasNoWayToChangeTheMethod(t *testing.T) {
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	if r := gtt(t, root, "dev-method", "set", "ddd"); r.code == 0 {
		t.Error("gtt dev-method is read-only: a ratified method changes by change request")
	}
}

// ratify writes a ratified Method Lock and its rules, as the bootstrap would.
func ratify(t *testing.T, root string) {
	t.Helper()
	arch := read(t, root, "gtt-domain/context/architecture.md")
	arch = regexp.MustCompile("(?s)```gtt-method\n.*?```").ReplaceAllString(arch,
		"```gtt-method\nstyle: vertical-slice\nstart: walking-skeleton\nstatus: ratified\nlocked_by: ADR-002\nverification: partial\n```")
	write(t, root, "gtt-domain/context/architecture.md", arch)
	stack := strings.Replace(read(t, root, "gtt-domain/context/stack.md"),
		"| Module | May depend on | Must not depend on | Rule |\n|---|---|---|---|\n| | | | |",
		"| Module | May depend on | Must not depend on | Rule |\n|---|---|---|---|\n"+
			"| src/features/* | src/shared/** | src/features/* (another slice) | VS-1 |\n| src/shared/** | - | src/features/** | VS-2 |", 1)
	write(t, root, "gtt-domain/context/stack.md", stack)
}

func TestCheckIsDeclaredOnlyWithoutAnAdapterForTheLanguage(t *testing.T) {
	root := project(t, nil) // a Go project: no adapter yet
	initProject(t, root, pristine, defaults...)
	if r := gtt(t, root, "check", "--json").must(t, 0).json(t); r["status"] != "not-declared" {
		t.Errorf("no method declared, nothing to check: %v", r["status"])
	}
	ratify(t, root)
	r := gtt(t, root, "check", "--json").must(t, 0).json(t)
	if r["status"] != "declared-only" || r["verification"] != "declared-only" {
		t.Errorf("without an adapter the method is declared-only, never a pass: %v", r)
	}
	if out := gtt(t, root, "check").must(t, 0).out; !strings.Contains(out, "not verified in the code") {
		t.Errorf("it says so plainly:\n%s", out)
	}
}

func TestCheckCannotBeDeterminedWithoutTheTool(t *testing.T) {
	root := project(t, map[string]string{"package.json": "{\"name\":\"shop\"}\n", "src/features/cart/index.js": "export const a = 1;\n"})
	initProject(t, root, pristine, defaults...)
	ratify(t, root)
	r := gtt(t, root, "check").must(t, 7)
	if !strings.Contains(r.err, "dependency-cruiser is not installed") {
		t.Errorf("a missing tool is never reported as a pass:\n%s", r.err)
	}
}

// The real tool: set GTT_TEST_NODE_MODULES to a node_modules directory that
// holds dependency-cruiser (npm install dependency-cruiser), with node on the PATH.
func TestCheckReportsAForbiddenDependencyAsAConflict(t *testing.T) {
	modules := os.Getenv("GTT_TEST_NODE_MODULES")
	node := ""
	if modules != "" {
		for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
			if _, err := os.Stat(filepath.Join(dir, "node")); err == nil {
				node = dir
				break
			}
		}
	}
	if modules == "" || node == "" {
		t.Skip("set GTT_TEST_NODE_MODULES to a node_modules with dependency-cruiser, and have node on the PATH")
	}
	root := project(t, map[string]string{
		"package.json":                   "{\"name\":\"shop\"}\n",
		"src/features/cart/index.js":     "import { fmt } from '../../shared/fmt.js';\nexport const cart = () => fmt();\n",
		"src/features/checkout/index.js": "export const total = () => 1;\n",
		"src/shared/fmt.js":              "export const fmt = () => '';\n",
	})
	t.Setenv("PATH", os.Getenv("PATH")+string(os.PathListSeparator)+node)
	if err := os.Symlink(modules, filepath.Join(root, "node_modules")); err != nil {
		t.Fatal(err)
	}
	initProject(t, root, pristine, defaults...)
	ratify(t, root)
	if r := gtt(t, root, "check", "--json").must(t, 0).json(t); r["status"] != "pass" || r["verification"] != "verified" {
		t.Errorf("a slice that uses only the shared kernel passes: %v", r)
	}
	write(t, root, "src/features/cart/index.js", "import { total } from '../checkout/index.js';\nexport const cart = () => total();\n")
	code, context := treeHash(t, filepath.Join(root, "src")), treeHash(t, filepath.Join(root, "gtt-domain"))
	r := gtt(t, root, "check").must(t, 1)
	if !strings.Contains(r.out, "CONFLICT VS-1: src/features/cart/index.js -> src/features/checkout/index.js") {
		t.Errorf("the conflict names the rule, the file and the forbidden dependency:\n%s", r.out)
	}
	if treeHash(t, filepath.Join(root, "src")) != code || treeHash(t, filepath.Join(root, "gtt-domain")) != context {
		t.Error("a conflict is reported, never corrected: the code or the context was modified")
	}
}
