package bootstrap

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

var (
	sectionRE = regexp.MustCompile(`^([a-z_]+):\s*$`)
	layerRE   = regexp.MustCompile(`^\s+(engine|domain):\s*([^\s#]+)`)
	entryRE   = regexp.MustCompile(`^\s+-\s*\{\s*path:\s*([^,}]+)`)
	trackRE   = regexp.MustCompile(`^\s+initial_sources:\s*\{\s*path:\s*([^,}]+)`)
)

// Layout reads from the scaffold manifest what the portable core is: the
// engine and domain layers, the entry points and, when declared, where the
// initial-source tracking record lives. Only installation
// structure is read; nothing else in the manifest is interpreted here.
func Layout(pkg ports.Package) (ports.CoreLayout, error) {
	manifest := pkg.Release.Scaffold.Manifest
	if manifest == "" {
		return ports.CoreLayout{}, &core.Error{Code: core.ExitIntegrity, Unmodified: true,
			What: "The Bootstrap release does not name its scaffold manifest."}
	}
	f, err := os.Open(filepath.Join(pkg.Root, filepath.FromSlash(manifest)))
	if err != nil {
		return ports.CoreLayout{}, &core.Error{Code: core.ExitIntegrity, Unmodified: true,
			What: "The Bootstrap scaffold manifest is missing.", Why: err.Error()}
	}
	defer f.Close()
	var layout ports.CoreLayout
	section := ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if m := sectionRE.FindStringSubmatch(line); m != nil {
			section = m[1]
			continue
		}
		switch section {
		case "layers":
			if m := layerRE.FindStringSubmatch(line); m != nil {
				path := strings.TrimSuffix(m[2], "/")
				layout.Paths = append(layout.Paths, path)
				if m[1] == "engine" {
					layout.Package = append(layout.Package, path)
				}
			}
		case "tracking":
			if m := trackRE.FindStringSubmatch(line); m != nil {
				layout.Tracking = strings.TrimSpace(m[1])
			}
		case "entry_points":
			if m := entryRE.FindStringSubmatch(line); m != nil {
				layout.Paths = append(layout.Paths, strings.TrimSpace(m[1]))
				layout.Package = append(layout.Package, strings.TrimSpace(m[1]))
			}
		}
	}
	if len(layout.Paths) < 2 {
		return layout, &core.Error{Code: core.ExitIntegrity, Unmodified: true,
			What: "The Bootstrap scaffold manifest does not declare its layers."}
	}
	return layout, sc.Err()
}
