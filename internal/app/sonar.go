package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// GTT Sonar: the read-only page of what is being done in the project. The snapshot, the page and its template
// belong to the Bootstrap (operations sonar.snapshot and sonar.render); the CLI generates and opens the page
// and keeps the list of repos of the portfolio. A repo whose Bootstrap does not offer sonar.v1 is a row that
// says so, and the rest goes on. Sonar changes nothing in a project but its derived page in .gtt/local/.

// SonarRepo is one repo of the portfolio and how it answered.
type SonarRepo struct {
	Path    string `json:"path"`
	Name    string `json:"name,omitempty"`
	Version string `json:"bootstrap_version,omitempty"`
	State   string `json:"state"` // ok | update | not-gtt | incompatible | missing
	Reason  string `json:"reason,omitempty"`
}

// SonarResult is `gtt sonar`.
type SonarResult struct {
	Schema    int         `json:"schema"`
	Supported bool        `json:"supported"`
	Page      string      `json:"page,omitempty"`
	Message   string      `json:"message,omitempty"`
	Repos     []SonarRepo `json:"repos,omitempty"`
}

// SonarList is `gtt sonar list`.
type SonarList struct {
	Schema int         `json:"schema"`
	Config string      `json:"config"`
	Repos  []SonarRepo `json:"repos"`
}

type sonarConfig struct {
	Schema int `json:"schema"`
	Repos  []struct {
		Path string `json:"path"`
	} `json:"repos"`
}

// SonarConfigPath is the portfolio list: ~/.config/gtt/sonar.json (XDG_CONFIG_HOME when set). It is the
// configuration of Sonar, not the state of a project.
func SonarConfigPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(base, "gtt", "sonar.json")
}

// SonarPortfolioDir is where the portfolio page lives: ~/.gtt/sonar/ (under the GTT home).
func (a *App) sonarPortfolioDir() string { return filepath.Join(a.Home, "sonar") }

func notSonar(version string) string {
	return "The installed GTT Bootstrap " + version + " does not offer GTT Sonar (sonar.v1): run `gtt update` to get it."
}

func supportsSonar(ctx context.Context, b ports.Bootstrap) bool {
	return supports(ctx, b, "sonar.snapshot") && supports(ctx, b, "sonar.render")
}

// Sonar generates the page of the current repo. A Bootstrap without sonar.v1 answers in one line and the
// command ends well: nothing is broken, the repo only needs an update.
func (a *App) Sonar(ctx context.Context, ro ResolveOptions) (SonarResult, error) {
	res := SonarResult{Schema: core.OutputSchema}
	if det := a.detect(ctx); !a.Factory.Installed(det.Root) {
		// Not inside a project: this folder holds projects. Find them and show them all, no list to keep.
		return a.SonarFolder(ctx, a.Cwd, ro)
	}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	if !supportsSonar(ctx, p.B) {
		// An older Bootstrap: read the project with the newer one (read only) and say so in the page.
		return a.sonarPortfolio(ctx, []string{p.Root}, ro)
	}
	out, err := p.B.Execute(ctx, ports.OperationRequest{Operation: "sonar.render"})
	if err != nil {
		return res, err
	}
	if !out.OK() {
		return res, &core.Error{Code: core.ExitFailure, What: "The page could not be generated.", Why: strings.TrimSpace(out.Stdout + "\n" + out.Stderr), Unmodified: true}
	}
	res.Supported, res.Page = true, filepath.Join(p.Root, ".gtt", "local", "sonar.html")
	return res, nil
}

// refreshSonar regenerates the page quietly after an act of the human, when the Bootstrap has Sonar. It never
// fails the act.
func (a *App) refreshSonar(root string) {
	defer func() { _ = recover() }()
	if !a.Factory.Installed(root) {
		return
	}
	ctx := context.Background()
	b := a.Factory.At(root)
	if !supportsSonar(ctx, b) {
		return
	}
	_, _ = b.Execute(ctx, ports.OperationRequest{Operation: "sonar.render", Args: map[string]string{"quiet": "true"}})
}

// ----------------------------------------------------------------------------------------- the portfolio

func readSonarConfig() (sonarConfig, error) {
	var c sonarConfig
	data, err := os.ReadFile(SonarConfigPath())
	if os.IsNotExist(err) {
		return sonarConfig{Schema: 1}, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, &core.Error{Code: core.ExitFailure, What: "The portfolio list is unreadable: " + SonarConfigPath(), Why: err.Error(), Unmodified: true,
			Next: "Fix or delete that file; it only lists repos."}
	}
	return c, nil
}

