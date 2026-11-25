package app

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// The Sonar server: `gtt sonar` serves the page of the project on 127.0.0.1 and writes, for the human, only what
// the human could write by hand, through the same declared operations the CLI runs (the closed list is in
// doors.go). Everything a page may do arrives at /api/act; everything else is read.
//
// Who may write: the agent runs on the same machine, so a request is honoured only with the cookie that the
// human's browser got by redeeming a one-time code (valid 60 s, used once). The secret that signs the code and
// the cookie lives in ~/.gtt/sonar/ with mode 0600; it is never served and never logged.

const (
	sonarCookie    = "gtt_sonar"
	sonarCodeTTL   = 60 * time.Second
	sonarIdleLimit = 8 * time.Hour
	sonarMaxBody   = 64 << 10
	sonarSnapTTL   = 2 * time.Second
	// SonarServeNeeds is the Bootstrap release that brings sonar.serve.v1, as the page and the CLI say it.
	SonarServeNeeds = "1.24"
	serveCapability = "sonar.serve.v1"
)

// SonarServer is the HTTP side. It is built by SonarServe (the daemon) and by the tests, which inject the clock.
type SonarServer struct {
	App    *App
	Dir    string // ~/.gtt/sonar: the secret, server.json, the rendered pages
	Secret []byte
	Now    func() time.Time
	// Render draws the page from the snapshot files; nil uses the Bootstrap (sonar.render, or its script).
	Render func(ctx context.Context, served bool, files []string, out string) error
	// Stop is called when the human asks the server to close (the daemon closes its listener).
	Stop func()

	mu      sync.Mutex
	port    int
	repos   []string
	used    map[string]time.Time
	lastUse time.Time
	cache   *snapSet
	pages   map[bool]*pageCache
	viewer  viewerPkg
	actMu   sync.Mutex
}

type pageCache struct {
	etag string
	html []byte
	csp  string
}

// repoState is how one repo answered the last time it was read.
type repoState struct {
	path     string // as registered
	snapRoot string // the repo.path its own snapshot reports (the page sends it back as `repo`)
	name     string
	version  string
	serve    bool // its Bootstrap offers sonar.serve.v1
	note     string
	b        ports.Bootstrap
	entry    map[string]any // {"snapshot": ...} or {"stale": ...}
	raw      []byte         // what the page renderer reads
}

type snapSet struct {
	at      time.Time
	repos   []*repoState
	payload []byte
	etag    string
}

func (s *SonarServer) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// SetPort tells the server the port it listens on (Host and Origin are checked against it).
func (s *SonarServer) SetPort(p int) { s.mu.Lock(); s.port = p; s.mu.Unlock() }

// Register adds repos to what the server shows (idempotent) and drops the cache.
func (s *SonarServer) Register(paths []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range paths {
		p = filepath.Clean(p)
		found := false
		for _, r := range s.repos {
			found = found || r == p
		}
		if !found {
			s.repos = append(s.repos, p)
		}
	}
	s.cache = nil
}

// ---------------------------------------------------------------------------------------- secret and codes

// LoadSonarSecret reads ~/.gtt/sonar/secret, creating it (mode 0600, in a 0700 folder) the first time.
func LoadSonarSecret(dir string) ([]byte, error) {
	path := filepath.Join(dir, "secret")
	if data, err := os.ReadFile(path); err == nil {
		if b, derr := hex.DecodeString(strings.TrimSpace(string(data))); derr == nil && len(b) >= 16 {
			return b, nil
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(hex.EncodeToString(b)+"\n"), 0o600); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return nil, err
	}
	return b, nil
}

func mac(secret []byte, parts ...string) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(m.Sum(nil))
}

// MintSonarCode makes a one-time code: when it was made, a nonce, and the signature of both. It needs the
// secret, so only a process that can read ~/.gtt/sonar/ can make one.
func MintSonarCode(secret []byte, now time.Time) string {
	nonce := make([]byte, 12)
	_, _ = rand.Read(nonce)
	ts, n := fmt.Sprint(now.UnixMilli()), hex.EncodeToString(nonce)
	return ts + "." + n + "." + mac(secret, "code", ts, n)
}

