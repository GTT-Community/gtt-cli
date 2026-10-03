package app

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/ade"
	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/methodology"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// MethodResult is `gtt method show`.
type MethodResult struct {
	Schema  int                  `json:"schema"`
	Plan    methodology.Plan     `json:"plan"`
	Choices []methodology.Choice `json:"choices,omitempty"`
}

// MethodShow reports the Method Plan in force, in the Bootstrap's words.
func (a *App) MethodShow(ctx context.Context) (MethodResult, error) {
	p, err := a.open(ctx)
	if err != nil {
		return MethodResult{}, err
	}
	plan, err := methodology.Load(ctx, p.B)
	if err != nil {
		return MethodResult{}, err
	}
	choices, err := methodology.Choices(ctx, p.B)
	return MethodResult{Schema: core.OutputSchema, Plan: plan, Choices: choices}, err
}

// MethodSet records a plan chosen by the human. Whether the transition is
// allowed is the Bootstrap's decision; a refusal is reported, not overridden.
func (a *App) MethodSet(ctx context.Context, id string) (MethodResult, error) {
	p, err := a.open(ctx)
	if err != nil {
		return MethodResult{}, err
	}
	choices, err := methodology.Choices(ctx, p.B)
	if err != nil {
		return MethodResult{}, err
	}
	current, err := methodology.Load(ctx, p.B)
	if err != nil {
		return MethodResult{}, err
	}
	if id == "" {
		if !a.Prompt.Interactive() {
			return MethodResult{}, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "The Method Plan must be chosen by a human.", Next: "Run: gtt method set <plan>"}
		}
		opts := make([]ports.Option, len(choices))
		for i, c := range choices {
			opts[i] = ports.Option{Label: c.Label, Detail: c.Summary}
		}
		i, err := a.Prompt.Select("GTT Method Plan\n\nChoose how GTT is applied to this project:", opts)
		if err != nil {
			return MethodResult{}, err
		}
		id = choices[i].ID
	}
	known := false
	for _, c := range choices {
		known = known || c.ID == id
	}
	if !known {
		return MethodResult{}, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Unknown Method Plan: " + id}
	}
	if current.Selected && current.Profile == id {
		return MethodResult{Schema: core.OutputSchema, Plan: current, Choices: choices}, nil
	}
	dry, err := p.B.Execute(ctx, ports.OperationRequest{Operation: "methodology.profile.set", Args: map[string]string{"profile": id}})
	if err != nil {
		return MethodResult{}, err
	}
	if !dry.OK() {
		return MethodResult{}, &core.Error{Code: core.ExitConflict, Unmodified: true,
			What: "This change requires the project's GTT change process.\n\nNo direct profile change was made.",
			Why:  strings.TrimSpace(dry.Stdout + "\n" + dry.Stderr)}
	}
	if _, err := p.B.Execute(ctx, ports.OperationRequest{Operation: "methodology.profile.set", Args: map[string]string{"profile": id}, Apply: true}); err != nil {
		return MethodResult{}, err
	}
	from := "not selected"
	if current.Selected {
		from = current.Profile
	}
	_ = p.Store.Record("method-plan", from+" -> "+id)
	a.log(p.Root, "method.set", "success", map[string]any{"from": from, "to": id})
	plan, err := methodology.Load(ctx, p.B)
	return MethodResult{Schema: core.OutputSchema, Plan: plan, Choices: choices}, err
}

