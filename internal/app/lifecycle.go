package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/ade"
	"github.com/GTT-Community/gtt-cli/internal/agents"
	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/fsx"
	"github.com/GTT-Community/gtt-cli/internal/lifecycle"
	"github.com/GTT-Community/gtt-cli/internal/methodology"
	"github.com/GTT-Community/gtt-cli/internal/ports"
	"github.com/GTT-Community/gtt-cli/internal/snapshot"
)

// BootstrapInfo identifies the Bootstrap a project runs.
type BootstrapInfo struct {
	ID             string `json:"id"`
	Version        string `json:"version"`
	Schema         int    `json:"schema"`
	Channel        string `json:"channel"`
	ScaffoldLayout int    `json:"scaffold_layout"`
	Canon          string `json:"canon"`
	Compatibility  string `json:"compatibility"`
}

func bootstrapInfo(rel ports.Release) BootstrapInfo {
	b := rel.Bootstrap
	return BootstrapInfo{ID: b.ID, Version: b.Version, Schema: b.SchemaVersion, Channel: b.Channel,
		ScaffoldLayout: rel.Scaffold.Version, Canon: rel.Canon, Compatibility: "PASS"}
}

// bootstrapStatus mirrors the fields of the Bootstrap status contract that
// the CLI presents. Classification (OPEN, BLOCKING, conflicts) is the
// Bootstrap's; the CLI only renders it.
type bootstrapStatus struct {
	ADE         adeState `json:"ade"`
	Methodology struct {
		Profile  string `json:"profile"`
		Language string `json:"language"`
		Source   string `json:"source"`
	} `json:"methodology"`
	Sources struct {
		Selected int `json:"selected"`
		Declared int `json:"declared"`
	} `json:"sources"`
	Governance struct {
		ChangeRequest string `json:"change_request"`
		Gaps          struct {
			Open     int `json:"open"`
			Blocking int `json:"blocking"`
		} `json:"gaps"`
		PendingProposals    []string `json:"pending_proposals"`
		UnresolvedConflicts int      `json:"unresolved_conflicts"`
	} `json:"governance"`
	Freeze struct {
		Frozen bool    `json:"frozen"`
		Since  *string `json:"since"`
	} `json:"freeze"`
	Validation struct {
		Run    bool   `json:"run"`
		Result string `json:"result"`
	} `json:"validation"`
	Session struct {
		Exists bool   `json:"exists"`
		File   string `json:"file"`
	} `json:"session"`
}

// StatusResult is `gtt status`.
type StatusResult struct {
	Schema    int                `json:"schema"`
	Project   string             `json:"project"`
	CLI       string             `json:"cli_version"`
	Bootstrap BootstrapInfo      `json:"bootstrap"`
	ADENames  map[string]string  `json:"ade_names"`
	Status    json.RawMessage    `json:"status"` // the Bootstrap status document, verbatim
	Agents    agents.Environment `json:"agents"`
	Sources   SourcesResult      `json:"initial_sources"`
	View      bootstrapStatus    `json:"-"`
}

// Status reports deterministic project state. It modifies nothing.
func (a *App) Status(ctx context.Context, withValidation bool) (StatusResult, error) {
	p, err := a.open(ctx)
	if err != nil {
		return StatusResult{}, err
	}
	res := StatusResult{Schema: core.OutputSchema, Project: p.Root, CLI: core.Version, Bootstrap: bootstrapInfo(p.Release), ADENames: map[string]string{}}
	args := map[string]string{}
	if withValidation {
		args["with_validation"] = "true"
	}
	out, err := a.query(ctx, p.B, "status", args, &res.View)
	if err != nil {
		return res, err
	}
	res.Status = json.RawMessage(out.Stdout)
	registry, err := ade.Registry(ctx, p.B)
	if err != nil {
		return res, err
	}
	for _, r := range registry {
		res.ADENames[r.ID] = r.Name
	}
	if res.Agents, err = a.agentEnvironment(ctx, p); err != nil {
		return res, err
	}
	if res.Sources, err = a.sourcesState(ctx, p); err != nil {
		return res, err
	}
	return res, nil
}

