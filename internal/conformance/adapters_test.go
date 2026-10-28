package conformance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// fakeRunner returns canned tool output, finds only the tools it is given and
// keeps the command it ran together with the configuration file it was handed.
type fakeRunner struct {
	tools  map[string]string
	result ports.Result
	err    error
	ran    *ports.Command
	config string
}

func (f *fakeRunner) Run(_ context.Context, c ports.Command) (ports.Result, error) {
	f.ran = &c
	for _, a := range c.Args {
		if i := strings.Index(a, "="); i >= 0 && strings.HasPrefix(a, "--config-file=") {
			a = a[i+1:]
		}
		if data, err := os.ReadFile(a); err == nil && filepath.IsAbs(a) {
			f.config = string(data)
		}
	}
	return f.result, f.err
}

func (f *fakeRunner) LookPath(name string) (string, error) {
	if p, ok := f.tools[name]; ok {
		return p, nil
	}
	return "", errors.New("not found")
}

// project writes the files (empty when the content is "") under a new root.
func project(t *testing.T, files ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, f := range files {
		p := filepath.Join(root, filepath.FromSlash(f))
		if strings.HasSuffix(f, "/") {
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

var sliceRows = []Row{
	{Rule: "VS-1", Module: "src/features/*", MustNot: "src/features/* (another slice)"},
	{Rule: "HX-1", Module: "src/domain", MustNot: "src/adapters"},
}

func TestTheRegistryPrefersTheSpecificManifest(t *testing.T) {
	cases := []struct {
		files []string
		id    string
	}{
		{[]string{"package.json"}, "dependency-cruiser"},
		{[]string{"go.mod", "package.json"}, "go-list"},
		{[]string{"pom.xml", "package.json"}, "jdeps"},
		{[]string{"build.gradle.kts"}, "jdeps"},
		{[]string{"composer.json", "package.json"}, "deptrac"},
		{[]string{"pyproject.toml", "package.json"}, "import-linter"},
		{[]string{"requirements-dev.txt"}, "import-linter"},
		{[]string{"App.csproj"}, ""}, // .NET: no CLI-able tool, no adapter
	}
	for _, c := range cases {
		got := ""
		if a := For(project(t, c.files...)); a != nil {
			got = a.ID()
		}
		if got != c.id {
			t.Errorf("%v: got %q want %q", c.files, got, c.id)
		}
	}
}

func TestEveryAdapterSaysHowToInstallItsTool(t *testing.T) {
	for _, a := range Adapters() {
		if a.Install() == "" {
			t.Errorf("%s has no install hint", a.ID())
		}
	}
}

func TestToolMissingIsEmpty(t *testing.T) {
	root := project(t)
	t.Setenv("JAVA_HOME", "")
	for _, a := range Adapters() {
		if got := a.Tool(root, &fakeRunner{}); got != "" {
			t.Errorf("%s: got %q", a.ID(), got)
		}
	}
	found := map[string]string{"go": "go", "jdeps": "jdeps", "deptrac": "deptrac", "lint-imports": "lint-imports", "depcruise": "depcruise"}
	for _, a := range Adapters() {
		if a.Tool(root, &fakeRunner{tools: found}) == "" {
			t.Errorf("%s: the tool on the PATH is not found", a.ID())
		}
	}
}

func TestExpandListsTheConcreteModules(t *testing.T) {
	root := project(t, "src/features/cart/", "src/features/orders/x/", "src/features/__pycache__/", "src/domain.py", "src/contexts/billing/internal/")
	cases := []struct {
		glob     string
		suffixes []string
		want     []string
	}{
		{"src/features/*", nil, []string{"src/features/cart", "src/features/orders"}},
		{"src/features/**", nil, []string{"src/features"}},
		{"src/contexts/*/internal", nil, []string{"src/contexts/billing/internal"}},
		{"src/domain", []string{".py"}, []string{"src/domain.py"}},
		{"src/missing", nil, nil},
	}
	for _, c := range cases {
		if got := Expand(root, c.glob, c.suffixes...); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v want %v", c.glob, got, c.want)
		}
	}
}

func TestMatcherNeverForbidsAModuleItself(t *testing.T) {
	ms := matchers(sliceRows, nil)
	edges := [][2]string{
		{"src/features/cart", "src/features/orders/api"}, // another slice
		{"src/features/cart/ui", "src/features/cart/db"}, // the same slice
		{"src/domain", "src/adapters/http"},
		{"src/adapters", "src/domain"},
	}
	want := []Conflict{
		{Rule: "VS-1", From: "src/features/cart", To: "src/features/orders/api"},
		{Rule: "HX-1", From: "src/domain", To: "src/adapters/http"},
	}
	if got := conflicts(ms, edges); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

// ------------------------------------------------------------------ Go

const goStream = `{"ImportPath": "example.com/shop/src/features/cart", "Imports": ["fmt", "example.com/shop/src/features/orders"]}
{"ImportPath": "example.com/shop/src/features/orders", "TestImports": ["example.com/shop/src/features/orders/internal"]}
{"ImportPath": "example.com/shop/src/domain", "XTestImports": ["example.com/shop/src/adapters"]}
`

func TestGoList(t *testing.T) {
	root := project(t, "src/features/cart/", "src/domain/")
	writeFile(t, root, "go.mod", "module example.com/shop // the shop\n\ngo 1.22\n")
	cases := []struct {
		name    string
		result  ports.Result
		want    []Conflict
		wantErr bool
	}{
		{"conflicts", ports.Result{Stdout: []byte(goStream)}, []Conflict{
			{Rule: "VS-1", From: "src/features/cart", To: "src/features/orders"},
			{Rule: "HX-1", From: "src/domain", To: "src/adapters"},
		}, false},
		{"clean", ports.Result{Stdout: []byte(`{"ImportPath": "example.com/shop/src/domain"}`)}, nil, false},
		{"unparseable", ports.Result{Stdout: []byte("go: not a module"), ExitCode: 1}, nil, true},
		{"package error", ports.Result{Stdout: []byte(`{"ImportPath": "example.com/shop/src/domain", "Error": {"Err": "syntax error"}}`)}, nil, true},
	}
	for _, c := range cases {
		r := &fakeRunner{result: c.result}
		out, err := GoList{}.Check(context.Background(), root, "go", sliceRows, r)
		if (err != nil) != c.wantErr || !reflect.DeepEqual(out.Conflicts, c.want) {
			t.Errorf("%s: got %+v, %v", c.name, out.Conflicts, err)
		}
		if r.ran == nil || !reflect.DeepEqual(r.ran.Args, []string{"list", "-e", "-json", "./..."}) {
			t.Errorf("%s: ran %+v", c.name, r.ran)
		}
	}
}

func TestGoListNeedsTheFoldersTheRulesName(t *testing.T) {
	root := project(t)
	writeFile(t, root, "go.mod", "module example.com/shop\n")
	if _, err := (GoList{}).Check(context.Background(), root, "go", sliceRows, &fakeRunner{}); err == nil {
		t.Error("no folder: cannot be determined")
	}
}

// --------------------------------------------------------------- jdeps

const jdepsReport = `classes -> java.base
classes -> not found
   com.acme.orders                                    -> com.acme.billing                                   classes
   com.acme.orders                                    -> com.acme.orders.internal                           classes
   com.acme.orders                                    -> java.util                                          java.base
   com.acme.domain                                    -> com.acme.adapters.http                             classes
`

func TestJdeps(t *testing.T) {
	rows := []Row{
		{Rule: "VS-1", Module: "src/main/java/com/acme/*", MustNot: "src/main/java/com/acme/* (another slice)"},
		{Rule: "HX-1", Module: "src/main/java/com/acme/domain", MustNot: "com.acme.adapters"},
	}
	root := project(t, "pom.xml", "src/main/java/com/acme/", "target/classes/com/acme/")
	cases := []struct {
		name    string
		result  ports.Result
		want    []Conflict
		wantErr bool
	}{
		{"conflicts", ports.Result{Stdout: []byte(jdepsReport)}, []Conflict{
			{Rule: "VS-1", From: "com.acme.orders", To: "com.acme.billing"},
			{Rule: "VS-1", From: "com.acme.domain", To: "com.acme.adapters.http"},
			{Rule: "HX-1", From: "com.acme.domain", To: "com.acme.adapters.http"},
		}, false},
		{"unparseable", ports.Result{Stdout: []byte("Error: invalid path"), ExitCode: 2}, nil, true},
	}
	for _, c := range cases {
		r := &fakeRunner{result: c.result}
		out, err := Jdeps{}.Check(context.Background(), root, "jdeps", rows, r)
		if (err != nil) != c.wantErr || !reflect.DeepEqual(out.Conflicts, c.want) {
			t.Errorf("%s: got %+v, %v", c.name, out.Conflicts, err)
		}
		if r.ran == nil || r.ran.Args[0] != "-verbose:package" || r.ran.Args[1] != filepath.Join(root, "target", "classes") {
			t.Errorf("%s: ran %+v", c.name, r.ran)
		}
	}
}

func TestJdepsNeedsCompiledClasses(t *testing.T) {
	root := project(t, "build.gradle", "src/main/kotlin/com/acme/")
	rows := []Row{{Rule: "HX-1", Module: "src/main/kotlin/com/acme/domain", MustNot: "src/main/kotlin/com/acme/adapters"}}
	_, err := Jdeps{}.Check(context.Background(), root, "jdeps", rows, &fakeRunner{})
	if err == nil || !strings.Contains(err.Error(), "build the project first") {
		t.Errorf("got %v", err)
	}
}

func TestPackageGlob(t *testing.T) {
	for in, want := range map[string]string{
		"src/main/java/com/acme/*":          "com/acme/*",
		"orders/src/main/kotlin/com/acme/x": "com/acme/x",
		"com.acme.adapters":                 "com/acme/adapters",
	} {
		if got := packageGlob(in); got != want {
			t.Errorf("%s: got %s", in, got)
		}
	}
}

// ------------------------------------------------------- import-linter

const linterBroken = `=============
Import Linter
=============

---------
Contracts
---------

Analyzed 12 files, 20 dependencies.
-----------------------------------

VS-1#1 BROKEN
VS-1#2 KEPT
HX-1#3 BROKEN

Contracts: 1 kept, 2 broken.


----------------
Broken contracts
----------------

VS-1#1
------

shop.features.cart is not allowed to import shop.features.orders:

- shop.features.cart.views -> shop.features.orders.models (l.3)


HX-1#3
------

shop.domain is not allowed to import shop.adapters:

`

func TestLinterConfigNamesModulesFromPaths(t *testing.T) {
	root := project(t, "src/shop/__init__.py", "src/shop/features/cart/", "src/shop/features/orders/", "src/shop/domain.py", "src/shop/adapters/")
	rows := []Row{
		{Rule: "VS-1", Module: "src/shop/features/*", MustNot: "src/shop/features/* (another slice)"},
		{Rule: "HX-1", Module: "src/shop/domain", MustNot: "src/shop/adapters"},
	}
	cfg, src, n := LinterConfig(root, rows)
	if !src || n != 3 {
		t.Fatalf("src layout %v, %d contracts:\n%s", src, n, cfg)
	}
	for _, want := range []string{
		"root_packages =\n    shop\n",
		"name = VS-1#1\ntype = forbidden\nsource_modules =\n    shop.features.cart\nforbidden_modules =\n    shop.features.orders\nallow_indirect_imports = True",
		"name = HX-1#3\ntype = forbidden\nsource_modules =\n    shop.domain\nforbidden_modules =\n    shop.adapters\n",
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("missing %q in:\n%s", want, cfg)
		}
	}
}

func TestImportLinter(t *testing.T) {
	root := project(t, "pyproject.toml", "shop/__init__.py", "shop/features/cart/", "shop/features/orders/", "shop/domain/", "shop/adapters/")
	rows := []Row{
		{Rule: "VS-1", Module: "shop/features/*", MustNot: "shop/features/* (another slice)"},
		{Rule: "HX-1", Module: "shop/domain", MustNot: "shop/adapters"},
	}
	cases := []struct {
		name    string
		result  ports.Result
		want    []Conflict
		wantErr bool
	}{
		{"conflicts", ports.Result{Stdout: []byte(linterBroken), ExitCode: 1}, []Conflict{
			{Rule: "VS-1", From: "shop.features.cart.views", To: "shop.features.orders.models"},
			{Rule: "HX-1", From: "shop.domain", To: "shop.adapters"},
		}, false},
		{"clean", ports.Result{Stdout: []byte("Contracts: 3 kept, 0 broken.\n")}, nil, false},
		{"unparseable", ports.Result{Stdout: []byte("Traceback (most recent call last):"), ExitCode: 1}, nil, true},
		{"broken but unreadable", ports.Result{Stdout: []byte("Contracts: 0 kept, 3 broken.\n"), ExitCode: 1}, nil, true},
	}
	for _, c := range cases {
		r := &fakeRunner{result: c.result}
		out, err := ImportLinter{}.Check(context.Background(), root, "lint-imports", rows, r)
		if (err != nil) != c.wantErr || !reflect.DeepEqual(out.Conflicts, c.want) {
			t.Errorf("%s: got %+v, %v", c.name, out.Conflicts, err)
		}
		if !strings.Contains(r.config, "[importlinter:contract:3]") || r.ran.Args[0] != "--config" {
			t.Errorf("%s: ran %+v with config %q", c.name, r.ran, r.config)
		}
	}
}

// ------------------------------------------------------------- deptrac

func TestDeptracConfigAllowsAllButTheForbidden(t *testing.T) {
	root := project(t, "src/features/cart/", "src/features/orders/", "src/domain/", "src/adapters/")
	cfg, forbidden, layers := DeptracConfig(root, sliceRows, []string{"src"})
	// layers in path order: L1 adapters, L2 domain, L3 cart, L4 orders
	if layers["L2"] != "src/domain" || layers["L3"] != "src/features/cart" {
		t.Fatalf("layers %v", layers)
	}
	want := map[string]string{"L3>L4": "VS-1", "L4>L3": "VS-1", "L2>L1": "HX-1"}
	if !reflect.DeepEqual(forbidden, want) {
		t.Errorf("forbidden %v", forbidden)
	}
	for _, line := range []string{
		"    - '" + filepath.Join(root, "src") + "'\n",
		"    - name: L3\n      collectors:\n        - type: directory\n          value: 'src/features/cart/.*'\n",
		"    L1: [L2, L3, L4]\n", "    L2: [L3, L4]\n", "    L3: [L1, L2]\n", "    L4: [L1, L2]\n",
	} {
		if !strings.Contains(cfg, line) {
			t.Errorf("missing %q in:\n%s", line, cfg)
		}
	}
}

func TestDeptrac(t *testing.T) {
	root := project(t, "composer.json", "src/features/cart/", "src/features/orders/", "src/domain/", "src/adapters/")
	report := `{"Report": {"Violations": 1, "Skipped violations": 0, "Uncovered": 0, "Allowed": 4, "Warnings": 0, "Errors": 0},
"files": {"` + filepath.ToSlash(filepath.Join(root, "src/features/cart/Checkout.php")) + `": {"violations": 1, "messages": [
{"message": "App\\Cart\\Checkout must not depend on App\\Orders\\Order (L3 on L4)", "line": 7, "type": "error"}]}}}`
	cases := []struct {
		name    string
		result  ports.Result
		want    []Conflict
		wantErr bool
	}{
		{"conflicts", ports.Result{Stdout: []byte("deptrac 2.0\n" + report), ExitCode: 1}, []Conflict{
			{Rule: "VS-1", From: "src/features/cart/Checkout.php", To: `App\Orders\Order`},
		}, false},
		{"clean", ports.Result{Stdout: []byte(`{"Report": {"Violations": 0, "Errors": 0}, "files": {}}`)}, nil, false},
		{"unparseable", ports.Result{Stdout: []byte("Command \"analyse\" is not defined."), ExitCode: 1}, nil, true},
		{"analysis errors", ports.Result{Stdout: []byte(`{"Report": {"Violations": 0, "Errors": 2}, "files": {}}`)}, nil, true},
		{"violations unreadable", ports.Result{Stdout: []byte(`{"Report": {"Violations": 2, "Errors": 0}, "files": {}}`)}, nil, true},
	}
	for _, c := range cases {
		r := &fakeRunner{result: c.result}
		out, err := Deptrac{}.Check(context.Background(), root, "deptrac", sliceRows, r)
		if (err != nil) != c.wantErr || !reflect.DeepEqual(out.Conflicts, c.want) {
			t.Errorf("%s: got %+v, %v", c.name, out.Conflicts, err)
		}
		if !strings.Contains(r.config, "ruleset:") || r.ran.Args[0] != "analyse" || r.ran.Args[2] != "--formatter=json" {
			t.Errorf("%s: ran %+v with config %q", c.name, r.ran, r.config)
		}
	}
}