// MethodCheck verifies that the plan configuration is coherent: a plan is
// selected and supported, every operating-policy value is one the CLI can
// execute, and the Bootstrap's own contract and validation checks pass.
func (a *App) MethodCheck(ctx context.Context) (DoctorResult, error) {
	res := DoctorResult{Schema: core.OutputSchema, Totals: map[string]int{}}
	add := func(name, level, detail string) {
		res.Checks = append(res.Checks, Check{Name: name, Level: level, Detail: firstLine(detail)})
		res.Totals[level]++
	}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	plan, err := methodology.Load(ctx, p.B)
	if err != nil {
		return res, err
	}
	switch {
	case !plan.Selected:
		add("Plan selected", Fail, "no Method Plan selected; run: gtt method set <plan>")
	case !contains(plan.Supported, plan.Profile):
		add("Plan selected", Fail, plan.Profile+" is not a plan this Bootstrap supports")
	default:
		add("Plan selected", Pass, plan.Detail.Label)
	}
	raw, err := p.B.Show(ctx, "profiles")
	if err != nil {
		return res, err
	}
	var doc struct {
		PolicyValues map[string]string `json:"policy_values"`
	}
	_ = json.Unmarshal(raw, &doc)
	var unknown []string
	for group, values := range map[string]map[string]string{"automation": plan.Detail.Policy.Automation,
		"confirmation": plan.Detail.Policy.Human.Confirmation, "collaboration": plan.Detail.Policy.Collaboration} {
		for key, value := range values {
			if _, ok := doc.PolicyValues[value]; !ok && key != "level" {
				unknown = append(unknown, group+"."+key+"="+value)
			}
		}
	}
	if len(unknown) > 0 {
		add("Automation rules", Fail, "values the Bootstrap does not define: "+strings.Join(unknown, ", "))
	} else {
		add("Automation rules", Pass, "every policy value is defined by the Bootstrap")
	}
	if plan.Decide(methodology.Destructive) != methodology.Confirm || plan.Decide(methodology.GovernedDecision) != methodology.Confirm {
		add("Compatible policies", Fail, "a destructive operation or governed decision would not be confirmed")
	} else {
		add("Compatible policies", Pass, "destructive operations and governed decisions are always confirmed")
	}
	if problems, err := p.B.CheckContracts(ctx); err != nil {
		add("Plan contract", CannotDetermine, err.Error())
	} else if len(problems) > 0 {
		add("Plan contract", Fail, strings.Join(problems, "; "))
	} else {
		add("Plan contract", Pass, "")
	}
	if v, err := a.validateProject(ctx, p.B); err != nil {
		add("Required artifacts and integrity", CannotDetermine, err.Error())
	} else if v.Result != "pass" {
		add("Required artifacts and integrity", Fail, strings.Join(v.Failing, ", "))
	} else {
		add("Required artifacts and integrity", Pass, "")
	}
	if res.Totals[Fail] > 0 {
		return res, core.Errorf(core.ExitFailure, "the Method Plan configuration is not coherent")
	}
	return res, nil
}

// ConfigPrimary changes the Primary ADE explicitly. The Bootstrap validates
// the change; other integrations are not reinstalled or removed.
func (a *App) ConfigPrimary(ctx context.Context, id string) (string, error) {
	p, err := a.open(ctx)
	if err != nil {
		return "", err
	}
	var st adeState
	if _, err := a.query(ctx, p.B, "ade.state", nil, &st); err != nil {
		return "", err
	}
	registry, err := ade.Registry(ctx, p.B)
	if err != nil {
		return "", err
	}
	if id == "" {
		if !a.Prompt.Interactive() {
			return "", &core.Error{Code: core.ExitUsage, Unmodified: true, What: "The Primary ADE must be chosen by a human.", Next: "Run: gtt config primary-ade <id>"}
		}
		a.Report.Info("Current Primary ADE:\n  %s\n\nParticipating ADEs:\n  %s", ade.Names(registry, []string{st.Primary}), ade.Names(registry, st.Participating))
		opts := make([]ports.Option, len(st.Participating))
		for i, x := range st.Participating {
			opts[i] = ports.Option{Label: ade.Names(registry, []string{x})}
		}
		i, err := a.Prompt.Select("Select new Primary:", opts)
		if err != nil {
			return "", err
		}
		id = st.Participating[i]
	}
	if id == st.Primary {
		return ade.Names(registry, []string{id}), nil
	}
	ok, err := a.confirm("Change the Primary ADE from "+ade.Names(registry, []string{st.Primary})+" to "+ade.Names(registry, []string{id})+"?", false)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", core.ErrCancelled
	}
	if _, err := a.mutate(ctx, p.B, "ade.set-primary", map[string]string{"ade": id}); err != nil {
		return "", err
	}
	_ = p.Store.Record("primary-ade", st.Primary+" -> "+id)
	a.log(p.Root, "config.primary-ade", "success", map[string]any{"from": st.Primary, "to": id})
	return ade.Names(registry, []string{id}), nil
}

// ConfigLanguage records the interaction language through the Bootstrap.
func (a *App) ConfigLanguage(ctx context.Context, lang string) error {
	p, err := a.open(ctx)
	if err != nil {
		return err
	}
	if !contains(p.Release.SupportedLanguages, lang) {
		return &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Unsupported language: " + lang,
			Why: "This Bootstrap supports: " + strings.Join(p.Release.SupportedLanguages, ", ")}
	}
	if _, err := a.mutate(ctx, p.B, "methodology.profile.set", map[string]string{"language": lang}); err != nil {
		return err
	}
	a.log(p.Root, "config.language", "success", map[string]any{"language": lang})
	return nil
}

// NextID is the artifact identity allocator: the Bootstrap finds the next
// free id; under a plan that resolves identity automatically the CLI simply
// reports the id to use, otherwise it reports a proposal.
type NextID struct {
	Schema     int             `json:"schema"`
	ID         string          `json:"id"`
	Resolution string          `json:"resolution"`
	Automatic  bool            `json:"automatic"`
	Detail     json.RawMessage `json:"detail"`
}

