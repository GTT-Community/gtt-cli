package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// actRequest is what the page sends to /api/act.
type actRequest struct {
	Repo string                     `json:"repo"`
	Op   string                     `json:"op"`
	Args map[string]json.RawMessage `json:"args"`
}

// text reads an argument as text; a number is its digits. Anything else is not text.
func (q actRequest) text(name string) string {
	raw, ok := q.Args[name]
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	return ""
}

// repoFor finds the repo an act is for: the path the page reports, or the registered one.
func (s *SonarServer) repoFor(set *snapSet, repo string) *repoState {
	if repo == "" && len(set.repos) == 1 {
		return set.repos[0]
	}
	for _, rs := range set.repos {
		if rs.path == repo || (rs.snapRoot != "" && rs.snapRoot == repo) {
			return rs
		}
	}
	if repo != "" {
		real, err := filepath.EvalSymlinks(repo)
		for _, rs := range set.repos {
			if r2, e2 := filepath.EvalSymlinks(rs.path); err == nil && e2 == nil && r2 == real {
				return rs
			}
		}
	}
	return nil
}

// act is the only way a page writes. The operation must be on the door's closed list; its arguments are
// mapped to the declared operation's; it runs in ITS repo, with ITS Bootstrap, as an act of the human.
func (s *SonarServer) act(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, sonarMaxBody)
	var q actRequest
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		deny(w, http.StatusBadRequest, "the request is not valid JSON (or it is too large)")
		return
	}
	if !DoorAllows(DoorSonar, q.Op) {
		deny(w, http.StatusForbidden, "this operation is not available from the page: "+q.Op)
		return
	}
	set := s.snapshots(r.Context())
	rs := s.repoFor(set, q.Repo)
	if rs == nil || rs.b == nil {
		deny(w, http.StatusBadRequest, "unknown repo")
		return
	}
	if !rs.serve {
		deny(w, http.StatusForbidden, "read only: "+strings.TrimPrefix(rs.note, rs.name+": "))
		return
	}
	req, cleanup, bad := s.mapAct(q)
	defer cleanup()
	if bad != "" {
		deny(w, http.StatusBadRequest, bad)
		return
	}
	s.actMu.Lock()
	res, err := rs.b.Execute(r.Context(), req)
	s.actMu.Unlock()
	if err != nil {
		code := http.StatusBadRequest
		msg := err.Error()
		var ce *core.Error
		if errors.As(err, &ce) {
			msg = ce.What
			if ce.Code == core.ExitCancelled {
				code = http.StatusForbidden
			}
		}
		deny(w, code, msg)
		return
	}
	text := strings.TrimSpace(res.Stdout + "\n" + res.Stderr)
	if res.ExitCode == 3 || strings.Contains(text, "STALE") {
		// Lo que guardas es lo que viste: the backlog changed while it was being written.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "stale", "message": text})
		return
	}
	if !res.OK() {
		deny(w, http.StatusBadRequest, refusalMessage(res))
		return
	}
	s.invalidate() // the next snapshot is read again, after the act
	s.App.log(rs.path, "sonar.act", "success", map[string]any{"op": q.Op, "ref": q.text("ref"), "via": "sonar"})
	out := map[string]any{"ok": true}
	if line := lastLine(res.Stdout); strings.HasPrefix(line, "{") {
		var doc map[string]any
		if json.Unmarshal([]byte(line), &doc) == nil {
			for k, v := range doc {
				out[k] = v
			}
			out["ok"] = true
		}
	} else if text != "" {
		out["output"] = text
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func firstMessage(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return "the Bootstrap refused it"
}

// refusalMessage is what the Bootstrap said when it refused: the "error" of its JSON line when it prints one,
// else the first line of what it wrote.
func refusalMessage(res ports.OperationResult) string {
	for _, l := range strings.Split(res.Stdout, "\n") {
		var doc struct {
			Error string `json:"error"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(l)), &doc) == nil && doc.Error != "" {
			return doc.Error
		}
	}
	if m := firstMessage(res.Stderr); strings.TrimSpace(res.Stderr) != "" {
		return m
	}
	return firstMessage(res.Stdout)
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// mapAct turns what the page sent into the request of the declared operation. Every act carries via=sonar, so
// the journal says who and by which door. The returned func removes any staged file.
func (s *SonarServer) mapAct(q actRequest) (req ports.OperationRequest, cleanup func(), bad string) {
	cleanup = func() {}
	args := map[string]string{"via": "sonar"}
	req = ports.OperationRequest{Operation: q.Op, Args: args, Apply: true, ConfirmedByHuman: true}
	switch q.Op {
	case OpBacklogSet:
		ref, sha := q.text("ref"), q.text("sha")
		if ref == "" || sha == "" || len(q.Args["fields"]) == 0 {
			return req, cleanup, "ref, sha and fields are needed"
		}
		var fields map[string]any
		if err := json.Unmarshal(q.Args["fields"], &fields); err != nil || len(fields) == 0 {
			return req, cleanup, "fields must be an object with at least one field"
		}
		enc, _ := json.Marshal(fields)
		args["ref"], args["sha"], args["fields"] = ref, sha, string(enc)
	case OpInboxAdd:
		text := q.text("text")
		if strings.TrimSpace(text) == "" {
			return req, cleanup, "the text is empty"
		}
		name, err := tempText(text) // the text travels in a file: no quote or newline can change what is stored
		if err != nil {
			return req, cleanup, "the text could not be staged"
		}
		cleanup = func() { _ = os.Remove(name) }
		args["file"] = name
		if about := q.text("about"); about != "" {
			args["about"] = about
		}
	case OpPendingAnswer:
		ref, answer := q.text("ref"), q.text("answer")
		if ref == "" || answer == "" {
			return req, cleanup, "ref and answer are needed"
		}
		args["ref"], args["reply"] = ref, answer
	case OpConsultSet:
		url := q.text("url")
		if strings.TrimSpace(url) == "" {
			return req, cleanup, "the URL of the thread is needed"
		}
		args["url"] = strings.TrimSpace(url)
		delete(args, "via") // the thread is the person's own state, not an act on the project: consult.set does not take via
		req.ConfirmedByHuman = false
	case OpSonarPrefs:
		if len(q.Args["set"]) == 0 {
			return req, cleanup, "set is needed"
		}
		var set map[string]any
		if err := json.Unmarshal(q.Args["set"], &set); err != nil {
			return req, cleanup, "set must be an object"
		}
		enc, _ := json.Marshal(set)
		args["set"] = string(enc)
		delete(args, "via") // the preferences are not an act on the project: sonar.prefs does not take via
		req.ConfirmedByHuman = false
	}
	return req, cleanup, ""
}

// --------------------------------------------------------------------------------------------------- admin

// admin is how `gtt sonar` talks to a server that already runs: it holds the secret, so it can say "also show
// this repo" or "close". The browser has neither the bearer nor a way to send it cross-site.
func (s *SonarServer) admin(w http.ResponseWriter, r *http.Request) {
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if r.Header.Get("Origin") != "" || got == "" || !constEq(got, s.adminToken()) {
		deny(w, http.StatusForbidden, "forbidden")
		return
	}
	s.touch()
	switch r.URL.Path {
	case "/_admin/open":
		r.Body = http.MaxBytesReader(w, r.Body, sonarMaxBody)
		var body struct {
			Repos []string `json:"repos"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			deny(w, http.StatusBadRequest, "not valid JSON")
			return
		}
		s.Register(body.Repos)
		s.App.keepWatchers(s.paths())
		set := s.snapshots(context.Background())
		var notes []string
		for _, rs := range set.repos {
			if rs.note != "" {
				notes = append(notes, rs.note)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "notes": notes, "repos": len(set.repos)})
	case "/_admin/stop":
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
		if s.Stop != nil {
			go s.Stop()
		}
	default:
		deny(w, http.StatusNotFound, "not found")
	}
}
