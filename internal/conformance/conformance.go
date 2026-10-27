// Package conformance verifies code against the dependency rules of a
// ratified development method, through the language's own tool. GTT has no
// import analyser of its own: an adapter turns the neutral rules the
// Bootstrap declares (method.rules) into that tool's configuration, runs it
// and maps what it reports back to the rules. It decides nothing: a
// violation is a conflict to report, never something to correct.
package conformance

import (
	"context"
	"regexp"
	"sort"
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
	// Check runs the tool. A non-nil error means the result could not be
	// determined, never that the code is fine.
	Check(ctx context.Context, root, tool string, rows []Row, runner ports.Runner) (Outcome, error)
}

// Adapters is the registry, in the order they are tried. Adding a language is
// adding an adapter here.
func Adapters() []Adapter {
	return []Adapter{DependencyCruiser{}}
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