// InspectResult is `gtt inspect`: topology and installation state.
type InspectResult struct {
	Schema       int                   `json:"schema"`
	ProjectRoot  string                `json:"project_root"`
	GitRoot      string                `json:"git_root,omitempty"`
	CLI          string                `json:"cli_version"`
	Bootstrap    BootstrapInfo         `json:"bootstrap"`
	Resolution   *lifecycle.Resolution `json:"resolution,omitempty"`
	Capabilities []ports.Capability    `json:"capabilities"`
	Operations   []string              `json:"operations"`
	Agents       agents.Environment    `json:"agents"`
	Plan         methodology.Plan      `json:"methodology"`
	Sources      json.RawMessage       `json:"sources"`
	Scaffold     []string              `json:"scaffold_paths"`
	OwnedByGTT   json.RawMessage       `json:"gtt_owned"`
	Templates    json.RawMessage       `json:"templates"`
	Frozen       bool                  `json:"frozen"`
	Session      bool                  `json:"session_exists"`
}

// Inspect shows where things are. It does not interpret architecture.
func (a *App) Inspect(ctx context.Context) (InspectResult, error) {
	p, err := a.open(ctx)
	if err != nil {
		return InspectResult{}, err
	}
	res := InspectResult{Schema: core.OutputSchema, ProjectRoot: p.Root, GitRoot: p.Detection.GitRoot, CLI: core.Version, Bootstrap: bootstrapInfo(p.Release)}
	st, err := p.Store.Load()
	if err != nil {
		return res, err
	}
	res.Resolution = st.Bootstrap
	if res.Capabilities, err = p.B.Capabilities(ctx); err != nil {
		return res, err
	}
	ops, err := p.B.Operations(ctx)
	if err != nil {
		return res, err
	}
	for id := range ops {
		res.Operations = append(res.Operations, id)
	}
	sortStrings(res.Operations)
	if res.Agents, err = a.agentEnvironment(ctx, p); err != nil {
		return res, err
	}
	if res.Plan, err = methodology.Load(ctx, p.B); err != nil {
		return res, err
	}
	raw := func(op string) (json.RawMessage, error) {
		out, err := a.query(ctx, p.B, op, nil, nil)
		return json.RawMessage(out.Stdout), err
	}
	if res.Sources, err = raw("source.list"); err != nil {
		return res, err
	}
	if res.OwnedByGTT, err = raw("export-policy"); err != nil {
		return res, err
	}
	if res.Templates, err = raw("template.list"); err != nil {
		return res, err
	}
	if layout, lerr := a.Installer.Layout(ports.Package{Root: p.Root, Release: p.Release}); lerr == nil {
		res.Scaffold = layout.Paths
	}
	var view bootstrapStatus
	if _, err := a.query(ctx, p.B, "status", nil, &view); err != nil {
		return res, err
	}
	res.Frozen, res.Session = view.Freeze.Frozen, view.Session.Exists
	return res, nil
}

// ValidateResult is `gtt validate`: the Bootstrap's verdict, unaltered.
type ValidateResult struct {
	Schema    int        `json:"schema"`
	Bootstrap string     `json:"bootstrap_version"`
	Result    string     `json:"result"`
	Report    validation `json:"report"`
}

// Validate delegates to the Bootstrap's single validation entry point. It
// is non-interactive, launches no ADE and modifies no governed artifact.
func (a *App) Validate(ctx context.Context) (ValidateResult, error) {
	p, err := a.open(ctx)
	if err != nil {
		return ValidateResult{}, err
	}
	v, err := a.validateProject(ctx, p.B)
	res := ValidateResult{Schema: core.OutputSchema, Bootstrap: p.Release.Bootstrap.Version, Result: v.Result, Report: v}
	if err != nil {
		return res, err
	}
	a.log(p.Root, "validate", v.Result, nil)
	if v.Result != "pass" {
		return res, core.Errorf(core.ExitFailure, "GTT validation failed: %s", strings.Join(v.Failing, ", "))
	}
	return res, nil
}

// ResumeResult is `gtt resume`: session context derived from project state.
type ResumeResult struct {
	Schema      int             `json:"schema"`
	Project     string          `json:"project"`
	Context     json.RawMessage `json:"session_context"` // the Bootstrap document, verbatim
	Text        string          `json:"-"`
	Interrupted string          `json:"interrupted_init,omitempty"`
	Handoff     *Handoff        `json:"handoff,omitempty"`
}

