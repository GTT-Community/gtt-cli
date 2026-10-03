package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/ade"
	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/export"
	"github.com/GTT-Community/gtt-cli/internal/fsx"
	"github.com/GTT-Community/gtt-cli/internal/lifecycle"
	"github.com/GTT-Community/gtt-cli/internal/methodology"
	"github.com/GTT-Community/gtt-cli/internal/ports"
	"github.com/GTT-Community/gtt-cli/internal/project"
	"github.com/GTT-Community/gtt-cli/internal/snapshot"
)

// InitOptions are the answers the human gave on the command line. Anything
// left empty is asked; without a terminal it is refused, never inferred.
type InitOptions struct {
	Resolve       ResolveOptions
	ADEs          []string
	Primary       string
	Language      string
	Method        string
	Sources       []string
	NoSources     bool
	Questionnaire string // "", "yes", "no"
	Launch        bool
	Resume        bool
	Rollback      bool
	Snapshot      string // explicit recovery snapshot to restore
	NoRestore     bool
}

// Handoff is what the CLI delivers to the Primary ADE: where the
// Bootstrap's own instructions are. The CLI writes no instructions.
type Handoff struct {
	PrimaryADE       string   `json:"primary_ade"`
	PrimaryName      string   `json:"primary_name"`
	InstructionEntry string   `json:"instruction_entry"`
	ReadFirst        []string `json:"read_first"`
	Questionnaire    string   `json:"questionnaire,omitempty"`
	Launchable       bool     `json:"launchable"`
}

// InitResult is the outcome of `gtt init`.
type InitResult struct {
	Schema      int                  `json:"schema"`
	Outcome     string               `json:"outcome"` // initialized | already-initialized | restored | rolled-back
	Project     string               `json:"project"`
	Bootstrap   lifecycle.Resolution `json:"bootstrap"`
	Selections  lifecycle.Selections `json:"selections"`
	Sources     []SourceReport       `json:"sources,omitempty"`
	Validation  string               `json:"validation"`
	Handoff     *Handoff             `json:"handoff,omitempty"`
	DesignState string               `json:"design_state"` // sources-selected | questionnaire | none
}

// Init is the primary orchestration workflow.
func (a *App) Init(ctx context.Context, o InitOptions) (InitResult, error) {
	res := InitResult{Schema: core.OutputSchema}
	det := a.detect(ctx)
	res.Project = det.Root
	store := lifecycle.Store{Root: det.Root}

	journal, err := store.LoadJournal()
	if err != nil {
		return res, err
	}
	if journal != nil {
		return a.interrupted(ctx, det.Root, journal, o)
	}
	if a.Factory.Installed(det.Root) {
		res.Outcome = "already-initialized"
		return res, nil
	}
	if !det.Confident {
		a.Report.Info("No project detected.")
		ok, err := a.confirm("Initialize GTT in this directory ("+det.Root+")?", false)
		if err != nil {
			return res, err
		}
		if !ok {
			return res, core.ErrCancelled
		}
	}

	a.Report.Info("GTT Project Initialization\n\nProject:\n  %s\n\nBootstrap:\n  Resolving GTT Bootstrap...", filepath.Base(det.Root))
	pkg, err := a.resolve(ctx, ports.ResolveRequest{Path: o.Resolve.Path, Version: o.Resolve.Version, Offline: o.Resolve.Offline})
	if err != nil {
		return res, err
	}
	res.Bootstrap = resolution(pkg)
	a.Report.Info("  Version: %s\n  Integrity: PASS\n  Compatibility: PASS", pkg.Release.Bootstrap.Version)
	catalog := a.Factory.At(pkg.Root)

	if conflicts, err := a.Installer.Conflicts(pkg, det.Root); err != nil {
		return res, err
	} else if len(conflicts) > 0 {
		return res, &core.Error{Code: core.ExitConflict, Unmodified: true,
			What: "GTT cannot be installed: these paths already exist in the project.", Why: strings.Join(conflicts, "\n"),
			Next: "Resolve the conflict explicitly. GTT never overwrites or merges them."}
	}

	if !o.NoRestore {
		if snap, ok := a.findSnapshot(det.Root, o.Snapshot); ok {
			restore, err := a.offerRestore(snap, o.Snapshot != "")
			if err != nil {
				return res, err
			}
			if restore {
				return a.restore(ctx, det.Root, pkg, snap, res)
			}
		}
	}

	sel, err := a.collect(ctx, det.Root, pkg, catalog, o)
	if err != nil {
		return res, err
	}
	res.Selections = sel
	registry, err := ade.Registry(ctx, catalog)
	if err != nil {
		return res, err
	}
	a.Report.Info("\nSummary:\n\n  Primary ADE: %s\n  Participating: %s\n  Sources: %d\n  Language: %s\n  Methodology: %s\n",
		ade.Names(registry, []string{sel.Primary}), ade.Names(registry, sel.Participating), len(sel.Sources), sel.Language, sel.Profile)
	if ok, err := a.confirm("Proceed?", true); err != nil {
		return res, err
	} else if !ok {
		return res, core.ErrCancelled
	}

	layout, err := a.Installer.Layout(pkg)
	if err != nil {
		return res, err
	}
	journal = &lifecycle.Journal{State: lifecycle.StateStaging, StartedAt: lifecycle.Timestamp(), Bootstrap: res.Bootstrap,
		Selections: sel, CorePaths: layout.Paths}
	if err := store.SaveJournal(journal); err != nil {
		return res, err
	}
	return a.install(ctx, det.Root, pkg, journal, res, o)
}

