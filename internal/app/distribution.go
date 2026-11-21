package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/export"
	"github.com/GTT-Community/gtt-cli/internal/fsx"
	"github.com/GTT-Community/gtt-cli/internal/lifecycle"
	"github.com/GTT-Community/gtt-cli/internal/ports"
	"github.com/GTT-Community/gtt-cli/internal/snapshot"
)

// ---------------------------------------------------------------- snapshot

// takeSnapshot asks the Bootstrap for a recovery snapshot and stores it
// outside the project.
func (a *App) takeSnapshot(ctx context.Context, p *Project, reason string) (snapshot.Info, error) {
	out, err := a.query(ctx, p.B, "recovery.snapshot", nil, nil)
	if err != nil {
		return snapshot.Info{}, err
	}
	if !out.OK() || !json.Valid([]byte(out.Stdout)) {
		return snapshot.Info{}, &core.Error{Code: core.ExitFailure, What: "The Bootstrap did not produce a recovery snapshot.", Why: strings.TrimSpace(out.Stderr)}
	}
	meta := snapshot.Meta{CreatedAt: lifecycle.Timestamp(), CLIVersion: core.Version, Reason: reason}
	if st, err := p.Store.Load(); err == nil && st.Bootstrap != nil {
		meta.BootstrapOrigin, meta.BootstrapChecksum = st.Bootstrap.Origin, st.Bootstrap.Checksum
	}
	info, err := snapshot.Store{Home: a.Home}.Save(p.Root, []byte(out.Stdout), meta)
	if err != nil {
		return info, err
	}
	_ = p.Store.Record("snapshot", info.Snapshot)
	a.log(p.Root, "snapshot", "success", map[string]any{"reason": reason})
	return info, nil
}

// SnapshotCreate is `gtt snapshot create`.
func (a *App) SnapshotCreate(ctx context.Context) (snapshot.Info, error) {
	p, err := a.open(ctx)
	if err != nil {
		return snapshot.Info{}, err
	}
	return a.takeSnapshot(ctx, p, "requested")
}

// SnapshotList is `gtt snapshot list`.
func (a *App) SnapshotList(ctx context.Context) []snapshot.Info {
	return snapshot.Store{Home: a.Home}.List(a.detect(ctx).Root)
}

// ------------------------------------------------------------ export/clean

type exportPolicy struct {
	CleanExport struct {
		Exclude              []string `json:"exclude"`
		RequiresConfirmation []string `json:"requires_confirmation"`
	} `json:"clean_export"`
}

// ExportResult is `gtt export --clean`.
type ExportResult struct {
	Schema int `json:"schema"`
	export.Result
	KeptForReview []string `json:"kept_requires_confirmation"`
}

// ExportClean writes a clean delivery copy. The development project is not
// modified; what is excluded is exactly what the Bootstrap policy lists.
func (a *App) ExportClean(ctx context.Context, dest string) (ExportResult, error) {
	res := ExportResult{Schema: core.OutputSchema}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	var pol exportPolicy
	if _, err := a.query(ctx, p.B, "export-policy", nil, &pol); err != nil {
		return res, err
	}
	if len(pol.CleanExport.Exclude) == 0 {
		return res, &core.Error{Code: core.ExitIncompatible, Unmodified: true, What: "The Bootstrap export policy lists nothing to exclude.",
			Why: "The CLI keeps no exclusion list of its own and will not guess one."}
	}
	if dest == "" {
		dest = filepath.Join("dist", filepath.Base(p.Root)+"-clean")
	}
	if !filepath.IsAbs(dest) {
		dest = filepath.Join(a.Cwd, dest)
	}
	if dest == p.Root {
		return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "The export destination cannot be the project itself."}
	}
	patterns := append([]string{}, pol.CleanExport.Exclude...)
	// Files GTT cannot prove it owns unchanged (an adopted ADE's, or ledger
	// files edited since install) are not excluded silently.
	if len(pol.CleanExport.RequiresConfirmation) > 0 {
		a.Report.Info("These files may be GTT-owned but were changed or adopted:\n  %s", strings.Join(pol.CleanExport.RequiresConfirmation, "\n  "))
		exclude := false
		if a.Prompt.Interactive() && !a.AssumeYes {
			if exclude, err = a.Prompt.Confirm("Exclude them from the clean export too?", false); err != nil {
				return res, err
			}
		}
		if exclude {
			patterns = append(patterns, pol.CleanExport.RequiresConfirmation...)
		} else {
			res.KeptForReview = pol.CleanExport.RequiresConfirmation
		}
	}
	patterns = append(patterns, lifecycle.JournalFile, ".gtt-staging-*", ".gtt-update-backup/")
	copied, err := export.Copy(p.Root, dest, patterns)
	res.Result = copied
	if err != nil {
		return res, &core.Error{Code: core.ExitFailure, What: "Clean export failed.", Why: err.Error()}
	}
	// A host AGENTS.md GTT joined is the host's: it is exported without GTT's block.
	if host, keep := fsx.HostText(filepath.Join(p.Root, fsx.SharedFile)); keep {
		if err := os.WriteFile(filepath.Join(dest, fsx.SharedFile), []byte(host), 0o644); err != nil {
			return res, &core.Error{Code: core.ExitFailure, What: "Clean export failed.", Why: err.Error()}
		}
	}
	// Nothing is recorded in the project: an export leaves it untouched.
	return res, nil
}

