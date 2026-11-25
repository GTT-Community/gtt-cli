package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// The Sonar server at the HTTP level (criteria 10 to 18 of the order): a fake Bootstrap stands for the real
// one, so a test can tell which operation an act reached and with which arguments.

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

type served struct {
	t        *testing.T
	b        *fakeBootstrap
	srv      *SonarServer
	ts       *httptest.Server
	clk      *clock
	root     string
	cookie   string
	snap     string // what sonar.snapshot answers; a test may change it
	rendered []bool
}

// js is a JSON string literal: a Windows path has backslashes, so it must be escaped, never pasted raw into JSON.
func js(s string) string { b, _ := json.Marshal(s); return string(b) }

func opsNamed(names ...string) map[string]ports.Operation {
	m := map[string]ports.Operation{}
	for _, n := range names {
		m[n] = ports.Operation{Version: 1}
	}
	return m
}

func serveFixture(t *testing.T, withServe bool) *served {
	t.Helper()
	b := &fakeBootstrap{compatible: true, version: "1.24.0", results: map[string]ports.OperationResult{}}
	b.ops = opsNamed("sonar.snapshot", "sonar.render", "findings.list", "backlog.set", "inbox.add", "pending.answer", "sonar.prefs")
	b.caps = []ports.Capability{{ID: "sonar.v1", Version: 1}}
	if withServe {
		b.caps = append(b.caps, ports.Capability{ID: "sonar.serve.v1", Version: 1})
	} else {
		b.version = "1.23.0"
	}
	a, _ := newApp(t, b)
	root := b.root
	s := &served{t: t, b: b, root: root, clk: &clock{t: time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)}}
	s.snap = `{"schema":"sonar.v1","generated_at":"2026-11-18T10:00:00Z","repo":{"path":` + js(root) + `,"name":"demo"},"readiness":{},"v":1}`
	b.results["sonar.snapshot"] = ports.OperationResult{Stdout: s.snap}
	dir := filepath.Join(t.TempDir(), "sonar")
	secret, err := LoadSonarSecret(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.srv = &SonarServer{App: a, Dir: dir, Secret: secret, Now: s.clk.now}
	s.srv.Render = func(_ context.Context, servedPage bool, files []string, out string) error {
		s.rendered = append(s.rendered, servedPage)
		page := `<html><head><meta http-equiv="Content-Security-Policy" content="default-src 'none'; connect-src 'self'"></head><body>` +
			map[bool]string{true: "SERVED", false: "STATIC"}[servedPage] + `</body></html>`
		return os.WriteFile(out, []byte(page), 0o644)
	}
	s.srv.Register([]string{root})
	s.ts = httptest.NewServer(s.srv.Handler())
	t.Cleanup(s.ts.Close)
	s.srv.SetPort(s.ts.Listener.Addr().(*net.TCPAddr).Port)
	return s
}

var noRedirect = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

// do sends a request; hdr overrides or adds headers ("-Name" removes one). The Origin and the JSON type are the
// page's own unless the test says otherwise.
func (s *served) do(method, path, body string, hdr map[string]string) (*http.Response, string) {
	s.t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, s.ts.URL+path, rd)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GTT-Sonar", "1")
		req.Header.Set("Origin", s.ts.URL)
	}
	if s.cookie != "" {
		req.Header.Set("Cookie", sonarCookie+"="+s.cookie)
	}
	for k, v := range hdr {
		if strings.HasPrefix(k, "-") {
			req.Header.Del(k[1:])
		} else if k == "Host" {
			req.Host = v
		} else {
			req.Header.Set(k, v)
		}
	}
	resp, err := noRedirect.Do(req)
	if err != nil {
		s.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, string(data)
}

