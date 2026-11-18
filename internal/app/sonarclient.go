package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// The two halves around the server: the daemon (`gtt sonar serve-daemon`, started detached by `gtt sonar`) and the
// client (`gtt sonar`, `gtt sonar stop`), who talk through ~/.gtt/sonar/server.json ({pid, port}, no secret) and
// the admin routes, which need the secret.

// SonarServerInfo is ~/.gtt/sonar/server.json: where the server listens. It holds no secret.
type SonarServerInfo struct {
	PID  int `json:"pid"`
	Port int `json:"port"`
}

func (a *App) sonarInfoPath() string { return filepath.Join(a.sonarPortfolioDir(), "server.json") }

func (a *App) readServerInfo() (SonarServerInfo, bool) {
	var info SonarServerInfo
	data, err := os.ReadFile(a.sonarInfoPath())
	if err != nil || json.Unmarshal(data, &info) != nil || info.Port == 0 {
		return info, false
	}
	return info, true
}

var sonarHTTP = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

// serverAlive asks the port whether a Sonar server answers there.
func serverAlive(port int) bool {
	c := &http.Client{Timeout: time.Second}
	resp, err := c.Get(fmt.Sprintf("http://127.0.0.1:%d/_ping", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	return resp.StatusCode == 200 && strings.Contains(string(body), `"sonar"`)
}

// keepWatchers makes sure each open repo has its watcher: the Bootstrap's own `gtt-sonar.sh watch`, which
// does nothing when one already runs. It is quiet and never fails the server.
func (a *App) keepWatchers(paths []string) {
	if a.Spawner == nil || a.Runner == nil {
		return
	}
	bash, err := a.Runner.LookPath("bash")
	if err != nil {
		return
	}
	for _, p := range paths {
		script := filepath.Join(p, ".gtt", "scripts", "gtt-sonar.sh")
		if _, err := os.Stat(script); err != nil {
			continue
		}
		_, _ = a.Spawner.Start(ports.Command{Path: bash, Args: []string{script, "watch"}, Dir: p})
	}
}

// SonarServe is the daemon: it listens on 127.0.0.1, writes {pid, port}, keeps the watchers, and closes
// after 8 hours without use or when told to. It returns when the server is closed.
func (a *App) SonarServe(ctx context.Context) error {
	dir := a.sonarPortfolioDir()
	secret, err := LoadSonarSecret(dir)
	if err != nil {
		return &core.Error{Code: core.ExitFailure, What: "The Sonar secret could not be read or created in " + dir, Why: err.Error()}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	srv := &SonarServer{App: a, Dir: dir, Secret: secret}
	srv.SetPort(ln.Addr().(*net.TCPAddr).Port)
	srv.touch()
	hs := &http.Server{Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 2 * time.Minute, MaxHeaderBytes: 16 << 10}
	done := make(chan struct{})
	var once = make(chan struct{}, 1)
	stop := func() {
		select {
		case once <- struct{}{}:
			sctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = hs.Shutdown(sctx)
			close(done)
		default:
		}
	}
	srv.Stop = stop
	info, _ := json.Marshal(SonarServerInfo{PID: os.Getpid(), Port: srv.port})
	if err := os.WriteFile(a.sonarInfoPath(), append(info, '\n'), 0o600); err != nil {
		return err
	}
	defer func() {
		if cur, ok := a.readServerInfo(); ok && cur.PID == os.Getpid() {
			_ = os.Remove(a.sonarInfoPath())
		}
	}()
	go func() {
		tick := time.NewTicker(time.Minute)
		defer tick.Stop()
		n := 0
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				stop()
				return
			case <-tick.C:
				if srv.idleExpired() {
					stop()
					return
				}
				if n++; n%15 == 0 { // the watcher ends on its own after a while: bring it back while the server lives
					a.keepWatchers(srv.paths())
				}
			}
		}
	}()
	if err := hs.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	<-done
	return nil
}

// SonarOpenOptions are the flags of `gtt sonar`.
type SonarOpenOptions struct {
	All    bool
	Log    bool
	NoOpen bool
}

// sonarServePaths are the repos this invocation shows: the project it is in, the portfolio (--all), or the
// projects found under the folder it is in.
func (a *App) sonarServePaths(ctx context.Context, all bool) ([]string, error) {
	if all {
		c, err := readSonarConfig()
		if err != nil {
			return nil, err
		}
		var paths []string
		for _, r := range c.Repos {
			paths = append(paths, r.Path)
		}
		if len(paths) > 0 {
			return paths, nil
		}
		return a.FindProjects(a.Cwd), nil
	}
	if det := a.detect(ctx); a.Factory.Installed(det.Root) {
		return []string{det.Root}, nil
	}
	return a.FindProjects(a.Cwd), nil
}

// SonarOpen starts the server (or finds the one that runs), tells it which repos to show, and opens the
// browser with a one-time code that the page exchanges for the human's cookie.
func (a *App) SonarOpen(ctx context.Context, o SonarOpenOptions) (SonarResult, error) {
	res := SonarResult{Schema: core.OutputSchema}
	paths, err := a.sonarServePaths(ctx, o.All)
	if err != nil {
		return res, err
	}
	if len(paths) == 0 {
		res.Message = "No GTT project in " + a.Cwd + " or the folders under it. Go to a folder with your projects, or to a project."
		return res, nil
	}
	dir := a.sonarPortfolioDir()
	secret, err := LoadSonarSecret(dir)
	if err != nil {
		return res, &core.Error{Code: core.ExitFailure, What: "The Sonar secret could not be read or created in " + dir, Why: err.Error(), Unmodified: true}
	}
	info, ok := a.readServerInfo()
	if !ok || !serverAlive(info.Port) {
		if info, err = a.startServer(); err != nil {
			return res, err
		}
	}
	notes, err := adminCall(info.Port, secret, "/_admin/open", map[string]any{"repos": paths})
	if err != nil {
		return res, &core.Error{Code: core.ExitFailure, What: "The Sonar server did not accept the repos.", Why: err.Error(), Unmodified: true,
			Next: "Close it with `gtt sonar stop` and run `gtt sonar` again."}
	}
	view := "plan"
	switch {
	case len(paths) > 1:
		view = "portfolio"
	case o.Log:
		view = "activity"
	}
	target := (&url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", info.Port), Path: "/enter",
		RawQuery: url.Values{"code": {MintSonarCode(secret, time.Now())}, "view": {view}}.Encode()}).String()
	res.Supported, res.Served, res.Port, res.Notes = true, true, info.Port, notes
	res.Message = fmt.Sprintf("Sonar is served at http://127.0.0.1:%d (it closes after 8 hours without use; close it now with: gtt sonar stop).", info.Port)
	if o.NoOpen {
		res.URL = target // valid 60 seconds, once
		return res, nil
	}
	if oerr := a.OpenPage(ctx, target); oerr != nil {
		res.Message += " The browser could not be opened (" + oerr.Error() + "): run `gtt sonar --no-open` and open the address it gives."
	}
	return res, nil
}

