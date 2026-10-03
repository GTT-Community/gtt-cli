package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

func release(mut func(*ports.Release)) ports.Release {
	var r ports.Release
	r.Bootstrap.ID, r.Bootstrap.Version, r.Bootstrap.SchemaVersion = core.BootstrapID, "1.0.0", 1
	r.RequiresCLICapabilities = []string{"contract.negotiate.v1", "operation.execute.v1"}
	if mut != nil {
		mut(&r)
	}
	return r
}

func TestLocalReasonsFailClosed(t *testing.T) {
	caps := []ports.Capability{{ID: "methodology.profile", Version: 1}, {ID: "something.new", Version: 7}}
	if r := LocalReasons(release(nil), caps); len(r) != 0 {
		t.Fatalf("a compatible pair must pass; an unknown offered capability is simply unused: %v", r)
	}
	cases := map[string]ports.Release{
		"capability": release(func(r *ports.Release) {
			r.RequiresCLICapabilities = append(r.RequiresCLICapabilities, "new-runtime-capability-X")
		}),
		"schema":  release(func(r *ports.Release) { r.Bootstrap.SchemaVersion = 2 }),
		"id":      release(func(r *ports.Release) { r.Bootstrap.ID = "other" }),
		"version": release(func(r *ports.Release) { r.Bootstrap.Version = "main" }),
	}
	for name, rel := range cases {
		if len(LocalReasons(rel, caps)) == 0 {
			t.Errorf("%s: must be refused", name)
		}
	}
	// A capability the CLI uses, offered at a version it does not implement.
	v2 := []ports.Capability{{ID: "methodology.profile", Version: 2}}
	reasons := LocalReasons(release(nil), v2)
	if len(reasons) != 1 || !strings.Contains(reasons[0], "methodology.profile.v2") {
		t.Errorf("got %v", reasons)
	}
}

func TestLayoutReadsLayersAndEntryPoints(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".gtt", "scaffold"), 0o755)
	manifest := `scaffold:
  version: 2
layers:
  engine: .gtt/                   # comment
  domain: gtt-domain/
  overlay: ade-native-paths
engine:
  - { path: .gtt/scripts/, kind: dir, required: true, role: services, with a comma }
entry_points:
  - { path: AGENTS.md, kind: file, required: true, role: contract }
  - { path: readme-gtt.md, kind: file }
overlays:
  - { id: claude, path: .claude/ }
tracking:
  initial_sources: { path: .gtt/tracking/initial-sources.json, schema: 1, maintained_by: cli }
`
	os.WriteFile(filepath.Join(root, ".gtt", "scaffold", "manifest.yaml"), []byte(manifest), 0o644)
	pkg := ports.Package{Root: root}
	pkg.Release.Scaffold.Manifest = ".gtt/scaffold/manifest.yaml"
	l, err := Layout(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(l.Paths, ","); got != ".gtt,gtt-domain,AGENTS.md,readme-gtt.md" {
		t.Errorf("paths: %s", got)
	}
	if got := strings.Join(l.Package, ","); got != ".gtt,AGENTS.md,readme-gtt.md" {
		t.Errorf("the governed domain is never package content: %s", got)
	}
	if l.Tracking != ".gtt/tracking/initial-sources.json" {
		t.Errorf("declared tracking record: %q", l.Tracking)
	}
}

// fakeRunner records every process the runtime would start.
type fakeRunner struct {
	calls [][]string
	reply func(args []string) ports.Result
}

func (f *fakeRunner) LookPath(name string) (string, error) { return "/bin/" + name, nil }
func (f *fakeRunner) Run(_ context.Context, c ports.Command) (ports.Result, error) {
	f.calls = append(f.calls, c.Args)
	return f.reply(c.Args), nil
}

func fakeTree(t *testing.T) string {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".gtt", "scripts"), 0o755)
	os.WriteFile(filepath.Join(root, filepath.FromSlash(entryPoint)), []byte("#!/bin/sh\n"), 0o755)
	return root
}