// collect gathers the human's operational choices. Nothing is installed here.
func (a *App) collect(ctx context.Context, root string, pkg ports.Package, catalog ports.Bootstrap, o InitOptions) (lifecycle.Selections, error) {
	var sel lifecycle.Selections
	need := func(what, flag string) error {
		return &core.Error{Code: core.ExitUsage, Unmodified: true, What: what + " must be chosen by a human.",
			Why: "GTT never infers it.", Next: "Run in a terminal, or pass " + flag + "."}
	}

	// ADEs: detection is an observation; participation is a choice.
	registry, err := ade.Registry(ctx, catalog)
	if err != nil {
		return sel, err
	}
	candidates := ade.Detect(registry, root, a.Runner)
	index := map[string]int{}
	options := make([]ports.Option, len(candidates))
	for i, c := range candidates {
		index[c.ID] = i
		signal := "no signal detected"
		if len(c.Signals) > 0 {
			sel.Detected = append(sel.Detected, c.ID)
			signal = "detected: " + strings.Join(c.Signals, ", ")
		}
		options[i] = ports.Option{Label: c.Name, Detail: signal}
	}
	switch {
	case len(o.ADEs) > 0:
		for _, id := range o.ADEs {
			if _, ok := index[id]; !ok {
				return sel, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Unknown ADE: " + id,
					Why: "Supported by this Bootstrap: " + ade.Names(nil, ids(registry))}
			}
		}
		sel.Participating = o.ADEs
	case a.Prompt.Interactive():
		picked, err := a.Prompt.MultiSelect("ADE candidates (detection is not participation).\nSelect the ADEs that will participate in this project:", options)
		if err != nil {
			return sel, err
		}
		for _, i := range picked {
			sel.Participating = append(sel.Participating, candidates[i].ID)
		}
	default:
		return sel, need("The participating ADEs", "--ade id[,id]")
	}
	if len(sel.Participating) == 0 {
		return sel, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "No participating ADE was selected.",
			Why: "A GTT project has exactly one Primary ADE, chosen among its participating ADEs."}
	}
	switch {
	case o.Primary != "":
		sel.Primary = o.Primary
	case len(sel.Participating) == 1:
		sel.Primary = sel.Participating[0] // the only participant: nothing to decide
	case a.Prompt.Interactive():
		opts := make([]ports.Option, len(sel.Participating))
		for i, id := range sel.Participating {
			opts[i] = ports.Option{Label: candidates[index[id]].Name}
		}
		i, err := a.Prompt.Select("Select the Primary ADE (a workflow identifier; it has no governance authority):", opts)
		if err != nil {
			return sel, err
		}
		sel.Primary = sel.Participating[i]
	default:
		return sel, need("The Primary ADE", "--primary id")
	}
	if !contains(sel.Participating, sel.Primary) {
		return sel, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "The Primary ADE must be one of the participating ADEs."}
	}

	// Language, among those the Bootstrap supports.
	langs := pkg.Release.SupportedLanguages
	switch {
	case o.Language != "":
		if !contains(langs, o.Language) {
			return sel, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Unsupported language: " + o.Language,
				Why: "This Bootstrap supports: " + strings.Join(langs, ", ")}
		}
		sel.Language = o.Language
	case len(langs) == 1:
		sel.Language = langs[0]
	case len(langs) == 0:
	case a.Prompt.Interactive():
		opts := make([]ports.Option, len(langs))
		for i, l := range langs {
			opts[i] = ports.Option{Label: l}
		}
		i, err := a.Prompt.Select("Language:", opts)
		if err != nil {
			return sel, err
		}
		sel.Language = langs[i]
	default:
		return sel, need("The language", "--language "+strings.Join(langs, "|"))
	}

	// Initial sources: the CLI lists candidates and records the selection.
	// It never decides that a document is authoritative.
	if err := a.collectSources(ctx, root, pkg, &sel, o); err != nil {
		return sel, err
	}

	// Method Plan: the names and descriptions come from the Bootstrap.
	choices, err := methodology.Choices(ctx, catalog)
	if err != nil {
		return sel, err
	}
	switch {
	case o.Method != "":
		found := false
		for _, c := range choices {
			found = found || c.ID == o.Method
		}
		if !found {
			return sel, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Unknown Method Plan: " + o.Method}
		}
		sel.Profile = o.Method
	case a.Prompt.Interactive():
		opts := make([]ports.Option, len(choices))
		for i, c := range choices {
			opts[i] = ports.Option{Label: c.Label, Detail: c.Summary}
		}
		i, err := a.Prompt.Select("GTT Method Plan\n\nChoose how GTT is applied to this project:", opts)
		if err != nil {
			return sel, err
		}
		sel.Profile = choices[i].ID
		a.Report.Info("Bootstrap will apply the semantics of the selected plan.")
	default:
		return sel, need("The Method Plan", "--method <plan>")
	}
	return sel, nil
}

