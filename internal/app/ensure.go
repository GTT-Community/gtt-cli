package app

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/fsx"
	"github.com/GTT-Community/gtt-cli/internal/lifecycle"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// BootstrapCheck is what `gtt init` found when a Bootstrap was already in
// the project.
type BootstrapCheck struct {
	Version  string `json:"version"`  // installed
	Required string `json:"required"` // the release this run requires
	Pinned   string `json:"pinned"`   // the release this CLI ships with
	Verified bool   `json:"verified"`
	// Modified are package files changed locally since they were installed;
	// Missing are package files that are gone. Neither is ever repaired
	// without the human's confirmation.
	Modified []string `json:"modified_locally,omitempty"`
	Missing  []string `json:"missing,omitempty"`
}

// ensure is `gtt init` in a project that already holds a Bootstrap: it must
// end with the required release present and verified, or fail. The matching
// release is reused and nothing is written; any other state is reported and
// changed only after an explicit human confirmation that --yes cannot give.
func (a *App) ensure(ctx context.Context, root string, o InitOptions, res InitResult) (InitResult, error) {
	b := a.Factory.At(root)
	rel, err := a.gate(ctx, b)
	if err != nil {
		return res, err
	}
	required := core.BootstrapPin
	switch {
	case o.Resolve.Path != "":
		pkg, err := a.Resolver.Resolve(ctx, o.Resolve.request())
		if err != nil {
			return res, err
		}
		required = pkg.Release.Bootstrap.Version
	case o.Resolve.Version != "":
		required = o.Resolve.Version
	}
	st, err := lifecycle.Store{Root: root}.Load()
	if err != nil {
		return res, err
	}
	modified, missing, err := a.packageState(root, rel, st)
	if err != nil {
		return res, err
	}
	check := &BootstrapCheck{Version: rel.Bootstrap.Version, Required: required, Pinned: core.BootstrapPin, Modified: modified, Missing: missing}
	res.Installed = check
	have, okHave := core.ParseSemver(check.Version)
	want, okWant := core.ParseSemver(required)
	if !okHave || !okWant {
		return res, &core.Error{Code: core.ExitIntegrity, Unmodified: true,
			What: "A Bootstrap version is not MAJOR.MINOR.PATCH (installed " + check.Version + ", required " + required + ")."}
	}
	cmp := core.CompareSemver(want, have)
	if cmp == 0 && len(missing) == 0 {
		problems, err := b.CheckContracts(ctx)
		if err != nil {
			return res, err
		}
		if len(problems) > 0 {
			return res, &core.Error{Code: core.ExitIntegrity, Unmodified: true,
				What: "The GTT Bootstrap " + check.Version + " installed in this project failed its own contract check.",
				Why:  strings.Join(problems, "\n"),
				Next: "Restore the Bootstrap files from version control, or reinstall: gtt clean, then gtt init."}
		}
		check.Verified = true
		res.Outcome = "already-initialized"
		return res, nil
	}
	if cmp < 0 {
		return res, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: "This project runs GTT Bootstrap " + check.Version + ", newer than the " + required + " this run requires.",
			Why:  "GTT CLI " + core.Version + " ships with GTT Bootstrap " + core.BootstrapPin + " and never downgrades a project.",
			Next: "Upgrade the GTT CLI to the release that ships with GTT Bootstrap " + check.Version + "."}
	}
	what := "This project runs GTT Bootstrap " + check.Version + "; this run requires " + required + "."
	question := "Update the GTT Bootstrap of this project from " + check.Version + " to " + required + "?"
	if cmp == 0 {
		what = fmt.Sprintf("GTT Bootstrap %s is incomplete in this project: %d file(s) are missing.\n  %s",
			check.Version, len(missing), strings.Join(missing, "\n  "))
		question = "Restore the missing files from GTT Bootstrap " + required + "?"
	}
	if !a.Prompt.Interactive() {
		return res, &core.Error{Code: core.ExitConflict, Unmodified: true, What: what,
			Why:  "Changing an installed Bootstrap needs an explicit human confirmation; --yes does not give it.",
			Next: "Run gtt init in a terminal and confirm, or run: gtt update"}
	}
	a.Report.Info("%s\n", what)
	ok, err := a.Prompt.Confirm(question, false)
	if err != nil {
		return res, err
	}
	if !ok {
		return res, core.ErrCancelled
	}
	upd, err := a.Update(ctx, UpdateOptions{Resolve: o.Resolve})
	res.Update = &upd
	if err != nil {
		return res, err
	}
	if upd.Outcome != "updated" {
		return res, core.Errorf(core.ExitFailure, "the Bootstrap is still not the required %s (update outcome: %s)", required, upd.Outcome)
	}
	check.Version, check.Verified, check.Missing, check.Modified = upd.To, true, nil, nil
	for _, path := range modified {
		if !contains(upd.Overwritten, path) {
			check.Modified = append(check.Modified, path)
		}
	}
	res.Outcome = "updated"
	return res, nil
}

// packageState compares the installed package files with the install ledger.
// Files the Bootstrap itself rewrites are left out: a difference there is
// not a customization. Without a ledger nothing can be told apart, and
// nothing is reported.
func (a *App) packageState(root string, rel ports.Release, st lifecycle.State) (modified, missing []string, err error) {
	if len(st.CoreLedger) == 0 {
		return nil, nil, nil
	}
	layout, err := a.Installer.Layout(ports.Package{Root: root, Release: rel})
	if err != nil {
		return nil, nil, err
	}
	skip := rewritten(layout, st)
	for path, sum := range st.CoreLedger {
		if !underAny(path, layout.Package) || skip(path) {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(path))
		switch now, herr := fsx.HashFile(full); {
		case herr != nil && !fsx.Exists(full):
			missing = append(missing, path)
		case herr != nil:
			return nil, nil, herr
		case now != sum:
			modified = append(modified, path)
		}
	}
	sort.Strings(modified)
	sort.Strings(missing)
	return modified, missing, nil
}

// rewritten tells which package paths the Bootstrap rewrites by itself: the
// ones its manifest declares as derived or as state, and the ones its own
// operations were seen to rewrite in this project.
func rewritten(layout ports.CoreLayout, st lifecycle.State) func(string) bool {
	seen := map[string]bool{}
	for _, p := range st.Rewritten {
		seen[p] = true
	}
	return func(path string) bool {
		if seen[path] {
			return true
		}
		for _, p := range layout.Rewritten {
			if path == p || (strings.HasSuffix(p, "/") && strings.HasPrefix(path, p)) {
				return true
			}
		}
		return false
	}
}

// noteRewritten records the files that no longer hold the content just
// installed: only the Bootstrap's own operations ran in between.
func noteRewritten(st *lifecycle.State, root string, installed map[string]string, paths []string) {
	known := map[string]bool{}
	for _, p := range st.Rewritten {
		known[p] = true
	}
	for _, path := range paths {
		if known[path] {
			continue
		}
		if now, err := fsx.HashFile(filepath.Join(root, filepath.FromSlash(path))); err == nil && now != installed[path] {
			st.Rewritten = append(st.Rewritten, path)
		}
	}
	sort.Strings(st.Rewritten)
}

// ledgerPaths lists the ledger entries that are package content.
func ledgerPaths(ledger map[string]string, layout ports.CoreLayout) []string {
	var out []string
	for path := range ledger {
		if underAny(path, layout.Package) {
			out = append(out, path)
		}
	}
	return out
}