func writeSonarConfig(c sonarConfig) error {
	c.Schema = 1
	path := SonarConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func absRepo(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// SonarAdd adds a repo to the portfolio list. The repo is not touched.
func (a *App) SonarAdd(_ context.Context, path string) (SonarList, error) {
	res := SonarList{Schema: core.OutputSchema, Config: SonarConfigPath()}
	abs, err := absRepo(path)
	if err != nil {
		return res, err
	}
	if info, serr := os.Stat(abs); serr != nil || !info.IsDir() {
		return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: fmt.Sprintf("%q is not a directory.", path)}
	}
	c, err := readSonarConfig()
	if err != nil {
		return res, err
	}
	for _, r := range c.Repos {
		if r.Path == abs {
			return a.SonarListRepos(context.Background())
		}
	}
	c.Repos = append(c.Repos, struct {
		Path string `json:"path"`
	}{Path: abs})
	if err := writeSonarConfig(c); err != nil {
		return res, &core.Error{Code: core.ExitFailure, What: "The portfolio list could not be written.", Why: err.Error(), Unmodified: true}
	}
	return a.SonarListRepos(context.Background())
}

// SonarRemove takes a repo out of the portfolio list. The repo is not touched.
func (a *App) SonarRemove(_ context.Context, path string) (SonarList, error) {
	res := SonarList{Schema: core.OutputSchema, Config: SonarConfigPath()}
	abs, err := absRepo(path)
	if err != nil {
		return res, err
	}
	c, err := readSonarConfig()
	if err != nil {
		return res, err
	}
	kept, found := c.Repos[:0:0], false
	for _, r := range c.Repos {
		if r.Path == abs {
			found = true
			continue
		}
		kept = append(kept, r)
	}
	if !found {
		return res, &core.Error{Code: core.ExitUsage, Unmodified: true, What: abs + " is not in the portfolio.", Next: "See: gtt sonar list"}
	}
	c.Repos = kept
	if err := writeSonarConfig(c); err != nil {
		return res, &core.Error{Code: core.ExitFailure, What: "The portfolio list could not be written.", Why: err.Error(), Unmodified: true}
	}
	return a.SonarListRepos(context.Background())
}

// SonarListRepos lists the portfolio with how each repo stands (read only).
func (a *App) SonarListRepos(ctx context.Context) (SonarList, error) {
	res := SonarList{Schema: core.OutputSchema, Config: SonarConfigPath()}
	c, err := readSonarConfig()
	if err != nil {
		return res, err
	}
	for _, r := range c.Repos {
		row := a.sonarProbe(ctx, r.Path)
		res.Repos = append(res.Repos, row.SonarRepo)
	}
	return res, nil
}

type sonarRow struct {
	SonarRepo
	b ports.Bootstrap
}

// sonarProbe asks one repo's own Bootstrap whether it offers Sonar.
func (a *App) sonarProbe(ctx context.Context, path string) sonarRow {
	row := sonarRow{SonarRepo: SonarRepo{Path: path, Name: filepath.Base(path)}}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		row.State, row.Reason = "missing", "the folder does not exist"
		return row
	}
	if !a.Factory.Installed(path) {
		row.State, row.Reason = "not-gtt", "GTT is not initialized here"
		return row
	}
	b := a.Factory.At(path)
	rel, err := a.gate(ctx, b)
	row.Version = rel.Bootstrap.Version
	if err != nil {
		row.State, row.Reason = "incompatible", "this Bootstrap and this CLI are not compatible"
		return row
	}
	if !supportsSonar(ctx, b) {
		row.State, row.Reason = "update", "Bootstrap "+rel.Bootstrap.Version+" has no sonar.v1"
		return row
	}
	row.State, row.b = "ok", b
	return row
}

// versionLess compares dotted numeric versions; anything unreadable counts as 0.
func versionLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(strings.SplitN(pa[i], "-", 2)[0])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(strings.SplitN(pb[i], "-", 2)[0])
		}
		if x != y {
			return x < y
		}
	}
	return false
}

