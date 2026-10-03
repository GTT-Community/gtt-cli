package app

import (
	"context"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/fsx"
	"github.com/GTT-Community/gtt-cli/internal/lifecycle"
	"github.com/GTT-Community/gtt-cli/internal/ports"
	"github.com/GTT-Community/gtt-cli/internal/tracking"
)

// DefaultSourceType is the Bootstrap source type recorded when the human
// did not state one.
const DefaultSourceType = "design-document"

// SourceReport is the outcome for one initial MD.
type SourceReport struct {
	Path    string `json:"path"`
	State   string `json:"state"`
	Outcome string `json:"outcome"` // applied | unchanged | pending | failed
	Reason  string `json:"reason,omitempty"`
}

// SourcesResult is the result of a tracking run.
type SourcesResult struct {
	Schema  int            `json:"schema"`
	Sources []SourceReport `json:"sources"`
	// validated holds the Bootstrap validation this run already performed.
	validated *validation
}

// Failed reports whether any document failed.
func (r SourcesResult) Failed() bool {
	for _, s := range r.Sources {
		if s.Outcome == "failed" {
			return true
		}
	}
	return false
}

type sourceList struct {
	Selected []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"selected"`
}

type projectDetection struct {
	DesignSources struct {
		Candidates []string `json:"candidates_at_root"`
	} `json:"design_sources"`
}

// ParseSource splits a "path[:type]" argument.
func ParseSource(arg string) lifecycle.Source {
	if i := strings.LastIndex(arg, ":"); i > 0 {
		return lifecycle.Source{Path: arg[:i], Type: arg[i+1:]}
	}
	return lifecycle.Source{Path: arg, Type: DefaultSourceType}
}

// processSources is the tracking flow: DISCOVERED -> READ -> APPLY ->
// VALIDATE -> marked. `authorized` are the documents the human selected in
// this run; a document already selected in the Bootstrap is authorised too.
// Anything else is only discovered and waits for the human. Tracking is
// updated without asking: it is deterministic bookkeeping.
func (a *App) processSources(ctx context.Context, p *Project, authorized []lifecycle.Source) (SourcesResult, error) {
	store := a.trackingStore(p)
	ledger, err := store.Load()
	if err != nil {
		return SourcesResult{}, err
	}
	var listed sourceList
	if _, err := a.query(ctx, p.B, "source.list", nil, &listed); err != nil {
		return SourcesResult{}, err
	}
	var det projectDetection
	if _, err := a.query(ctx, p.B, "project.detect", nil, &det); err != nil {
		return SourcesResult{}, err
	}
	selected := map[string]string{}
	for _, s := range listed.Selected {
		selected[s.Path] = s.Type
	}
	auth := map[string]string{}
	order := []string{}
	add := func(path, typ string) {
		if _, ok := auth[path]; !ok {
			order = append(order, path)
		}
		auth[path] = typ
	}
	for _, s := range authorized {
		add(s.Path, s.Type)
	}
	for _, s := range listed.Selected {
		if _, ok := auth[s.Path]; !ok {
			add(s.Path, s.Type)
		}
	}
	pending := []string{}
	for _, c := range det.DesignSources.Candidates {
		if _, ok := auth[c]; !ok {
			pending = append(pending, c)
		}
	}

	result := SourcesResult{Schema: core.OutputSchema}
	var applied []*tracking.Entry
	for _, path := range order {
		if _, err := fsx.SafeJoin(p.Root, path); err != nil {
			result.Sources = append(result.Sources, SourceReport{Path: path, State: tracking.Failed, Outcome: "failed", Reason: err.Error()})
			continue
		}
		e, err := ledger.Observe(p.Root, path)
		if err != nil {
			result.Sources = append(result.Sources, SourceReport{Path: path, State: tracking.Failed, Outcome: "failed", Reason: "cannot be read: " + err.Error()})
			continue
		}
		if _, inBootstrap := selected[path]; e.Done() && inBootstrap {
			result.Sources = append(result.Sources, SourceReport{Path: path, State: e.State, Outcome: "unchanged"})
			continue
		}
		e.MarkRead()
		if _, inBootstrap := selected[path]; !inBootstrap {
			args := map[string]string{"path": path, "type": auth[path], "by": "gtt-cli"}
			if _, err := a.mutate(ctx, p.B, "source.select", args); err != nil {
				e.MarkFailed(err.Error())
				result.Sources = append(result.Sources, SourceReport{Path: path, State: e.State, Outcome: "failed", Reason: firstLine(err.Error())})
				continue
			}
		}
		if err := e.MarkApplied(); err != nil {
			return result, err
		}
		applied = append(applied, e)
	}

	if len(applied) > 0 {
		// Validate before marking: READ + APPLIED is recorded only when the
		// Bootstrap confirms the selection and its validation passes.
		var after sourceList
		if _, err := a.query(ctx, p.B, "source.list", nil, &after); err != nil {
			return result, err
		}
		present := map[string]bool{}
		for _, s := range after.Selected {
			present[s.Path] = true
		}
		v, err := a.validateProject(ctx, p.B)
		if err != nil {
			return result, err
		}
		result.validated = &v
		for _, e := range applied {
			switch {
			case !present[e.Path]:
				e.MarkFailed("the Bootstrap does not list it as a selected source")
			case v.Result != "pass":
				e.MarkFailed("Bootstrap validation failed: " + strings.Join(v.Failing, ", "))
			default:
				if err := e.MarkValidated(); err != nil {
					return result, err
				}
				result.Sources = append(result.Sources, SourceReport{Path: e.Path, State: e.State, Outcome: "applied"})
				continue
			}
			result.Sources = append(result.Sources, SourceReport{Path: e.Path, State: e.State, Outcome: "failed", Reason: e.Error})
		}
	}
	for _, path := range pending {
		e, err := ledger.Observe(p.Root, path)
		if err != nil {
			continue
		}
		result.Sources = append(result.Sources, SourceReport{Path: path, State: e.State, Outcome: "pending",
			Reason: "not selected: selecting an initial source is the human's decision"})
	}
	if err := store.Save(ledger); err != nil {
		return result, err
	}
	a.log(p.Root, "sources.track", outcome(!result.Failed()), map[string]any{"documents": len(result.Sources)})
	return result, nil
}

// SourcesApply processes the initial MDs. Paths named by the human are
// thereby selected.
func (a *App) SourcesApply(ctx context.Context, paths []string) (SourcesResult, error) {
	p, err := a.open(ctx)
	if err != nil {
		return SourcesResult{}, err
	}
	var authorized []lifecycle.Source
	for _, arg := range paths {
		authorized = append(authorized, ParseSource(arg))
	}
	res, err := a.processSources(ctx, p, authorized)
	if err != nil {
		return res, err
	}
	if res.Failed() {
		return res, core.Errorf(core.ExitFailure, "one or more initial sources were not applied")
	}
	return res, nil
}

// SourcesList reports the tracking state without changing anything.
func (a *App) SourcesList(ctx context.Context) (SourcesResult, error) {
	p, err := a.open(ctx)
	if err != nil {
		return SourcesResult{}, err
	}
	return a.sourcesState(ctx, p)
}

func (a *App) sourcesState(ctx context.Context, p *Project) (SourcesResult, error) {
	ledger, err := a.trackingStore(p).Load()
	if err != nil {
		return SourcesResult{}, err
	}
	var listed sourceList
	if _, err := a.query(ctx, p.B, "source.list", nil, &listed); err != nil {
		return SourcesResult{}, err
	}
	var det projectDetection
	if _, err := a.query(ctx, p.B, "project.detect", nil, &det); err != nil {
		return SourcesResult{}, err
	}
	result := SourcesResult{Schema: core.OutputSchema}
	seen := map[string]bool{}
	report := func(path string, selected bool) {
		if seen[path] {
			return
		}
		seen[path] = true
		e := ledger.Entries[path]
		sum, herr := fsx.HashFile(p.Root + "/" + path)
		switch {
		case herr != nil:
			result.Sources = append(result.Sources, SourceReport{Path: path, State: tracking.Failed, Outcome: "failed", Reason: "file is missing"})
		case e == nil && selected:
			result.Sources = append(result.Sources, SourceReport{Path: path, State: tracking.Discovered, Outcome: "pending", Reason: "selected but not yet tracked; run: gtt sources apply"})
		case e == nil:
			result.Sources = append(result.Sources, SourceReport{Path: path, State: tracking.Discovered, Outcome: "pending", Reason: "not selected"})
		case e.SHA256 != sum:
			result.Sources = append(result.Sources, SourceReport{Path: path, State: tracking.Discovered, Outcome: "pending", Reason: "content changed since it was applied; run: gtt sources apply"})
		case e.Done() && selected:
			result.Sources = append(result.Sources, SourceReport{Path: path, State: e.State, Outcome: "unchanged"})
		case e.State == tracking.Failed:
			result.Sources = append(result.Sources, SourceReport{Path: path, State: e.State, Outcome: "failed", Reason: firstLine(e.Error)})
		default:
			result.Sources = append(result.Sources, SourceReport{Path: path, State: e.State, Outcome: "pending", Reason: "not selected"})
		}
	}
	for _, s := range listed.Selected {
		report(s.Path, true)
	}
	for _, c := range det.DesignSources.Candidates {
		report(c, false)
	}
	return result, nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func outcome(ok bool) string {
	if ok {
		return "success"
	}
	return "failure"
}

var _ = ports.OperationRequest{}

// trackingStore keeps the record where the Bootstrap declares it, or at the
// CLI's own location when the Bootstrap declares none. A declared path that
// escapes the project is ignored rather than followed.
func (a *App) trackingStore(p *Project) tracking.Store {
	store := tracking.Store{Root: p.Root}
	layout, err := a.Installer.Layout(ports.Package{Root: p.Root, Release: p.Release})
	if err != nil || layout.Tracking == "" {
		return store
	}
	if _, err := fsx.SafeJoin(p.Root, layout.Tracking); err == nil {
		store.Path = layout.Tracking
	}
	return store
}