type cleanPlan struct {
	WouldRemove          []string `json:"would_remove"`
	RequiresConfirmation []string `json:"requires_confirmation"`
}

// CleanResult is `gtt clean`.
type CleanResult struct {
	Schema   int      `json:"schema"`
	Removed  []string `json:"removed"`
	Kept     []string `json:"kept"`
	Snapshot string   `json:"snapshot,omitempty"`
}

// Clean removes GTT from the current project: exactly the ownership set
// the Bootstrap reports, and only after an explicit human confirmation.
func (a *App) Clean(ctx context.Context, noSnapshot bool) (CleanResult, error) {
	res := CleanResult{Schema: core.OutputSchema}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	var plan cleanPlan
	if _, err := a.query(ctx, p.B, "clean.plan", nil, &plan); err != nil {
		return res, err
	}
	var st adeState
	if _, err := a.query(ctx, p.B, "ade.state", nil, &st); err != nil {
		return res, err
	}
	a.Report.Info("WARNING\n\nThis operation removes GTT from this project.\n\nPrimary ADE:\n  %s\n\nBootstrap:\n  %s\n\nIt would remove %d path(s). A recovery snapshot is recommended.\n",
		st.Primary, p.Release.Bootstrap.Version, len(plan.WouldRemove))
	a.Report.Detail(strings.Join(plan.WouldRemove, "\n"))

	snap := !noSnapshot
	if snap && a.Prompt.Interactive() && !a.AssumeYes {
		if snap, err = a.Prompt.Confirm("Create GTT recovery snapshot first?", true); err != nil {
			return res, err
		}
	}
	ok, err := a.confirm("Continue with destructive clean?", false)
	if err != nil {
		return res, err
	}
	if !ok {
		return res, core.ErrCancelled
	}
	if snap {
		info, err := a.takeSnapshot(ctx, p, "before clean")
		if err != nil {
			return res, err
		}
		res.Snapshot = info.Snapshot
	}
	// Validate every path before removing anything.
	targets := make([]string, 0, len(plan.WouldRemove))
	for _, rel := range plan.WouldRemove {
		if strings.ContainsAny(rel, "*?[") {
			matches, _ := filepath.Glob(filepath.Join(p.Root, filepath.FromSlash(rel)))
			for _, m := range matches {
				r, _ := filepath.Rel(p.Root, m)
				targets = append(targets, filepath.ToSlash(r))
			}
			continue
		}
		targets = append(targets, strings.TrimSuffix(rel, "/"))
	}
	for _, rel := range targets {
		if _, err := fsx.SafeJoin(p.Root, rel); err != nil {
			return res, &core.Error{Code: core.ExitIntegrity, Unmodified: true, What: "The clean plan contains an unsafe path; nothing was removed.", Why: err.Error()}
		}
	}
	// ADE overlays first, through the Bootstrap: it removes only the files
	// its ledger says GTT installed, and keeps the ones changed since.
	if st.Configured {
		out, err := a.mutate(ctx, p.B, "ade.remove", map[string]string{"all": "true"})
		if err != nil {
			return res, err
		}
		for _, line := range strings.Split(out.Stdout, "\n") {
			if f := strings.Fields(line); len(f) >= 3 && f[0] == "KEEP" {
				res.Kept = append(res.Kept, f[2])
			}
		}
	}
	sort.Slice(targets, func(i, j int) bool { return len(targets[i]) > len(targets[j]) })
	for _, rel := range targets {
		full := filepath.Join(p.Root, filepath.FromSlash(rel))
		if !fsx.Exists(full) || contains(res.Kept, rel) {
			continue
		}
		if fsx.IsShared(rel) {
			if err := fsx.StripOrRemove(full); err != nil {
				return res, &core.Error{Code: core.ExitFailure, What: "Clean stopped: could not take GTT's block out of " + rel, Why: err.Error()}
			}
			if fsx.Exists(full) {
				res.Kept = append(res.Kept, rel+" (host content kept; GTT's block removed)")
			} else {
				res.Removed = append(res.Removed, rel)
			}
			continue
		}
		if err := os.RemoveAll(full); err != nil {
			return res, &core.Error{Code: core.ExitFailure, What: "Clean stopped: could not remove " + rel, Why: err.Error()}
		}
		res.Removed = append(res.Removed, rel)
	}
	pruneEmptyParents(p.Root, targets)
	sort.Strings(res.Removed)
	return res, nil
}