// ArtifactNextID asks the Bootstrap for the next free artifact id.
func (a *App) ArtifactNextID(ctx context.Context, kind, requested string) (NextID, error) {
	p, err := a.open(ctx)
	if err != nil {
		return NextID{}, err
	}
	args := map[string]string{"kind": kind}
	if requested != "" {
		args["requested"] = requested
	}
	var doc struct {
		ID         string `json:"id"`
		Resolution string `json:"resolution"`
	}
	out, err := a.query(ctx, p.B, "artifact.next-id", args, &doc)
	if err != nil {
		return NextID{}, err
	}
	if !out.OK() {
		return NextID{}, &core.Error{Code: core.ExitFailure, What: "The Bootstrap could not allocate an id.", Why: strings.TrimSpace(out.Stderr)}
	}
	plan, err := methodology.Load(ctx, p.B)
	if err != nil {
		return NextID{}, err
	}
	// The allocation is recorded so that a resolved collision stays traceable.
	// Updating references to a renumbered artifact is the Bootstrap's
	// (reconcile): the CLI never writes the governed domain.
	a.log(p.Root, "artifact.next-id", "ok", map[string]any{"kind": kind, "requested": requested, "id": doc.ID, "resolution": doc.Resolution})
	return NextID{Schema: core.OutputSchema, ID: doc.ID, Resolution: doc.Resolution,
		Automatic: plan.Decide(methodology.IdentityResolution) == methodology.Automatic, Detail: json.RawMessage(out.Stdout)}, nil
}

// SyncResult is `gtt sync`.
type SyncResult struct {
	Schema   int              `json:"schema"`
	Agents   AgentsSyncResult `json:"agents"`
	Sources  SourcesResult    `json:"initial_sources"`
	Maintain string           `json:"maintain"`
	Pending  bool             `json:"human_action_pending"`
}

// Sync does the deterministic upkeep in one run: agent context, initial
// source tracking, and the Bootstrap's own maintenance (protection
// registry, index, validation). Nothing here is a governed decision.
func (a *App) Sync(ctx context.Context, opts ResolveOptions) (SyncResult, error) {
	res := SyncResult{Schema: core.OutputSchema}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	if res.Agents, err = a.syncAgents(ctx, p, nil, opts); err != nil {
		return res, err
	}
	// Registry, index and validation first: tracking marks a source as
	// applied only against a project that validates.
	out, err := p.B.Execute(ctx, ports.OperationRequest{Operation: "maintain", Apply: true})
	if err != nil {
		return res, err
	}
	res.Maintain = strings.TrimSpace(out.Stdout)
	a.Report.Detail(out.Stderr)
	a.log(p.Root, "sync", outcome(out.OK()), nil)
	if !out.OK() {
		res.Pending = true
		return res, &core.Error{Code: core.ExitFailure, What: "Maintenance stopped: a human action is pending.", Why: strings.TrimSpace(out.Stdout + "\n" + out.Stderr)}
	}
	if res.Sources, err = a.processSources(ctx, p, nil); err != nil {
		return res, err
	}
	if res.Sources.Failed() {
		return res, core.Errorf(core.ExitFailure, "one or more initial sources were not applied")
	}
	return res, nil
}

// VersionResult is `gtt version`.
type VersionResult struct {
	Schema       int      `json:"schema"`
	CLI          string   `json:"cli_version"`
	Capabilities []string `json:"cli_capabilities"`
	Schemas      []int    `json:"bootstrap_schemas"`
	Bootstrap    *struct {
		Version       string `json:"version"`
		Schema        int    `json:"schema"`
		Channel       string `json:"channel"`
		Compatibility string `json:"compatibility"`
	} `json:"bootstrap,omitempty"`
}

// Version reports the CLI release and, in a GTT project, the installed
// Bootstrap. The two versions are independent.
func (a *App) Version(ctx context.Context) VersionResult {
	res := VersionResult{Schema: core.OutputSchema, CLI: core.Version, Capabilities: core.CLICapabilities, Schemas: core.BootstrapSchemas}
	det := a.detect(ctx)
	if !a.Factory.Installed(det.Root) {
		return res
	}
	b := a.Factory.At(det.Root)
	rel, err := b.Release(ctx)
	if err != nil {
		return res
	}
	info := &struct {
		Version       string `json:"version"`
		Schema        int    `json:"schema"`
		Channel       string `json:"channel"`
		Compatibility string `json:"compatibility"`
	}{rel.Bootstrap.Version, rel.Bootstrap.SchemaVersion, rel.Bootstrap.Channel, "REFUSE"}
	if n, err := b.Negotiate(ctx); err == nil && n.Compatible {
		info.Compatibility = "PASS"
	}
	res.Bootstrap = info
	return res
}
