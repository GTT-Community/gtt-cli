// Package arch enforces the dependency rules of the governed architecture
// map (gtt-domain/context/stack.md, section 5) and the boundary with the
// Bootstrap: no package may hold GTT semantics.
package arch

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const module = "github.com/GTT-Community/gtt-cli/internal/"

var (
	kernel   = []string{"core", "fsx", "ports"}
	services = []string{"ade", "project", "methodology", "tracking", "agents", "lifecycle", "snapshot", "export"}
	adapters = []string{"bootstrap", "execution", "ui"}
)

// allowed lists, per package, the internal packages it may import.
func allowed() map[string][]string {
	rules := map[string][]string{
		"core": nil, "fsx": nil, "ports": nil, "output": nil,
		"app":     append(append([]string{}, kernel...), services...),
		"cli":     append([]string{"app", "output", "agents"}, kernel...),
		"compose": append(append(append([]string{"app", "cli", "output"}, kernel...), services...), adapters...),
	}
	for _, s := range services {
		rules[s] = append(append([]string{}, kernel...), services...)
	}
	for _, a := range adapters {
		rules[a] = kernel
	}
	return rules
}

func imports(t *testing.T, dir string) map[string]bool {
	out := map[string]bool{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, e.Name()), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			out[strings.Trim(imp.Path.Value, `"`)] = true
		}
	}
	return out
}

func TestDependenciesPointInward(t *testing.T) {
	root := filepath.Join("..", "..", "internal")
	rules := allowed()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		pkg := e.Name()
		permitted, known := rules[pkg]
		if !known {
			t.Errorf("internal/%s is not in the architecture map: add it to stack.md section 5 through the change process", pkg)
			continue
		}
		ok := map[string]bool{}
		for _, p := range permitted {
			ok[p] = true
		}
		for imp := range imports(t, filepath.Join(root, pkg)) {
			if strings.HasPrefix(imp, module) {
				if dep := strings.TrimPrefix(imp, module); !ok[dep] {
					t.Errorf("internal/%s must not import internal/%s", pkg, dep)
				}
			}
			if strings.Contains(imp, "spf13/cobra") && pkg != "cli" {
				t.Errorf("Cobra is confined to internal/cli; internal/%s imports it", pkg)
			}
			if (imp == "os/exec") && pkg != "execution" {
				t.Errorf("process execution is confined to internal/execution; internal/%s imports os/exec", pkg)
			}
		}
	}
}

func TestNoGovernancePackages(t *testing.T) {
	for _, pkg := range []string{"governance", "architecture", "adr", "reasoning", "think", "gtt-rules"} {
		if _, err := os.Stat(filepath.Join("..", "..", "internal", pkg)); err == nil {
			t.Errorf("internal/%s must not exist: GTT semantics belong to the Bootstrap", pkg)
		}
	}
}

// The CLI reaches the Bootstrap through one path only: its contract entry
// point. No other Bootstrap script may be named in non-test Go source.
func TestOnlyTheContractEntryPointIsKnown(t *testing.T) {
	for _, dir := range []string{"internal", "cmd"} {
		filepath.Walk(filepath.Join("..", "..", dir), func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			data, _ := os.ReadFile(p)
			for _, line := range strings.Split(string(data), "\n") {
				code := line
				if i := strings.Index(code, "//"); i >= 0 {
					code = code[:i]
				}
				for _, m := range []string{"gtt-ade.sh", "gtt-template.sh", "gtt-validate.sh", "gtt-freeze.sh", "gtt-status.sh",
					"gtt-project.sh", "gtt-index.sh", "gtt-query.sh", "gtt-check-", "gtt-maintain.sh", "ade.json", "methodology.json", "selected-sources.json"} {
					if strings.Contains(code, m) {
						t.Errorf("%s names a Bootstrap internal (%s): use a declared operation", p, m)
					}
				}
			}
			return nil
		})
	}
}
