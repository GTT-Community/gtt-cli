// Package bootstrap is the adapter to GTT Bootstrap: it resolves, verifies
// and installs a catalog, and runs declared operations through the
// Bootstrap's contract entry point. It holds no GTT semantics.
package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// entryPoint is the single Bootstrap path the CLI knows: the contract entry
// point. Every other implementation is reached through declared operations.
const entryPoint = ".gtt/scripts/gtt-contract.sh"

// Exit codes of the contract entry point.
const (
	contractRefused = 3
	contractHuman   = 4
	contractInvalid = 5
)

// Factory builds runtimes.
type Factory struct{ Runner ports.Runner }

// At binds a runtime to a Bootstrap tree.
func (f Factory) At(root string) ports.Bootstrap { return &Runtime{root: root, runner: f.Runner} }

// Installed reports whether root holds the contract entry point.
func (f Factory) Installed(root string) bool {
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(entryPoint)))
	return err == nil && !info.IsDir()
}

// Runtime talks to one Bootstrap tree.
type Runtime struct {
	root   string
	runner ports.Runner

	once sync.Once
	ops  map[string]ports.Operation
	err  error

	relOnce sync.Once
	rel     ports.Release
	relErr  error
}

// Root returns the bound directory.
func (r *Runtime) Root() string { return r.root }

func (r *Runtime) call(ctx context.Context, args ...string) (ports.Result, error) {
	bash, err := r.runner.LookPath("bash")
	if err != nil {
		return ports.Result{}, &core.Error{Code: core.ExitUnavailable, What: "bash is not available.",
			Why: "GTT Bootstrap services are invoked through bash.", Next: "Install bash and retry."}
	}
	if _, err := os.Stat(filepath.Join(r.root, filepath.FromSlash(entryPoint))); err != nil {
		return ports.Result{}, &core.Error{Code: core.ExitIntegrity, What: "GTT Bootstrap contract entry point not found.",
			Why: "Expected " + entryPoint + " under " + r.root + "."}
	}
	// The Bootstrap tree is content, not a scratch area: keep the interpreter
	// from leaving bytecode caches in a project or in a verified catalog.
	// GTT_BASH tells the Bootstrap which bash the CLI resolved, so that its
	// own scripts run the same one (on Windows, Git Bash rather than WSL's).
	res, err := r.runner.Run(ctx, ports.Command{Path: bash, Args: append([]string{entryPoint}, args...), Dir: r.root,
		Env: []string{"PYTHONDONTWRITEBYTECODE=1", "GTT_BASH=" + bash}})
	if err != nil {
		return res, &core.Error{Code: core.ExitUnavailable, What: "Could not run the GTT Bootstrap contract.", Err: err, Why: err.Error()}
	}
	return res, nil
}

func (r *Runtime) query(ctx context.Context, v any, args ...string) error {
	res, err := r.call(ctx, args...)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return &core.Error{Code: core.ExitIntegrity, What: "GTT Bootstrap contract query failed: " + strings.Join(args, " "),
			Why: strings.TrimSpace(string(res.Stderr))}
	}
	if err := json.Unmarshal(res.Stdout, v); err != nil {
		return &core.Error{Code: core.ExitIntegrity, What: "GTT Bootstrap returned an unreadable contract: " + strings.Join(args, " "), Why: err.Error()}
	}
	return nil
}

// Release returns the release descriptor.
func (r *Runtime) Release(ctx context.Context) (ports.Release, error) {
	r.relOnce.Do(func() { r.relErr = r.query(ctx, &r.rel, "release", "--json") })
	return r.rel, r.relErr
}

// Negotiate asks the Bootstrap whether this CLI is compatible and then
// applies the CLI-side checks. Anything unknown is an incompatibility.
func (r *Runtime) Negotiate(ctx context.Context) (ports.Negotiation, error) {
	schemas := make([]string, len(core.BootstrapSchemas))
	for i, s := range core.BootstrapSchemas {
		schemas[i] = strconv.Itoa(s)
	}
	res, err := r.call(ctx, "negotiate", "--cli-version", core.Version,
		"--cli-capabilities", strings.Join(core.CLICapabilities, ","),
		"--cli-schemas", strings.Join(schemas, ","), "--json")
	if err != nil {
		return ports.Negotiation{}, err
	}
	var n ports.Negotiation
	if jerr := json.Unmarshal(res.Stdout, &n); jerr != nil || (res.ExitCode != 0 && res.ExitCode != contractRefused) {
		return ports.Negotiation{Reasons: []string{"the Bootstrap did not return a negotiation result"}}, nil
	}
	if res.ExitCode == contractRefused {
		n.Compatible = false
	}
	rel, err := r.Release(ctx)
	if err != nil {
		return n, err
	}
	caps, err := r.Capabilities(ctx)
	if err != nil {
		return n, err
	}
	n.Reasons = append(n.Reasons, LocalReasons(rel, caps)...)
	n.Compatible = n.Compatible && len(n.Reasons) == 0
	return n, nil
}

