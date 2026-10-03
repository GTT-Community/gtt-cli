package app

import (
	"context"
	"encoding/json"
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
	Snapshot  string   `json:"snapshot,omitempty"`
}

// Update replaces the Bootstrap package content of a project with another
// release, transactionally. Project state - ADEs, Primary, plan, language,
// sources, governed artifacts - is not package content and is not touched;
// a package file changed since install is preserved, never overwritten.
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
	pkg, err := a.resolve(ctx, ports.ResolveRequest{Path: o.Resolve.Path, Version: o.Resolve.Version, Offline: o.Resolve.Offline, Refresh: true})
	if err != nil {
		return res, err
	}
	res.To = pkg.Release.Bootstrap.Version
	a.Report.Info("Target:\n  %s\n\nChecking compatibility...\nPASS\n", res.To)
	cur, _ := core.ParseSemver(res.From)
	tgt, _ := core.ParseSemver(res.To)
	sameContent := st.Bootstrap != nil && st.Bootstrap.Checksum == pkg.Checksum
	if core.CompareSemver(tgt, cur) == 0 && (sameContent || st.Bootstrap == nil) {
		res.Outcome = "already-current"
		return res, nil
	}
	if core.CompareSemver(tgt, cur) < 0 {
		return res, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: "Bootstrap " + res.To + " is older than the installed " + res.From + ".", Why: "Downgrading is not supported."}
	}
	// Migration gate: this CLI can only carry a project across releases
	// that keep the same schema and scaffold layout.
	a.Report.Info("Checking migration...")
	if pkg.Release.Bootstrap.SchemaVersion != p.Release.Bootstrap.SchemaVersion || pkg.Release.Scaffold.Version != p.Release.Scaffold.Version {
		return res, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: "Bootstrap " + res.To + " changes the schema or the scaffold layout.",
			Why:  "That needs a migration contract this CLI does not have.", Next: "Upgrade the GTT CLI."}
	}
	a.Report.Info("PASS\n")
	ledger := st.CoreLedger
	if len(ledger) == 0 {
		if ledger, err = a.baseLedger(ctx, o.Base, res.From); err != nil {
			return res, err
		}
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
		for _, rel := range files {
			if target[rel], err = fsx.HashFile(filepath.Join(pkg.Root, filepath.FromSlash(rel))); err != nil {
				return res, err
			}
		}
	}
	for rel, sum := range target {
		now, herr := fsx.HashFile(filepath.Join(p.Root, filepath.FromSlash(rel)))
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
		if _, still := target[rel]; still || !underAny(rel, layout.Package) {
			continue
		}
		if now, herr := fsx.HashFile(filepath.Join(p.Root, filepath.FromSlash(rel))); herr == nil && now == sum {
			res.Removed = append(res.Removed, rel)
		}
	}
	for _, list := range [][]string{res.Added, res.Replaced, res.Preserved, res.Removed} {
		sort.Strings(list)
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
		if !dry.OK() {
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
	a.Report.Info("PASS\n\nStaging update...")
	if err := makeBackup(ctx, a, p, backup, layout.Package); err != nil {
		os.RemoveAll(backup)
		return res, err
	}
	fail := func(err error) (UpdateResult, error) {
		rb := restoreBackup(p.Root, backup)
		res.Outcome = "rolled-back"
		e := &core.Error{Code: core.CodeOf(err), What: "The update failed and was rolled back.", Why: err.Error(), Unmodified: rb == nil, Err: err}
		if rb != nil {
			e.What, e.Next = "The update failed and the rollback was incomplete.", "Run: gtt update --rollback\n"+rb.Error()
		}
		return res, e
	}
	for _, rel := range append(append([]string{}, res.Added...), res.Replaced...) {
		if err := fsx.CopyFile(filepath.Join(pkg.Root, filepath.FromSlash(rel)), filepath.Join(p.Root, filepath.FromSlash(rel)), true); err != nil {
			return fail(err)
		}
	}
	for _, rel := range res.Removed {
		if err := os.Remove(filepath.Join(p.Root, filepath.FromSlash(rel))); err != nil {
			return fail(err)
		}
	}
	a.Report.Info("PASS\n\nValidating...")
	updated := a.Factory.At(p.Root) // the updated Bootstrap: a fresh runtime
	if _, err := a.gate(ctx, updated); err != nil {
		return fail(err)
	}
	if ade.Configured {
		out, err := updated.Execute(ctx, ports.OperationRequest{Operation: "ade.update", Args: map[string]string{"from": pkg.Root}, Apply: true})
		if err != nil {
			return fail(err)
		}
		if !out.OK() {
			return fail(core.Errorf(core.ExitFailure, "ADE update failed: %s", strings.TrimSpace(out.Stdout+"\n"+out.Stderr)))
		}
	}
	if out, err := updated.Execute(ctx, ports.OperationRequest{Operation: "index", Apply: true}); err != nil || !out.OK() {
		return fail(core.Errorf(core.ExitFailure, "the index could not be rebuilt after the update"))
	}
	v, err := a.validateProject(ctx, updated)
	if err != nil {
		return fail(err)
	}
	if v.Result != "pass" {
		return fail(core.Errorf(core.ExitFailure, "Bootstrap validation failed after the update: %s", strings.Join(append(v.Failing, v.Messages...), "; ")))
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
	st.History = append(st.History, lifecycle.Event{At: lifecycle.Timestamp(), Event: "update", Detail: res.From + " -> " + res.To})
	if err := p.Store.Save(st); err != nil {
		return fail(err)
	}
	if err := os.RemoveAll(backup); err != nil {
		return res, err
	}
	a.log(p.Root, "update", "success", map[string]any{"from": res.From, "to": res.To})
	res.Outcome = "updated"
	return res, nil
}

// baseLedger rebuilds an install ledger from the catalog of the installed
// version, for projects that were not installed by this CLI.
func (a *App) baseLedger(ctx context.Context, base, version string) (map[string]string, error) {
	if base == "" {
		return nil, &core.Error{Code: core.ExitConflict, Unmodified: true,
			What: "This project has no install ledger, so local changes to GTT files cannot be told apart from package content.",
			Why:  "It was not installed by the gtt CLI.",
			Next: "Pass --base /path/to/the/gtt-bootstrap-" + version + " catalog it was installed from."}
	}
	pkg, err := a.Resolver.Resolve(ctx, ports.ResolveRequest{Path: base, Version: version})
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
		for _, rel := range files {
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
