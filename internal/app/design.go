package app

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/fsx"
)

// The design a reset returns to the Bootstrap's templates: the governed
// context, the Epic designs, the ADRs and the backlog (Canon 3.0 §19).
var designPaths = []string{"gtt-domain/context", "gtt-domain/adr", "gtt-domain/backlog.md"}

// What elicitation left behind and a reset discards with the design.
var elicitationPaths = []string{"gtt-domain/proposals/bootstrap"}

// DesignResetResult is `gtt design reset`.
type DesignResetResult struct {
	Schema   int      `json:"schema"`
	Restored []string `json:"restored"`
	Removed  []string `json:"removed"`
	Backup   string   `json:"backup"`
}

// DesignReset starts the design over, before the first freeze: the governed
// context, Epic designs, ADRs and backlog go back to the Bootstrap's
// templates, and the elicitation working copies and SOURCE-BRIEF go away.
// Setup choices and recorded sources stay. Everything changed is copied to a
// local backup first. After a freeze there is no reset: a change to the
// ratified design is a change request.
func (a *App) DesignReset(ctx context.Context, opts ResolveOptions) (DesignResetResult, error) {
	res := DesignResetResult{Schema: core.OutputSchema}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	if fsx.Exists(filepath.Join(p.Root, "gtt-domain", ".frozen")) {
		return res, &core.Error{Code: core.ExitConflict, Unmodified: true,
			What: "The design is frozen: there is no reset of a ratified design.",
			Why:  "After a freeze the design changes only through the change flow (Canon 3.0 §18-19).",
			Next: "Write what must change in gtt-domain/change-request.md, then run: gtt change"}
	}
	pkg, err := a.catalogFor(ctx, p, opts)
	if err != nil {
		return res, err
	}
	template := map[string]string{} // rel -> package file
	for _, top := range designPaths {
		src := filepath.Join(pkg.Root, filepath.FromSlash(top))
		info, err := os.Stat(src)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			template[top] = src
			continue
		}
		files, err := fsx.ListFiles(src)
		if err != nil {
			return res, &core.Error{Code: core.ExitIntegrity, Unmodified: true, What: "The Bootstrap templates are unsafe to read.", Why: err.Error()}
		}
		for _, f := range files {
			template[top+"/"+f] = filepath.Join(src, filepath.FromSlash(f))
		}
	}
	// What the project has now, and what of it a reset changes.
	var restore, remove []string
	for rel, src := range template {
		now, herr := fsx.HashFile(filepath.Join(p.Root, filepath.FromSlash(rel)))
		want, _ := fsx.HashFile(src)
		if herr != nil || now != want {
			restore = append(restore, rel)
		}
	}
	for _, top := range append(append([]string{}, designPaths...), elicitationPaths...) {
		full := filepath.Join(p.Root, filepath.FromSlash(top))
		info, err := os.Stat(full)
		if err != nil || !info.IsDir() {
			continue
		}
		files, _ := fsx.ListFiles(full)
		for _, f := range files {
			if rel := top + "/" + f; template[rel] == "" {
				remove = append(remove, rel)
			}
		}
	}
	briefs, _ := filepath.Glob(filepath.Join(p.Root, "SOURCE-BRIEF.*"))
	for _, b := range briefs {
		remove = append(remove, filepath.Base(b))
	}
	sort.Strings(restore)
	sort.Strings(remove)
	if len(restore)+len(remove) == 0 {
		a.Report.Info("The design is already at the Bootstrap's templates: nothing to reset.")
		return res, nil
	}
	a.Report.Info("Design reset (before the first freeze)\n\nBack to the templates: %d file(s)\nRemoved: %d file(s) (Epic designs, ADR drafts, SOURCE-BRIEF, questionnaire or assessment copies)\n"+
		"Kept: setup choices (ADEs, autonomy profile, development method, language) and recorded sources.\n", len(restore), len(remove))
	a.Report.Detail(strings.Join(append(append([]string{}, restore...), remove...), "\n"))
	ok, err := a.confirm("Reset the design? A backup is kept under .gtt/local/design-reset/.", false)
	if err != nil {
		return res, err
	}
	if !ok {
		return res, core.ErrCancelled
	}
	res.Backup = filepath.ToSlash(filepath.Join(".gtt", "local", "design-reset", time.Now().UTC().Format("20060102T150405Z")))
	for _, rel := range append(append([]string{}, restore...), remove...) {
		from := filepath.Join(p.Root, filepath.FromSlash(rel))
		if !fsx.Exists(from) {
			continue
		}
		if err := fsx.CopyFile(from, filepath.Join(p.Root, filepath.FromSlash(res.Backup), filepath.FromSlash(rel)), true); err != nil {
			return res, &core.Error{Code: core.ExitFailure, Unmodified: true, What: "Could not back up the design; nothing was reset.", Why: err.Error()}
		}
	}
	for _, rel := range remove {
		if err := os.Remove(filepath.Join(p.Root, filepath.FromSlash(rel))); err != nil {
			return res, &core.Error{Code: core.ExitFailure, What: "Design reset stopped: could not remove " + rel, Why: err.Error(),
				Next: "The design before the reset is in " + res.Backup}
		}
		res.Removed = append(res.Removed, rel)
	}
	pruneEmptyParents(p.Root, remove)
	for _, rel := range restore {
		if err := fsx.CopyFile(template[rel], filepath.Join(p.Root, filepath.FromSlash(rel)), true); err != nil {
			return res, &core.Error{Code: core.ExitFailure, What: "Design reset stopped: could not restore " + rel, Why: err.Error(),
				Next: "The design before the reset is in " + res.Backup}
		}
		res.Restored = append(res.Restored, rel)
	}
	// Artifact identity is derived: retire what went away, register what came back.
	if _, err := a.mutate(ctx, p.B, "reconcile", map[string]string{"retire_missing": "true"}); err != nil {
		a.Report.Detail("reconcile after design reset: " + err.Error())
	}
	if _, err := a.mutate(ctx, p.B, "index", nil); err != nil {
		a.Report.Detail("index after design reset: " + err.Error())
	}
	a.log(p.Root, "design.reset", "ok", map[string]any{"restored": len(res.Restored), "removed": len(res.Removed)})
	return res, nil
}
