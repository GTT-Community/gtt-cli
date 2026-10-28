// Package conformance verifies code against the dependency rules of a
// ratified development method, through the language's own tool. GTT has no
// import analyser of its own: an adapter turns the neutral rules the
// Bootstrap declares (method.rules) into that tool's configuration, runs it
// and maps what it reports back to the rules. It decides nothing: a
// violation is a conflict to report, never something to correct.
package conformance

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Row is one materialized rule: the project's real paths, as stack.md
// section 5 writes them.
type Row struct {
	Rule     string `json:"rule"`
	Module   string `json:"module"`
	MustNot  string `json:"must_not_depend_on"`
	MayUse   string `json:"may_depend_on"`
	PackRule string `json:"pack_rule,omitempty"`
}

// Conflict is one forbidden dependency found in the code.
type Conflict struct {
	Rule string `json:"rule"`
	From string `json:"from"`
	To   string `json:"to"`
}

// Outcome is what an adapter found.
type Outcome struct {
	Tool      string     `json:"tool"`
	Conflicts []Conflict `json:"conflicts"`
}

// Adapter verifies the rows for one language.
type Adapter interface {
	ID() string
	Language() string
	// Applies says whether the project is written in this adapter's language.
	Applies(root string) bool
	// Tool locates the language's tool; "" means it is not installed.
	Tool(root string, runner ports.Runner) string
	// Install says how to install the tool, for the human to run.
	Install() string
	// Check runs the tool. A non-nil error means the result could not be
	// determined, never that the code is fine.
	Check(ctx context.Context, root, tool string, rows []Row, runner ports.Runner) (Outcome, error)
}

// Adapters is the registry, in the order they are tried. Adding a language is
// adding an adapter here. JavaScript/TypeScript comes last: a package.json is
// often only the front-end or tooling of a project written in another
// language, while go.mod, pom.xml, composer.json or pyproject.toml are not.
// .NET has no adapter: its architecture test tools are libraries, not CLIs.
func Adapters() []Adapter {
	return []Adapter{GoList{}, Jdeps{}, Deptrac{}, ImportLinter{}, DependencyCruiser{}}
}

// For returns the adapter for the project's language, or nil.
func For(root string) Adapter {
	for _, a := range Adapters() {
		if a.Applies(root) {
			return a
		}
	}
	return nil
}

// Supported lists the languages that have an adapter.
func Supported() []string {
	var out []string
	for _, a := range Adapters() {
		out = append(out, a.Language())
	}
	return out
}

var annotation = regexp.MustCompile(`\s*\([^)]*\)`)

// Paths splits a cell into path globs: comma separated, annotations such as
// "(another slice)" dropped, "-" and empty entries skipped.
func Paths(cell string) []string {
	var out []string
	for _, part := range strings.Split(annotation.ReplaceAllString(cell, ""), ",") {
		part = strings.Trim(strings.TrimSpace(part), "`")
		if part != "" && part != "-" {
			out = append(out, strings.TrimPrefix(part, "./"))
		}
	}
	return out
}

// Pattern turns a path glob into an anchored regular expression over paths
// relative to the project root. `**` is any depth, `*` one path segment.
// With capture, each `*` is a group, so a rule can refer to "the same one".
func Pattern(glob string, capture bool) string {
	glob = strings.TrimSuffix(strings.TrimSuffix(glob, "/**"), "/")
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(glob); i++ {
		switch {
		case strings.HasPrefix(glob[i:], "**"):
			b.WriteString(".*")
			i++
		case glob[i] == '*':
			if capture {
				b.WriteString("([^/]+)")
			} else {
				b.WriteString("[^/]+")
			}
		default:
			b.WriteString(regexp.QuoteMeta(string(glob[i])))
		}
	}
	b.WriteString("(/|$)")
	return b.String()
}

// Sibling is the pattern of "the same one" as the captured from-module:
// `src/features/*` -> `^src/features/$1(/|$)`.
func Sibling(glob string) string {
	glob = strings.TrimSuffix(strings.TrimSuffix(glob, "/**"), "/")
	var b strings.Builder
	b.WriteString("^")
	group := 0
	for i := 0; i < len(glob); i++ {
		switch {
		case strings.HasPrefix(glob[i:], "**"):
			b.WriteString(".*")
			i++
		case glob[i] == '*':
			group++
			b.WriteString("$" + string(rune('0'+group)))
		default:
			b.WriteString(regexp.QuoteMeta(string(glob[i])))
		}
	}
	b.WriteString("(/|$)")
	return b.String()
}

// Roots are the top-level folders the rows name, which is what the tool scans.
func Roots(rows []Row) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range rows {
		for _, g := range append(Paths(r.Module), Paths(r.MustNot)...) {
			top := strings.SplitN(g, "/", 2)[0]
			if top != "" && !strings.ContainsAny(top, "*?[") && !seen[top] {
				seen[top] = true
				out = append(out, top)
			}
		}
	}
	sort.Strings(out)
	return out
}

// exists says whether any of the files exists under root.
func exists(root string, files ...string) bool {
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			return true
		}
	}
	return false
}