// Resume reconstructs operational context from actual state: the Bootstrap
// derives it from the repository, never from an agent's recollection.
func (a *App) Resume(ctx context.Context) (ResumeResult, error) {
	det := a.detect(ctx)
	res := ResumeResult{Schema: core.OutputSchema, Project: det.Root}
	if j, _ := (lifecycle.Store{Root: det.Root}).LoadJournal(); j != nil {
		res.Interrupted = j.State
		if !a.Factory.Installed(det.Root) {
			return res, &core.Error{Code: core.ExitConflict, Unmodified: true,
				What: "Incomplete GTT initialization detected (state " + j.State + ").", Next: "Run: gtt init --resume   or   gtt init --rollback"}
		}
	}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	out, err := a.query(ctx, p.B, "session-context", nil, nil)
	if err != nil {
		return res, err
	}
	res.Context = json.RawMessage(out.Stdout)
	if text, terr := p.B.Execute(ctx, ports.OperationRequest{Operation: "session-context.text"}); terr == nil {
		res.Text = text.Stdout
	}
	var st adeState
	if _, err := a.query(ctx, p.B, "ade.state", nil, &st); err != nil {
		return res, err
	}
	if st.Primary != "" {
		if h, herr := a.handoff(ctx, p, st.Primary); herr == nil {
			res.Handoff = &h
		}
	}
	return res, nil
}

// Freeze is a facade over the Bootstrap freeze. Ratification is a human
// act: it is asked in a terminal and cannot be pre-answered.
func (a *App) Freeze(ctx context.Context) (string, error) {
	p, err := a.open(ctx)
	if err != nil {
		return "", err
	}
	if !a.Prompt.Interactive() {
		return "", &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Freeze is a human ratification.",
			Why: "It must be confirmed by a person in a terminal; it is never run unattended or by an agent.", Next: "Run `gtt freeze` yourself."}
	}
	a.Report.Info("Freeze ratifies the governed context of this project. There is no unfreeze:\nafter it, governed context changes only through the governed change process.\n")
	ok, err := a.Prompt.Confirm("Freeze the governed context now?", false)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", core.ErrCancelled
	}
	out, err := p.B.Execute(ctx, ports.OperationRequest{Operation: "freeze", Apply: true, ConfirmedByHuman: true})
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(out.Stdout + "\n" + out.Stderr)
	a.log(p.Root, "freeze", outcome(out.OK()), nil)
	switch out.ExitCode {
	case 0:
		return text, nil
	case 2:
		return text, &core.Error{Code: core.ExitConflict, Unmodified: true, What: "The project is already frozen.", Why: text}
	default:
		return text, &core.Error{Code: core.ExitFailure, Unmodified: true, What: "The Bootstrap refused the freeze.", Why: text}
	}
}

// Check levels of `gtt doctor`.
const (
	Pass            = "PASS"
	Warn            = "WARN"
	Fail            = "FAIL"
	CannotDetermine = "CANNOT DETERMINE"
)

// Check is one diagnostic.
type Check struct {
	Name   string `json:"name"`
	Level  string `json:"level"`
	Detail string `json:"detail,omitempty"`
}

// DoctorResult is `gtt doctor`.
type DoctorResult struct {
	Schema int            `json:"schema"`
	Checks []Check        `json:"checks"`
	Totals map[string]int `json:"totals"`
}

