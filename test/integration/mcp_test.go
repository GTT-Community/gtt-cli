package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GTT-Community/gtt-cli/internal/app"
	"github.com/GTT-Community/gtt-cli/internal/cli"
	"github.com/GTT-Community/gtt-cli/internal/compose"
)

// `gtt mcp` against the real Bootstrap and two demo projects (criteria 41 to 45 end to end). Skipped when the
// Bootstrap has no demo fixture or does not offer findings and code (set GTT_TEST_BOOTSTRAP).

func realSnapshot(t *testing.T, proj string) map[string]any {
	t.Helper()
	cmd := exec.Command("bash", filepath.Join(proj, ".gtt", "scripts", "gtt-sonar.sh"), "snapshot")
	cmd.Dir = proj
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

type mcpSession struct {
	t  *testing.T
	cs *mcp.ClientSession
}

func startMCP(t *testing.T, cwd string, repos ...string) *mcpSession {
	t.Helper()
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("GTT_HOME", filepath.Join(t.TempDir(), "gtt-home"))
	t.Setenv("GTT_NO_AUTO_UPGRADE", "1")
	var rs []string
	for _, r := range repos {
		rs = append(rs, fmt.Sprintf(`{"path":%q}`, r))
	}
	_ = os.MkdirAll(filepath.Join(xdg, "gtt"), 0o755)
	_ = os.WriteFile(filepath.Join(xdg, "gtt", "sonar.json"), []byte(`{"schema":1,"repos":[`+strings.Join(rs, ",")+`]}`), 0o644)
	a, _, err := compose.Wire(compose.Streams{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard})(cli.Options{Dir: cwd})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)
	ct, st := mcp.NewInMemoryTransports()
	ss, err := cli.NewMCPServer(a).Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "it", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return &mcpSession{t: t, cs: cs}
}

// call returns the summary line and the decoded JSON of a tool.
func (s *mcpSession) call(tool string, args map[string]any) (string, map[string]any) {
	s.t.Helper()
	res, err := s.cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil || res.IsError || len(res.Content) != 2 {
		s.t.Fatalf("%s: %v %+v", tool, err, res)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(res.Content[1].(*mcp.TextContent).Text), &m); err != nil {
		s.t.Fatal(err)
	}
	return res.Content[0].(*mcp.TextContent).Text, m
}

func refsOf(v any, key string) []string {
	var out []string
	for _, x := range v.([]any) {
		out = append(out, fmt.Sprint(x.(map[string]any)[key]))
	}
	sort.Strings(out)
	return out
}

func TestMCPAgainstTheDemoProjects(t *testing.T) {
	if !strings.Contains(read(t, pristine, ".gtt/contract/capabilities.json"), "findings.v1") {
		t.Skip("this Bootstrap has no findings.v1")
	}
	p1, p2 := demoProject(t), demoProject(t)
	s := startMCP(t, p1, p1, p2)

	tools, err := s.cs.ListTools(context.Background(), nil)
	if err != nil || len(tools.Tools) != 7 {
		t.Fatalf("tools: %v %v", err, tools)
	}
	for _, tl := range tools.Tools {
		if !tl.Annotations.ReadOnlyHint {
			t.Errorf("%s is not read only", tl.Name)
		}
	}

	snap := realSnapshot(t, p1)
	name := filepath.Base(p1) // both demos are called "proj": go by path
	sum, d := s.call("gtt_projects", nil)
	t.Log(sum)
	rows := d["projects"].([]any)
	if len(rows) != 2 {
		t.Fatalf("two projects, got %v", rows)
	}
	if !reflect.DeepEqual(rows[0].(map[string]any)["counts"], snap["readiness"].(map[string]any)["counts"]) {
		t.Errorf("counts differ from the snapshot: %v", rows[0])
	}

	// the same data as the snapshot
	_, d = s.call("gtt_epic", map[string]any{"repo": p1, "epic": "EPIC-001"})
	var wantStories []string
	for _, st := range snap["stories"].([]any) {
		if m := st.(map[string]any); m["epic"] == "EPIC-001" {
			wantStories = append(wantStories, fmt.Sprint(m["ref"]))
		}
	}
	sort.Strings(wantStories)
	if got := refsOf(d["stories"], "ref"); !reflect.DeepEqual(got, wantStories) {
		t.Errorf("stories %v, snapshot %v", got, wantStories)
	}
	for _, e := range snap["epics"].([]any) {
		if m := e.(map[string]any); m["ref"] == "EPIC-001" {
			got := d["epic"].(map[string]any)
			for _, k := range []string{"pct", "estimate_h", "active_h", "eta", "eta_range", "start"} {
				if !reflect.DeepEqual(got[k], m[k]) {
					t.Errorf("EPIC-001 %s = %v, snapshot %v", k, got[k], m[k])
				}
			}
		}
	}

	_, d = s.call("gtt_activity", map[string]any{"repo": p1})
	if !reflect.DeepEqual(d["now"], snap["activity"].(map[string]any)["now"]) {
		t.Errorf("activity now differs: %v", d["now"])
	}

	sum, d = s.call("gtt_ready", map[string]any{"repo": p1, "ref": "STORY-003"})
	t.Log(sum)
	if d["situation"] == nil {
		t.Errorf("gtt_ready has no situation: %v", d)
	}
	var wantCodes []string
	for _, f := range snap["findings"].([]any) {
		if m := f.(map[string]any); m["ref"] == "STORY-003" {
			wantCodes = append(wantCodes, fmt.Sprint(m["code"]))
		}
	}
	sort.Strings(wantCodes)
	if got := refsOf(d["lacks"], "code"); !reflect.DeepEqual(got, wantCodes) {
		t.Errorf("STORY-003 lacks %v, the snapshot's findings say %v", got, wantCodes)
	}

	// needs_you without repo joins both repos (criterion 45); with repo, one
	sum, d = s.call("gtt_needs_you", nil)
	t.Log(sum)
	byRepo := map[string]int{}
	for _, it := range d["items"].([]any) {
		byRepo[fmt.Sprint(it.(map[string]any)["repo"])]++
	}
	want := len(snap["needs_you"].([]any))
	for _, f := range snap["findings"].([]any) {
		m := f.(map[string]any)
		if m["owner"] == "human" && (m["plane"] == "definition" || m["plane"] == "governance") {
			want++
		}
	}
	if want == 0 || len(d["items"].([]any)) != 2*want || d["page"].(map[string]any)["total"].(float64) != float64(2*want) {
		t.Errorf("joined needs_you: %d items, want %d (twice %d) in %v", len(d["items"].([]any)), 2*want, want, byRepo)
	}
	_, d = s.call("gtt_needs_you", map[string]any{"repo": p2})
	if len(d["items"].([]any)) != want {
		t.Errorf("one repo: %d items, want %d", len(d["items"].([]any)), want)
	}

	_, d = s.call("gtt_findings", map[string]any{"repo": p1, "owner": "human"})
	for _, f := range d["findings"].([]any) {
		if f.(map[string]any)["owner"] != "human" {
			t.Errorf("the owner filter let through %v", f)
		}
	}

	if strings.Contains(read(t, pristine, ".gtt/contract/capabilities.json"), `"code.v1"`) {
		sum, d = s.call("gtt_code", map[string]any{"repo": p1})
		t.Log(sum)
		if d["modules"] == nil || len(d["modules"].([]any)) == 0 {
			t.Errorf("gtt_code has no modules: %v", d)
		}
	}
	_ = name
}

// The walk of the table against the declared operations of the real Bootstrap (criterion 43).
func TestMCPTableAgainstTheRealOperations(t *testing.T) {
	decl, err := app.ReadOperationsFile(filepath.Join(pristine, ".gtt", "contract", "operations.json"))
	if err != nil {
		t.Skip("no operations.json")
	}
	// an older Bootstrap may not declare code.map or findings.list yet
	if problems := app.MCPTableProblems(app.MCPToolTable(), decl, true); len(problems) > 0 {
		t.Errorf("%s", strings.Join(problems, "\n"))
	}
}