// present are the Roots that exist in the project. None is an error: the
// rules name folders the project does not have, so nothing can be verified.
func present(root string, rows []Row) ([]string, error) {
	roots := Roots(rows)
	var out []string
	for _, r := range roots {
		if exists(root, r) {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("none of the folders the rules name exists in the project (%s)", strings.Join(roots, ", "))
	}
	return out, nil
}

// matcher decides whether a dependency between two paths breaks one
// (row, module glob, forbidden glob) of the rules, for the adapters whose tool
// only reports dependencies and leaves the rules to GTT.
type matcher struct {
	rule     string
	from, to *regexp.Regexp
	capture  *regexp.Regexp // set when the forbidden glob is "another one of the same kind"
	sibling  string
}

// matchers builds one matcher per (row, module glob, forbidden glob); glob
// maps a glob into the space of the paths the tool reports (identity when nil).
func matchers(rows []Row, glob func(string) string) []matcher {
	if glob == nil {
		glob = func(g string) string { return g }
	}
	var out []matcher
	for _, r := range rows {
		for _, from := range Paths(r.Module) {
			for _, to := range Paths(r.MustNot) {
				from, to := glob(from), glob(to)
				m := matcher{rule: r.Rule, from: regexp.MustCompile(Pattern(from, false)), to: regexp.MustCompile(Pattern(to, false))}
				if sameKind(from, to) {
					m.capture, m.sibling = regexp.MustCompile(Pattern(from, true)), Sibling(from)
				}
				out = append(out, m)
			}
		}
	}
	return out
}

// breaks says whether the dependency from -> to is forbidden. A module is
// never forbidden from depending on itself.
func (m matcher) breaks(from, to string) bool {
	if !m.from.MatchString(from) || !m.to.MatchString(to) {
		return false
	}
	if m.capture == nil {
		return true
	}
	groups := m.capture.FindStringSubmatch(from)
	self := m.sibling
	for i := len(groups) - 2; i >= 1; i-- { // the last group is the (/|$) anchor
		self = strings.ReplaceAll(self, "$"+strconv.Itoa(i), regexp.QuoteMeta(groups[i]))
	}
	return !regexp.MustCompile(self).MatchString(to)
}

// conflicts maps the dependencies a tool reported onto the rules.
func conflicts(ms []matcher, edges [][2]string) []Conflict {
	var out []Conflict
	for _, e := range edges {
		for _, m := range ms {
			if m.breaks(e[0], e[1]) {
				out = append(out, Conflict{Rule: m.rule, From: e[0], To: e[1]})
			}
		}
	}
	return out
}

// Expand lists the project's folders a glob names, relative to root. A glob
// without wildcards names itself, when it exists (or, with a suffix such as
// ".py", when that file exists). Hidden folders, `__*__`, node_modules and
// vendor are never part of a wildcard.
func Expand(root, glob string, suffixes ...string) []string {
	glob = strings.TrimSuffix(strings.TrimSuffix(glob, "/**"), "/")
	if !strings.Contains(glob, "*") {
		for _, s := range append([]string{""}, suffixes...) {
			if exists(root, glob+s) {
				return []string{glob + s}
			}
		}
		return nil
	}
	exact := regexp.MustCompile(strings.TrimSuffix(Pattern(glob, false), "(/|$)") + "$")
	static := glob[:strings.Index(glob, "*")]
	if i := strings.LastIndex(static, "/"); i >= 0 {
		static = static[:i]
	} else {
		static = ""
	}
	segments := func(p string) int {
		if p == "" {
			return 0
		}
		return strings.Count(p, "/") + 1
	}
	var out []string
	_ = filepath.WalkDir(filepath.Join(root, static), func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if rel = filepath.ToSlash(rel); rel == "." {
			rel = ""
		}
		name := d.Name()
		if rel != static && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "__") || name == "node_modules" || name == "vendor") {
			return filepath.SkipDir
		}
		if exact.MatchString(rel) {
			out = append(out, rel)
		}
		if !strings.Contains(glob, "**") && segments(rel) >= segments(glob) {
			return filepath.SkipDir
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// ban is one concrete module and the concrete modules it must not depend on,
// for the tools whose configuration names modules rather than patterns.
type ban struct {
	Rule string
	From string
	To   []string
}

// bans expands the rows into concrete modules on disk. A forbidden glob of
// the same kind as the module glob excludes the module itself.
func bans(root string, rows []Row, suffixes ...string) []ban {
	var out []ban
	for _, r := range rows {
		for _, from := range Paths(r.Module) {
			for _, to := range Paths(r.MustNot) {
				tos := Expand(root, to, suffixes...)
				for _, f := range Expand(root, from, suffixes...) {
					b := ban{Rule: r.Rule, From: f}
					for _, t := range tos {
						if t == f || (sameKind(from, to) && strings.HasPrefix(t, f+"/")) {
							continue
						}
						b.To = append(b.To, t)
					}
					if len(b.To) > 0 {
						out = append(out, b)
					}
				}
			}
		}
	}
	return out
}