const registry = `{"operations":{
 "status":{"version":1,"args":[],"mutates":false,"human_authority":false,"outputs":{"format":"json"}},
 "freeze":{"version":1,"args":[],"mutates":true,"human_authority":true,"outputs":{"format":"text"}},
 "source.select":{"version":1,"args":[{"name":"path","type":"path"},{"name":"apply","type":"bool"}],"mutates":true,"human_authority":false,"outputs":{"format":"text"}}}}`

func TestExecuteRunsDeclaredOperationsOnly(t *testing.T) {
	fr := &fakeRunner{reply: func(args []string) ports.Result {
		if args[1] == "operations" {
			return ports.Result{Stdout: []byte(registry)}
		}
		return ports.Result{Stdout: []byte(`{"exit_code":0,"stdout":"ok","stderr":"","outputs":{"format":"text"}}`)}
	}}
	b := Factory{Runner: fr}.At(fakeTree(t))
	ctx := context.Background()

	_, err := b.Execute(ctx, ports.OperationRequest{Operation: "unfreeze"})
	var ce *core.Error
	if !errors.As(err, &ce) || ce.Code != core.ExitIncompatible || !ce.Unmodified {
		t.Fatalf("an undeclared operation must be refused, unmodified: %v", err)
	}
	if _, err := b.Execute(ctx, ports.OperationRequest{Operation: "status", Args: map[string]string{"x; rm -rf /": "1"}}); err == nil {
		t.Error("an undeclared argument must be refused")
	}
	if _, err := b.Execute(ctx, ports.OperationRequest{Operation: "freeze", Apply: true}); core.CodeOf(err) != core.ExitCancelled {
		t.Errorf("human-authority operations need the human: %v", err)
	}
	if _, err := b.Execute(ctx, ports.OperationRequest{Operation: "freeze", ConfirmedByHuman: true}); err == nil {
		t.Error("a mutating operation without a dry run must be requested with apply")
	}
	for _, call := range fr.calls {
		if call[1] == "run" {
			t.Fatalf("nothing may have been run so far: %v", call)
		}
	}

	if _, err := b.Execute(ctx, ports.OperationRequest{Operation: "source.select", Args: map[string]string{"path": "a b; echo x"}}); err != nil {
		t.Fatal(err)
	}
	last := fr.calls[len(fr.calls)-1]
	want := []string{entryPoint, "run", "source.select", "path=a b; echo x", "--envelope"}
	if strings.Join(last, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("a dry run passes structured argv and no apply:\n got %q\nwant %q", last, want)
	}
	b.Execute(ctx, ports.OperationRequest{Operation: "source.select", Args: map[string]string{"path": "a"}, Apply: true})
	if last = fr.calls[len(fr.calls)-1]; last[len(last)-2] != "apply=true" {
		t.Errorf("apply must be explicit: %q", last)
	}
	b.Execute(ctx, ports.OperationRequest{Operation: "freeze", Apply: true, ConfirmedByHuman: true})
	if last = fr.calls[len(fr.calls)-1]; last[len(last)-2] != "confirmed_by_human=true" {
		t.Errorf("human confirmation is passed only when given: %q", last)
	}
}

func TestBootstrapRefusalsMapToCLIExitCodes(t *testing.T) {
	for code, want := range map[int]core.ExitCode{contractInvalid: core.ExitIncompatible, contractHuman: core.ExitCancelled} {
		fr := &fakeRunner{reply: func(args []string) ports.Result {
			if args[1] == "operations" {
				return ports.Result{Stdout: []byte(registry)}
			}
			return ports.Result{ExitCode: code, Stderr: []byte("refused")}
		}}
		_, err := Factory{Runner: fr}.At(fakeTree(t)).Execute(context.Background(), ports.OperationRequest{Operation: "status"})
		if core.CodeOf(err) != want {
			t.Errorf("contract exit %d: got %v", code, err)
		}
	}
}
