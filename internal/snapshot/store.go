// Package snapshot stores recovery snapshots. The Bootstrap defines what a
// snapshot contains; this package only keeps the document, outside the
// project, so that it survives `gtt clean`.
package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GTT-Community/gtt-cli/internal/fsx"
)

// Meta is the CLI's own record around a Bootstrap snapshot.
type Meta struct {
	Schema            int    `json:"schema"`
	ProjectRoot       string `json:"project_root"`
	CreatedAt         string `json:"created_at"`
	CLIVersion        string `json:"cli_version"`
	BootstrapOrigin   string `json:"bootstrap_origin,omitempty"`
	BootstrapChecksum string `json:"bootstrap_checksum,omitempty"`
	Reason            string `json:"reason"`
}

// Info describes one stored snapshot.
type Info struct {
	Dir      string `json:"dir"`
	Snapshot string `json:"snapshot"` // path of the Bootstrap snapshot document
	Meta     Meta   `json:"meta"`
	// Summary fields read from the Bootstrap document for display only.
	BootstrapVersion string   `json:"bootstrap_version"`
	Primary          string   `json:"primary_ade"`
	Participating    []string `json:"participating"`
	Profile          string   `json:"methodology_profile"`
	Language         string   `json:"language"`
}

// Store keeps snapshots under the user-level GTT home.
type Store struct{ Home string }

// Key identifies a project by its absolute path.
func Key(projectRoot string) string {
	sum := sha256.Sum256([]byte(projectRoot))
	return filepath.Base(projectRoot) + "-" + hex.EncodeToString(sum[:6])
}

func (s Store) dir(projectRoot string) string {
	return filepath.Join(s.Home, "snapshots", Key(projectRoot))
}

// Save stores a Bootstrap snapshot document verbatim plus the CLI metadata.
func (s Store) Save(projectRoot string, document []byte, meta Meta) (Info, error) {
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	dir := filepath.Join(s.dir(projectRoot), strings.ReplaceAll(stamp, ".", ""))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Info{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, "snapshot.json"), document, 0o644); err != nil {
		return Info{}, err
	}
	meta.Schema, meta.ProjectRoot = 1, projectRoot
	if err := fsx.WriteJSON(filepath.Join(dir, "meta.json"), meta); err != nil {
		return Info{}, err
	}
	return Read(dir)
}

// Read loads one snapshot directory.
func Read(dir string) (Info, error) {
	info := Info{Dir: dir, Snapshot: filepath.Join(dir, "snapshot.json")}
	_ = fsx.ReadJSON(filepath.Join(dir, "meta.json"), &info.Meta)
	return info, describe(&info)
}

// ReadFile loads a snapshot given directly as a file (portable recovery).
func ReadFile(path string) (Info, error) {
	info := Info{Dir: filepath.Dir(path), Snapshot: path}
	return info, describe(&info)
}

func describe(info *Info) error {
	data, err := os.ReadFile(info.Snapshot)
	if err != nil {
		return err
	}
	var doc struct {
		Bootstrap struct {
			Version string `json:"version"`
		} `json:"bootstrap"`
		ADE struct {
			Primary       string   `json:"primary"`
			Participating []string `json:"participating"`
		} `json:"ade"`
		Methodology struct {
			Profile  string `json:"profile"`
			Language string `json:"language"`
		} `json:"methodology"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	info.BootstrapVersion, info.Primary, info.Participating = doc.Bootstrap.Version, doc.ADE.Primary, doc.ADE.Participating
	info.Profile, info.Language = doc.Methodology.Profile, doc.Methodology.Language
	return nil
}

// List returns the snapshots of a project, newest first.
func (s Store) List(projectRoot string) []Info {
	entries, err := os.ReadDir(s.dir(projectRoot))
	if err != nil {
		return nil
	}
	var out []Info
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if info, err := Read(filepath.Join(s.dir(projectRoot), e.Name())); err == nil {
			out = append(out, info)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Dir > out[j].Dir })
	return out
}
