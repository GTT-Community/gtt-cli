package ade

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// fakeRunner knows a fixed set of binaries and records what it is asked to run.
type fakeRunner struct {
	onPath map[string]string
	ran    []ports.Command
	result ports.Result
	err    error
}

func (f *fakeRunner) LookPath(name string) (string, error) {
	if p, ok := f.onPath[name]; ok {
		return p, nil
	}
	return "", errors.New("not found")
}

func (f *fakeRunner) Run(_ context.Context, c ports.Command) (ports.Result, error) {
	f.ran = append(f.ran, c)
	return f.result, f.err
}

func TestDetectReportsSignalsForEveryRegistryADE(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	registry := []ports.ADE{
		{ID: "claude", DetectPaths: []string{".claude"}, Binary: "claude"},
		{ID: "kiro", DetectPaths: []string{".kiro"}, Binary: "kiro"},
		{ID: "codex", Binary: "codex"},
	}
	got := Detect(registry, root, &fakeRunner{onPath: map[string]string{"claude": "/bin/claude", "codex": "/bin/codex"}})
	want := [][]string{{"path:.claude", "binary:claude"}, nil, {"binary:codex"}}
	if len(got) != len(registry) {
		t.Fatalf("every registry ADE is reported, detected or not: got %d", len(got))
	}
	for i, c := range got {
		if c.ID != registry[i].ID || !reflect.DeepEqual(c.Signals, want[i]) {
			t.Errorf("%s: signals %v, want %v", registry[i].ID, c.Signals, want[i])
		}
	}
}

func TestExecutorStartsTheDeclaredBinaryWithNoArguments(t *testing.T) {
	r := &fakeRunner{onPath: map[string]string{"claude": "/bin/claude"}}
	e := Executor{ADE: ports.ADE{ID: "claude", Name: "Claude Code", Binary: "claude"}, Runner: r}
	if e.ID() != "claude" || !e.Available() {
		t.Fatalf("id %q, available %v", e.ID(), e.Available())
	}
	if err := e.Execute(context.Background(), ports.HandoffContract{ProjectRoot: "/project", InstructionEntry: "AGENTS.md"}); err != nil {
		t.Fatal(err)
	}
	want := ports.Command{Path: "/bin/claude", Dir: "/project", Interactive: true}
	if len(r.ran) != 1 || !reflect.DeepEqual(r.ran[0], want) {
		t.Errorf("the CLI passes no instructions to the ADE: ran %+v, want %+v", r.ran, want)
	}
}

func TestExecutorFailures(t *testing.T) {
	ctx, contract := context.Background(), ports.HandoffContract{ProjectRoot: "/project"}
	ade := ports.ADE{ID: "claude", Name: "Claude Code", Binary: "claude"}
	found := map[string]string{"claude": "/bin/claude"}
	for name, c := range map[string]struct {
		executor Executor
		code     core.ExitCode
		ran      int
	}{
		"no binary declared":  {Executor{ADE: ports.ADE{ID: "codex", Name: "Codex"}, Runner: &fakeRunner{}}, core.ExitUnavailable, 0},
		"binary not on PATH":  {Executor{ADE: ade, Runner: &fakeRunner{}}, core.ExitUnavailable, 0},
		"process not started": {Executor{ADE: ade, Runner: &fakeRunner{onPath: found, err: errors.New("boom")}}, core.ExitUnavailable, 1},
		"non-zero exit":       {Executor{ADE: ade, Runner: &fakeRunner{onPath: found, result: ports.Result{ExitCode: 2}}}, core.ExitFailure, 1},
	} {
		err := c.executor.Execute(ctx, contract)
		if err == nil || core.CodeOf(err) != c.code {
			t.Errorf("%s: got %v, want exit code %d", name, err, c.code)
		}
		if n := len(c.executor.Runner.(*fakeRunner).ran); n != c.ran {
			t.Errorf("%s: ran %d process(es), want %d", name, n, c.ran)
		}
	}
}

func TestNames(t *testing.T) {
	registry := []ports.ADE{{ID: "claude", Name: "Claude Code"}, {ID: "kiro", Name: "Kiro"}}
	for want, ids := range map[string][]string{
		"none":                 nil,
		"Claude Code, unknown": {"claude", "unknown"},
	} {
		if got := Names(registry, ids); got != want {
			t.Errorf("Names(%v) = %q, want %q", ids, got, want)
		}
	}
}
