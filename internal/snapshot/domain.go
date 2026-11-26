package snapshot

import (
	"archive/zip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// keepDomainBackups is how many zips of a project's gtt-domain are kept; the oldest go first.
const keepDomainBackups = 5

// domainRoots are the project folders a domain backup holds: the project's governed domain and its ideas.
var domainRoots = []string{"gtt-domain", "docs/ideas"}

// SaveDomain zips the project's gtt-domain/ and docs/ideas/ into <home>/backups/<project>/, outside the project, so that a
// disk, a clone or a move that loses the project does not lose its content. It returns the path ("" when there is nothing
// to keep). Symbolic links are skipped, and what is stored is only ever relative to the project.
func (s Store) SaveDomain(projectRoot string) (string, error) {
	dir := filepath.Join(s.Home, "backups", Key(projectRoot))
	var files []string
	for _, top := range domainRoots {
		base := filepath.Join(projectRoot, filepath.FromSlash(top))
		_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			files = append(files, path)
			return nil
		})
	}
	if len(files) == 0 {
		return "", nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	var out, tmp string
	if err := claimName(s.now(), time.Second, func(at time.Time) (err error) {
		out = filepath.Join(dir, at.Format("20060102T150405Z")+"-gtt-domain.zip")
		tmp, err = claimZip(out)
		return err
	}); err != nil {
		return "", err
	}
	if err := writeZip(tmp, projectRoot, files); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, out); err != nil {
		os.Remove(tmp)
		return "", err
	}
	prune(dir)
	return out, nil
}

// claimZip reserves out by creating its temporary file exclusively, and answers
// fs.ErrExist when that name is being written or already holds a backup. The
// order matters: only the holder of the temporary file may look for the backup.
func claimZip(out string) (string, error) {
	tmp := out + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if _, err := os.Lstat(out); !errors.Is(err, fs.ErrNotExist) {
		os.Remove(tmp)
		if err == nil {
			err = fs.ErrExist
		}
		return "", err
	}
	return tmp, nil
}

func writeZip(dest, root string, files []string) (err error) {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	zw := zip.NewWriter(f)
	for _, path := range files {
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		w, werr := zw.Create(filepath.ToSlash(rel))
		if werr != nil {
			return werr
		}
		src, oerr := os.Open(path)
		if oerr != nil {
			return oerr
		}
		_, cerr := io.Copy(w, src)
		src.Close()
		if cerr != nil {
			return cerr
		}
	}
	return zw.Close()
}

func prune(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var zips []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "-gtt-domain.zip") {
			zips = append(zips, e.Name())
		}
	}
	sort.Strings(zips)
	for len(zips) > keepDomainBackups {
		_ = os.Remove(filepath.Join(dir, zips[0]))
		zips = zips[1:]
	}
}
