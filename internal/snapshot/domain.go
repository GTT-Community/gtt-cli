package snapshot

import (
	"archive/zip"
	"io"
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
	out := filepath.Join(dir, time.Now().UTC().Format("20060102T150405Z")+"-gtt-domain.zip")
	tmp := out + ".tmp"
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