// SonarAll builds the portfolio page: each repo answers with its own Bootstrap (sonar.snapshot), and the
// newest of those Bootstraps draws the page (sonar.render). A repo that cannot answer is a row that says
// "update GTT in this repo"; the rest goes on.
func (a *App) SonarAll(ctx context.Context, ro ResolveOptions) (SonarResult, error) {
	c, err := readSonarConfig()
	if err != nil {
		return SonarResult{Schema: core.OutputSchema}, err
	}
	if len(c.Repos) == 0 {
		return a.SonarFolder(ctx, a.Cwd, ro)
	}
	paths := make([]string, 0, len(c.Repos))
	for _, r := range c.Repos {
		paths = append(paths, r.Path)
	}
	return a.sonarPortfolio(ctx, paths, ro)
}

func (a *App) sonarPortfolio(ctx context.Context, paths []string, ro ResolveOptions) (SonarResult, error) {
	res := SonarResult{Schema: core.OutputSchema}
	dir := a.sonarPortfolioDir()
	snapDir := filepath.Join(dir, "snapshots")
	if err := os.RemoveAll(snapDir); err != nil {
		return res, &core.Error{Code: core.ExitFailure, What: "The portfolio folder is not writable: " + dir, Why: err.Error(), Unmodified: true}
	}
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		return res, &core.Error{Code: core.ExitFailure, What: "The portfolio folder is not writable: " + dir, Why: err.Error(), Unmodified: true}
	}
	var files []string
	var drawer ports.Bootstrap
	var viewer viewerPkg
	newest := ""
	for i, path := range paths {
		row := a.sonarProbe(ctx, path)
		file := filepath.Join(snapDir, fmt.Sprintf("repo-%02d.json", i+1))
		if row.State == "ok" {
			out, qerr := row.b.Execute(ctx, ports.OperationRequest{Operation: "sonar.snapshot"})
			switch {
			case qerr != nil || !out.OK() || !json.Valid([]byte(out.Stdout)):
				row.State, row.Reason = "update", "its snapshot could not be read"
			default:
				if werr := os.WriteFile(file, []byte(out.Stdout), 0o644); werr != nil {
					return res, &core.Error{Code: core.ExitFailure, What: "A snapshot could not be written.", Why: werr.Error(), Unmodified: true}
				}
				files = append(files, file)
				if drawer == nil || versionLess(newest, row.Version) {
					drawer, newest = row.b, row.Version
				}
			}
		}
		if row.State == "update" {
			if snap, ok := a.readWithViewer(ctx, &viewer, ro, row); ok {
				row.State, row.Reason = "read", "read with Bootstrap "+viewer.pkg.Release.Bootstrap.Version+" (the project has "+row.Version+"): gtt update gives it the hook and ideas"
				// The page says so too: the snapshot carries who read it and what to run to get the complete one.
				var doc map[string]any
				if json.Unmarshal(snap, &doc) == nil {
					doc["compat"] = map[string]string{"read_with": viewer.pkg.Release.Bootstrap.Version, "installed": row.Version, "command": "cd '" + row.Path + "' && gtt update"}
					if out, merr := json.Marshal(doc); merr == nil {
						snap = out
					}
				}
				if werr := os.WriteFile(file, snap, 0o644); werr != nil {
					return res, &core.Error{Code: core.ExitFailure, What: "A snapshot could not be written.", Why: werr.Error(), Unmodified: true}
				}
				files = append(files, file)
				res.Repos = append(res.Repos, row.SonarRepo)
				continue
			}
		}
		if row.State != "ok" {
			stale := map[string]any{"stale": map[string]string{"path": row.Path, "name": row.Name, "reason": row.Reason, "key": "", "command": "cd '" + row.Path + "' && gtt update"}}
			data, _ := json.Marshal(stale)
			if werr := os.WriteFile(file, data, 0o644); werr != nil {
				return res, &core.Error{Code: core.ExitFailure, What: "A snapshot could not be written.", Why: werr.Error(), Unmodified: true}
			}
			files = append(files, file)
		}
		res.Repos = append(res.Repos, row.SonarRepo)
	}
	if drawer == nil && viewer.ok {
		drawer = a.Factory.At(viewer.pkg.Root)
	}
	if drawer == nil {
		res.Message = "No repo of the portfolio has a Bootstrap with GTT Sonar and the newer one could not be fetched: run `gtt update` in them."
		return res, nil
	}
	page := filepath.Join(dir, "index.html")
	out, err := drawer.Execute(ctx, ports.OperationRequest{Operation: "sonar.render", Args: map[string]string{"files": strings.Join(files, ","), "out": page}})
	if err != nil {
		return res, err
	}
	if !out.OK() {
		return res, &core.Error{Code: core.ExitFailure, What: "The portfolio page could not be generated.", Why: strings.TrimSpace(out.Stdout + "\n" + out.Stderr), Unmodified: true}
	}
	res.Supported, res.Page = true, page
	return res, nil
}