func (a *App) collectSources(ctx context.Context, root string, pkg ports.Package, sel *lifecycle.Selections, o InitOptions) error {
	owned, err := ownedPatterns(ctx, a.Factory.At(pkg.Root))
	if err != nil {
		return err
	}
	docs := project.Documents(root, func(rel string) bool { return export.Excluded(rel, owned) })
	switch {
	case len(o.Sources) > 0:
		for _, arg := range o.Sources {
			s := ParseSource(arg)
			full, err := fsx.SafeJoin(root, s.Path)
			if err != nil {
				return &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Invalid source path.", Why: err.Error()}
			}
			if info, err := os.Stat(full); err != nil || info.IsDir() {
				return &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Source document not found: " + s.Path}
			}
			sel.Sources = append(sel.Sources, s)
		}
	case o.NoSources:
	case !a.Prompt.Interactive():
		if len(docs) > 0 {
			return &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Initial source documents must be chosen by a human.",
				Why: "Candidates: " + strings.Join(docs, ", "), Next: "Pass --sources path[,path] or --no-sources."}
		}
	case len(docs) == 1:
		a.Report.Info("\nInitial project source detected:\n\n  %s\n", docs[0])
		ok, err := a.Prompt.Confirm("Use this as initial source material?", true)
		if err != nil {
			return err
		}
		if ok {
			sel.Sources = []lifecycle.Source{{Path: docs[0], Type: DefaultSourceType}}
		}
	case len(docs) > 1:
		opts := make([]ports.Option, len(docs))
		for i, d := range docs {
			opts[i] = ports.Option{Label: d}
		}
		picked, err := a.Prompt.MultiSelect("Candidate source documents (selection is not authority):", opts)
		if err != nil {
			return err
		}
		for _, i := range picked {
			sel.Sources = append(sel.Sources, lifecycle.Source{Path: docs[i], Type: DefaultSourceType})
		}
	}
	if len(sel.Sources) > 0 {
		return nil
	}
	// No sufficient design source: offer the Bootstrap-owned questionnaire.
	switch {
	case o.Questionnaire == "yes":
		sel.Questionnaire = true
	case o.Questionnaire == "no" || !a.Prompt.Interactive():
	default:
		a.Report.Info("\nNo sufficient design source document was detected.\n\nGTT Bootstrap provides an Initial Design Questionnaire.\n" +
			"It is owned by Bootstrap and will be materialized as project source material.\n")
		ok, err := a.Prompt.Confirm("Create questionnaire?", true)
		if err != nil {
			return err
		}
		sel.Questionnaire = ok
	}
	return nil
}

