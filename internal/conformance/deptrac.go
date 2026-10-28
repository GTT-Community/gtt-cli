package conformance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Deptrac is the adapter for PHP: https://github.com/deptrac/deptrac. It uses
// the project's own installation (vendor/bin) or one on the PATH, and never
// installs it.
type Deptrac struct{}

func (Deptrac) ID() string       { return "deptrac" }
func (Deptrac) Language() string { return "php" }
func (Deptrac) Install() string  { return "composer require --dev deptrac/deptrac" }

func (Deptrac) Applies(root string) bool { return exists(root, "composer.json") }

func (Deptrac) Tool(root string, runner ports.Runner) string {
	local := filepath.Join(root, "vendor", "bin", "deptrac")
	if info, err := os.Stat(local); err == nil && !info.IsDir() {
		return local
	}
	if p, err := runner.LookPath("deptrac"); err == nil {
		return p
	}
	return ""
}

// yamlQuote writes a YAML single-quoted scalar.
func yamlQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// DeptracConfig is the deptrac configuration for the rows. Deptrac's ruleset
// lists what a layer MAY depend on, so every concrete module is a layer and
// each layer may depend on every other layer except those the rules forbid
// it. forbidden maps "layer>layer" back to the rule.
func DeptracConfig(root string, rows []Row, paths []string) (cfg string, forbidden map[string]string, layers map[string]string) {
	bs := bans(root, rows)
	set := map[string]bool{}
	for _, x := range bs {
		set[x.From] = true
		for _, t := range x.To {
			set[t] = true
		}
	}
	var dirs []string
	for d := range set {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	name := map[string]string{}
	layers = map[string]string{}
	for i, d := range dirs {
		name[d] = fmt.Sprintf("L%d", i+1)
		layers[name[d]] = d
	}
	forbidden = map[string]string{}
	for _, x := range bs {
		for _, t := range x.To {
			if _, ok := forbidden[name[x.From]+">"+name[t]]; !ok {
				forbidden[name[x.From]+">"+name[t]] = x.Rule
			}
		}
	}
	var b strings.Builder
	b.WriteString("deptrac:\n  paths:\n")
	for _, p := range paths {
		fmt.Fprintf(&b, "    - %s\n", yamlQuote(filepath.Join(root, p)))
	}
	b.WriteString("  layers:\n")
	for _, d := range dirs {
		fmt.Fprintf(&b, "    - name: %s\n      collectors:\n        - type: directory\n          value: %s\n", name[d], yamlQuote(regexp.QuoteMeta(d)+"/.*"))
	}
	b.WriteString("  ruleset:\n")
	for _, d := range dirs {
		var allowed []string
		for _, o := range dirs {
			if _, banned := forbidden[name[d]+">"+name[o]]; o != d && !banned {
				allowed = append(allowed, name[o])
			}
		}
		fmt.Fprintf(&b, "    %s: [%s]\n", name[d], strings.Join(allowed, ", "))
	}
	return b.String(), forbidden, layers
}

var deptracMessage = regexp.MustCompile(`(\S+) must not depend on (\S+) \((\S+) on (\S+)\)`)

// deptracConflicts reads deptrac's JSON report: each violation names the
// depending class, the class it depends on and the two layers.
func deptracConflicts(root string, report []byte, forbidden map[string]string) ([]Conflict, error) {
	start, end := strings.IndexByte(string(report), '{'), strings.LastIndexByte(string(report), '}')
	if start < 0 || end < start {
		return nil, fmt.Errorf("no JSON report")
	}
	var r struct {
		Report map[string]int `json:"Report"`
		Files  map[string]struct {
			Messages []struct {
				Message string `json:"message"`
			} `json:"messages"`
		} `json:"files"`
	}
	if err := json.Unmarshal(report[start:end+1], &r); err != nil {
		return nil, err
	}
	if r.Report == nil {
		return nil, fmt.Errorf("no Report section")
	}
	if r.Report["Errors"] > 0 {
		return nil, fmt.Errorf("deptrac reported %d error(s) while analysing", r.Report["Errors"])
	}
	files := make([]string, 0, len(r.Files))
	for f := range r.Files {
		files = append(files, f)
	}
	sort.Strings(files)
	var out []Conflict
	for _, f := range files {
		rel, err := filepath.Rel(root, f)
		if err != nil || strings.HasPrefix(rel, "..") {
			rel = f
		}
		for _, m := range r.Files[f].Messages {
			v := deptracMessage.FindStringSubmatch(m.Message)
			if v == nil {
				continue
			}
			rule, ok := forbidden[v[3]+">"+v[4]]
			if !ok {
				return nil, fmt.Errorf("violation outside the generated rules: %s", m.Message)
			}
			out = append(out, Conflict{Rule: rule, From: filepath.ToSlash(rel), To: v[2]})
		}
	}
	if len(out) < r.Report["Violations"] {
		return nil, fmt.Errorf("%d violation(s) reported but only %d could be read", r.Report["Violations"], len(out))
	}
	return out, nil
}

func (d Deptrac) Check(ctx context.Context, root, tool string, rows []Row, runner ports.Runner) (Outcome, error) {
	out := Outcome{Tool: d.ID()}
	paths, err := present(root, rows)
	if err != nil {
		return out, err
	}
	cfg, forbidden, _ := DeptracConfig(root, rows, paths)
	if len(forbidden) == 0 {
		return out, nil // the modules the rules name have nothing they must not use
	}
	dir, err := os.MkdirTemp("", "conformance-")
	if err != nil {
		return out, err
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "deptrac.yaml")
	if err := os.WriteFile(file, []byte(cfg), 0o600); err != nil {
		return out, err
	}
	res, err := runner.Run(ctx, ports.Command{Path: tool, Dir: root,
		Args: []string{"analyse", "--config-file=" + file, "--formatter=json", "--no-progress"}})
	if err != nil {
		return out, err
	}
	found, err := deptracConflicts(root, res.Stdout, forbidden)
	if err != nil {
		return out, fmt.Errorf("%s produced no readable report (exit %d): %s", d.ID(), res.ExitCode, why(res.Stderr, err))
	}
	out.Conflicts = found
	return out, nil
}
