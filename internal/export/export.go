// Package export writes a clean delivery copy of a project. What is
// excluded comes from the Bootstrap export policy; this package holds no
// exclusion list of its own.
package export

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/fsx"
)

// Matches reports whether a slash-separated relative path is covered by a
// policy pattern: `dir/` covers the directory and everything under it, a
// pattern with a wildcard is matched against the whole path, anything else
// is an exact path.
func Matches(rel, pattern string) bool {
	if strings.HasSuffix(pattern, "/") {
		dir := strings.TrimSuffix(pattern, "/")
		return rel == dir || strings.HasPrefix(rel, pattern)
	}
	if strings.ContainsAny(pattern, "*?[") {
		ok, err := path.Match(pattern, rel)
		return err == nil && ok
	}
	return rel == pattern
}

// Excluded reports whether any pattern covers rel.
func Excluded(rel string, patterns []string) bool {
	for _, p := range patterns {
		if Matches(rel, p) {
			return true
		}
	}
	return false
}

// Result summarises an export.
type Result struct {
	Destination string `json:"destination"`
	Copied      int    `json:"files_copied"`
	Excluded    int    `json:"files_excluded"`
}

// Copy writes every file of src that no pattern covers into dst. The source
// is never modified. dst must not exist or must be empty; version-control
// metadata and dst itself (when inside src) are never copied.
func Copy(src, dst string, patterns []string) (Result, error) {
	res := Result{Destination: dst}
	if entries, err := os.ReadDir(dst); err == nil && len(entries) > 0 {
		return res, fmt.Errorf("destination %s exists and is not empty", dst)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return res, err
	}
	dstRel := ""
	if rel, err := filepath.Rel(src, dst); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		dstRel = filepath.ToSlash(rel)
	}
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || rel == dstRel {
				return filepath.SkipDir
			}
			if Excluded(rel, patterns) {
				res.Excluded += countFiles(p)
				return filepath.SkipDir
			}
			return nil
		}
		if Excluded(rel, patterns) {
			res.Excluded++
			return nil
		}
		target := filepath.Join(dst, filepath.FromSlash(rel))
		if d.Type()&fs.ModeSymlink != 0 {
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			res.Copied++
			return os.Symlink(link, target)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		res.Copied++
		return fsx.CopyFile(p, target, false)
	})
	return res, err
}

func countFiles(dir string) int {
	n := 0
	filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}