// LocalReasons are the CLI's own reasons to refuse a Bootstrap.
func LocalReasons(rel ports.Release, caps []ports.Capability) []string {
	var reasons []string
	if rel.Bootstrap.ID != core.BootstrapID {
		reasons = append(reasons, fmt.Sprintf("unknown Bootstrap id %q (this CLI operates %s)", rel.Bootstrap.ID, core.BootstrapID))
	}
	if _, ok := core.ParseSemver(rel.Bootstrap.Version); !ok {
		reasons = append(reasons, "the Bootstrap release version is not MAJOR.MINOR.PATCH")
	}
	schemaOK := false
	for _, s := range core.BootstrapSchemas {
		schemaOK = schemaOK || s == rel.Bootstrap.SchemaVersion
	}
	if !schemaOK {
		reasons = append(reasons, fmt.Sprintf("Bootstrap schema %d is not supported by this CLI", rel.Bootstrap.SchemaVersion))
	}
	have := map[string]bool{}
	for _, c := range core.CLICapabilities {
		have[c] = true
	}
	for _, need := range rel.RequiresCLICapabilities {
		if !have[need] {
			reasons = append(reasons, fmt.Sprintf("Bootstrap requires capability: %s (this CLI supports: %s)", need, strings.Join(core.CLICapabilities, ", ")))
		}
	}
	for _, c := range caps {
		if want, used := core.SupportedCapabilities[c.ID]; used && c.Version != want {
			reasons = append(reasons, fmt.Sprintf("Bootstrap offers %s.v%d; this CLI supports %s.v%d", c.ID, c.Version, c.ID, want))
		}
	}
	return reasons
}

// Capabilities returns the capability registry.
func (r *Runtime) Capabilities(ctx context.Context) ([]ports.Capability, error) {
	var doc struct {
		Capabilities []ports.Capability `json:"capabilities"`
	}
	err := r.query(ctx, &doc, "capabilities", "--json")
	return doc.Capabilities, err
}

// Operations returns the operation registry (cached per runtime).
func (r *Runtime) Operations(ctx context.Context) (map[string]ports.Operation, error) {
	r.once.Do(func() {
		var doc struct {
			Operations map[string]ports.Operation `json:"operations"`
		}
		r.err = r.query(ctx, &doc, "operations", "--json")
		r.ops = doc.Operations
	})
	return r.ops, r.err
}

// Show returns one contract document as raw JSON.
func (r *Runtime) Show(ctx context.Context, contract string) (json.RawMessage, error) {
	var raw json.RawMessage
	err := r.query(ctx, &raw, "show", contract)
	return raw, err
}

// CheckContracts returns the problems the Bootstrap finds in its own contracts.
func (r *Runtime) CheckContracts(ctx context.Context) ([]string, error) {
	res, err := r.call(ctx, "check", "--json")
	if err != nil {
		return nil, err
	}
	var doc struct {
		OK       bool     `json:"ok"`
		Problems []string `json:"problems"`
	}
	if jerr := json.Unmarshal(res.Stdout, &doc); jerr != nil {
		return []string{"the contract check returned no readable result: " + strings.TrimSpace(string(res.Stderr))}, nil
	}
	if !doc.OK && len(doc.Problems) == 0 {
		doc.Problems = []string{"the contract check failed without detail"}
	}
	return doc.Problems, nil
}

// Execute runs one declared operation. An operation the Bootstrap does not
// declare is refused here, before anything is run.
func (r *Runtime) Execute(ctx context.Context, req ports.OperationRequest) (ports.OperationResult, error) {
	ops, err := r.Operations(ctx)
	if err != nil {
		return ports.OperationResult{}, err
	}
	op, ok := ops[req.Operation]
	if !ok {
		return ports.OperationResult{}, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: fmt.Sprintf("Operation %q is not declared by this Bootstrap.", req.Operation),
			Why:  "The CLI executes declared Bootstrap operations only and does not emulate a missing one.",
			Next: "Use a Bootstrap release that provides it, or upgrade the CLI/Bootstrap pair."}
	}
	if op.HumanAuthority && !req.ConfirmedByHuman {
		return ports.OperationResult{}, &core.Error{Code: core.ExitCancelled, Unmodified: true,
			What: fmt.Sprintf("%q needs a human decision.", req.Operation)}
	}
	args := []string{"run", req.Operation}
	keys := make([]string, 0, len(req.Args))
	for k := range req.Args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !op.HasArg(k) {
			return ports.OperationResult{}, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
				What: fmt.Sprintf("Operation %q does not declare the argument %q.", req.Operation, k)}
		}
		args = append(args, k+"="+req.Args[k])
	}
	if op.Mutates && !op.HasArg("apply") && !req.Apply {
		return ports.OperationResult{}, core.Errorf(core.ExitFailure, "operation %q has no dry run; it must be requested with apply", req.Operation)
	}
	if req.Apply && op.HasArg("apply") {
		args = append(args, "apply=true")
	}
	if op.HumanAuthority {
		args = append(args, "confirmed_by_human=true")
	}
	args = append(args, "--envelope")
	res, err := r.call(ctx, args...)
	if err != nil {
		return ports.OperationResult{}, err
	}
	switch res.ExitCode {
	case contractInvalid:
		return ports.OperationResult{}, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: fmt.Sprintf("The Bootstrap refused %q: invalid or undeclared.", req.Operation), Why: strings.TrimSpace(string(res.Stderr))}
	case contractHuman:
		return ports.OperationResult{}, &core.Error{Code: core.ExitCancelled, Unmodified: true,
			What: fmt.Sprintf("%q needs a human decision.", req.Operation), Why: strings.TrimSpace(string(res.Stderr))}
	}
	var env struct {
		ExitCode int    `json:"exit_code"`
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		Applied  bool   `json:"applied"`
		Outputs  struct {
			Format string `json:"format"`
		} `json:"outputs"`
	}
	if jerr := json.Unmarshal(res.Stdout, &env); jerr != nil {
		return ports.OperationResult{}, &core.Error{Code: core.ExitFailure,
			What: fmt.Sprintf("The Bootstrap returned no envelope for %q.", req.Operation), Why: strings.TrimSpace(string(res.Stderr))}
	}
	return ports.OperationResult{Operation: req.Operation, ExitCode: env.ExitCode, Stdout: env.Stdout, Stderr: env.Stderr,
		Format: env.Outputs.Format, Applied: env.Applied}, nil
}