// install runs the transaction: stage and commit the core, install the
// participating ADEs, record the selections through the Bootstrap, validate.
// Any failure rolls the project back to where it started.
func (a *App) install(ctx context.Context, root string, pkg ports.Package, j *lifecycle.Journal, res InitResult, o InitOptions) (InitResult, error) {
	store := lifecycle.Store{Root: root}
	sel := j.Selections
	res.Selections, res.Bootstrap, res.Project = sel, j.Bootstrap, root
	step := func(name, state string, fn func() error) error {
		if j.Done(name) {
			return nil
		}
		j.State = state
		if err := store.SaveJournal(j); err != nil {
			return err
		}
		if err := fn(); err != nil {
			return err
		}
		j.Completed = append(j.Completed, name)
		return store.SaveJournal(j)
	}
	fail := func(err error) (InitResult, error) {
		rbErr := a.rollback(ctx, root, j)
		res.Outcome = "rolled-back"
		e := &core.Error{Code: core.CodeOf(err), What: "GTT initialization failed and was rolled back.", Why: err.Error(), Unmodified: rbErr == nil, Err: err}
		if rbErr != nil {
			e.What = "GTT initialization failed and the rollback was incomplete."
			e.Next = "Run: gtt init --rollback\n" + rbErr.Error()
		}
		return res, e
	}

	a.Report.Info("Installing GTT Core...")
	if err := step("core", lifecycle.StateStaging, func() error {
		ledger, err := a.Installer.Install(pkg, root)
		j.CoreLedger = ledger
		return err
	}); err != nil {
		return fail(err)
	}
	p := &Project{Root: root, B: a.Factory.At(root), Release: pkg.Release, Store: store}

	a.Report.Info("Installing ADE integrations...")
	if err := step("ade", lifecycle.StateInstalling, func() error {
		_, err := a.mutate(ctx, p.B, "ade.install", map[string]string{"from": pkg.Root,
			"participating": strings.Join(sel.Participating, ","), "primary": sel.Primary})
		return err
	}); err != nil {
		return fail(err)
	}
	if err := step("methodology", lifecycle.StateInstalling, func() error {
		args := map[string]string{"profile": sel.Profile}
		if sel.Language != "" {
			args["language"] = sel.Language
		}
		_, err := a.mutate(ctx, p.B, "methodology.profile.set", args)
		return err
	}); err != nil {
		return fail(err)
	}
	questionnaire, err := initialDesign(ctx, p.B)
	if err != nil {
		return fail(err)
	}
	if sel.Questionnaire {
		if err := step("questionnaire", lifecycle.StateInstalling, func() error {
			_, err := a.mutate(ctx, p.B, "template.materialize", map[string]string{"id": questionnaire.Template})
			return err
		}); err != nil {
			return fail(err)
		}
	}
	if err := step("index", lifecycle.StateInstalling, func() error { return a.reindex(ctx, p.B) }); err != nil {
		return fail(err)
	}

	a.Report.Info("Validating...")
	j.State = lifecycle.StateValidating
	if err := store.SaveJournal(j); err != nil {
		return fail(err)
	}
	tracked, err := a.processSources(ctx, p, sel.Sources)
	if err != nil {
		return fail(err)
	}
	res.Sources = tracked.Sources
	var v validation
	if tracked.validated != nil {
		v = *tracked.validated // the source step already validated this state
	} else if v, err = a.validateProject(ctx, p.B); err != nil {
		return fail(err)
	}
	if tracked.Failed() && v.Result == "pass" {
		return fail(core.Errorf(core.ExitFailure, "an initial source could not be applied: %s", describeFailures(tracked)))
	}
	res.Validation = v.Result
	if v.Result != "pass" {
		return fail(&core.Error{Code: core.ExitFailure, What: "Bootstrap validation failed.", Why: strings.Join(append(v.Failing, v.Messages...), "\n")})
	}

	st, _ := store.Load()
	st.Bootstrap, st.CoreLedger = &j.Bootstrap, j.CoreLedger
	st.History = append(st.History, lifecycle.Event{At: lifecycle.Timestamp(), Event: "init", Detail: "Bootstrap " + j.Bootstrap.Version})
	if err := store.Save(st); err != nil {
		return fail(err)
	}
	j.State = lifecycle.StateInitialized
	if err := store.ClearJournal(); err != nil {
		return res, err
	}
	a.log(root, "init", "success", map[string]any{"bootstrap": j.Bootstrap.Version, "primary": sel.Primary})

	a.Report.Info("Generating Bootstrap handoff...")
	res.Outcome = "initialized"
	switch {
	case len(sel.Sources) > 0:
		res.DesignState = "sources-selected"
	case sel.Questionnaire:
		res.DesignState = "questionnaire"
	default:
		res.DesignState = "none"
	}
	h, err := a.handoff(ctx, p, sel.Primary)
	if err != nil {
		return res, err
	}
	if sel.Questionnaire {
		h.Questionnaire = questionnaire.Output
	}
	res.Handoff = &h
	return res, nil
}