// Doctor runs CLI-native operational diagnostics. It reports; it never
// repairs anything.
func (a *App) Doctor(ctx context.Context) (DoctorResult, error) {
	res := DoctorResult{Schema: core.OutputSchema, Totals: map[string]int{}}
	add := func(name, level, detail string) {
		res.Checks = append(res.Checks, Check{Name: name, Level: level, Detail: firstLine(detail)})
		res.Totals[level]++
	}
	finish := func() (DoctorResult, error) {
		if res.Totals[Fail] > 0 {
			return res, core.Errorf(core.ExitFailure, "gtt doctor found %d failing check(s)", res.Totals[Fail])
		}
		return res, nil
	}
	check := func(name string, err error, okDetail string) bool {
		if err != nil {
			add(name, Fail, err.Error())
			return false
		}
		add(name, Pass, okDetail)
		return true
	}

	add("CLI", Pass, core.Version)
	det := a.detect(ctx)
	if det.Confident {
		add("Project root", Pass, det.Root+" ("+det.Basis+")")
	} else {
		add("Project root", Warn, det.Root+" (no project marker; using the working directory)")
	}
	switch {
	case !a.Git.Available():
		add("Git", Warn, "git is not installed")
	case det.GitRoot == "":
		add("Git", Warn, "not a Git repository")
	default:
		add("Git", Pass, det.GitRoot)
	}
	if _, err := a.Runner.LookPath("bash"); err != nil {
		add("Bootstrap runtime (bash)", Fail, "bash not found on PATH")
		return finish()
	}
	if j, _ := (lifecycle.Store{Root: det.Root}).LoadJournal(); j != nil {
		add("Initialization", Fail, "incomplete (state "+j.State+"); run: gtt init --resume | --rollback")
	}
	if !a.Factory.Installed(det.Root) {
		add("Bootstrap package", Fail, "GTT is not initialized here; run: gtt init")
		return finish()
	}
	b := a.Factory.At(det.Root)
	rel, err := b.Release(ctx)
	if !check("Bootstrap package", err, rel.Bootstrap.ID+" "+rel.Bootstrap.Version) {
		return finish()
	}
	if problems, err := b.CheckContracts(ctx); err != nil {
		add("Bootstrap integrity", CannotDetermine, err.Error())
	} else if len(problems) > 0 {
		add("Bootstrap integrity", Fail, strings.Join(problems, "; "))
	} else {
		add("Bootstrap integrity", Pass, "contracts consistent")
	}
	store := lifecycle.Store{Root: det.Root}
	if st, err := store.Load(); err != nil {
		add("Installed core files", CannotDetermine, err.Error())
	} else if len(st.CoreLedger) == 0 {
		add("Installed core files", CannotDetermine, "no install ledger: this project was not installed by the gtt CLI")
	} else {
		changed := 0
		for rel, sum := range st.CoreLedger {
			if now, herr := fsx.HashFile(filepath.Join(det.Root, filepath.FromSlash(rel))); herr != nil || now != sum {
				changed++
			}
		}
		add("Installed core files", Pass, itoa(len(st.CoreLedger)-changed)+" unchanged, "+itoa(changed)+" changed or project state since install")
	}
	n, err := b.Negotiate(ctx)
	if err != nil {
		add("CLI compatibility", CannotDetermine, err.Error())
		return finish()
	}
	if !n.Compatible {
		add("CLI compatibility", Fail, strings.Join(n.Reasons, "; "))
		return finish()
	}
	add("CLI compatibility", Pass, "CLI "+core.Version+" / Bootstrap "+rel.Bootstrap.Version)

	p := &Project{Root: det.Root, Detection: det, B: b, Release: rel, Store: store}
	env, err := a.agentEnvironment(ctx, p)
	switch {
	case err != nil:
		add("ADE state", CannotDetermine, err.Error())
	case !env.Configured:
		add("ADE state", Fail, "no ADE state recorded")
	default:
		add("ADE state", Pass, "")
		primary := ""
		for _, ag := range env.Agents {
			if ag.Primary && ag.Enabled {
				primary = ag.Name
			}
		}
		if primary == "" {
			add("Primary ADE", Fail, "no Primary ADE among the participating ADEs")
		} else {
			add("Primary ADE", Pass, primary)
		}
		if env.OK() {
			add("ADE integrations", Pass, "")
		} else {
			add("ADE integrations", Fail, strings.Join(env.Problems, "; ")+" (run: gtt agents sync)")
		}
		if len(env.Undeclared) > 0 {
			add("Undeclared ADEs detected", Warn, strings.Join(env.Undeclared, ", ")+" (not participating; nothing was changed)")
		}
	}
	plan, err := methodology.Load(ctx, b)
	switch {
	case err != nil:
		add("Methodology profile contract", CannotDetermine, err.Error())
	case !plan.Selected:
		add("Methodology profile contract", Warn, "no Method Plan selected; run: gtt method set <plan>")
	default:
		add("Methodology profile contract", Pass, plan.Detail.Label)
	}
	if layout, err := a.Installer.Layout(ports.Package{Root: det.Root, Release: rel}); err != nil {
		add("GTT scaffold", Fail, err.Error())
	} else {
		var missing []string
		for _, path := range layout.Paths {
			if !fsx.Exists(filepath.Join(det.Root, filepath.FromSlash(path))) {
				missing = append(missing, path)
			}
		}
		if len(missing) > 0 {
			add("GTT scaffold", Fail, "missing: "+strings.Join(missing, ", "))
		} else {
			add("GTT scaffold", Pass, "")
		}
	}
	if probe, err := os.CreateTemp(det.Root, ".gtt-doctor-"); err != nil {
		add("Permissions", Fail, "the project root is not writable")
	} else {
		probe.Close()
		os.Remove(probe.Name())
		add("Permissions", Pass, "")
	}
	if v, err := a.validateProject(ctx, b); err != nil {
		add("Bootstrap validation", CannotDetermine, err.Error())
	} else if v.Result != "pass" {
		add("Bootstrap validation", Fail, strings.Join(v.Failing, ", "))
	} else {
		add("Bootstrap validation", Pass, "")
	}
	if src, err := a.sourcesState(ctx, p); err != nil {
		add("Initial sources tracking", CannotDetermine, err.Error())
	} else {
		pending := 0
		for _, s := range src.Sources {
			if s.Outcome != "unchanged" {
				pending++
			}
		}
		if pending > 0 {
			add("Initial sources tracking", Warn, itoa(pending)+" document(s) pending; see: gtt sources")
		} else {
			add("Initial sources tracking", Pass, "")
		}
	}
	var view bootstrapStatus
	if _, err := a.query(ctx, b, "status", nil, &view); err != nil {
		add("Session state", CannotDetermine, err.Error())
	} else if !view.Session.Exists {
		add("Session state", Warn, "no session file yet")
	} else {
		add("Session state", Pass, "")
	}
	return finish()
}