func (s *SonarServer) sessionToken() string { return mac(s.Secret, "session") }
func (s *SonarServer) adminToken() string   { return mac(s.Secret, "admin") }

// redeem accepts a code once, within 60 seconds of its making.
func (s *SonarServer) redeem(code string) bool {
	parts := strings.Split(code, ".")
	if len(parts) != 3 {
		return false
	}
	want := mac(s.Secret, "code", parts[0], parts[1])
	if subtle.ConstantTimeCompare([]byte(want), []byte(parts[2])) != 1 {
		return false
	}
	var ms int64
	if _, err := fmt.Sscan(parts[0], &ms); err != nil {
		return false
	}
	now := s.now()
	age := now.Sub(time.UnixMilli(ms))
	if age > sonarCodeTTL || age < -5*time.Second {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.used == nil {
		s.used = map[string]time.Time{}
	}
	for k, t := range s.used { // forget what can no longer be redeemed anyway
		if now.Sub(t) > 2*sonarCodeTTL {
			delete(s.used, k)
		}
	}
	if _, again := s.used[parts[1]]; again {
		return false
	}
	s.used[parts[1]] = now
	return true
}

// ----------------------------------------------------------------------------------------------- handler

// Handler is the whole server.
func (s *SonarServer) Handler() http.Handler { return http.HandlerFunc(s.serve) }

func (s *SonarServer) touch() { s.mu.Lock(); s.lastUse = s.now(); s.mu.Unlock() }

// IdleSince is when the server was last used.
func (s *SonarServer) IdleSince() time.Time { s.mu.Lock(); defer s.mu.Unlock(); return s.lastUse }

func (s *SonarServer) hosts() []string {
	s.mu.Lock()
	p := s.port
	s.mu.Unlock()
	if p == 0 {
		return nil
	}
	return []string{fmt.Sprintf("127.0.0.1:%d", p), fmt.Sprintf("localhost:%d", p)}
}

func (s *SonarServer) hostOK(r *http.Request) bool {
	for _, h := range s.hosts() {
		if strings.EqualFold(r.Host, h) {
			return true
		}
	}
	return false
}

func (s *SonarServer) authed(r *http.Request) bool {
	c, err := r.Cookie(sonarCookie)
	return err == nil && subtle.ConstantTimeCompare([]byte(c.Value), []byte(s.sessionToken())) == 1
}

func deny(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": msg})
}

func (s *SonarServer) serve(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Frame-Options", "DENY")
	if !s.hostOK(r) {
		deny(w, http.StatusForbidden, "forbidden host")
		return
	}
	origin := r.Header.Get("Origin")
	if origin != "" && !strings.EqualFold(origin, "http://"+r.Host) {
		deny(w, http.StatusForbidden, "forbidden origin")
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		s.get(w, r)
	case http.MethodPost:
		if strings.HasPrefix(r.URL.Path, "/_admin/") {
			s.admin(w, r)
			return
		}
		// Every POST, whatever its path or body, needs the human's cookie, the header and the same origin.
		if !s.authed(r) {
			deny(w, http.StatusForbidden, "read only: this page was not opened with gtt sonar")
			return
		}
		if origin == "" || r.Header.Get("X-GTT-Sonar") == "" {
			deny(w, http.StatusForbidden, "forbidden request")
			return
		}
		if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
			deny(w, http.StatusForbidden, "forbidden content type")
			return
		}
		if r.URL.Path != "/api/act" {
			deny(w, http.StatusNotFound, "not found")
			return
		}
		s.touch()
		s.act(w, r)
	default:
		deny(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *SonarServer) get(w http.ResponseWriter, r *http.Request) {
	switch p := r.URL.Path; {
	case p == "/_ping":
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"gtt":"sonar"}`))
	case p == "/enter":
		s.touch()
		s.enter(w, r)
	case p == "/":
		s.touch()
		s.page(w, r)
	case p == "/api/snapshot":
		s.snapshotAPI(w, r)
	case strings.HasPrefix(p, "/fonts/"):
		s.font(w, r, strings.TrimPrefix(p, "/fonts/"))
	default:
		deny(w, http.StatusNotFound, "not found")
	}
}

// enter exchanges the one-time code for the cookie and sends the browser to the page.
func (s *SonarServer) enter(w http.ResponseWriter, r *http.Request) {
	if !s.redeem(r.URL.Query().Get("code")) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("This code is not valid or has expired. Run `gtt sonar` again.\n"))
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sonarCookie, Value: s.sessionToken(), Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
	view := r.URL.Query().Get("view")
	if !regexp.MustCompile(`^[a-z]{1,16}$`).MatchString(view) {
		view = "plan"
	}
	http.Redirect(w, r, "/#/view="+view, http.StatusFound)
}

// ------------------------------------------------------------------------------------------------ snapshots

// supportsServe: the Bootstrap offers sonar.serve.v1 and the operations it names.
// operationHasArg reports whether the Bootstrap declares the argument of the operation.
func operationHasArg(ctx context.Context, b ports.Bootstrap, op, arg string) bool {
	ops, err := b.Operations(ctx)
	if err != nil {
		return false
	}
	o, ok := ops[op]
	return ok && o.HasArg(arg)
}

func supportsServe(ctx context.Context, b ports.Bootstrap) bool {
	caps, err := b.Capabilities(ctx)
	if err != nil {
		return false
	}
	found := false
	for _, c := range caps {
		found = found || c.ID == serveCapability
	}
	if !found {
		return false
	}
	for _, op := range append([]string{"sonar.snapshot"}, doorOps[DoorSonar]...) {
		if op == OpConsultSet {
			continue // optional: a Bootstrap without it serves the page as before, and the page says the thread is registered from the terminal
		}
		if !supports(ctx, b, op) {
			return false
		}
	}
	return true
}

func (s *SonarServer) paths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.repos...)
}

// snapshots reads every repo with its own Bootstrap, at most once every 2 seconds, and shares the answer.
func (s *SonarServer) snapshots(ctx context.Context) *snapSet {
	s.mu.Lock()
	if c := s.cache; c != nil && s.now().Sub(c.at) < sonarSnapTTL {
		s.mu.Unlock()
		return c
	}
	s.mu.Unlock()
	paths := s.paths()
	set := &snapSet{at: s.now(), repos: make([]*repoState, len(paths))}
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			set.repos[i] = s.read(ctx, p, len(paths) == 1)
		}(i, p)
	}
	wg.Wait()
	var entries []json.RawMessage
	var sums bytes.Buffer
	for _, rs := range set.repos {
		raw, _ := json.Marshal(rs.entry)
		entries = append(entries, raw)
		// the tag follows the content, not the moment it was read
		sums.Write(stableForTag(rs.entry))
	}
	sum := sha256.Sum256(sums.Bytes())
	set.etag = `"` + hex.EncodeToString(sum[:8]) + `"`
	set.payload, _ = json.Marshal(map[string]any{"schema": "sonar.page.v1", "generated_at": set.at.UTC().Format("2006-01-02T15:04:05Z"),
		"repos": entries, "mode": "served"})
	s.mu.Lock()
	// An act (or Register) that happened while reading wins: the cache stays empty, the next read is fresh.
	if s.cache == nil && len(s.repos) == len(paths) {
		s.cache = set
	}
	s.mu.Unlock()
	return set
}

// stableForTag is the entry without the moments that change on every read.
func stableForTag(entry map[string]any) []byte {
	cp := map[string]any{}
	for k, v := range entry {
		if m, ok := v.(map[string]any); ok {
			c := map[string]any{}
			for kk, vv := range m {
				if kk != "generated_at" {
					c[kk] = vv
				}
			}
			cp[k] = c
		} else {
			cp[k] = v
		}
	}
	b, _ := json.Marshal(cp)
	return b
}

func staleEntry(path, name, reason string) map[string]any {
	return map[string]any{"stale": map[string]string{"path": path, "name": name, "reason": reason, "key": "", "command": "cd '" + path + "' && gtt update"}}
}

// read asks one repo's own Bootstrap for its snapshot. A repo that cannot serve is shown all the same, read
// only, with the release it needs; one that cannot be read at all is a row that says so.
//
// withDocs asks for the governed documents too (the Design view reads them): only when the page holds one
// project, as the page itself says; a portfolio would carry up to 900 KB of them per project on every read.
func (s *SonarServer) read(ctx context.Context, path string, withDocs bool) *repoState {
	a := s.App
	rs := &repoState{path: path, name: filepath.Base(path)}
	row := a.sonarProbe(ctx, path)
	rs.version = row.Version
	finish := func(entry map[string]any) *repoState {
		rs.entry = entry
		rs.raw, _ = json.Marshal(entry)
		return rs
	}
	if row.State != "ok" && row.State != "update" {
		return finish(staleEntry(path, rs.name, row.Reason))
	}
	var data []byte
	if row.State == "ok" {
		rs.b = row.b
		req := ports.OperationRequest{Operation: "sonar.snapshot"}
		if withDocs && operationHasArg(ctx, row.b, "sonar.snapshot", "docs") { // an older Bootstrap does not declare it: it goes without
			req.Args = map[string]string{"docs": "true"}
		}
		out, err := row.b.Execute(ctx, req)
		if err == nil && out.OK() && json.Valid([]byte(out.Stdout)) {
			data = []byte(out.Stdout)
		} else {
			return finish(staleEntry(path, rs.name, "its snapshot could not be read"))
		}
		rs.serve = supportsServe(ctx, row.b)
	} else {
		var ok bool
		data, ok = a.readWithViewer(ctx, &s.viewer, ResolveOptions{}, row, withDocs)
		if !ok {
			return finish(staleEntry(path, rs.name, row.Reason))
		}
	}
	var doc map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return finish(staleEntry(path, rs.name, "its snapshot is unreadable"))
	}
	if repo, ok := doc["repo"].(map[string]any); ok {
		rs.snapRoot, _ = repo["path"].(string)
	}
	if !rs.serve {
		rs.note = fmt.Sprintf("%s: read only. Its Bootstrap %s has no %s (needs %s or newer): run `gtt update` there.", rs.name, orUnknown(row.Version), serveCapability, SonarServeNeeds)
		if doc["degraded"] == nil { // the page edits nothing in a degraded repo
			doc["degraded"] = map[string]any{"project": row.Version, "tool": "", "missing": []string{"serve"}, "needs": SonarServeNeeds, "command": "cd '" + path + "' && gtt update"}
		}
		if doc["compat"] == nil && row.State == "update" && s.viewer.ok {
			doc["compat"] = map[string]string{"read_with": s.viewer.pkg.Release.Bootstrap.Version, "installed": row.Version, "command": "cd '" + path + "' && gtt update"}
		}
	}
	return finish(map[string]any{"snapshot": doc})
}

func orUnknown(v string) string {
	if v == "" {
		return "(unknown)"
	}
	return v
}

func (s *SonarServer) invalidate() { s.mu.Lock(); s.cache = nil; s.mu.Unlock() }

func (s *SonarServer) snapshotAPI(w http.ResponseWriter, r *http.Request) {
	set := s.snapshots(r.Context())
	w.Header().Set("ETag", set.etag)
	if match := r.Header.Get("If-None-Match"); match != "" && match == set.etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(set.payload)
}

// ------------------------------------------------------------------------------------------------- the page

var cspMeta = regexp.MustCompile(`<meta[^>]+http-equiv="Content-Security-Policy"[^>]+content="([^"]*)"`)

func (s *SonarServer) serveDir() string { return filepath.Join(s.Dir, "serve") }

// page serves the page: with the human's cookie it is the served one (forms, live); without it, the static
// one (read only, commands to copy).
func (s *SonarServer) page(w http.ResponseWriter, r *http.Request) {
	served := s.authed(r)
	set := s.snapshots(r.Context())
	pg, err := s.drawPage(r.Context(), served, set)
	if err != nil {
		var ce *core.Error
		msg := err.Error()
		if errors.As(err, &ce) {
			msg = ce.What
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("The page could not be drawn: " + msg + "\n"))
		return
	}
	if pg.csp != "" {
		w.Header().Set("Content-Security-Policy", pg.csp)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(pg.html)
}

func (s *SonarServer) drawPage(ctx context.Context, served bool, set *snapSet) (*pageCache, error) {
	s.mu.Lock()
	if c := s.pages[served]; c != nil && c.etag == set.etag {
		s.mu.Unlock()
		return c, nil
	}
	s.mu.Unlock()
	dir := filepath.Join(s.serveDir(), map[bool]string{true: "served", false: "static"}[served])
	snapDir := filepath.Join(s.serveDir(), "snapshots")
	if err := os.MkdirAll(snapDir, 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	var files []string
	for i, rs := range set.repos {
		f := filepath.Join(snapDir, fmt.Sprintf("repo-%02d.json", i+1))
		if err := os.WriteFile(f, rs.raw, 0o600); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	out := filepath.Join(dir, "index.html")
	render := s.Render
	if render == nil {
		render = s.bootstrapRender(set)
	}
	if err := render(ctx, served, files, out); err != nil {
		return nil, err
	}
	html, err := os.ReadFile(out)
	if err != nil {
		return nil, err
	}
	pg := &pageCache{etag: set.etag, html: html}
	if m := cspMeta.FindSubmatch(html); m != nil {
		pg.csp = string(m[1])
	}
	s.mu.Lock()
	if s.pages == nil {
		s.pages = map[bool]*pageCache{}
	}
	s.pages[served] = pg
	s.mu.Unlock()
	return pg, nil
}

// bootstrapRender draws with the newest Bootstrap that can: for the served page one that offers
// sonar.serve.v1, for the static page any with sonar.render.
func (s *SonarServer) bootstrapRender(set *snapSet) func(context.Context, bool, []string, string) error {
	return func(ctx context.Context, served bool, files []string, out string) error {
		var drawer ports.Bootstrap
		newest := ""
		for _, rs := range set.repos {
			if rs.b == nil || (served && !rs.serve) || !supports(ctx, rs.b, "sonar.render") {
				continue
			}
			if drawer == nil || versionLess(newest, rs.version) {
				drawer, newest = rs.b, rs.version
			}
		}
		if drawer == nil && !served {
			if s.viewer.ok {
				drawer = s.App.Factory.At(s.viewer.pkg.Root)
			}
		}
		if drawer == nil {
			return &core.Error{Code: core.ExitIncompatible, What: "no Bootstrap here can draw the page (needs " + SonarServeNeeds + " or newer): run `gtt update`"}
		}
		args := map[string]string{"files": strings.Join(files, ","), "out": out}
		ops, _ := drawer.Operations(ctx)
		if op, ok := ops["sonar.render"]; ok && (!served || op.HasArg("served")) {
			if served {
				args["served"] = "true"
			}
			res, err := drawer.Execute(ctx, ports.OperationRequest{Operation: "sonar.render", Args: args})
			if err != nil {
				return err
			}
			if !res.OK() {
				return errors.New(strings.TrimSpace(res.Stdout + " " + res.Stderr))
			}
			return nil
		}
		// sonar.render does not declare `served` yet: the Bootstrap's own script, as the viewer does.
		bash, err := s.App.Runner.LookPath("bash")
		if err != nil {
			return err
		}
		argv := []string{filepath.Join(drawer.Root(), ".gtt", "scripts", "gtt-sonar.sh"), "render", "--served", "--out", out}
		res, err := s.App.Runner.Run(ctx, ports.Command{Path: bash, Args: append(argv, files...), Dir: drawer.Root()})
		if err != nil {
			return err
		}
		if res.ExitCode != 0 {
			return errors.New(strings.TrimSpace(string(res.Stdout) + " " + string(res.Stderr)))
		}
		return nil
	}
}

// font serves the fonts the rendered page carries next to it. Only a plain file name inside fonts/.
func (s *SonarServer) font(w http.ResponseWriter, r *http.Request, name string) {
	if name == "" || name != filepath.Base(name) || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".woff2") {
		deny(w, http.StatusNotFound, "not found")
		return
	}
	for _, d := range []string{"served", "static"} {
		if data, err := os.ReadFile(filepath.Join(s.serveDir(), d, "fonts", name)); err == nil {
			w.Header().Set("Content-Type", "font/woff2")
			w.Header().Set("Cache-Control", "max-age=3600")
			_, _ = w.Write(data)
			return
		}
	}
	deny(w, http.StatusNotFound, "not found")
}

func constEq(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }
