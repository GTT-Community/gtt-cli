package conformance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Jdeps is the adapter for Java and Kotlin: jdeps, which ships with the JDK.
// It reads compiled classes, not sources, so the project must be built
// first; the rules' source folders (src/main/java/com/acme/orders) are
// matched as packages (com.acme.orders).
type Jdeps struct{}

func (Jdeps) ID() string       { return "jdeps" }
func (Jdeps) Language() string { return "java/kotlin" }
func (Jdeps) Install() string  { return "install a JDK 11 or later: jdeps ships with it" }

func (Jdeps) Applies(root string) bool {
	return exists(root, "pom.xml", "build.gradle", "build.gradle.kts")
}

func (Jdeps) Tool(_ string, runner ports.Runner) string {
	if home := os.Getenv("JAVA_HOME"); home != "" {
		for _, name := range []string{"jdeps", "jdeps.exe"} {
			if info, err := os.Stat(filepath.Join(home, "bin", name)); err == nil && !info.IsDir() {
				return filepath.Join(home, "bin", name)
			}
		}
	}
	if p, err := runner.LookPath("jdeps"); err == nil {
		return p
	}
	return ""
}

// classDirs are the compiled class folders of Maven and Gradle, at the root
// and one module below it.
func classDirs(root string) []string {
	var out []string
	for _, prefix := range []string{"", "*/"} {
		for _, d := range []string{"target/classes", "build/classes/java/main", "build/classes/kotlin/main"} {
			found, _ := filepath.Glob(filepath.Join(root, filepath.FromSlash(prefix+d)))
			for _, f := range found {
				if info, err := os.Stat(f); err == nil && info.IsDir() {
					out = append(out, f)
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

var sourceRoot = regexp.MustCompile(`^(.*/)?src/(main|test)/(java|kotlin)/`)

// packageGlob maps a source folder glob to a package glob with "/" between
// the parts: src/main/java/com/acme/* -> com/acme/*. A glob written as a
// package (com.acme.orders) is accepted too.
func packageGlob(glob string) string {
	if loc := sourceRoot.FindStringIndex(glob); loc != nil {
		return glob[loc[1]:]
	}
	if !strings.Contains(glob, "/") {
		return strings.ReplaceAll(glob, ".", "/")
	}
	return glob
}

var jdepsLine = regexp.MustCompile(`^\s+(\S+)\s+->\s+(\S+)`)

// jdepsEdges reads `jdeps -verbose:package` into dependencies between
// packages, written with "/" so the rules' patterns apply.
func jdepsEdges(report string) ([][2]string, error) {
	var edges [][2]string
	read := false
	for _, line := range strings.Split(report, "\n") {
		if strings.Contains(line, "->") {
			read = true
		}
		if m := jdepsLine.FindStringSubmatch(line); m != nil && m[1] != m[2] {
			edges = append(edges, [2]string{strings.ReplaceAll(m[1], ".", "/"), strings.ReplaceAll(m[2], ".", "/")})
		}
	}
	if !read {
		return nil, errors.New("no dependency listed")
	}
	return edges, nil
}

func (j Jdeps) Check(ctx context.Context, root, tool string, rows []Row, runner ports.Runner) (Outcome, error) {
	out := Outcome{Tool: j.ID()}
	if _, err := present(root, rows); err != nil {
		return out, err
	}
	dirs := classDirs(root)
	if len(dirs) == 0 {
		return out, errors.New("no compiled classes (target/classes or build/classes): jdeps reads classes, " +
			"build the project first (mvn compile, or gradle classes)")
	}
	res, err := runner.Run(ctx, ports.Command{Path: tool, Dir: root, Args: append([]string{"-verbose:package"}, dirs...)})
	if err != nil {
		return out, err
	}
	edges, err := jdepsEdges(string(res.Stdout))
	if err != nil || res.ExitCode != 0 {
		return out, fmt.Errorf("%s produced no readable report (exit %d): %s", j.ID(), res.ExitCode, why(res.Stderr, err))
	}
	for _, c := range conflicts(matchers(rows, packageGlob), edges) {
		out.Conflicts = append(out.Conflicts, Conflict{Rule: c.Rule, From: strings.ReplaceAll(c.From, "/", "."), To: strings.ReplaceAll(c.To, "/", ".")})
	}
	return out, nil
}