// pruneEmptyParents removes directories left empty by the removal of
// GTT-owned files (never a directory that still holds host files).
func pruneEmptyParents(root string, removed []string) {
	for _, rel := range removed {
		for dir := filepath.Dir(filepath.Join(root, filepath.FromSlash(rel))); dir != root && strings.HasPrefix(dir, root); dir = filepath.Dir(dir) {
			if os.Remove(dir) != nil {
				break
			}
		}
	}
}

// ------------------------------------------------------------------ update

const updateBackup = ".gtt-update-backup"

// UpdateOptions are the flags of `gtt update`.
type UpdateOptions struct {
	Resolve  ResolveOptions
	Base     string // catalog of the installed version, when no install ledger exists
	Rollback bool
}

// UpdateResult is `gtt update`.
type UpdateResult struct {
	Schema    int      `json:"schema"`
	Outcome   string   `json:"outcome"` // updated | already-current | rolled-back
	From      string   `json:"from"`
	To        string   `json:"to"`
	Replaced  []string `json:"replaced"`
	Added     []string `json:"added"`
	Removed   []string `json:"removed"`
	Preserved []string `json:"preserved_modified"`
	// Overwritten are locally modified files the human confirmed replacing.
	Overwritten []string `json:"replaced_modified,omitempty"`
	Snapshot    string   `json:"snapshot,omitempty"`
	// Inbox is the migration of a 1.22 project to the inbox, when the new Bootstrap offers it.
	Inbox *InboxMigration `json:"inbox,omitempty"`
	// Findings are what the new Bootstrap's validators report on the project's own content (a project
	// older than the validators fails them). They are work to do, never a reason to leave the project on the old release.
	Findings []string `json:"findings,omitempty"`
	// DomainBackup is the zip of the project's gtt-domain/ kept outside the project before the update.
	DomainBackup string `json:"domain_backup,omitempty"`
	// Domain says whether the project's gtt-domain must be brought to the format of the new Bootstrap: that is the
	// agent's work (it reads the content), and the order reaches it as the next action. The update never touches gtt-domain/.
	Domain *DomainState `json:"domain,omitempty"`
	// Reconciled says the identity of files moved or removed since the install was brought up to date.
	Reconciled bool `json:"reconciled,omitempty"`
}

// DomainState is the Bootstrap's answer to `domain.status`.
type DomainState struct {
	Needed bool     `json:"needed"`
	Why    []string `json:"why,omitempty"`
	Epics  int      `json:"epics"`
}