// rollback removes what an init transaction created.
func (a *App) rollback(ctx context.Context, root string, j *lifecycle.Journal) error {
	if j.Done("ade") || a.Factory.Installed(root) {
		// Best effort: let the Bootstrap remove the overlays it installed.
		b := a.Factory.At(root)
		if _, err := b.Execute(ctx, ports.OperationRequest{Operation: "ade.remove", Args: map[string]string{"all": "true"}, Apply: true}); err != nil {
			a.Report.Detail("ade.remove during rollback: " + err.Error())
		}
	}
	if err := a.Installer.Remove(root, ports.CoreLayout{Paths: j.CorePaths}); err != nil {
		return err
	}
	matches, _ := filepath.Glob(filepath.Join(root, ".gtt-staging-*"))
	for _, m := range matches {
		os.RemoveAll(m)
	}
	return lifecycle.Store{Root: root}.ClearJournal()
}

// interrupted handles a recorded, unfinished init. It uses the recorded
// transaction state; it never infers completion from directories existing.
func (a *App) interrupted(ctx context.Context, root string, j *lifecycle.Journal, o InitOptions) (InitResult, error) {
	res := InitResult{Schema: core.OutputSchema, Project: root, Bootstrap: j.Bootstrap, Selections: j.Selections}
	choice := -1
	switch {
	case o.Resume:
		choice = 0
	case o.Rollback:
		choice = 1
	case a.Prompt.Interactive():
		var err error
		choice, err = a.Prompt.Select(fmt.Sprintf("Incomplete GTT initialization detected.\n\nState:\n%s", j.State),
			[]ports.Option{{Label: "Resume"}, {Label: "Roll back"}, {Label: "Inspect"}, {Label: "Abort"}})
		if err != nil {
			return res, err
		}
	default:
		return res, &core.Error{Code: core.ExitConflict, Unmodified: true,
			What: "Incomplete GTT initialization detected (state " + j.State + ").",
			Next: "Run: gtt init --resume   or   gtt init --rollback"}
	}
	switch choice {
	case 0:
		pkg, err := a.resolve(ctx, ports.ResolveRequest{Path: o.Resolve.Path, Recorded: j.Bootstrap.Origin,
			Version: j.Bootstrap.Version, Offline: o.Resolve.Offline})
		if err != nil {
			return res, err
		}
		return a.install(ctx, root, pkg, j, res, o)
	case 1:
		if err := a.rollback(ctx, root, j); err != nil {
			return res, err
		}
		res.Outcome = "rolled-back"
		return res, nil
	case 2:
		data, _ := json.MarshalIndent(j, "", "  ")
		a.Report.Info("%s", data)
		return res, &core.Error{Code: core.ExitConflict, Unmodified: true, What: "The initialization is still incomplete.",
			Next: "Run: gtt init --resume   or   gtt init --rollback"}
	}
	return res, core.ErrCancelled
}

