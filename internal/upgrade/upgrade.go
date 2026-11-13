// Package upgrade replaces the gtt binary with another release. It is an adapter: it knows how each install
// channel (npm, uv, pipx, go install, the curl and PowerShell installers) puts the binary on the machine and
// how to take the next one from the release assets, verified, with a way back. It knows nothing of projects.
package upgrade

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Repo is the only repository releases are taken from, unless the human declares a mirror.
const Repo = "GTT-Community/gtt-cli"

const maxBinary = 256 << 20 // a release binary is a few MB; anything near this is not one

// Upgrader implements ports.SelfUpgrader.
type Upgrader struct {
	Runner ports.Runner
	// Base is the mirror holding the release assets (GTT_DOWNLOAD_BASE); empty means GitHub releases.
	Base string
	// Exe is the binary being replaced; empty means the running one.
	Exe string
	// HTTP is the client; nil means a default one with timeouts.
	HTTP *http.Client
	// GOOS and GOARCH override the platform (tests); empty means this machine.
	GOOS, GOARCH string
	// GoBins are the directories `go install` writes to (GOBIN, GOPATH/bin); empty means read the environment.
	GoBins []string
}

var _ ports.SelfUpgrader = (*Upgrader)(nil)

func (u *Upgrader) goos() string   { return pick(u.GOOS, runtime.GOOS) }
func (u *Upgrader) goarch() string { return pick(u.GOARCH, runtime.GOARCH) }

func pick(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (u *Upgrader) client() *http.Client {
	if u.HTTP != nil {
		return u.HTTP
	}
	return &http.Client{Timeout: 2 * time.Minute}
}

func (u *Upgrader) exe() string {
	exe := u.Exe
	if exe == "" {
		exe, _ = os.Executable()
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return exe
}

// ---------------------------------------------------------------- source

// checkURL accepts https, and plain http only for a mirror on this machine (it carries nothing secret and
// the download is verified against checksums.txt anyway).
func checkURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return core.Errorf(core.ExitUsage, "not a valid download URL: %q", raw)
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" {
		switch u.Hostname() {
		case "127.0.0.1", "localhost", "::1":
			return nil
		}
	}
	return core.Errorf(core.ExitUsage, "refusing %q: downloads must use https (http only for a mirror on this machine)", raw)
}

func (u *Upgrader) base(target string) string {
	if u.Base != "" {
		return strings.TrimRight(u.Base, "/")
	}
	return "https://github.com/" + Repo + "/releases/download/v" + target
}

func (u *Upgrader) get(ctx context.Context, raw string, limit int64) ([]byte, error) {
	if err := checkURL(raw); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gtt-cli")
	resp, err := u.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %s", raw, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s is larger than %d bytes", raw, limit)
	}
	return data, nil
}

var tagRE = regexp.MustCompile(`/tag/v?(\d+\.\d+\.\d+)$`)

// Latest asks GitHub which release "latest" points to; with a mirror, the mirror's VERSION file says it.
func (u *Upgrader) Latest(ctx context.Context) (string, error) {
	unavailable := func(err error) error {
		return &core.Error{Code: core.ExitUnavailable, Unmodified: true, Err: err,
			What: "Could not find out the latest gtt release.", Why: err.Error(),
			Next: "Check the connection, or name the release yourself: gtt upgrade --to <version>"}
	}
	if u.Base != "" {
		data, err := u.get(ctx, strings.TrimRight(u.Base, "/")+"/VERSION", 64)
		if err != nil {
			return "", unavailable(fmt.Errorf("the mirror has no VERSION file (%v); name the release with --to", err))
		}
		v := strings.TrimPrefix(strings.TrimSpace(string(data)), "v")
		if _, ok := core.ParseSemver(v); !ok {
			return "", unavailable(fmt.Errorf("the mirror's VERSION is not MAJOR.MINOR.PATCH: %q", v))
		}
		return v, nil
	}
	raw := "https://github.com/" + Repo + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, raw, nil)
	if err != nil {
		return "", unavailable(err)
	}
	req.Header.Set("User-Agent", "gtt-cli")
	c := *u.client()
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := c.Do(req)
	if err != nil {
		return "", unavailable(err)
	}
	resp.Body.Close()
	m := tagRE.FindStringSubmatch(resp.Header.Get("Location"))
	if m == nil {
		return "", unavailable(fmt.Errorf("%s did not point to a release (%s)", raw, resp.Status))
	}
	return m[1], nil
}

