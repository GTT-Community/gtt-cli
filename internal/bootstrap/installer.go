package bootstrap

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/fsx"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Installer stages and commits the portable core.
type Installer struct{}

// Layout returns the core layout of a package.
func (Installer) Layout(pkg ports.Package) (ports.CoreLayout, error) { return Layout(pkg) }

// Conflicts lists core paths that already exist in the project. A conflict
// is reported, never merged.
func (Installer) Conflicts(pkg ports.Package, projectRoot string) ([]string, error) {
	layout, err := Layout(pkg)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, p := range layout.Paths {
		target, err := fsx.SafeJoin(projectRoot, p)
		if err != nil {
			return nil, &core.Error{Code: core.ExitIntegrity, Unmodified: true, What: "The Bootstrap manifest declares an unsafe path.", Why: err.Error()}
		}
		if fsx.Exists(target) {
			out = append(out, p)
		}
	}
	return out, nil
}

// Install stages the core next to the project, then commits it with one
// rename per top-level path. On any failure everything it created is removed.
func (i Installer) Install(pkg ports.Package, projectRoot string) (map[string]string, error) {
	layout, err := Layout(pkg)
	if err != nil {
		return nil, err
	}
	if conflicts, err := i.Conflicts(pkg, projectRoot); err != nil {
		return nil, err
	} else if len(conflicts) > 0 {
		return nil, &core.Error{Code: core.ExitConflict, Unmodified: true,
			What: "GTT cannot be installed: these paths already exist.", Why: strings.Join(conflicts, "\n"),
			Next: "Resolve the conflict explicitly; GTT never overwrites or merges them."}
	}
	stage, err := os.MkdirTemp(projectRoot, ".gtt-staging-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stage)
	ledger := map[string]string{}
	for _, p := range layout.Paths {
		src := filepath.Join(pkg.Root, filepath.FromSlash(p))
		info, err := os.Stat(src)
		if err != nil {
			return nil, &core.Error{Code: core.ExitIntegrity, Unmodified: true, What: "The Bootstrap package lacks a declared core path: " + p}
		}
		files := []string{""}
		if info.IsDir() {
			if files, err = fsx.ListFiles(src); err != nil {
				return nil, &core.Error{Code: core.ExitIntegrity, Unmodified: true, What: "The Bootstrap package is unsafe to copy.", Why: err.Error()}
			}
		}
		for _, rel := range files {
			from, to, key := src, filepath.Join(stage, filepath.FromSlash(p)), p
			if rel != "" {
				from, to, key = filepath.Join(src, filepath.FromSlash(rel)), filepath.Join(to, filepath.FromSlash(rel)), p+"/"+rel
			}
			if err := fsx.CopyFile(from, to, false); err != nil {
				return nil, err
			}
			if ledger[key], err = fsx.HashFile(to); err != nil {
				return nil, err
			}
		}
	}
	var committed []string
	for _, p := range layout.Paths {
		target := filepath.Join(projectRoot, filepath.FromSlash(p))
		if err := os.Rename(filepath.Join(stage, filepath.FromSlash(p)), target); err != nil {
			for _, done := range committed {
				os.RemoveAll(done)
			}
			return nil, &core.Error{Code: core.ExitFailure, Unmodified: true, What: "Could not commit the GTT core; the installation was rolled back.", Why: err.Error()}
		}
		committed = append(committed, target)
	}
	return ledger, nil
}

// Remove deletes the core paths from a project (rollback of Install).
func (Installer) Remove(projectRoot string, layout ports.CoreLayout) error {
	for _, p := range layout.Paths {
		target, err := fsx.SafeJoin(projectRoot, p)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(target); err != nil {
			return err
		}
	}
	return nil
}