type questionnaireContract struct {
	Template string
	Output   string
}

// initialDesign reads the questionnaire contract from the Bootstrap: which
// template to request and where it materializes.
func initialDesign(ctx context.Context, b ports.Bootstrap) (questionnaireContract, error) {
	raw, err := b.Show(ctx, "initial-design")
	if err != nil {
		return questionnaireContract{}, err
	}
	var doc struct {
		Scaffold struct {
			InitialDesign struct {
				Questionnaire struct {
					Output  string `json:"output"`
					Handoff struct {
						Template string `json:"template"`
					} `json:"handoff"`
				} `json:"questionnaire"`
			} `json:"initial_design"`
		} `json:"scaffold"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return questionnaireContract{}, &core.Error{Code: core.ExitIntegrity, What: "The Bootstrap questionnaire contract is unreadable.", Why: err.Error()}
	}
	q := doc.Scaffold.InitialDesign.Questionnaire
	if q.Handoff.Template == "" {
		return questionnaireContract{}, &core.Error{Code: core.ExitIncompatible, What: "The Bootstrap declares no Initial Design Questionnaire."}
	}
	return questionnaireContract{Template: q.Handoff.Template, Output: q.Output}, nil
}

// handoff builds the delivery for the Primary ADE from the Bootstrap's ADE
// registry and elicitation contract.
func (a *App) handoff(ctx context.Context, p *Project, primary string) (Handoff, error) {
	h := Handoff{PrimaryADE: primary}
	registry, err := ade.Registry(ctx, p.B)
	if err != nil {
		return h, err
	}
	for _, r := range registry {
		if r.ID == primary {
			h.PrimaryName, h.InstructionEntry = r.Name, r.Entry
			h.Launchable = ade.Executor{ADE: r, Runner: a.Runner}.Available()
		}
	}
	layout, err := a.Installer.Layout(ports.Package{Root: p.Root, Release: p.Release})
	if err == nil {
		for _, path := range layout.Paths {
			if info, serr := os.Stat(filepath.Join(p.Root, filepath.FromSlash(path))); serr == nil && !info.IsDir() {
				h.ReadFirst = append(h.ReadFirst, path)
			}
		}
	}
	return h, nil
}

// LaunchPrimary starts the Primary ADE in the project. It is only ever
// called after the human asked for it.
func (a *App) LaunchPrimary(ctx context.Context) error {
	p, err := a.open(ctx)
	if err != nil {
		return err
	}
	var st adeState
	if _, err := a.query(ctx, p.B, "ade.state", nil, &st); err != nil {
		return err
	}
	registry, err := ade.Registry(ctx, p.B)
	if err != nil {
		return err
	}
	for _, r := range registry {
		if r.ID == st.Primary {
			return ade.Executor{ADE: r, Runner: a.Runner}.Execute(ctx, ports.HandoffContract{ProjectRoot: p.Root, InstructionEntry: r.Entry})
		}
	}
	return core.Errorf(core.ExitConflict, "no Primary ADE is recorded for this project")
}

// findSnapshot looks for a recovery snapshot of this project.
func (a *App) findSnapshot(root, explicit string) (snapshot.Info, bool) {
	if explicit != "" {
		info, err := snapshot.ReadFile(explicit)
		return info, err == nil
	}
	list := snapshot.Store{Home: a.Home}.List(root)
	if len(list) == 0 {
		return snapshot.Info{}, false
	}
	return list[0], true
}

func (a *App) offerRestore(s snapshot.Info, explicit bool) (bool, error) {
	a.Report.Info("\nGTT recovery snapshot detected.\n\nPrevious Bootstrap:\n  %s\n\nPrimary ADE:\n  %s\n\nMethodology:\n  %s\n",
		s.BootstrapVersion, s.Primary, s.Profile)
	if explicit {
		return true, nil
	}
	if !a.Prompt.Interactive() && !a.AssumeYes {
		a.Report.Info("Not restoring without a human decision (pass --snapshot <file> to restore, or --no-restore to silence this).")
		return false, nil
	}
	return a.confirm("Restore GTT installation?", true)
}

// restore reinstalls the core and lets the Bootstrap restore the snapshot.
func (a *App) restore(ctx context.Context, root string, pkg ports.Package, s snapshot.Info, res InitResult) (InitResult, error) {
	layout, err := a.Installer.Layout(pkg)
	if err != nil {
		return res, err
	}
	store := lifecycle.Store{Root: root}
	j := &lifecycle.Journal{State: lifecycle.StateStaging, StartedAt: lifecycle.Timestamp(), Bootstrap: res.Bootstrap, CorePaths: layout.Paths}
	if err := store.SaveJournal(j); err != nil {
		return res, err
	}
	fail := func(err error) (InitResult, error) {
		rb := a.rollback(ctx, root, j)
		return res, &core.Error{Code: core.CodeOf(err), What: "GTT recovery failed and was rolled back.", Why: err.Error(), Unmodified: rb == nil, Err: err}
	}
	a.Report.Info("Installing GTT Core...")
	ledger, err := a.Installer.Install(pkg, root)
	if err != nil {
		return fail(err)
	}
	j.CoreLedger, j.Completed, j.State = ledger, []string{"core"}, lifecycle.StateInstalling
	if err := store.SaveJournal(j); err != nil {
		return fail(err)
	}
	p := &Project{Root: root, B: a.Factory.At(root), Release: pkg.Release, Store: store}
	a.Report.Info("Restoring GTT configuration...")
	if _, err := a.mutate(ctx, p.B, "recovery.restore", map[string]string{"snapshot": s.Snapshot, "from": pkg.Root}); err != nil {
		return fail(err)
	}
	if err := a.reindex(ctx, p.B); err != nil {
		return fail(err)
	}
	a.Report.Info("Validating...")
	v, err := a.validateProject(ctx, p.B)
	if err != nil {
		return fail(err)
	}
	res.Validation = v.Result
	if v.Result != "pass" {
		return fail(&core.Error{Code: core.ExitFailure, What: "Bootstrap validation failed after restore.", Why: strings.Join(append(v.Failing, v.Messages...), "\n")})
	}
	st, _ := store.Load()
	st.Bootstrap, st.CoreLedger = &j.Bootstrap, ledger
	st.History = append(st.History, lifecycle.Event{At: lifecycle.Timestamp(), Event: "restore", Detail: "snapshot " + s.Snapshot})
	if err := store.Save(st); err != nil {
		return fail(err)
	}
	if err := store.ClearJournal(); err != nil {
		return res, err
	}
	tracked, err := a.processSources(ctx, p, nil)
	if err == nil {
		res.Sources = tracked.Sources
	}
	res.Outcome = "restored"
	res.Selections = lifecycle.Selections{Participating: s.Participating, Primary: s.Primary, Profile: s.Profile, Language: s.Language}
	h, err := a.handoff(ctx, p, s.Primary)
	if err == nil {
		res.Handoff = &h
	}
	a.log(root, "restore", "success", map[string]any{"bootstrap": j.Bootstrap.Version})
	return res, nil
}

// ownedPatterns asks a Bootstrap which paths GTT owns (static ownership).
func ownedPatterns(ctx context.Context, b ports.Bootstrap) ([]string, error) {
	raw, err := b.Show(ctx, "export-policy")
	if err != nil {
		return nil, err
	}
	var doc struct {
		Ownership struct {
			Static []struct {
				Pattern string `json:"pattern"`
			} `json:"static"`
		} `json:"ownership"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, &core.Error{Code: core.ExitIntegrity, What: "The Bootstrap export policy is unreadable.", Why: err.Error()}
	}
	out := make([]string, len(doc.Ownership.Static))
	for i, s := range doc.Ownership.Static {
		out[i] = s.Pattern
	}
	return out, nil
}

func describeFailures(r SourcesResult) string {
	var parts []string
	for _, s := range r.Sources {
		if s.Outcome == "failed" {
			parts = append(parts, s.Path+" ("+s.Reason+")")
		}
	}
	return strings.Join(parts, "; ")
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func ids(registry []ports.ADE) []string {
	out := make([]string, len(registry))
	for i, r := range registry {
		out[i] = r.ID
	}
	return out
}