// AuditResult is `gtt audit`: operational traceability, not governance.
type AuditResult struct {
	Schema     int                   `json:"schema"`
	Generated  string                `json:"generated_at"`
	Project    string                `json:"project"`
	CLI        string                `json:"cli_version"`
	Bootstrap  BootstrapInfo         `json:"bootstrap"`
	Resolution *lifecycle.Resolution `json:"resolution,omitempty"`
	Status     json.RawMessage       `json:"status"`
	Sources    json.RawMessage       `json:"sources"`
	Tracking   SourcesResult         `json:"initial_sources"`
	Validation validation            `json:"validation"`
	Snapshots  []snapshot.Info       `json:"recovery_snapshots"`
	History    []lifecycle.Event     `json:"history"`
	View       bootstrapStatus       `json:"-"`
	ADENames   map[string]string     `json:"ade_names"`
}

// Audit collects traceability information from the Bootstrap and from the
// CLI's own records.
func (a *App) Audit(ctx context.Context) (AuditResult, error) {
	p, err := a.open(ctx)
	if err != nil {
		return AuditResult{}, err
	}
	res := AuditResult{Schema: core.OutputSchema, Generated: lifecycle.Timestamp(), Project: p.Root, CLI: core.Version,
		Bootstrap: bootstrapInfo(p.Release), ADENames: map[string]string{}}
	st, err := p.Store.Load()
	if err != nil {
		return res, err
	}
	res.Resolution, res.History = st.Bootstrap, st.History
	out, err := a.query(ctx, p.B, "status", nil, &res.View)
	if err != nil {
		return res, err
	}
	res.Status = json.RawMessage(out.Stdout)
	src, err := a.query(ctx, p.B, "source.list", nil, nil)
	if err != nil {
		return res, err
	}
	res.Sources = json.RawMessage(src.Stdout)
	if res.Tracking, err = a.sourcesState(ctx, p); err != nil {
		return res, err
	}
	if res.Validation, err = a.validateProject(ctx, p.B); err != nil {
		return res, err
	}
	registry, err := ade.Registry(ctx, p.B)
	if err != nil {
		return res, err
	}
	for _, r := range registry {
		res.ADENames[r.ID] = r.Name
	}
	res.Snapshots = snapshot.Store{Home: a.Home}.List(p.Root)
	return res, nil
}