// Update replaces the Bootstrap package content of a project with another
// release, transactionally. Project state - ADEs, Primary, plan, language,
// sources, governed artifacts - is not package content and is not touched;
// a package file changed since install is preserved: its difference is
// shown, and it is replaced only when the human confirms it in a terminal
// (--yes does not).
func (a *App) Update(ctx context.Context, o UpdateOptions) (UpdateResult, error) {
	res := UpdateResult{Schema: core.OutputSchema}
	det := a.detect(ctx)
	backup := filepath.Join(det.Root, updateBackup)
	if o.Rollback {
		if !fsx.Exists(backup) {
			return res, &core.Error{Code: core.ExitConflict, Unmodified: true, What: "There is no interrupted update to roll back."}
		}
		res.Outcome = "rolled-back"
		return res, restoreBackup(det.Root, backup)
	}
	if fsx.Exists(backup) {
		return res, &core.Error{Code: core.ExitConflict, Unmodified: true, What: "An interrupted update was detected.",
			Next: "Run: gtt update --rollback"}
	}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	res.From = p.Release.Bootstrap.Version
	a.Report.Info("Current Bootstrap:\n  %s\n", res.From)
	st, err := p.Store.Load()
	if err != nil {
		return res, err
	}
	req := o.Resolve.request()
	req.Refresh = true
	pkg, err := a.resolve(ctx, req)
	if err != nil {
		return res, err
	}
	res.To = pkg.Release.Bootstrap.Version
	a.Report.Info("Target:\n  %s\n\nChecking compatibility...\nPASS\n", res.To)
	cur, _ := core.ParseSemver(res.From)
	tgt, _ := core.ParseSemver(res.To)
	same := core.CompareSemver(tgt, cur) == 0
	if core.CompareSemver(tgt, cur) < 0 {
		return res, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: "Bootstrap " + res.To + " is older than the installed " + res.From + ".", Why: "Downgrading is not supported."}
	}
	// Migration gate: this CLI can only carry a project across releases
	// that keep the same schema and scaffold layout.
	if !same {
		a.Report.Info("Checking migration...")
	}
	if pkg.Release.Bootstrap.SchemaVersion != p.Release.Bootstrap.SchemaVersion || pkg.Release.Scaffold.Version != p.Release.Scaffold.Version {
		return res, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: "Bootstrap " + res.To + " changes the schema or the scaffold layout.",
			Why:  "That needs a migration contract this CLI does not have.", Next: "Upgrade the GTT CLI."}
	}
	if !same {
		a.Report.Info("PASS\n")
	}
	ledger := st.CoreLedger
	rebuilt := false
	if len(ledger) == 0 {
		if ledger, err = a.baseLedger(ctx, o.Base, res.From); err != nil {
			if same {
				// The same release and no way to compare: as before, nothing to do, and nothing to complain about.
				res.Outcome = "already-current"
				return res, nil
			}
			return res, err
		}
		rebuilt = true
	}
	layout, err := a.Installer.Layout(pkg)
	if err != nil {
		return res, err
	}
	target := map[string]string{} // rel -> sha256 in the target package
	for _, top := range layout.Package {
		files, err := packageFiles(pkg.Root, top)
		if err != nil {
			return res, &core.Error{Code: core.ExitIntegrity, Unmodified: true, What: "The target Bootstrap is unsafe to read.", Why: err.Error()}
		}
		for _, rel := range withoutState(files, layout.State) {
			if target[rel], err = fsx.HashFile(filepath.Join(pkg.Root, filepath.FromSlash(rel))); err != nil {
				return res, err
			}
		}
	}
	for rel, sum := range target {
		now, herr := fsx.HashFile(filepath.Join(p.Root, filepath.FromSlash(rel)))
		if fsx.IsShared(rel) {
			if block, ok := fsx.BlockHash(filepath.Join(p.Root, filepath.FromSlash(rel))); ok {
				now = block // the host's lines around GTT's block are not GTT's to compare
			}
		}
		switch {
		case herr != nil:
			res.Added = append(res.Added, rel)
		case now == sum:
		case ledger[rel] == now:
			res.Replaced = append(res.Replaced, rel)
		default:
			res.Preserved = append(res.Preserved, rel)
		}
	}
	for rel, sum := range ledger {
		// A catalog file GTT once shipped among the project's proposals (the staged session adapters
		// before 1.8) and no longer ships: removed when untouched, so it stops posing as the project's own.
		retired := strings.HasPrefix(rel, "gtt-domain/proposals/") && !fsx.Exists(filepath.Join(pkg.Root, filepath.FromSlash(rel)))
		if _, still := target[rel]; still || (!underAny(rel, layout.Package) && !retired) {
			continue
		}
		if now, herr := fsx.HashFile(filepath.Join(p.Root, filepath.FromSlash(rel))); herr == nil && now == sum {
			res.Removed = append(res.Removed, rel)
		}
	}
	if err := a.offerReplace(p.Root, pkg, layout, st, &res); err != nil {
		return res, err
	}
	for _, list := range [][]string{res.Added, res.Replaced, res.Preserved, res.Removed} {
		sort.Strings(list)
	}
	if same && len(res.Added)+len(res.Replaced)+len(res.Removed) == 0 {
		// The release is already here and complete; what differs was kept.
		if rebuilt && len(st.CoreLedger) == 0 {
			// The ledger rebuilt from the published release is kept, so the next update needs no network for it.
			st.CoreLedger = ledger
			_ = p.Store.Save(st)
		}
		res.Outcome = "already-current"
		return res, nil
	}
	// ADE overlays: ask the Bootstrap what an update would do before any change.
	var ade adeState
	if _, err := a.query(ctx, p.B, "ade.state", nil, &ade); err != nil {
		return res, err
	}
	if ade.Configured {
		dry, err := p.B.Execute(ctx, ports.OperationRequest{Operation: "ade.update", Args: map[string]string{"from": pkg.Root}})
		if err != nil {
			return res, err
		}
		// The check above is made by the Bootstrap the project has now, which may predate the rule that a file edited
		// by the project is kept (the catalog's version saved beside it as .gtt-new). The real step below runs with the
		// updated Bootstrap, so a conflict that is only a local edit does not stop the update; any other one still does.
		if !dry.OK() && !onlyLocalEdits(dry.Stdout, func(rel string) bool { return sameFile(pkg.Root, p.Root, rel) }) {
			return res, &core.Error{Code: core.ExitConflict, Unmodified: true, What: "The ADE integrations cannot be updated without a conflict.",
				Why: strings.TrimSpace(dry.Stdout), Next: "Resolve the listed files; GTT does not overwrite local changes."}
		}
	}

	a.Report.Info("Creating recovery snapshot...")
	snap, err := a.takeSnapshot(ctx, p, "before update to "+res.To)
	if err != nil {
		return res, err
	}
	res.Snapshot = snap.Snapshot
	// The domain is never touched by an update, but a zip of it kept outside the project is the one thing that survives a lost
	// disk or a bad move, and the agent's migration that follows starts from a project that has one. Never a reason to stop.
	if zipped, zerr := (snapshot.Store{Home: a.Home}).SaveDomain(p.Root); zerr != nil {
		a.Report.Info("The gtt-domain backup could not be written (%v); the update goes on.", zerr)
	} else {
		res.DomainBackup = zipped
	}
	a.Report.Info("PASS\n\nStaging update...")
	if err := makeBackup(ctx, a, p, backup, layout.Package); err != nil {
		os.RemoveAll(backup)
		return res, err
	}
	var adeAdded []string // files the ADE step creates: a rollback removes them too
	fail := func(err error) (UpdateResult, error) {
		rb := restoreBackup(p.Root, backup)
		for _, rel := range adeAdded {
			if abs, serr := fsx.SafeJoin(p.Root, rel); serr == nil {
				os.Remove(abs)
			}
		}
		pruneEmptyParents(p.Root, adeAdded)
		res.Outcome = "rolled-back"
		e := &core.Error{Code: core.CodeOf(err), What: "The update failed and was rolled back.", Why: err.Error(), Unmodified: rb == nil, Err: err}
		if rb != nil {
			e.What, e.Next = "The update failed and the rollback was incomplete.", "Run: gtt update --rollback\n"+rb.Error()
		}
		return res, e
	}
	for _, rel := range append(append([]string{}, res.Added...), res.Replaced...) {
		from, to := filepath.Join(pkg.Root, filepath.FromSlash(rel)), filepath.Join(p.Root, filepath.FromSlash(rel))
		if fsx.IsShared(rel) {
			if _, joined := fsx.BlockHash(to); joined {
				text, err := os.ReadFile(from)
				if err == nil {
					err = fsx.WriteMerged(to, string(text))
				}
				if err != nil {
					return fail(err)
				}
				continue
			}
		}
		if err := fsx.CopyFile(from, to, true); err != nil {
			return fail(err)
		}
	}
	for _, rel := range res.Removed {
		if err := os.Remove(filepath.Join(p.Root, filepath.FromSlash(rel))); err != nil {
			return fail(err)
		}
	}
	pruneEmptyParents(p.Root, res.Removed)
	a.Report.Info("PASS\n\nValidating...")
	updated := a.Factory.At(p.Root) // the updated Bootstrap: a fresh runtime
	if _, err := a.gate(ctx, updated); err != nil {
		return fail(err)
	}
	if ade.Configured {
		if plan, perr := updated.Execute(ctx, ports.OperationRequest{Operation: "ade.update", Args: map[string]string{"from": pkg.Root}}); perr == nil {
			adeAdded = adeCreated(plan.Stdout)
		}
		out, err := updated.Execute(ctx, ports.OperationRequest{Operation: "ade.update", Args: map[string]string{"from": pkg.Root}, Apply: true})
		if err != nil {
			return fail(err)
		}
		if !out.OK() {
			return fail(core.Errorf(core.ExitFailure, "ADE update failed: %s", strings.TrimSpace(out.Stdout+"\n"+out.Stderr)))
		}
	}
	// Files the project moved or deleted since they were registered: the identity follows them (a move is
	// recorded with its old path; nothing is retired). When it cannot, that is a finding, not a failure.
	if _, rerr := a.mutate(ctx, updated, "reconcile", nil); rerr != nil {
		res.Findings = append(res.Findings, "identity: "+strings.TrimSpace(rerr.Error())+" (run: bash .gtt/scripts/gtt-reconcile.sh)")
	} else {
		res.Reconciled = true
	}
	if out, err := updated.Execute(ctx, ports.OperationRequest{Operation: "index", Apply: true}); err != nil || !out.OK() {
		// The index is derived from the project's own documents. When it cannot be rebuilt the cause is almost always their
		// content (two staged files with one name, a registered file that is gone), not the mechanics: the core is updated,
		// the finding says what to do, and the project is never left on the old Bootstrap because of it.
		why := "the index could not be rebuilt"
		if err == nil {
			if first := strings.TrimSpace(strings.SplitN(strings.TrimSpace(out.Stderr+"\n"+out.Stdout), "\n", 2)[0]); first != "" {
				why = first
			}
		}
		res.Findings = append(res.Findings, "index: "+why+" (fix what it names, then run: gtt sync)")
	}
	// The mechanics (copy, contracts, ADE files, index) are what a failed update rolls back. What the validators
	// say about the project's own content is reported: the update is done and the findings are work.
	v, err := a.validateProject(ctx, updated)
	if err != nil {
		return fail(err)
	}
	if v.Result != "pass" {
		res.Findings = append(res.Findings, append(v.Failing, v.Messages...)...)
	}
	// Commit: record the new release and drop the backup.
	st, _ = p.Store.Load()
	rec := resolution(pkg)
	st.Bootstrap = &rec
	if st.CoreLedger == nil {
		st.CoreLedger = map[string]string{}
	}
	for _, rel := range res.Removed {
		delete(st.CoreLedger, rel)
	}
	for rel, sum := range target {
		st.CoreLedger[rel] = sum
	}
	noteRewritten(&st, p.Root, target, append(append([]string{}, res.Added...), res.Replaced...))
	st.History = append(st.History, lifecycle.Event{At: lifecycle.Timestamp(), Event: "update", Detail: res.From + " -> " + res.To})
	if err := p.Store.Save(st); err != nil {
		return fail(err)
	}
	if err := os.RemoveAll(backup); err != nil {
		return res, err
	}
	a.log(p.Root, "update", "success", map[string]any{"from": res.From, "to": res.To})
	res.Outcome = "updated"
	// What the new Bootstrap asks of the project's own gtt-domain is told, never done here: it needs reading the content.
	var dom DomainState
	if _, derr := a.query(ctx, updated, "domain.status", nil, &dom); derr == nil && dom.Needed {
		res.Domain = &dom
	}
	// The update is done and stays done: a migration that cannot run is said, never a failure of the update.
	if m, merr := a.migrateInbox(ctx, updated, p.Root); m.Offered {
		res.Inbox = &m
		if merr != nil {
			res.Inbox.Pending = true
			res.Inbox.Report = merr.Error()
		}
	}
	return res, nil
}

