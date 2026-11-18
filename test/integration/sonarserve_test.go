package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/cli"
	"github.com/GTT-Community/gtt-cli/internal/compose"
)

// The Sonar server against a real Bootstrap (set GTT_TEST_BOOTSTRAP to the catalog; the default stays
// ../gtt-bootstrap). Without a Bootstrap that offers sonar.serve.v1 these tests are skipped.

// The Sonar door never reaches a ratification verb nor a human-authority operation outside the four the
// order allows, whatever this Bootstrap declares.
func TestSonarDoorHasNoHumanAuthorityBeyondTheFour(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(pristine, ".gtt", "contract", "operations.json"))
	if err != nil {
		t.Skip("no operations.json")
	}
	var doc struct {
		Operations map[string]struct {
			HumanAuthority bool `json:"human_authority"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	for _, op := range app.DoorOperations(app.DoorSonar) {
		decl, ok := doc.Operations[op]
		if !ok {
			continue // an older Bootstrap does not declare it: the server then shows the repo read only
		}
		if app.RatificationVerb(op) {
			t.Errorf("%s is a ratification verb", op)
		}
		if decl.HumanAuthority && !app.HumanAuthorityAllowed(op) {
			t.Errorf("%s has human authority and is not one of the allowed acts", op)
		}
	}
	// the other way round: every human-authority operation the Bootstrap declares is out of the door unless allowed
	for op, decl := range doc.Operations {
		if decl.HumanAuthority && app.DoorAllows(app.DoorSonar, op) && !app.HumanAuthorityAllowed(op) {
			t.Errorf("the Sonar door reaches %s, a human-authority operation that is not one of the four", op)
		}
	}
	for _, verb := range []string{"approve", "promote", "run.execute", "undo", "freeze", "adr.adopt", "epic.approve", "promotion.apply", "promotion.undo"} {
		if app.DoorAllows(app.DoorSonar, verb) {
			t.Errorf("the Sonar door reaches %s", verb)
		}
	}
}

func demoProject(t *testing.T) string {
	t.Helper()
	script := filepath.Join(pristine, ".gtt", "tests", "fixtures", "sonar-demo.py")
	if _, err := os.Stat(script); err != nil {
		t.Skip("this Bootstrap has no sonar-demo fixture")
	}
	if !exists(pristine, ".gtt/contract/capabilities.json") || !strings.Contains(read(t, pristine, ".gtt/contract/capabilities.json"), "sonar.serve.v1") {
		t.Skip("this Bootstrap does not offer sonar.serve.v1")
	}
	for _, tool := range []string{"python3", "bash", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not available")
		}
	}
	cmd := exec.Command("python3", script, "--full")
	cmd.Env = append(os.Environ(), "GTT_SONAR_NO_WATCH=1")
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("the fixture could not be built: %v", err)
	}
	proj, _ := bufio.NewReader(bytes.NewReader(out)).ReadString('\n')
	proj = strings.TrimSpace(proj)
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(proj)) })
	return proj
}

type liveServer struct {
	t      *testing.T
	srv    *app.SonarServer
	ts     *httptest.Server
	cookie string
	secret []byte
}

func startServer(t *testing.T, proj string) *liveServer {
	t.Helper()
	t.Setenv("GTT_HOME", filepath.Join(t.TempDir(), "gtt-home"))
	t.Setenv("GTT_SONAR_NO_WATCH", "1")
	a, _, err := compose.Wire(compose.Streams{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard})(cli.Options{Dir: proj})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "sonar")
	secret, err := app.LoadSonarSecret(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := &app.SonarServer{App: a, Dir: dir, Secret: secret}
	srv.Register([]string{proj})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	srv.SetPort(ts.Listener.Addr().(*net.TCPAddr).Port)
	return &liveServer{t: t, srv: srv, ts: ts, secret: secret}
}

func (s *liveServer) do(method, path, body string, hdr map[string]string) (*http.Response, []byte) {
	s.t.Helper()
	req, _ := http.NewRequest(method, s.ts.URL+path, strings.NewReader(body))
	if method == "POST" {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GTT-Sonar", "1")
		req.Header.Set("Origin", s.ts.URL)
	}
	if s.cookie != "" {
		req.Header.Set("Cookie", "gtt_sonar="+s.cookie)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
	if err != nil {
		s.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func (s *liveServer) login() {
	s.t.Helper()
	resp, _ := s.do("GET", "/enter?code="+app.MintSonarCode(s.secret, time.Now()), "", nil)
	for _, c := range resp.Cookies() {
		if c.Name == "gtt_sonar" {
			s.cookie = c.Value
		}
	}
	if resp.StatusCode != 302 || s.cookie == "" {
		s.t.Fatalf("the one-time code was not accepted: %d", resp.StatusCode)
	}
}

// shaOf is the sha256 of a story's block, as the page would send it back: read from /api/snapshot.
func (s *liveServer) shaOf(ref string) string {
	s.t.Helper()
	resp, body := s.do("GET", "/api/snapshot", "", nil)
	if resp.StatusCode != 200 {
		s.t.Fatalf("snapshot: %d %s", resp.StatusCode, body)
	}
	var page struct {
		Repos []struct {
			Snapshot struct {
				Stories []struct{ Ref, Sha string } `json:"stories"`
			} `json:"snapshot"`
		} `json:"repos"`
	}
	if err := json.Unmarshal(body, &page); err != nil || len(page.Repos) != 1 {
		s.t.Fatalf("the page JSON: %v %.200s", err, body)
	}
	for _, st := range page.Repos[0].Snapshot.Stories {
		if st.Ref == ref {
			if st.Sha == "" {
				s.t.Fatalf("%s has no sha in the snapshot", ref)
			}
			return st.Sha
		}
	}
	s.t.Fatalf("%s is not in the snapshot", ref)
	return ""
}

func (s *liveServer) act(proj, op string, args map[string]any) (int, map[string]any) {
	s.t.Helper()
	body, _ := json.Marshal(map[string]any{"repo": proj, "op": op, "args": args})
	resp, data := s.do("POST", "/api/act", string(body), nil)
	var out map[string]any
	_ = json.Unmarshal(data, &out)
	return resp.StatusCode, out
}

// Criteria 10, 11 and 13 end to end: the real server, the real Bootstrap, the demo project with its planted problems.
func TestSonarServerFillsAStoryAndRefusesAStaleOne(t *testing.T) {
	proj := demoProject(t)
	s := startServer(t, proj)
	backlog := filepath.Join(proj, "gtt-domain", "backlog.md")
	before, _ := os.ReadFile(backlog)
	if !strings.Contains(string(before), "**Estimate:** ?") {
		t.Fatalf("the fixture plants `Estimate: ?` in STORY-003")
	}

	// without the cookie: read only, whatever is asked
	sha := s.shaOf("STORY-003")
	if code, _ := s.act(proj, "backlog.set", map[string]any{"ref": "STORY-003", "sha": sha, "fields": map[string]any{"Estimate": "3h"}}); code != 403 {
		t.Fatalf("without the cookie: %d", code)
	}
	if after, _ := os.ReadFile(backlog); !bytes.Equal(before, after) {
		t.Fatal("a refused act wrote something")
	}

	// the page: static without the cookie, served with it
	if resp, page := s.do("GET", "/", "", nil); resp.StatusCode != 200 || strings.Contains(string(page), `"mode": "served"`) || strings.Contains(string(page), `"mode":"served"`) {
		t.Fatalf("without the cookie the page is the static one: %d", resp.StatusCode)
	}
	s.login()
	resp, page := s.do("GET", "/", "", nil)
	if resp.StatusCode != 200 || !strings.Contains(string(page), `served`) || !strings.Contains(resp.Header.Get("Content-Security-Policy"), "connect-src 'self'") {
		t.Fatalf("with the cookie the served page, with its CSP: %d %q", resp.StatusCode, resp.Header.Get("Content-Security-Policy"))
	}

	// the Estimate of STORY-003 is saved: that line changes, no other
	code, out := s.act(proj, "backlog.set", map[string]any{"ref": "STORY-003", "sha": sha, "fields": map[string]any{"Estimate": "3h"}})
	if code != 200 || out["ok"] != true {
		t.Fatalf("backlog.set: %d %v", code, out)
	}
	after, _ := os.ReadFile(backlog)
	old, now := strings.Split(string(before), "\n"), strings.Split(string(after), "\n")
	if len(old) != len(now) {
		t.Fatalf("the line count must not change: %d -> %d", len(old), len(now))
	}
	changed := 0
	for i := range old {
		if old[i] != now[i] {
			changed++
			if !strings.Contains(now[i], "Estimate") || !strings.Contains(now[i], "3h") {
				t.Errorf("an unexpected line changed: %q -> %q", old[i], now[i])
			}
		}
	}
	if changed != 1 {
		t.Errorf("only the Estimate line changes, %d lines did", changed)
	}

	// the finding of that story is gone in the next snapshot, without waiting for the cache
	_, body := s.do("GET", "/api/snapshot", "", nil)
	if strings.Contains(string(body), `"code":"DEF-STORY-ESTIMATE"`) && strings.Contains(string(body), `"ref":"STORY-003"`) {
		var page struct {
			Repos []struct {
				Snapshot struct {
					Findings []struct{ Code, Ref string } `json:"findings"`
				} `json:"snapshot"`
			} `json:"repos"`
		}
		_ = json.Unmarshal(body, &page)
		for _, f := range page.Repos[0].Snapshot.Findings {
			if f.Code == "DEF-STORY-ESTIMATE" && f.Ref == "STORY-003" {
				t.Errorf("the Estimate finding of STORY-003 is still there after the act")
			}
		}
	}

	// the old sha is stale now: 409 and nothing is written
	frozen, _ := os.ReadFile(backlog)
	code, out = s.act(proj, "backlog.set", map[string]any{"ref": "STORY-003", "sha": sha, "fields": map[string]any{"Estimate": "99h"}})
	if code != 409 || out["error"] != "stale" {
		t.Fatalf("a stale sha: %d %v", code, out)
	}
	if again, _ := os.ReadFile(backlog); !bytes.Equal(frozen, again) {
		t.Fatal("a stale act wrote something")
	}

	// a field Sonar never writes is refused by the Bootstrap, not by luck
	fresh := s.shaOf("STORY-003")
	code, out = s.act(proj, "backlog.set", map[string]any{"ref": "STORY-003", "sha": fresh, "fields": map[string]any{"Status": "Done"}})
	if code != 400 || out["ok"] == true {
		t.Fatalf("Status is never written: %d %v", code, out)
	}
	if again, _ := os.ReadFile(backlog); !bytes.Equal(frozen, again) {
		t.Fatal("Status was written")
	}

	// and nothing that ratifies exists in the server
	if code, _ := s.act(proj, "freeze", map[string]any{}); code != 403 {
		t.Errorf("freeze: %d", code)
	}
}

// Criterion 14 on the server side: an input about a ref becomes an IDEA that says what it is about, with the
// journal's `via`.
func TestSonarServerSendsAnInputAboutARef(t *testing.T) {
	proj := demoProject(t)
	s := startServer(t, proj)
	s.login()
	code, out := s.act(proj, "inbox.add", map[string]any{"text": "the sync must also cover deletions", "about": "EPIC-001"})
	if code != 200 || out["ok"] != true {
		t.Fatalf("inbox.add --about: %d %v", code, out)
	}
	var found string
	filepath.WalkDir(filepath.Join(proj, "docs", "ideas"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".md") {
			if data, _ := os.ReadFile(p); strings.Contains(string(data), "the sync must also cover deletions") {
				found = string(data)
			}
		}
		return nil
	})
	if found == "" {
		t.Fatal("the IDEA was not written under docs/ideas/")
	}
	if !strings.Contains(found, "EPIC-001") || !(strings.Contains(found, "About:") || strings.Contains(found, "Sobre:")) {
		t.Errorf("the IDEA says what it is about:\n%s", found)
	}
}
