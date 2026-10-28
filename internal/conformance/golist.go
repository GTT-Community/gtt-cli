package conformance

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// GoList is the adapter for Go. It needs no extra tool: `go list` reports
// each package's imports, and the rules are matched against the packages'
// folders, the module path of go.mod removed.
type GoList struct{}

func (GoList) ID() string       { return "go-list" }
func (GoList) Language() string { return "go" }
func (GoList) Install() string  { return "install Go from https://go.dev/dl/" }

func (GoList) Applies(root string) bool { return exists(root, "go.mod") }

func (GoList) Tool(_ string, runner ports.Runner) string {
	if p, err := runner.LookPath("go"); err == nil {
		return p
	}
	return ""
}

// modulePath reads the module path that go.mod declares.
func modulePath(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if rest, ok := strings.CutPrefix(line, "module"); ok && (rest == "" || rest[0] == ' ' || rest[0] == '\t') {
			rest, _, _ = strings.Cut(rest, "//")
			if m := strings.Trim(strings.TrimSpace(rest), `"`+"`"); m != "" {
				return m, nil
			}
		}
	}
	return "", errors.New("go.mod declares no module path")
}

// goPackage is what `go list -json` says of one package.
type goPackage struct {
	ImportPath   string
	Imports      []string
	TestImports  []string
	XTestImports []string
	Error        *struct{ Err string }
}

// goEdges reads the `go list -json` stream into dependencies between the
// module's own package folders, relative to the project root.
func goEdges(module string, stream []byte) ([][2]string, error) {
	local := func(path string) (string, bool) {
		if path == module {
			return ".", true
		}
		rel, ok := strings.CutPrefix(path, module+"/")
		return rel, ok
	}
	var edges [][2]string
	dec := json.NewDecoder(bytes.NewReader(stream))
	n := 0
	for {
		var p goPackage
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
		n++
		from, ok := local(p.ImportPath)
		if !ok {
			continue
		}
		if p.Error != nil {
			return nil, fmt.Errorf("package %s cannot be loaded: %s", p.ImportPath, firstLine(p.Error.Err))
		}
		seen := map[string]bool{}
		for _, imp := range append(append(append([]string{}, p.Imports...), p.TestImports...), p.XTestImports...) {
			if to, ok := local(imp); ok && to != from && !seen[to] {
				seen[to] = true
				edges = append(edges, [2]string{from, to})
			}
		}
	}
	if n == 0 {
		return nil, errors.New("no package listed")
	}
	return edges, nil
}

func (g GoList) Check(ctx context.Context, root, tool string, rows []Row, runner ports.Runner) (Outcome, error) {
	out := Outcome{Tool: g.ID()}
	if _, err := present(root, rows); err != nil {
		return out, err
	}
	module, err := modulePath(root)
	if err != nil {
		return out, err
	}
	res, err := runner.Run(ctx, ports.Command{Path: tool, Dir: root, Args: []string{"list", "-e", "-json", "./..."}})
	if err != nil {
		return out, err
	}
	edges, err := goEdges(module, res.Stdout)
	if err != nil || res.ExitCode != 0 {
		return out, fmt.Errorf("go list produced no readable report (exit %d): %s", res.ExitCode, why(res.Stderr, err))
	}
	out.Conflicts = conflicts(matchers(rows, nil), edges)
	return out, nil
}