// offerReplace shows how each locally modified package file differs from the
// target release and asks whether to replace them. The answer is the
// human's alone: without a terminal, and with --yes, they are kept.
func (a *App) offerReplace(root string, pkg ports.Package, layout ports.CoreLayout, st lifecycle.State, res *UpdateResult) error {
	skip := rewritten(layout, st)
	var custom, kept []string
	for _, rel := range res.Preserved {
		if skip(rel) {
			kept = append(kept, rel)
		} else {
			custom = append(custom, rel)
		}
	}
	if len(custom) == 0 {
		return nil
	}
	sort.Strings(custom)
	a.Report.Info("Bootstrap files modified in this project (%d):\n", len(custom))
	for _, rel := range custom {
		a.Report.Info("%s", fsx.DiffFiles(filepath.Join(root, filepath.FromSlash(rel)), filepath.Join(pkg.Root, filepath.FromSlash(rel)),
			rel+" (this project)", rel+" (GTT Bootstrap "+res.To+")", 40))
	}
	replace := false
	if a.Prompt.Interactive() {
		// No answer (end of input) is not a confirmation: the files are kept.
		var err error
		if replace, err = a.Prompt.Confirm(fmt.Sprintf("Replace these %d file(s) with the content of GTT Bootstrap %s? Your changes to them will be lost.",
			len(custom), res.To), false); err != nil && !errors.Is(err, core.ErrCancelled) {
			return err
		}
	}
	if !replace {
		a.Report.Info("Kept as they are: a locally modified file is never replaced without your confirmation.\n")
		return nil
	}
	res.Replaced, res.Overwritten, res.Preserved = append(res.Replaced, custom...), custom, kept
	return nil
}