// startServer re-executes this binary in the background as the daemon and waits until it answers.
func (a *App) startServer() (SonarServerInfo, error) {
	var info SonarServerInfo
	if a.Spawner == nil {
		return info, &core.Error{Code: core.ExitUnavailable, Unmodified: true, What: "This gtt cannot start the Sonar server here.", Next: "Use: gtt sonar --static"}
	}
	exe, err := os.Executable()
	if err != nil {
		return info, err
	}
	_ = os.Remove(a.sonarInfoPath())
	if _, err := a.Spawner.Start(ports.Command{Path: exe, Args: []string{"sonar", "serve-daemon"}, Env: []string{"GTT_HOME=" + a.Home}}); err != nil {
		return info, &core.Error{Code: core.ExitFailure, Unmodified: true, What: "The Sonar server could not be started.", Why: err.Error(), Next: "Use: gtt sonar --static"}
	}
	for i := 0; i < 200; i++ {
		if cur, ok := a.readServerInfo(); ok && serverAlive(cur.Port) {
			return cur, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return info, &core.Error{Code: core.ExitUnavailable, Unmodified: true, What: "The Sonar server did not answer in time.", Next: "Try again, or use: gtt sonar --static"}
}

func adminCall(port int, secret []byte, path string, body any) ([]string, error) {
	data, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d%s", port, path), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mac(secret, "admin"))
	resp, err := sonarHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		OK    bool     `json:"ok"`
		Error string   `json:"error"`
		Notes []string `json:"notes"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out)
	if resp.StatusCode != 200 || !out.OK {
		return nil, fmt.Errorf("HTTP %d %s", resp.StatusCode, out.Error)
	}
	return out.Notes, nil
}

// SonarStop closes the server that runs, if any.
func (a *App) SonarStop(_ context.Context) (SonarResult, error) {
	res := SonarResult{Schema: core.OutputSchema}
	info, ok := a.readServerInfo()
	if !ok || !serverAlive(info.Port) {
		_ = os.Remove(a.sonarInfoPath())
		res.Message = "No Sonar server is running."
		return res, nil
	}
	secret, err := LoadSonarSecret(a.sonarPortfolioDir())
	if err != nil {
		return res, err
	}
	if _, err := adminCall(info.Port, secret, "/_admin/stop", map[string]any{}); err != nil {
		return res, &core.Error{Code: core.ExitFailure, What: "The Sonar server did not close.", Why: err.Error()}
	}
	for i := 0; i < 60 && serverAlive(info.Port); i++ {
		time.Sleep(50 * time.Millisecond)
	}
	res.Message = "The Sonar server is closed."
	return res, nil
}

// idleExpired: 8 hours without use. Polling the snapshot is not use: a forgotten tab must not keep the server alive.
func (s *SonarServer) idleExpired() bool { return s.now().Sub(s.IdleSince()) > sonarIdleLimit }