// ---------------------------------------------------------------- channel

// Channel names how this binary got onto the machine, from where it lives.
func Channel(exe string, goBins []string) string {
	p := filepath.ToSlash(exe)
	switch {
	case strings.Contains(p, "/node_modules/@gtt-community/"):
		return "npm"
	case strings.Contains(p, "/uv/tools/"):
		return "uv"
	case strings.Contains(p, "/pipx/venvs/"):
		return "pipx"
	}
	dir := filepath.ToSlash(filepath.Dir(exe))
	for _, b := range goBins {
		if b != "" && filepath.ToSlash(filepath.Clean(b)) == dir {
			return "go"
		}
	}
	return "direct"
}

func (u *Upgrader) goBins() []string {
	if u.GoBins != nil {
		return u.GoBins
	}
	var out []string
	if b := os.Getenv("GOBIN"); b != "" {
		out = append(out, b)
	}
	for _, p := range filepath.SplitList(os.Getenv("GOPATH")) {
		out = append(out, filepath.Join(p, "bin"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, "go", "bin"))
	}
	return out
}

// installer is the command a human runs when the CLI cannot do it for them.
func (u *Upgrader) installer(target string) string {
	if u.goos() == "windows" {
		return fmt.Sprintf(`$env:GTT_VERSION="%s"; irm https://raw.githubusercontent.com/%s/main/install.ps1 | iex`, target, Repo)
	}
	return fmt.Sprintf("curl -fsSL https://raw.githubusercontent.com/%s/main/install.sh | GTT_VERSION=%s sh", Repo, target)
}

func writable(dir string) bool {
	f, err := os.CreateTemp(dir, ".gtt-write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

// Plan decides, from where the running binary lives, how it would be replaced.
func (u *Upgrader) Plan(target string) ports.UpgradePlan {
	exe := u.exe()
	p := ports.UpgradePlan{Channel: Channel(exe, u.goBins()), Exe: exe, Target: target}
	manual := func(reason, hint string) ports.UpgradePlan {
		p.Manual, p.Reason, p.Hint = true, reason, hint
		return p
	}
	switch p.Channel {
	case "npm":
		p.Command = []string{"npm", "install", "-g", "@gtt-community/gtt-cli@" + target}
	case "uv":
		p.Command = []string{"uv", "tool", "install", "--force", "gtt-cli==" + target}
	case "pipx":
		p.Command = []string{"pipx", "install", "--force", "gtt-cli==" + target}
	case "go":
		return manual("this gtt was built by `go install`, and the CLI does not run the Go toolchain",
			fmt.Sprintf("go install github.com/%s/cmd/gtt@v%s", Repo, target))
	default:
		if u.Base != "" {
			if err := checkURL(u.Base); err != nil {
				return manual(err.Error(), u.installer(target))
			}
		}
		if !writable(filepath.Dir(exe)) {
			return manual("no permission to write in "+filepath.Dir(exe), u.installer(target))
		}
		return p
	}
	if u.Runner == nil {
		return manual("no way to run "+p.Command[0], strings.Join(p.Command, " "))
	}
	if _, err := u.Runner.LookPath(p.Command[0]); err != nil {
		return manual(p.Command[0]+" is not on the PATH", strings.Join(p.Command, " "))
	}
	return p
}

// ---------------------------------------------------------------- apply

// Apply runs the package manager command, or replaces the binary itself.
func (u *Upgrader) Apply(ctx context.Context, plan ports.UpgradePlan) error {
	if plan.Manual {
		return core.Errorf(core.ExitUsage, "a manual plan is not applied by the CLI: %s", plan.Hint)
	}
	if len(plan.Command) > 0 {
		res, err := u.Runner.Run(ctx, ports.Command{Path: plan.Command[0], Args: plan.Command[1:]})
		if err != nil || res.ExitCode != 0 {
			why := strings.TrimSpace(string(res.Stderr))
			if err != nil {
				why = err.Error()
			}
			return &core.Error{Code: core.ExitFailure, Unmodified: true,
				What: "`" + strings.Join(plan.Command, " ") + "` failed.", Why: why,
				Next: "Run it yourself and read its output: " + strings.Join(plan.Command, " ")}
		}
		return nil
	}
	return u.replaceBinary(ctx, plan)
}

func (u *Upgrader) archiveName() string {
	ext := ".tar.gz"
	if u.goos() == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("gtt_%s_%s%s", u.goos(), u.goarch(), ext)
}

func (u *Upgrader) replaceBinary(ctx context.Context, plan ports.UpgradePlan) error {
	fail := func(what, why string) error {
		return &core.Error{Code: core.ExitFailure, Unmodified: true, What: what, Why: why,
			Next: "Nothing was changed. Install by hand: " + u.installer(plan.Target)}
	}
	name := u.archiveName()
	base := u.base(plan.Target)
	archive, err := u.get(ctx, base+"/"+name, maxBinary)
	if err != nil {
		return fail("Could not download "+name+".", err.Error())
	}
	sums, err := u.get(ctx, base+"/checksums.txt", 1<<20)
	if err != nil {
		return fail("Could not download checksums.txt.", err.Error())
	}
	want := ""
	for _, line := range strings.Split(string(sums), "\n") {
		if f := strings.Fields(line); len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			want = strings.ToLower(f[0])
		}
	}
	if want == "" {
		return fail(name+" is not listed in checksums.txt.", "The release does not carry this platform, or the mirror is incomplete.")
	}
	got := sha256.Sum256(archive)
	if hex.EncodeToString(got[:]) != want {
		return fail("The checksum of "+name+" does not match.", "expected "+want+", got "+hex.EncodeToString(got[:])+". The download was discarded.")
	}
	binName := "gtt"
	if u.goos() == "windows" {
		binName = "gtt.exe"
	}
	bin, err := extract(archive, name, binName)
	if err != nil {
		return fail("Could not read "+name+".", err.Error())
	}

	dir := filepath.Dir(plan.Exe)
	tmp, err := os.CreateTemp(dir, ".gtt-new-*")
	if err != nil {
		return fail("Cannot write next to "+plan.Exe+".", err.Error())
	}
	next := tmp.Name()
	defer os.Remove(next)
	_, werr := tmp.Write(bin)
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Chmod(next, 0o755)
	}
	if werr != nil {
		return fail("Cannot write the new binary.", werr.Error())
	}

	old := plan.Exe + ".old"
	os.Remove(old)
	if err := os.Rename(plan.Exe, old); err != nil {
		return fail("Cannot move the current binary aside.", err.Error())
	}
	if err := os.Rename(next, plan.Exe); err != nil {
		os.Rename(old, plan.Exe)
		return fail("Cannot put the new binary in place.", err.Error())
	}
	// The new binary must answer to `gtt version` with the release asked for; otherwise the old one comes back.
	res, rerr := u.Runner.Run(ctx, ports.Command{Path: plan.Exe, Args: []string{"version"}})
	if rerr != nil || res.ExitCode != 0 || !bytes.Contains(res.Stdout, []byte(plan.Target)) {
		os.Remove(plan.Exe)
		os.Rename(old, plan.Exe)
		return &core.Error{Code: core.ExitFailure,
			What: "The new binary did not answer as gtt " + plan.Target + "; the previous one was restored.",
			Why:  strings.TrimSpace(string(res.Stdout) + string(res.Stderr)),
			Next: "Install by hand: " + u.installer(plan.Target)}
	}
	return nil
}

// extract returns the named file from a .tar.gz or .zip, whatever folder it sits in.
func extract(archive []byte, archiveName, file string) ([]byte, error) {
	if strings.HasSuffix(archiveName, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if filepath.Base(f.Name) == file && !f.FileInfo().IsDir() {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(io.LimitReader(rc, maxBinary))
			}
		}
		return nil, fmt.Errorf("%s is not in the archive", file)
	}
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("%s is not in the archive", file)
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag == tar.TypeReg && filepath.Base(h.Name) == file {
			return io.ReadAll(io.LimitReader(tr, maxBinary))
		}
	}
}
