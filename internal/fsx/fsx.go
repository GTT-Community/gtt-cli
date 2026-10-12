// Package fsx provides path safety, hashing and copy primitives.
package fsx

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// IgnoredDirs are never hashed, copied or exported as package content.
var IgnoredDirs = map[string]bool{".git": true, "__pycache__": true}

// rooted reports a path that is not relative on some platform: absolute,
// carrying a volume name (C:, \\server\share), or starting with a separator.
// On Windows "/etc/passwd" is not absolute, yet it names the current drive's
// root: it is never accepted as a path relative to the project.
func rooted(p string) bool {
	return filepath.IsAbs(p) || filepath.VolumeName(p) != "" || strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) ||
		(len(p) >= 2 && p[1] == ':')
}

// escapes reports whether a cleaned relative path leaves its base.
func escapes(clean string) bool {
	return clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

// SafeJoin joins rel under root and rejects absolute or rooted paths,
// traversal and any existing symlink component that resolves outside root.
func SafeJoin(root, rel string) (string, error) {
	if rel == "" || rooted(rel) || strings.HasPrefix(rel, "~") {
		return "", fmt.Errorf("unsafe path %q: must be relative to the project", rel)
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if escapes(clean) {
		return "", fmt.Errorf("unsafe path %q: escapes the project", rel)
	}
	full := filepath.Join(root, clean)
	if err := withinResolved(root, full); err != nil {
		return "", err
	}
	return full, nil
}

// withinResolved verifies that the deepest existing ancestor of full, once
// symlinks are resolved, is still inside root.
func withinResolved(root, full string) error {
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	probe := full
	for {
		real, err := filepath.EvalSymlinks(probe)
		if err == nil {
			rel, rerr := filepath.Rel(rootReal, real)
			if rerr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return fmt.Errorf("unsafe path %q: resolves outside the project through a symlink", full)
			}
			return nil
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return fmt.Errorf("unsafe path %q", full)
		}
		probe = parent
	}
}

// linkStaysInside checks a symlink's target as written, whether or not it
// exists: a dangling link that points outside root is as unsafe as a live
// one, and cannot be caught by resolving it.
func linkStaysInside(root, link string) error {
	target, err := os.Readlink(link)
	if err != nil {
		return err
	}
	if rooted(target) {
		return fmt.Errorf("unsafe symlink %q: points to %q, outside the project", link, target)
	}
	rel, err := filepath.Rel(root, filepath.Join(filepath.Dir(link), target))
	if err != nil || escapes(rel) {
		return fmt.Errorf("unsafe symlink %q: points to %q, outside the project", link, target)
	}
	return nil
}

// HashFile returns the hex sha256 of a file.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ListFiles returns every regular file under root as slash-separated paths
// relative to root, sorted. Symlinks are reported through the error: a
// package or an export source containing one that escapes root is unsafe.
func ListFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && IgnoredDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".pyc") {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			if err := linkStaysInside(root, p); err != nil {
				return err
			}
			if err := withinResolved(root, p); err != nil {
				return err
			}
		}
		rel, _ := filepath.Rel(root, p)
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out, err
}

// HashTree returns one sha256 over the sorted (path, content hash) pairs of
// the given files under root.
func HashTree(root string, files []string) (string, error) {
	h := sha256.New()
	for _, rel := range files {
		sum, err := HashFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s\x00%s\n", rel, sum)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// CopyFile copies src to dst preserving the mode; it creates parent
// directories and refuses to overwrite unless overwrite is true.
func CopyFile(src, dst string, overwrite bool) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if overwrite {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, flags, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// Exists reports whether a path exists (following nothing).
func Exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// WriteJSON writes v atomically as indented JSON.
func WriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ReadJSON reads a JSON file into v; a missing file returns os.ErrNotExist.
func ReadJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// DiffFiles returns a compact line diff from the file at oldPath to the file
// at newPath, in unified style with two lines of context and at most limit
// lines of output. Binary or very large files are summarized, not printed.
func DiffFiles(oldPath, newPath, oldLabel, newLabel string, limit int) string {
	a, errA := os.ReadFile(oldPath)
	b, errB := os.ReadFile(newPath)
	head := "--- " + oldLabel + "\n+++ " + newLabel + "\n"
	switch {
	case errA != nil:
		return head + "(not present: " + oldLabel + ")\n"
	case errB != nil:
		return head + "(not present: " + newLabel + ")\n"
	case bytes.IndexByte(a, 0) >= 0 || bytes.IndexByte(b, 0) >= 0:
		return head + "(binary files differ)\n"
	}
	x, y := strings.Split(strings.TrimSuffix(string(a), "\n"), "\n"), strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if len(x)*len(y) > 4_000_000 {
		return head + fmt.Sprintf("(%d lines against %d: too large to show)\n", len(x), len(y))
	}
	// Longest common subsequence, then walk it back into edit operations.
	lcs := make([][]int, len(x)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(y)+1)
	}
	for i := len(x) - 1; i >= 0; i-- {
		for j := len(y) - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	type op struct {
		kind byte // ' ', '-', '+'
		text string
	}
	var ops []op
	for i, j := 0, 0; i < len(x) || j < len(y); {
		switch {
		case i < len(x) && j < len(y) && x[i] == y[j]:
			ops = append(ops, op{' ', x[i]})
			i, j = i+1, j+1
		case i < len(x) && (j == len(y) || lcs[i+1][j] >= lcs[i][j+1]):
			ops = append(ops, op{'-', x[i]})
			i++
		default:
			ops = append(ops, op{'+', y[j]})
			j++
		}
	}
	const context = 2
	keep := make([]bool, len(ops))
	for i, o := range ops {
		if o.kind == ' ' {
			continue
		}
		for k := max(0, i-context); k <= min(len(ops)-1, i+context); k++ {
			keep[k] = true
		}
	}
	var out strings.Builder
	out.WriteString(head)
	shown, gap := 0, false
	for i, o := range ops {
		if !keep[i] {
			gap = true
			continue
		}
		if shown == limit {
			out.WriteString("... (diff truncated)\n")
			break
		}
		if gap && shown > 0 {
			out.WriteString("...\n")
		}
		gap = false
		out.WriteString(string(o.kind) + o.text + "\n")
		shown++
	}
	return out.String()
}