// baseLedger rebuilds an install ledger from the catalog of the installed
// version, for projects that were not installed by this CLI.
func (a *App) baseLedger(ctx context.Context, base, version string) (map[string]string, error) {
	var pkg ports.Package
	var err error
	if base != "" {
		pkg, err = a.Resolver.Resolve(ctx, ports.ResolveRequest{Path: base, Version: version})
	} else {
		// No ledger and no --base (a project that was cloned, moved, shared, or installed by an older CLI): the
		// installed release is named by the project itself, so its catalog is the published tag, from the cache
		// or the source. Nothing is asked of the human.
		a.Report.Info("This project has no install ledger: rebuilding it from the published Bootstrap %s...", version)
		pkg, err = a.resolve(ctx, ports.ResolveRequest{Version: version})
		if err != nil {
			return nil, &core.Error{Code: core.ExitConflict, Unmodified: true,
				What: "This project has no install ledger and the Bootstrap " + version + " it runs could not be obtained to rebuild it.",
				Why:  err.Error(),
				Next: "Check the network, or pass --base /path/to/the/gtt-bootstrap-" + version + " catalog it was installed from."}
		}
	}
	if err != nil {
		return nil, err
	}
	layout, err := a.Installer.Layout(pkg)
	if err != nil {
		return nil, err
	}
	ledger := map[string]string{}
	for _, top := range layout.Package {
		files, err := packageFiles(pkg.Root, top)
		if err != nil {
			return nil, err
		}
		for _, rel := range withoutState(files, layout.State) {
			if ledger[rel], err = fsx.HashFile(filepath.Join(pkg.Root, filepath.FromSlash(rel))); err != nil {
				return nil, err
			}
		}
	}
	return ledger, nil
}