// OpenPage opens the generated page in the default browser. It is a convenience: when it cannot, the path
// printed by the command is enough.
func (a *App) OpenPage(ctx context.Context, target string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name, args = "open", []string{target}
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", target}
	default:
		name, args = "xdg-open", []string{target}
	}
	path, err := a.Runner.LookPath(name)
	if err != nil {
		return err
	}
	res, err := a.Runner.Run(ctx, ports.Command{Path: path, Args: args})
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("%s exited with %d", name, res.ExitCode)
	}
	return nil
}

// scanSkip are folders that never hold a project worth listing.
var scanSkip = map[string]bool{"node_modules": true, "vendor": true, "dist": true, "build": true, "target": true, "__pycache__": true}

// FindProjects lists the GTT projects under root, to two levels, without descending into a project.
func (a *App) FindProjects(root string) []string {
	var found []string
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || scanSkip[e.Name()] {
				continue
			}
			child := filepath.Join(dir, e.Name())
			if a.Factory.Installed(child) {
				found = append(found, child)
			} else if depth < 2 {
				walk(child, depth+1)
			}
		}
	}
	if a.Factory.Installed(root) {
		return []string{root}
	}
	walk(root, 1)
	sort.Strings(found)
	return dropWorktrees(found)
}

// dropWorktrees keeps one entry per repository: a linked Git worktree (its .git is a file pointing at the main
// checkout) shows the same state as the repo, so it is listed only when its main checkout is not in the list.
func dropWorktrees(paths []string) []string {
	main := func(p string) string {
		data, err := os.ReadFile(filepath.Join(p, ".git"))
		if err != nil {
			return "" // a directory: the main checkout itself, or not Git at all
		}
		line := strings.TrimSpace(string(data))
		if !strings.HasPrefix(line, "gitdir:") {
			return ""
		}
		gitdir := filepath.ToSlash(strings.TrimSpace(strings.TrimPrefix(line, "gitdir:")))
		if i := strings.Index(gitdir, "/.git/worktrees/"); i >= 0 {
			return filepath.FromSlash(gitdir[:i])
		}
		return ""
	}
	listed := map[string]bool{}
	for _, p := range paths {
		listed[p] = true
	}
	seen := map[string]bool{}
	var out []string
	for _, p := range paths {
		if m := main(p); m != "" {
			if listed[m] || seen[m] {
				continue
			}
			seen[m] = true
		}
		out = append(out, p)
	}
	return out
}

// SonarFolder shows every GTT project found under a folder: no list to keep, just go there and run it.
func (a *App) SonarFolder(ctx context.Context, root string, ro ResolveOptions) (SonarResult, error) {
	paths := a.FindProjects(root)
	if len(paths) == 0 {
		return SonarResult{Schema: core.OutputSchema, Message: "No GTT project in " + root + " or the folders under it. Go to a folder with your projects, or to a project."}, nil
	}
	return a.sonarPortfolio(ctx, paths, ro)
}

type viewerPkg struct {
	tried bool
	ok    bool
	pkg   ports.Package
	// Release is the viewer's release descriptor.
	Release ports.Release
}

// readWithViewer snapshots a project whose own Bootstrap has no Sonar, using the Bootstrap this CLI ships.
// The snapshot only reads: nothing is written in the project. When the newer Bootstrap cannot be obtained
// (offline, no cache) the project stays a row that says to update.
func (a *App) readWithViewer(ctx context.Context, v *viewerPkg, ro ResolveOptions, row sonarRow) ([]byte, bool) {
	if !v.tried {
		v.tried = true
		if pkg, err := a.resolve(ctx, ro.request()); err == nil && supportsSonar(ctx, a.Factory.At(pkg.Root)) {
			v.pkg, v.ok, v.Release = pkg, true, pkg.Release
		}
	}
	if !v.ok {
		return nil, false
	}
	bash, err := a.Runner.LookPath("bash")
	if err != nil {
		return nil, false
	}
	script := filepath.Join(v.pkg.Root, ".gtt", "scripts", "gtt-sonar.sh")
	out, err := a.Runner.Run(ctx, ports.Command{Path: bash, Args: []string{script, "snapshot", "--json"}, Dir: row.Path})
	if err != nil || out.ExitCode != 0 || !json.Valid(out.Stdout) {
		return nil, false
	}
	return out.Stdout, true
}
