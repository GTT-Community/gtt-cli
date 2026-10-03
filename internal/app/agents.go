package app

import (
	"context"
	"sort"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/ade"
	"github.com/GTT-Community/gtt-cli/internal/agents"
	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/methodology"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

type adeState struct {
	Configured    bool     `json:"configured"`
	Participating []string `json:"participating"`
	Primary       string   `json:"primary"`
	Excluded      []string `json:"excluded"`
	Integrations  []struct {
		ADE     string `json:"ade"`
		Level   string `json:"level"`
		Message string `json:"message"`
	} `json:"integrations"`
}

type adeDetection struct {
	Candidates []struct {
		ID      string   `json:"id"`
		Name    string   `json:"name"`
		Signals []string `json:"signals"`
		Status  string   `json:"status"`
	} `json:"candidates"`
}

// agentEnvironment compares the project's declared agents with what is
// detected here and with the state of each declared agent's context. It is
// read-only and uses no LLM: registry, participation and integration
// validity all come from the Bootstrap.
func (a *App) agentEnvironment(ctx context.Context, p *Project) (agents.Environment, error) {
	env := agents.Environment{Schema: core.OutputSchema}
	var st adeState
	if _, err := a.query(ctx, p.B, "ade.state", nil, &st); err != nil {
		return env, err
	}
	var det adeDetection
	if _, err := a.query(ctx, p.B, "ade.detect", nil, &det); err != nil {
		return env, err
	}
	env.Configured = st.Configured
	in := func(list []string, id string) bool {
		for _, x := range list {
			if x == id {
				return true
			}
		}
		return false
	}
	level := map[string][2]string{}
	for _, i := range st.Integrations {
		level[i.ADE] = [2]string{i.Level, i.Message}
	}
	if !st.Configured {
		env.Problems = append(env.Problems, "no ADE state is recorded for this project")
	}
	for _, c := range det.Candidates {
		ag := agents.Agent{ID: c.ID, Name: c.Name, Signals: c.Signals, Detected: len(c.Signals) > 0,
			Enabled: in(st.Participating, c.ID), Primary: st.Primary == c.ID, Excluded: in(st.Excluded, c.ID), Context: agents.NotRequired}
		if ag.Enabled {
			lv := level[c.ID]
			ag.Detail = lv[1]
			if lv[0] == "PASS" {
				ag.Context = agents.Synchronized
			} else {
				ag.Context = agents.Missing
				env.Problems = append(env.Problems, c.Name+": "+lv[1])
			}
		} else if ag.Detected {
			env.Undeclared = append(env.Undeclared, c.ID)
		}
		env.Agents = append(env.Agents, ag)
	}
	sort.Slice(env.Agents, func(i, j int) bool { return env.Agents[i].ID < env.Agents[j].ID })
	return env, a.describeAgents(ctx, p, &env)
}

// describeAgents adds what each agent's adapter declares and the state of
// its managed artifacts, from the Bootstrap registry and ownership ledger.
func (a *App) describeAgents(ctx context.Context, p *Project, env *agents.Environment) error {
	entries, err := ade.Registry(ctx, p.B)
	if err != nil {
		return err
	}
	var owned struct {
		Surfaces []struct {
			ADE    string `json:"ade"`
			Path   string `json:"path"`
			Status string `json:"status"`
			Basis  string `json:"basis"`
		} `json:"surfaces"`
	}
	if _, err := a.query(ctx, p.B, "ade.owned", nil, &owned); err != nil {
		return err
	}
	registry := agents.NewRegistry(entries)
	for i := range env.Agents {
		ag := &env.Agents[i]
		if ad, ok := registry.Adapter(ag.ID); ok {
			ag.AdapterVersion, ag.Entry, ag.Locations = ad.Version(), ad.InstructionEntry(), ad.ContextLocations()
		}
		for _, s := range owned.Surfaces {
			if s.ADE == ag.ID {
				ag.Artifacts = append(ag.Artifacts, agents.Artifact{Path: s.Path, Status: agents.ArtifactStatus(s.Status), Basis: s.Basis})
			}
		}
	}
	return nil
}

// AgentsCheck reports the agent environment. Nothing is modified.
func (a *App) AgentsCheck(ctx context.Context) (agents.Environment, error) {
	p, err := a.open(ctx)
	if err != nil {
		return agents.Environment{}, err
	}
	env, err := a.agentEnvironment(ctx, p)
	if err != nil {
		return env, err
	}
	if !env.OK() {
		return env, core.Errorf(core.ExitFailure, "agent context is not consistent: %s", strings.Join(env.Problems, "; "))
	}
	return env, nil
}

// AgentsSyncResult reports a synchronisation.
type AgentsSyncResult struct {
	Schema      int                `json:"schema"`
	Environment agents.Environment `json:"environment"`
	Synced      []string           `json:"synchronized"`
	Added       []string           `json:"added"`
	Plan        string             `json:"plan,omitempty"`
	Source      string             `json:"source,omitempty"`
	Changed     bool               `json:"changed"`
}

// AgentsSync restores the managed context of declared agents from the
// authorised source: the Bootstrap catalog the project was installed from.
// It is idempotent. `add` names agents the human wants to incorporate.
func (a *App) AgentsSync(ctx context.Context, add []string, opts ResolveOptions) (AgentsSyncResult, error) {
	p, err := a.open(ctx)
	if err != nil {
		return AgentsSyncResult{}, err
	}
	return a.syncAgents(ctx, p, add, opts)
}

func (a *App) syncAgents(ctx context.Context, p *Project, add []string, opts ResolveOptions) (AgentsSyncResult, error) {
	res := AgentsSyncResult{Schema: core.OutputSchema}
	env, err := a.agentEnvironment(ctx, p)
	if err != nil {
		return res, err
	}
	res.Environment = env
	store := agents.Store{Root: p.Root}
	if env.OK() && len(add) == 0 {
		return res, store.Save(env, nil) // nothing to do: stable result
	}
	plan, err := methodology.Load(ctx, p.B)
	if err != nil {
		return res, err
	}
	pkg, err := a.catalogFor(ctx, p, opts)
	if err != nil {
		return res, err
	}
	res.Source = pkg.Origin
	synced := map[string]bool{}

	if len(add) > 0 {
		var st adeState
		if _, err := a.query(ctx, p.B, "ade.state", nil, &st); err != nil {
			return res, err
		}
		registry, err := ade.Registry(ctx, p.B)
		if err != nil {
			return res, err
		}
		known := map[string]bool{}
		for _, r := range registry {
			known[r.ID] = true
		}
		participating := append([]string{}, st.Participating...)
		var reinclude []string
		for _, id := range add {
			if !known[id] {
				return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Unknown ADE: " + id,
					Why: "It is not in the Bootstrap ADE registry; GTT does not synchronise an environment it has no integration for."}
			}
			for _, x := range st.Excluded {
				if x == id {
					reinclude = append(reinclude, id)
				}
			}
			participating = append(participating, id)
		}
		// Incorporating a new participant is never silent, in any plan.
		ok, err := a.confirm("Add "+ade.Names(registry, add)+" as participating ADE(s) of this project?", false)
		if err != nil {
			return res, err
		}
		if !ok {
			return res, core.ErrCancelled
		}
		primary := st.Primary
		if primary == "" {
			primary = participating[0]
		}
		args := map[string]string{"from": pkg.Root, "participating": strings.Join(participating, ","), "primary": primary}
		if len(reinclude) > 0 {
			args["reinclude"] = strings.Join(reinclude, ",")
		}
		if _, err := a.mutate(ctx, p.B, "ade.install", args); err != nil {
			return res, err
		}
		res.Added, res.Changed = add, true
		for _, id := range add {
			synced[id] = true
		}
	}

	if !env.OK() {
		dry, err := p.B.Execute(ctx, ports.OperationRequest{Operation: "ade.update", Args: map[string]string{"from": pkg.Root}})
		if err != nil {
			return res, err
		}
		res.Plan = strings.TrimSpace(dry.Stdout)
		if !dry.OK() {
			// A conflict (a local file GTT did not install, or one changed
			// locally) is never overwritten: it is the human's to resolve.
			return res, &core.Error{Code: core.ExitConflict, What: "Agent context cannot be synchronised automatically.",
				Why: res.Plan, Next: "Resolve the listed conflicts; GTT does not overwrite local files."}
		}
		if plan.Decide(methodology.AgentContextSync) != methodology.Automatic && plan.Decide(methodology.AgentContextSync) != methodology.AutomaticSafe {
			a.Report.Info("%s", res.Plan)
			ok, err := a.confirm("Synchronise the agent context from "+pkg.Origin+"?", true)
			if err != nil {
				return res, err
			}
			if !ok {
				return res, core.ErrCancelled
			}
		}
		applied, err := p.B.Execute(ctx, ports.OperationRequest{Operation: "ade.update", Args: map[string]string{"from": pkg.Root}, Apply: true})
		if err != nil {
			return res, err
		}
		if !applied.OK() {
			return res, &core.Error{Code: core.ExitFailure, What: "Agent context synchronisation failed.", Why: strings.TrimSpace(applied.Stdout + "\n" + applied.Stderr)}
		}
		res.Changed = true
		for _, ag := range env.Agents {
			if ag.Enabled && ag.Context != agents.Synchronized {
				synced[ag.ID] = true
			}
		}
	}

	after, err := a.agentEnvironment(ctx, p)
	if err != nil {
		return res, err
	}
	res.Environment = after
	for id := range synced {
		res.Synced = append(res.Synced, id)
	}
	sort.Strings(res.Synced)
	// One trace entry per synchronised agent: who, with which adapter, from
	// where, to where and with what result. Paths and counts only.
	for _, ag := range after.Agents {
		if synced[ag.ID] {
			a.log(p.Root, "agents.sync", outcome(ag.Context == agents.Synchronized), map[string]any{"agent": ag.ID,
				"adapter_version": ag.AdapterVersion, "source": pkg.Origin, "destination": ag.Locations, "files": len(ag.Artifacts)})
		}
	}
	if err := store.Save(after, synced); err != nil {
		return res, err
	}
	if !after.OK() {
		return res, core.Errorf(core.ExitFailure, "agent context is still inconsistent after synchronisation: %s", strings.Join(after.Problems, "; "))
	}
	return res, nil
}