// packageFiles lists the files of one top-level package path, relative to root.
func packageFiles(root, top string) ([]string, error) {
	full := filepath.Join(root, filepath.FromSlash(top))
	info, err := os.Stat(full)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{top}, nil
	}
	files, err := fsx.ListFiles(full)
	if err != nil {
		return nil, err
	}
	for i := range files {
		files[i] = top + "/" + files[i]
	}
	return files, nil
}

// withoutState drops what the manifest declares per-project state: a package
// never ships it, even when a local catalog holds it.
func withoutState(files, state []string) []string {
	out := files[:0:0]
	for _, rel := range files {
		skip := false
		for _, s := range state {
			if rel == strings.TrimSuffix(s, "/") || (strings.HasSuffix(s, "/") && strings.HasPrefix(rel, s)) {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, rel)
		}
	}
	return out
}

func underAny(rel string, tops []string) bool {
	for _, t := range tops {
		if rel == t || strings.HasPrefix(rel, t+"/") {
			return true
		}
	}
	return false
}

// makeBackup copies everything an update may change: the package paths and
// the ADE files the Bootstrap ledger owns.
func makeBackup(ctx context.Context, a *App, p *Project, backup string, tops []string) error {
	var files []string
	for _, top := range tops {
		if !fsx.Exists(filepath.Join(p.Root, filepath.FromSlash(top))) {
			continue
		}
		list, err := packageFiles(p.Root, top)
		if err != nil {
			return err
		}
		files = append(files, list...)
	}
	var owned struct {
		Surfaces []struct {
			Path string `json:"path"`
		} `json:"surfaces"`
	}
	if _, err := a.query(ctx, p.B, "ade.owned", nil, &owned); err != nil {
		return err
	}
	for _, s := range owned.Surfaces {
		if _, err := fsx.SafeJoin(p.Root, s.Path); err == nil && fsx.Exists(filepath.Join(p.Root, filepath.FromSlash(s.Path))) {
			files = append(files, s.Path)
		}
	}
	for _, rel := range files {
		if err := fsx.CopyFile(filepath.Join(p.Root, filepath.FromSlash(rel)), filepath.Join(backup, "files", filepath.FromSlash(rel)), true); err != nil {
			return err
		}
	}
	return fsx.WriteJSON(filepath.Join(backup, "manifest.json"), map[string]any{"schema": 1, "files": files, "package_paths": tops})
}

