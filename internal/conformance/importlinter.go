package conformance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// ImportLinter is the adapter for Python: https://github.com/seddonym/import-linter.
// It uses the project's virtual environment (.venv, venv) or a `lint-imports`
// on the PATH, and never installs it.
type ImportLinter struct{}

func (ImportLinter) ID() string       { return "import-linter" }
func (ImportLinter) Language() string { return "python" }
func (ImportLinter) Install() string  { return "pip install import-linter" }

func (ImportLinter) Applies(root string) bool {
	if exists(root, "pyproject.toml", "setup.py", "setup.cfg") {
		return true
	}
	found, _ := filepath.Glob(filepath.Join(root, "requirements*.txt"))
	return len(found) > 0
}

func (ImportLinter) Tool(root string, runner ports.Runner) string {
	for _, venv := range []string{".venv", "venv"} {
		for _, bin := range []string{"bin/lint-imports", "Scripts/lint-imports.exe"} {
			local := filepath.Join(root, venv, filepath.FromSlash(bin))
			if info, err := os.Stat(local); err == nil && !info.IsDir() {
				return local
			}
		}
	}
	if p, err := runner.LookPath("lint-imports"); err == nil {
		return p
	}
	return ""
}

// pyModule turns a project path into a dotted module name. In a src layout
// (src/ without __init__.py) the src folder is the import root, not a package.
func pyModule(root, path string) (module string, src bool) {
	path = strings.TrimSuffix(path, ".py")
	if rest, ok := strings.CutPrefix(path, "src/"); ok && !exists(root, "src/__init__.py") {
		path, src = rest, true
	}
	return strings.ReplaceAll(path, "/", "."), src
}

// LinterConfig is the import-linter configuration for the rows: one forbidden
// contract per concrete module and what it must not import, direct imports
// only (as for the other languages). The contract is named after the rule,
// so the report maps back to it. src reports a src layout.
func LinterConfig(root string, rows []Row) (cfg string, src bool, contracts int) {
	var b strings.Builder
	roots := map[string]bool{}
	for i, x := range bans(root, rows, ".py") {
		from, s := pyModule(root, x.From)
		src = src || s
		roots[strings.SplitN(from, ".", 2)[0]] = true
		fmt.Fprintf(&b, "\n[importlinter:contract:%d]\nname = %s#%d\ntype = forbidden\nsource_modules =\n    %s\nforbidden_modules =\n", i+1, x.Rule, i+1, from)
		for _, t := range x.To {
			to, s := pyModule(root, t)
			src = src || s
			roots[strings.SplitN(to, ".", 2)[0]] = true
			fmt.Fprintf(&b, "    %s\n", to)
		}
		b.WriteString("allow_indirect_imports = True\n")
		contracts++
	}
	var names []string
	for r := range roots {
		names = append(names, r)
	}
	sort.Strings(names)
	return "[importlinter]\nroot_packages =\n    " + strings.Join(names, "\n    ") + "\n" + b.String(), src, contracts
}

var (
	linterSummary = regexp.MustCompile(`Contracts:\s*(\d+) kept,\s*(\d+) broken`)
	linterPair    = regexp.MustCompile(`^(\S+) is not allowed to import (\S+?):?$`)
	linterImport  = regexp.MustCompile(`^-\s+(\S+)\s+->\s+(\S+)`)
	linterRule    = regexp.MustCompile(`^(\S+)#\d+$`)
)

// linterConflicts reads the lint-imports report. Under each broken contract
// (its name underlined with dashes) every "X is not allowed to import Y"
// is followed by the imports that break it; when none is listed, the pair
// itself is the conflict.
func linterConflicts(report string) ([]Conflict, error) {
	m := linterSummary.FindStringSubmatch(report)
	if m == nil {
		return nil, fmt.Errorf("no contract summary")
	}
	broken, _ := strconv.Atoi(m[2])
	lines := strings.Split(strings.ReplaceAll(report, "\r\n", "\n"), "\n")
	var out []Conflict
	rule, section := "", false
	var pair *Conflict
	detailed := false
	flush := func() {
		if pair != nil && !detailed {
			out = append(out, *pair)
		}
		pair, detailed = nil, false
	}
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		underlined := i+1 < len(lines) && line != "" && strings.Trim(strings.TrimSpace(lines[i+1]), "-") == "" && strings.TrimSpace(lines[i+1]) != ""
		switch {
		case underlined && line == "Broken contracts":
			section = true
		case !section:
		case underlined:
			flush()
			if r := linterRule.FindStringSubmatch(line); r != nil {
				rule = r[1]
			} else {
				rule = line
			}
		case linterPair.MatchString(line):
			flush()
			p := linterPair.FindStringSubmatch(line)
			pair = &Conflict{Rule: rule, From: p[1], To: p[2]}
		case pair != nil && linterImport.MatchString(line):
			p := linterImport.FindStringSubmatch(line)
			out = append(out, Conflict{Rule: rule, From: p[1], To: p[2]})
			detailed = true
		}
	}
	flush()
	if broken > 0 && len(out) == 0 {
		return nil, fmt.Errorf("%d broken contract(s) reported but none could be read", broken)
	}
	return out, nil
}

func (l ImportLinter) Check(ctx context.Context, root, tool string, rows []Row, runner ports.Runner) (Outcome, error) {
	out := Outcome{Tool: l.ID()}
	if _, err := present(root, rows); err != nil {
		return out, err
	}
	cfg, src, contracts := LinterConfig(root, rows)
	if contracts == 0 {
		return out, nil // the modules the rules name have nothing they must not import
	}
	dir, err := os.MkdirTemp("", "conformance-")
	if err != nil {
		return out, err
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "importlinter.ini")
	if err := os.WriteFile(file, []byte(cfg), 0o600); err != nil {
		return out, err
	}
	path := []string{root}
	if src {
		path = []string{filepath.Join(root, "src"), root}
	}
	if prev := os.Getenv("PYTHONPATH"); prev != "" {
		path = append(path, prev)
	}
	res, err := runner.Run(ctx, ports.Command{Path: tool, Dir: root,
		Args: []string{"--config", file, "--no-cache"},
		Env:  []string{"PYTHONPATH=" + strings.Join(path, string(os.PathListSeparator))}})
	if err != nil {
		return out, err
	}
	found, err := linterConflicts(string(res.Stdout))
	if err != nil || res.ExitCode > 1 || (res.ExitCode == 1 && len(found) == 0) {
		return out, fmt.Errorf("%s produced no readable report (exit %d): %s", l.ID(), res.ExitCode, why(res.Stderr, err))
	}
	out.Conflicts = found
	return out, nil
}