// login redeems a fresh code, as the browser does, and keeps the cookie.
func (s *served) login() {
	s.t.Helper()
	resp, _ := s.do("GET", "/enter?code="+MintSonarCode(s.srv.Secret, s.clk.now()), "", nil)
	if resp.StatusCode != 302 {
		s.t.Fatalf("the code was not accepted: %d", resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		if c.Name == sonarCookie {
			s.cookie = c.Value
		}
	}
	if s.cookie == "" {
		s.t.Fatal("no cookie")
	}
}

func (s *served) act(op string, args map[string]any) (*http.Response, map[string]any) {
	s.t.Helper()
	body, _ := json.Marshal(map[string]any{"repo": s.root, "op": op, "args": args})
	resp, text := s.do("POST", "/api/act", string(body), nil)
	var out map[string]any
	_ = json.Unmarshal([]byte(text), &out)
	return resp, out
}

func (s *served) requestsOf(op string) []ports.OperationRequest {
	var out []ports.OperationRequest
	for _, r := range s.b.requests {
		if r.Operation == op {
			out = append(out, r)
		}
	}
	return out
}

func TestCodeIsRedeemedOnceForAHttpOnlyStrictCookie(t *testing.T) {
	s := serveFixture(t, true)
	code := MintSonarCode(s.srv.Secret, s.clk.now())
	resp, _ := s.do("GET", "/enter?code="+code+"&view=inbox", "", nil)
	if resp.StatusCode != 302 || resp.Header.Get("Location") != "/#/view=inbox" {
		t.Fatalf("the code opens the page: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	var ck *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == sonarCookie {
			ck = c
		}
	}
	if ck == nil || !ck.HttpOnly || ck.SameSite != http.SameSiteStrictMode || ck.Value == "" {
		t.Fatalf("the cookie is HttpOnly and SameSite=Strict: %+v", ck)
	}
	if strings.Contains(ck.Value, string(s.srv.Secret)) {
		t.Error("the cookie is derived from the secret, never the secret")
	}
	if resp2, _ := s.do("GET", "/enter?code="+code, "", nil); resp2.StatusCode != 403 || len(resp2.Cookies()) != 0 {
		t.Errorf("a code cannot be used twice: %d", resp2.StatusCode)
	}
	if resp3, _ := s.do("GET", "/enter?code=garbage", "", nil); resp3.StatusCode != 403 {
		t.Errorf("a made-up code is refused: %d", resp3.StatusCode)
	}
	other := MintSonarCode([]byte("another secret, not the server's"), s.clk.now())
	if resp4, _ := s.do("GET", "/enter?code="+other, "", nil); resp4.StatusCode != 403 {
		t.Errorf("a code signed with another secret is refused: %d", resp4.StatusCode)
	}
}

func TestCodeExpiresAfterSixtySeconds(t *testing.T) {
	s := serveFixture(t, true)
	late := MintSonarCode(s.srv.Secret, s.clk.now())
	fresh := MintSonarCode(s.srv.Secret, s.clk.now())
	s.clk.add(59 * time.Second)
	if resp, _ := s.do("GET", "/enter?code="+fresh, "", nil); resp.StatusCode != 302 {
		t.Errorf("at 59 seconds it still opens: %d", resp.StatusCode)
	}
	s.clk.add(2 * time.Second)
	if resp, _ := s.do("GET", "/enter?code="+late, "", nil); resp.StatusCode != 403 {
		t.Errorf("at 61 seconds it is gone: %d", resp.StatusCode)
	}
}

func TestEveryPostWithoutTheCookieIs403(t *testing.T) {
	s := serveFixture(t, true)
	for _, path := range []string{"/api/act", "/api/anything", "/", "/enter", "/api/snapshot"} {
		for _, body := range []string{`{"repo":` + js(s.root) + `,"op":"backlog.set","args":{}}`, "", "not json"} {
			if resp, _ := s.do("POST", path, body, nil); resp.StatusCode != 403 {
				t.Errorf("POST %s without the cookie: %d, want 403", path, resp.StatusCode)
			}
		}
	}
	if resp, _ := s.do("POST", "/api/act", "{}", map[string]string{"Cookie": sonarCookie + "=forged"}); resp.StatusCode != 403 {
		t.Errorf("a forged cookie: %d", resp.StatusCode)
	}
	for _, r := range s.b.requests {
		if r.Operation != "sonar.snapshot" {
			t.Errorf("nothing but reading reached the Bootstrap: %v", r)
		}
	}
}

func TestPostNeedsHeaderOriginAndJSON(t *testing.T) {
	s := serveFixture(t, true)
	s.login()
	body := `{"repo":` + js(s.root) + `,"op":"sonar.prefs","args":{"set":{"view":"plan"}}}`
	for name, hdr := range map[string]map[string]string{
		"no X-GTT-Sonar":     {"-X-GTT-Sonar": ""},
		"another Origin":     {"Origin": "http://evil.example"},
		"no Origin":          {"-Origin": ""},
		"form content type":  {"Content-Type": "application/x-www-form-urlencoded"},
		"text content type":  {"Content-Type": "text/plain"},
		"another Host":       {"Host": "evil.example"},
		"Origin of the port": {"Origin": "http://127.0.0.1:1"},
	} {
		if resp, _ := s.do("POST", "/api/act", body, hdr); resp.StatusCode != 403 {
			t.Errorf("%s: %d, want 403", name, resp.StatusCode)
		}
	}
	if len(s.requestsOf("sonar.prefs")) != 0 {
		t.Error("none of those reached the Bootstrap")
	}
	s.b.results["sonar.prefs:apply"] = ok("saved\n")
	if resp, _ := s.do("POST", "/api/act", body, nil); resp.StatusCode != 200 {
		t.Errorf("the page's own request is accepted: %d", resp.StatusCode)
	}
}

func TestHostAndOriginAreCheckedOnReadsToo(t *testing.T) {
	s := serveFixture(t, true)
	if resp, _ := s.do("GET", "/api/snapshot", "", map[string]string{"Host": "evil.example"}); resp.StatusCode != 403 {
		t.Errorf("DNS rebinding (another Host): %d", resp.StatusCode)
	}
	if resp, _ := s.do("GET", "/api/snapshot", "", map[string]string{"Origin": "http://evil.example"}); resp.StatusCode != 403 {
		t.Errorf("a read from another origin: %d", resp.StatusCode)
	}
	if resp, _ := s.do("GET", "/api/snapshot", "", map[string]string{"Host": "localhost:" + strings.Split(s.ts.URL, ":")[2]}); resp.StatusCode != 200 {
		t.Errorf("localhost is the same machine: %d", resp.StatusCode)
	}
	if resp, _ := s.do("GET", "/api/snapshot", "", map[string]string{"Origin": s.ts.URL}); resp.StatusCode != 200 {
		t.Errorf("the page's own origin reads: %d", resp.StatusCode)
	}
}

func TestOnlyTheClosedListOfOperationsIsServed(t *testing.T) {
	s := serveFixture(t, true)
	s.login()
	for _, op := range []string{"approve", "promote", "run.execute", "undo", "freeze", "adr.adopt", "inbox.promote", "validation.run", "idea.drop", "", "BACKLOG.SET", "backlog.set "} {
		resp, out := s.act(op, map[string]any{"ref": "x"})
		if resp.StatusCode != 403 || out["ok"] == true {
			t.Errorf("%q: %d, want 403", op, resp.StatusCode)
		}
	}
	for _, r := range s.b.requests {
		if r.Operation != "sonar.snapshot" {
			t.Errorf("nothing was run: %v", r)
		}
	}
}

func TestBacklogSetReachesTheOperationAsAnActOfTheHuman(t *testing.T) {
	s := serveFixture(t, true)
	s.login()
	s.b.results["backlog.set:apply"] = ports.OperationResult{Stdout: `{"ok":true,"ref":"STORY-003","changed":["Estimate"],"left":[{"code":"DEF-STORY-DONE-WHEN"}],"sha":"abc"}` + "\n"}
	resp, out := s.act("backlog.set", map[string]any{"ref": "STORY-003", "sha": "deadbeef", "fields": map[string]any{"Estimate": "2h", "Done when": "it syncs"}})
	if resp.StatusCode != 200 || out["ok"] != true || out["left"] == nil || out["sha"] != "abc" {
		t.Fatalf("the answer carries what is left: %d %v", resp.StatusCode, out)
	}
	reqs := s.requestsOf("backlog.set")
	if len(reqs) != 1 || !reqs[0].Apply || !reqs[0].ConfirmedByHuman {
		t.Fatalf("one applied act of the human: %+v", reqs)
	}
	a := reqs[0].Args
	var fields map[string]string
	if err := json.Unmarshal([]byte(a["fields"]), &fields); err != nil || fields["Estimate"] != "2h" || fields["Done when"] != "it syncs" {
		t.Errorf("fields travel as a JSON string: %q", a["fields"])
	}
	if a["ref"] != "STORY-003" || a["sha"] != "deadbeef" || a["via"] != "sonar" || len(a) != 4 {
		t.Errorf("ref, sha, fields and via=sonar, nothing else: %v", a)
	}
	if resp, _ := s.act("backlog.set", map[string]any{"ref": "STORY-003", "sha": "x"}); resp.StatusCode != 400 {
		t.Errorf("without fields it is a bad request: %d", resp.StatusCode)
	}
}

func TestStaleBacklogIs409AndWritesNothingMore(t *testing.T) {
	s := serveFixture(t, true)
	s.login()
	s.b.results["backlog.set:apply"] = ports.OperationResult{ExitCode: 3, Stderr: "gtt-backlog: STALE: STORY-003 changed since the page showed it\n"}
	resp, out := s.act("backlog.set", map[string]any{"ref": "STORY-003", "sha": "old", "fields": map[string]any{"Estimate": "1h"}})
	if resp.StatusCode != 409 || out["error"] != "stale" || out["ok"] == true {
		t.Errorf("stale: %d %v", resp.StatusCode, out)
	}
	// the word is enough, whatever the exit code
	s.b.results["backlog.set:apply"] = ports.OperationResult{ExitCode: 1, Stdout: "STALE"}
	if resp, _ := s.act("backlog.set", map[string]any{"ref": "STORY-003", "sha": "old", "fields": map[string]any{"Estimate": "1h"}}); resp.StatusCode != 409 {
		t.Errorf("STALE in the output: %d", resp.StatusCode)
	}
}

func TestARefusalIs400WithItsMessage(t *testing.T) {
	s := serveFixture(t, true)
	s.login()
	s.b.results["backlog.set:apply"] = ports.OperationResult{ExitCode: 1, Stdout: `{"ok": false, "ref": "STORY-003", "error": "Status is not a field Sonar may write"}` + "\n", Stderr: "\ngtt-backlog: Status is not a field Sonar may write\n"}
	resp, out := s.act("backlog.set", map[string]any{"ref": "STORY-003", "sha": "x", "fields": map[string]any{"Status": "Done"}})
	if resp.StatusCode != 400 || out["error"] != "Status is not a field Sonar may write" {
		t.Errorf("refused: %d %v", resp.StatusCode, out)
	}
}

func TestTheOtherThreeActsMapToTheirOperations(t *testing.T) {
	s := serveFixture(t, true)
	s.login()
	s.b.results["pending.answer:apply"] = ok("answered\n")
	s.b.results["inbox.add:apply"] = ok("IDEA-007 written\n")
	s.b.results["sonar.prefs:apply"] = ok("saved\n")
	if resp, out := s.act("pending.answer", map[string]any{"ref": "ASK-014", "answer": "1"}); resp.StatusCode != 200 || out["ok"] != true {
		t.Fatalf("answer: %d %v", resp.StatusCode, out)
	}
	r := s.requestsOf("pending.answer")[0]
	if r.Args["ref"] != "ASK-014" || r.Args["reply"] != "1" || r.Args["via"] != "sonar" || !r.Apply || !r.ConfirmedByHuman || len(r.Args) != 3 {
		t.Errorf("pending.answer: %+v", r)
	}
	text := "an idea with 'quotes', $dollars and\na newline"
	var staged string
	if resp, out := s.act("inbox.add", map[string]any{"text": text, "about": "EPIC-002"}); resp.StatusCode != 200 || out["ok"] != true {
		t.Fatalf("inbox.add: %d %v", resp.StatusCode, out)
	}
	r = s.requestsOf("inbox.add")[0]
	if r.Args["about"] != "EPIC-002" || r.Args["via"] != "sonar" || r.Args["file"] == "" || r.Args["text"] != "" {
		t.Errorf("inbox.add: %+v", r)
	}
	staged = r.Args["file"]
	if _, err := os.Stat(staged); err == nil {
		t.Error("the staged text file is removed after the act")
	}
	if resp, _ := s.act("inbox.add", map[string]any{"text": "  "}); resp.StatusCode != 400 {
		t.Errorf("an empty idea: %d", resp.StatusCode)
	}
	s.b.results["consult.set:apply"] = ok("thread registered\n")
	if resp, out := s.act("consult.set", map[string]any{"url": " https://claude.ai/chat/0b6e1f2a-4c3d-4e5f-8a9b-1c2d3e4f5a6b "}); resp.StatusCode != 200 || out["ok"] != true {
		t.Fatalf("consult.set: %d %v", resp.StatusCode, out)
	}
	r = s.requestsOf("consult.set")[0]
	if r.Args["url"] != "https://claude.ai/chat/0b6e1f2a-4c3d-4e5f-8a9b-1c2d3e4f5a6b" || r.Args["via"] != "" || len(r.Args) != 1 || !r.Apply {
		t.Errorf("consult.set: %+v", r)
	}
	if resp, _ := s.act("consult.set", map[string]any{"url": "  "}); resp.StatusCode != 400 {
		t.Errorf("an empty thread URL: %d", resp.StatusCode)
	}
	if resp, out := s.act("sonar.prefs", map[string]any{"set": map[string]any{"view": "inbox"}}); resp.StatusCode != 200 || out["ok"] != true {
		t.Fatalf("prefs: %d %v", resp.StatusCode, out)
	}
	r = s.requestsOf("sonar.prefs")[0]
	if r.Args["set"] != `{"view":"inbox"}` || r.Args["via"] != "" {
		t.Errorf("sonar.prefs: %+v", r)
	}
}

func TestInboxAddTextReachesTheOperationAsAFile(t *testing.T) {
	s := serveFixture(t, true)
	s.login()
	text := "exactly 'this'\n$HOME and \"quotes\""
	var got string
	s.b.results["inbox.add:apply"] = ok("IDEA-1\n")
	// read the staged file while the operation "runs": the fake records requests, the file is gone later
	fb := &fileSpy{fakeBootstrap: s.b}
	s.srv.App.Factory = spyFactory{fb}
	s.srv.invalidate()
	if resp, _ := s.act("inbox.add", map[string]any{"text": text}); resp.StatusCode != 200 {
		t.Fatalf("%d", resp.StatusCode)
	}
	got = fb.content
	if got != text {
		t.Errorf("the idea is stored as written: %q", got)
	}
}

type fileSpy struct {
	*fakeBootstrap
	content string
}

func (f *fileSpy) Execute(ctx context.Context, req ports.OperationRequest) (ports.OperationResult, error) {
	if req.Operation == "inbox.add" && req.Args["file"] != "" {
		data, _ := os.ReadFile(req.Args["file"])
		f.content = string(data)
	}
	return f.fakeBootstrap.Execute(ctx, req)
}

type spyFactory struct{ b ports.Bootstrap }

func (f spyFactory) At(string) ports.Bootstrap { return f.b }
func (f spyFactory) Installed(root string) bool {
	return root == f.b.Root()
}

func TestSnapshotETagAnd304AndFreshAfterAnAct(t *testing.T) {
	s := serveFixture(t, true)
	resp, body := s.do("GET", "/api/snapshot", "", nil)
	etag := resp.Header.Get("ETag")
	if resp.StatusCode != 200 || etag == "" || !strings.Contains(body, `"schema":"sonar.page.v1"`) || !strings.Contains(body, `"mode":"served"`) {
		t.Fatalf("the page JSON with its tag: %d %q %.200s", resp.StatusCode, etag, body)
	}
	var page struct {
		Repos []struct {
			Snapshot map[string]any `json:"snapshot"`
		} `json:"repos"`
	}
	if err := json.Unmarshal([]byte(body), &page); err != nil || len(page.Repos) != 1 || page.Repos[0].Snapshot["schema"] != "sonar.v1" {
		t.Fatalf("a repo with its snapshot: %v %s", err, body)
	}
	if r2, _ := s.do("GET", "/api/snapshot", "", map[string]string{"If-None-Match": etag}); r2.StatusCode != 304 {
		t.Errorf("the same content is 304: %d", r2.StatusCode)
	}
	// time passes and the snapshot is read again: only generated_at differs, so the tag holds
	s.clk.add(10 * time.Second)
	s.b.results["sonar.snapshot"] = ports.OperationResult{Stdout: strings.Replace(s.snap, "10:00:00", "10:00:10", 1)}
	if r3, _ := s.do("GET", "/api/snapshot", "", map[string]string{"If-None-Match": etag}); r3.StatusCode != 304 {
		t.Errorf("a new moment is not new content: %d", r3.StatusCode)
	}
	// an act: the next snapshot is read at once (inside the 2 s cache) and carries the new content
	s.login()
	s.b.results["backlog.set:apply"] = ok(`{"ok":true,"left":[]}`)
	s.b.results["sonar.snapshot"] = ports.OperationResult{Stdout: strings.Replace(s.snap, `"v":1`, `"v":2`, 1)}
	if resp, _ := s.act("backlog.set", map[string]any{"ref": "STORY-003", "sha": "x", "fields": map[string]any{"Estimate": "1h"}}); resp.StatusCode != 200 {
		t.Fatalf("act: %d", resp.StatusCode)
	}
	r4, body4 := s.do("GET", "/api/snapshot", "", map[string]string{"If-None-Match": etag})
	if r4.StatusCode != 200 || !strings.Contains(body4, `"v":2`) || r4.Header.Get("ETag") == etag {
		t.Errorf("fresh after the act: %d %s", r4.StatusCode, r4.Header.Get("ETag"))
	}
}

func TestAnOlderBootstrapIsShownReadOnlyAndSaysWhichRelease(t *testing.T) {
	s := serveFixture(t, false)
	s.login()
	resp, body := s.do("GET", "/api/snapshot", "", nil)
	if resp.StatusCode != 200 || !strings.Contains(body, `"degraded"`) || !strings.Contains(body, SonarServeNeeds) {
		t.Fatalf("the page is served, degraded, and says what it needs: %d %.300s", resp.StatusCode, body)
	}
	r, out := s.act("backlog.set", map[string]any{"ref": "STORY-003", "sha": "x", "fields": map[string]any{"Estimate": "1h"}})
	msg, _ := out["error"].(string)
	if r.StatusCode != 403 || !strings.Contains(msg, SonarServeNeeds) || !strings.Contains(msg, "gtt update") {
		t.Errorf("an act on it is refused with the release it needs: %d %q", r.StatusCode, msg)
	}
	if len(s.requestsOf("backlog.set")) != 0 {
		t.Error("nothing was written")
	}
	if notes := s.srv.snapshots(context.Background()).repos[0].note; !strings.Contains(notes, "1.23.0") || !strings.Contains(notes, SonarServeNeeds) {
		t.Errorf("one line for the terminal: %q", notes)
	}
}

func TestPageIsServedOnlyWithTheCookieAndKeepsItsCSP(t *testing.T) {
	s := serveFixture(t, true)
	resp, body := s.do("GET", "/", "", nil)
	if resp.StatusCode != 200 || !strings.Contains(body, "STATIC") {
		t.Errorf("without the cookie the page is the static one (read only): %d %s", resp.StatusCode, body)
	}
	s.login()
	resp, body = s.do("GET", "/", "", nil)
	if resp.StatusCode != 200 || !strings.Contains(body, "SERVED") {
		t.Errorf("with the cookie, the served one: %d %s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Security-Policy"); got != "default-src 'none'; connect-src 'self'" {
		t.Errorf("the CSP is the page's own: %q", got)
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" || resp.Header.Get("Referrer-Policy") != "no-referrer" {
		t.Error("hardening headers")
	}
}

func TestTheSecretIsNeverServedNorWorldReadable(t *testing.T) {
	s := serveFixture(t, true)
	s.login()
	secret := string(bytes.TrimSpace(mustRead(t, filepath.Join(s.srv.Dir, "secret"))))
	for _, p := range []string{"/secret", "/../secret", "/serve/../secret", "/fonts/../secret", "/fonts/secret", "/server.json", "/api/secret", "/_admin/open"} {
		resp, body := s.do("GET", p, "", nil)
		if resp.StatusCode == 200 || strings.Contains(body, secret) {
			t.Errorf("GET %s: %d", p, resp.StatusCode)
		}
	}
	if _, body := s.do("GET", "/api/snapshot", "", nil); strings.Contains(body, secret) {
		t.Error("the snapshot never carries the secret")
	}
	if runtime.GOOS != "windows" { // Windows has no Unix permission bits (Go reports 0666/0777): access there follows the ACL of the user's profile
		if info, err := os.Stat(filepath.Join(s.srv.Dir, "secret")); err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("the secret file is 0600: %v %v", info, err)
		}
		if info, _ := os.Stat(s.srv.Dir); info.Mode().Perm() != 0o700 {
			t.Errorf("its folder is 0700: %v", info.Mode())
		}
	}
	if resp, _ := s.do("POST", "/_admin/open", "{}", map[string]string{"Authorization": "Bearer " + s.srv.sessionToken(), "-Origin": ""}); resp.StatusCode != 403 {
		t.Errorf("the cookie is not the admin token: %d", resp.StatusCode)
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestOversizedBodyIsRefused(t *testing.T) {
	s := serveFixture(t, true)
	s.login()
	big := `{"repo":"x","op":"inbox.add","args":{"text":"` + strings.Repeat("a", sonarMaxBody) + `"}}`
	if resp, _ := s.do("POST", "/api/act", big, nil); resp.StatusCode != 400 {
		t.Errorf("more than 64 KB: %d", resp.StatusCode)
	}
	if len(s.requestsOf("inbox.add")) != 0 {
		t.Error("nothing was written")
	}
}

func TestIdleServerClosesAfterEightHoursOfNoUse(t *testing.T) {
	s := serveFixture(t, true)
	s.srv.touch()
	s.clk.add(7 * time.Hour)
	s.do("GET", "/api/snapshot", "", nil) // polling is not use
	if s.srv.idleExpired() {
		t.Error("7 hours is not yet")
	}
	s.clk.add(61 * time.Minute)
	if !s.srv.idleExpired() {
		t.Error("8 hours without use closes it")
	}
	s.srv.touch()
	if s.srv.idleExpired() {
		t.Error("an act or a visit is use")
	}
}

// The daemon binds 127.0.0.1 only, writes {pid, port} without the secret, and `stop` (the admin route, with
// the secret) closes it and removes the file.
func TestDaemonBindsLoopbackWritesNoSecretAndStops(t *testing.T) {
	b := &fakeBootstrap{compatible: true, results: map[string]ports.OperationResult{}}
	a, _ := newApp(t, b)
	done := make(chan error, 1)
	go func() { done <- a.SonarServe(context.Background()) }()
	var info SonarServerInfo
	for i := 0; i < 100; i++ {
		if cur, ok := a.readServerInfo(); ok && serverAlive(cur.Port) {
			info = cur
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if info.Port == 0 || info.PID != os.Getpid() {
		t.Fatalf("server.json has {pid, port}: %+v", info)
	}
	raw := string(mustRead(t, a.sonarInfoPath()))
	secret := strings.TrimSpace(string(mustRead(t, filepath.Join(a.sonarPortfolioDir(), "secret"))))
	if strings.Contains(raw, secret) || strings.Count(raw, ":") != 2 {
		t.Errorf("server.json carries no secret: %s", raw)
	}
	// reachable by loopback, not by any other address of this machine
	addrs, _ := net.InterfaceAddrs()
	for _, ad := range addrs {
		if ipn, ok := ad.(*net.IPNet); ok && !ipn.IP.IsLoopback() && ipn.IP.To4() != nil {
			c, err := net.DialTimeout("tcp", net.JoinHostPort(ipn.IP.String(), jint(info.Port)), 300*time.Millisecond)
			if err == nil {
				c.Close()
				t.Errorf("the server answers on %s: it must listen on 127.0.0.1 only", ipn.IP)
			}
		}
	}
	sec, _ := LoadSonarSecret(a.sonarPortfolioDir())
	if _, err := adminCall(info.Port, []byte("wrong"), "/_admin/stop", map[string]any{}); err == nil {
		t.Error("stop needs the secret")
	}
	if _, err := adminCall(info.Port, sec, "/_admin/stop", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Error(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the server did not close")
	}
	if _, ok := a.readServerInfo(); ok {
		t.Error("server.json is removed when it closes")
	}
}

func jint(n int) string { b, _ := json.Marshal(n); return string(b) }

// The Design view reads the governed documents: a served page of one project must ask for them (it did not, and the
// view said "documents are not part of the portfolio view"), and a Bootstrap that does not declare the argument is read without it.
func TestOneProjectIsReadWithItsDocumentsWhenTheBootstrapDeclaresThem(t *testing.T) {
	s := serveFixture(t, true)
	s.b.ops["sonar.snapshot"] = ports.Operation{Version: 1, Args: []ports.OperationArg{{Name: "docs"}}}
	s.login()
	if resp, _ := s.do("GET", "/api/snapshot", "", nil); resp.StatusCode != 200 {
		t.Fatalf("snapshot: %d", resp.StatusCode)
	}
	reqs := s.requestsOf("sonar.snapshot")
	if len(reqs) == 0 || reqs[0].Args["docs"] != "true" {
		t.Errorf("a single project is read with its documents: %v", reqs)
	}
}

func TestAnOlderBootstrapIsReadWithoutTheDocsArgument(t *testing.T) {
	s := serveFixture(t, true) // its sonar.snapshot declares no arguments
	s.login()
	if resp, _ := s.do("GET", "/api/snapshot", "", nil); resp.StatusCode != 200 {
		t.Fatalf("snapshot: %d", resp.StatusCode)
	}
	for _, r := range s.requestsOf("sonar.snapshot") {
		if len(r.Args) != 0 {
			t.Errorf("an undeclared argument would be refused by the Bootstrap: %v", r.Args)
		}
	}
}