// restoreBackup puts every backed-up file back and removes package files
// that the interrupted update added.
func restoreBackup(root, backup string) error {
	var m struct {
		Files        []string `json:"files"`
		PackagePaths []string `json:"package_paths"`
	}
	if err := fsx.ReadJSON(filepath.Join(backup, "manifest.json"), &m); err != nil {
		return err
	}
	had := map[string]bool{}
	for _, rel := range m.Files {
		had[rel] = true
	}
	for _, top := range m.PackagePaths {
		if !fsx.Exists(filepath.Join(root, filepath.FromSlash(top))) {
			continue
		}
		now, err := packageFiles(root, top)
		if err != nil {
			return err
		}
		for _, rel := range now {
			if !had[rel] {
				os.Remove(filepath.Join(root, filepath.FromSlash(rel)))
			}
		}
	}
	for _, rel := range m.Files {
		if err := fsx.CopyFile(filepath.Join(backup, "files", filepath.FromSlash(rel)), filepath.Join(root, filepath.FromSlash(rel)), true); err != nil {
			return err
		}
	}
	return os.RemoveAll(backup)
}

// ReleaseKeygen creates a release signing key pair in dir.
func (a *App) ReleaseKeygen(dir, id string) (ports.KeyPair, error) {
	return a.Signer.Keygen(dir, id)
}

// ReleaseSign signs a Bootstrap package directory with a release key. It is
// a publisher's operation: it writes only the package's signature file.
func (a *App) ReleaseSign(ctx context.Context, root, keyFile string) (ports.SignResult, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return ports.SignResult{}, err
	}
	return a.Signer.Sign(ctx, abs, keyFile)
}

// onlyLocalEdits is true when a failed `ade.update` check lists conflicts that are all "modified locally and
// changed in the catalog", or a file GTT did not record whose content is exactly the catalog's (`same` says so: a
// leftover of an update that was interrupted, which the Bootstrap being installed records instead of refusing),
// and nothing else (no PROBLEM line, no file that is foreign to GTT).
func onlyLocalEdits(out string, same func(rel string) bool) bool {
	conflicts := 0
	for _, line := range strings.Split(out, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "PROBLEM"):
			return false
		case strings.HasPrefix(t, "CONFLICT"):
			switch {
			case strings.Contains(t, "modified locally and changed in the catalog"):
			case strings.Contains(t, "exists but was not installed by GTT") && same != nil && len(strings.Fields(t)) > 2 && same(strings.Fields(t)[2]):
			default:
				return false
			}
			conflicts++
		}
	}
	return conflicts > 0
}

// sameFile is true when the project's file and the package's have the same bytes.
func sameFile(pkgRoot, projRoot, rel string) bool {
	if _, err := fsx.SafeJoin(projRoot, rel); err != nil {
		return false
	}
	a, errA := fsx.HashFile(filepath.Join(projRoot, filepath.FromSlash(rel)))
	b, errB := fsx.HashFile(filepath.Join(pkgRoot, filepath.FromSlash(rel)))
	return errA == nil && errB == nil && a == b
}

// adeCreated lists the files an `ade.update` plan will create (its ADD and RESTORE lines), so that an update that fails
// afterwards can take them away again: the backup holds what existed, never what the new Bootstrap adds.
func adeCreated(plan string) []string {
	var out []string
	for _, line := range strings.Split(plan, "\n") {
		f := strings.Fields(line)
		if len(f) >= 3 && (f[0] == "ADD" || f[0] == "RESTORE") {
			out = append(out, f[2])
		}
	}
	return out
}
