package conformance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// DependencyCruiser is the adapter for JavaScript and TypeScript:
// https://github.com/sverweij/dependency-cruiser. It uses the project's own
// installation (node_modules/.bin) or one on the PATH, and never downloads it.
type DependencyCruiser struct{}

func (DependencyCruiser) ID() string       { return "dependency-cruiser" }
func (DependencyCruiser) Language() string { return "javascript/typescript" }

func (DependencyCruiser) Applies(root string) bool {
	for _, f := range []string{"package.json", "tsconfig.json"} {
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			return true
		}
	}
	return false
}

func (DependencyCruiser) Install() string { return "npm install --save-dev dependency-cruiser" }

func (DependencyCruiser) Tool(root string, runner ports.Runner) string {
	local := filepath.Join(root, "node_modules", ".bin", "depcruise")
	if info, err := os.Stat(local); err == nil && !info.IsDir() {
		return local
	}
	if p, err := runner.LookPath("depcruise"); err == nil {
		return p
	}
	return ""
}

type cruiseRule struct {
	Name     string            `json:"name"`
	Severity string            `json:"severity"`
	Comment  string            `json:"comment,omitempty"`
	From     map[string]string `json:"from"`
	To       map[string]string `json:"to"`
}

// sameKind says whether a forbidden glob names other modules of the same kind
// as the module glob - `src/features/*` against `src/features/*`, or
// `src/contexts/*` against `src/contexts/*/internal`. A module is never
// forbidden from depending on itself: only the others are.
func sameKind(from, to string) bool {
	from, to = strings.TrimSuffix(strings.TrimSuffix(from, "/**"), "/"), strings.TrimSuffix(strings.TrimSuffix(to, "/**"), "/")
	return strings.Contains(from, "*") && (to == from || strings.HasPrefix(to, from+"/"))
}

// Config is the dependency-cruiser configuration for the rows: one forbidden
// rule per (row, module glob, forbidden glob). A forbidden glob equal to the
// module glob means "another one of the same kind" (a slice must not depend
// on another slice), and so does one below it (a context must not reach into
// another context's internals): the module itself is excluded through a group
// match.
func Config(root string, rows []Row) map[string]any {
	var rules []cruiseRule
	for i, r := range rows {
		for _, from := range Paths(r.Module) {
			for _, to := range Paths(r.MustNot) {
				rule := cruiseRule{Name: fmt.Sprintf("%s#%d", r.Rule, i+1), Severity: "error",
					Comment: r.Module + " must not depend on " + r.MustNot,
					From:    map[string]string{"path": Pattern(from, false)}, To: map[string]string{"path": Pattern(to, false)}}
				if sameKind(from, to) {
					rule.From["path"] = Pattern(from, true)
					rule.To["pathNot"] = Sibling(from)
				}
				rules = append(rules, rule)
			}
		}
	}
	options := map[string]any{"doNotFollow": map[string]string{"path": "node_modules"}}
	if _, err := os.Stat(filepath.Join(root, "tsconfig.json")); err == nil {
		options["tsConfig"] = map[string]string{"fileName": "tsconfig.json"}
	}
	return map[string]any{"forbidden": rules, "options": options}
}

func (d DependencyCruiser) Check(ctx context.Context, root, tool string, rows []Row, runner ports.Runner) (Outcome, error) {
	out := Outcome{Tool: d.ID()}
	roots := Roots(rows)
	var present []string
	for _, r := range roots {
		if _, err := os.Stat(filepath.Join(root, r)); err == nil {
			present = append(present, r)
		}
	}
	if len(present) == 0 {
		return out, fmt.Errorf("none of the folders the rules name exists in the project (%s)", strings.Join(roots, ", "))
	}
	dir, err := os.MkdirTemp("", "conformance-")
	if err != nil {
		return out, err
	}
	defer os.RemoveAll(dir)
	cfg := filepath.Join(dir, "dependency-cruiser.json")
	data, _ := json.MarshalIndent(Config(root, rows), "", "  ")
	if err := os.WriteFile(cfg, data, 0o600); err != nil {
		return out, err
	}
	res, err := runner.Run(ctx, ports.Command{Path: tool, Dir: root,
		Args: append([]string{"--config", cfg, "--output-type", "json"}, present...)})
	if err != nil {
		return out, err
	}
	var report struct {
		Summary struct {
			Violations []struct {
				From string `json:"from"`
				To   string `json:"to"`
				Rule struct {
					Name string `json:"name"`
				} `json:"rule"`
			} `json:"violations"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(res.Stdout, &report); err != nil {
		return out, fmt.Errorf("%s produced no readable report (exit %d): %s", d.ID(), res.ExitCode, firstLine(string(res.Stderr)))
	}
	for _, v := range report.Summary.Violations {
		rule := strings.SplitN(v.Rule.Name, "#", 2)[0]
		out.Conflicts = append(out.Conflicts, Conflict{Rule: rule, From: v.From, To: v.To})
	}
	return out, nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// why is what a tool's failure says: the reading error, then its stderr.
func why(stderr []byte, err error) string {
	var parts []string
	if err != nil {
		parts = append(parts, err.Error())
	}
	if line := firstLine(string(stderr)); line != "" {
		parts = append(parts, line)
	}
	return strings.Join(parts, "; ")
}
