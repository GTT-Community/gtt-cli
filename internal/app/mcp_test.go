package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Criteria 42 to 45 of the order (the data side of `gtt mcp`) and the rule of 12.1: the tools read the same data as
// the snapshot, an older Bootstrap is an answer and not a failure, needs_you joins the portfolio, and no tool
// reaches an operation that writes, one with human authority, or a ratification verb.

type pathFactory map[string]*fakeBootstrap

func (f pathFactory) At(p string) ports.Bootstrap {
	if b, ok := f[p]; ok {
		return b
	}
	return nil
}
func (f pathFactory) Installed(p string) bool { _, ok := f[p]; return ok }

const mcpSnapshotJSON = `{"schema":"sonar.v1","lang":"%s","repo":{"path":"%s","name":"%s","key":"K"},
"project":{"phase":"DEV"},"suite_version":"1.26.0","readiness":{"counts":{"blocked":1,"mine":2,"ready":3}},
"summary":{"epics":{"total":1,"done":0},"stories_done":1,"stories_total":2,"stories_blocked":0,"in_progress":1,"progress_pct":50},
"epics":[{"ref":"EPIC-001","title":"Base","raw_status":"In Progress","derived":"in_progress","pct":50,"estimate_h":8.0,"active_h":2.0,"real_h":4.0,"start":"2026-11-17","end":null,"eta":"2026-11-19","eta_range":["2026-11-19","2026-11-21"]}],
"stories":[{"ref":"STORY-001","epic":"EPIC-001","title":"Schema","status":"Done","estimate_h":2.0,"active_h":null,"ready":"ready"},
{"ref":"STORY-002","epic":"EPIC-001","title":"Endpoint","status":"In Progress","estimate_h":2.0,"active_h":1.5,"ready":"ready"}],
"needs_you":[{"kind":"ask","ref":"ASK-001","title":"Approve payments?","how":"decide","command":"gtt answer ASK-001"}],
"findings":[{"code":"DEF-EPIC-APPROVAL","plane":"definition","level":"WARNING","ref":"EPIC-001","owner":"human","text":"not approved"},
{"code":"DEF-STORY-ESTIMATE","plane":"definition","level":"WARNING","ref":"STORY-003","owner":"agent","text":"no estimate"},
{"code":"GTT-DEPENDS","plane":"code","level":"WARNING","ref":"src/a","owner":"human","text":"dependency"}],
"activity":{"now":{"story":"STORY-002","active_h":1.5,"estimate_h":2.0},"last":[{"actor":"agent","at":"2026-11-18T01:00:00Z","ref":"STORY-002","text":"started"}],"journal":true,"sources":["hook"]},
"working_now":[{"ref":"STORY-002","since_h":1.5}]}`

const mcpFindingsJSON = `{"findings":[{"code":"DEF-STORY-ESTIMATE","plane":"definition","level":"WARNING","ref":"STORY-003","epic":"EPIC-001","where":"backlog.md:37","owner":"agent","for":null,"source":"gtt_backlog","text":"no estimate","fix":{"chat":"x"}},
{"code":"DEF-EPIC-APPROVAL","plane":"definition","level":"WARNING","ref":"EPIC-001","owner":"human","text":"not approved"}],
"readiness":{"EPIC-001":{"kind":"epic","status":"In Progress","situation":"to_define"},"STORY-003":{"kind":"story","epic":"EPIC-001","status":"Planned","situation":"to_define"}},"counts":{"to_define":2}}`

const mcpCodeJSON = `{"modules":[{"id":"src/a","path":"src/a/","layer":1,"files":2,"may":["src/core/"],"must_not":["src/b/"]}],
"edges":[{"from":"src/a/","to":"src/core/","count":1,"allowed":true},{"from":"src/a/","to":"src/b/","count":2,"example":"src/a/x.py:3","allowed":false}],
"design_to_code":{"EPIC-001":[{"item":"F-1","text":"x","stories":["STORY-001"],"built":true},{"item":"F-2","text":"y","built":false}]},"outside":[]}`

// mcpRepo builds a fake repo; newer=false is a Bootstrap 1.24 with no findings and no code map.
func mcpFake(t *testing.T, name, lang string, newer bool) (*fakeBootstrap, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(filepath.Join(root, ".gtt"), 0o755); err != nil {
		t.Fatal(err)
	}
	b := &fakeBootstrap{root: root, compatible: true, version: "1.26.0", results: map[string]ports.OperationResult{}}
	b.ops = opsNamed("sonar.snapshot", "sonar.render")
	b.caps = []ports.Capability{{ID: "sonar.v1", Version: 1}}
	b.results["sonar.snapshot"] = ports.OperationResult{Stdout: fmt.Sprintf(mcpSnapshotJSON, lang, root, name)}
	if newer {
		for k, v := range opsNamed("findings.list", "code.map") {
			b.ops[k] = v
		}
		b.caps = append(b.caps, ports.Capability{ID: "findings.v1", Version: 1}, ports.Capability{ID: "code.v1", Version: 1})
		b.results["findings.list"] = ports.OperationResult{Stdout: mcpFindingsJSON}
		b.results["code.map"] = ports.OperationResult{Stdout: mcpCodeJSON}
	} else {
		b.version = "1.24.0"
	}
	return b, root
}

func mcpApp(t *testing.T, repos ...*fakeBootstrap) *App {
	t.Helper()
	pf := pathFactory{}
	var cfg []string
	for _, b := range repos {
		pf[b.root] = b
		cfg = append(cfg, fmt.Sprintf(`{"path":%q}`, b.root))
	}
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	_ = os.MkdirAll(filepath.Join(xdg, "gtt"), 0o755)
	_ = os.WriteFile(filepath.Join(xdg, "gtt", "sonar.json"), []byte(`{"schema":1,"repos":[`+strings.Join(cfg, ",")+`]}`), 0o644)
	cwd := t.TempDir() // not a project: the portfolio is the list
	return &App{Factory: pf, Git: noGit{}, Report: silent{}, Cwd: cwd, Home: t.TempDir()}
}

func data(t *testing.T, ans MCPAnswer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(ans.JSON()), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func call(t *testing.T, a *App, tool string, args MCPArgs) (MCPAnswer, map[string]any) {
	t.Helper()
	ans, err := a.MCPCall(context.Background(), tool, args)
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	if ans.Summary == "" || strings.Contains(ans.Summary, "\n") {
		t.Errorf("%s: the summary must be exactly one line, got %q", tool, ans.Summary)
	}
	return ans, data(t, ans)
}

func refs(items any, key string) []string {
	var out []string
	for _, x := range items.([]any) {
		out = append(out, x.(map[string]any)[key].(string))
	}
	return out
}

// Criterion 42: each tool returns the data of the snapshot of the same repo, in the language of the project.
func TestMCPToolsReturnTheSnapshotData(t *testing.T) {
	b, _ := mcpFake(t, "demo", "es", true)
	a := mcpApp(t, b)
	var snap map[string]any
	_ = json.Unmarshal([]byte(b.results["sonar.snapshot"].Stdout), &snap)

	ans, d := call(t, a, ToolProjects, MCPArgs{})
	row := d["projects"].([]any)[0].(map[string]any)
	if !reflect.DeepEqual(row["counts"], snap["readiness"].(map[string]any)["counts"]) || row["phase"] != "DEV" || row["bootstrap"] != "1.26.0" {
		t.Errorf("gtt_projects differs from the snapshot: %v", row)
	}
	if !strings.Contains(ans.Summary, "proyectos") {
		t.Errorf("the summary must speak the project's language (es): %q", ans.Summary)
	}

	_, d = call(t, a, ToolEpic, MCPArgs{Epic: "EPIC-001"})
	e := d["epic"].(map[string]any)
	se := snap["epics"].([]any)[0].(map[string]any)
	for _, k := range []string{"ref", "pct", "estimate_h", "active_h", "eta", "eta_range", "start", "end"} {
		if !reflect.DeepEqual(e[k], se[k]) {
			t.Errorf("gtt_epic %s = %v, snapshot has %v", k, e[k], se[k])
		}
	}
	if got := refs(d["stories"], "ref"); !reflect.DeepEqual(got, []string{"STORY-001", "STORY-002"}) {
		t.Errorf("stories of the Epic: %v", got)
	}

	_, d = call(t, a, ToolActivity, MCPArgs{})
	if !reflect.DeepEqual(d["now"], snap["activity"].(map[string]any)["now"]) || len(d["last"].([]any)) != 1 {
		t.Errorf("gtt_activity differs from the snapshot: %v", d)
	}

	_, d = call(t, a, ToolFindings, MCPArgs{Plane: "definition"})
	if got := refs(d["findings"], "code"); !reflect.DeepEqual(got, []string{"DEF-STORY-ESTIMATE", "DEF-EPIC-APPROVAL"}) {
		t.Errorf("gtt_findings: %v", got)
	}
	var sent ports.OperationRequest
	for _, r := range b.requests {
		if r.Operation == "findings.list" {
			sent = r
		}
	}
	if sent.Args["plane"] != "definition" {
		t.Errorf("the plane filter must reach findings.list: %v", sent)
	}

	_, d = call(t, a, ToolReady, MCPArgs{Ref: "EPIC-001"})
	if d["situation"] != "to_define" || d["kind"] != "epic" || len(d["lacks"].([]any)) != 2 {
		t.Errorf("gtt_ready: %v", d)
	}
	if up := d["up_to"].(map[string]any); up["human"].(float64) != 1 || up["agent"].(float64) != 1 {
		t.Errorf("whom it is up to: %v", up)
	}

	_, d = call(t, a, ToolCode, MCPArgs{})
	dis := d["disallowed_dependencies"].([]any)
	if len(dis) != 1 || dis[0].(map[string]any)["to"] != "src/b/" || len(d["modules"].([]any)) != 1 {
		t.Errorf("gtt_code: %v", d)
	}
}

// Every tool reads through `query`: only the three declared read operations, never Apply, never a mutation.
func TestMCPNeverMutatesAndStaysInItsTable(t *testing.T) {
	b, _ := mcpFake(t, "demo", "en", true)
	a := mcpApp(t, b)
	for _, tool := range MCPToolOrder {
		args := MCPArgs{Ref: "EPIC-001"}
		if _, err := a.MCPCall(context.Background(), tool, args); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range b.requests {
		if r.Apply {
			t.Errorf("%s was applied: a tool never mutates", r.Operation)
		}
		if !DoorAllows(DoorMCP, r.Operation) {
			t.Errorf("%s is not an operation of the MCP door", r.Operation)
		}
	}
	// a tool asking for an operation outside its row is refused before it reaches the Bootstrap
	n := len(b.requests)
	if err := a.mcpRun(context.Background(), b, ToolEpic, OpCodeMap, nil, nil); err == nil || len(b.requests) != n {
		t.Errorf("gtt_epic must not reach code.map: %v", err)
	}
	if err := a.mcpRun(context.Background(), b, ToolCode, "backlog.set", nil, nil); err == nil || len(b.requests) != n {
		t.Errorf("no tool reaches backlog.set: %v", err)
	}
}

// Criterion 44: an older Bootstrap is a normal answer that says which release is needed.
func TestMCPOlderBootstrapSaysWhatItNeeds(t *testing.T) {
	b, _ := mcpFake(t, "old", "en", false)
	a := mcpApp(t, b)
	for tool, want := range map[string]string{ToolFindings: "1.25", ToolReady: "1.25", ToolCode: "1.26"} {
		ans, d := call(t, a, tool, MCPArgs{Ref: "EPIC-001"})
		if !strings.Contains(ans.Summary, want) || d["supported"] != false || d["needs_bootstrap"] != want {
			t.Errorf("%s: %q %v, want it to name %s", tool, ans.Summary, d, want)
		}
	}
	for _, r := range b.requests {
		if r.Operation == "findings.list" || r.Operation == "code.map" {
			t.Errorf("an operation the Bootstrap does not offer was asked for: %s", r.Operation)
		}
	}
	// a Bootstrap with no Sonar at all
	b2, _ := mcpFake(t, "older", "es", false)
	b2.ops = map[string]ports.Operation{}
	b2.caps = nil
	a2 := mcpApp(t, b2)
	ans, d := call(t, a2, ToolEpic, MCPArgs{})
	if !strings.Contains(ans.Summary, "1.25") || d["supported"] != false {
		t.Errorf("no Sonar: %q", ans.Summary)
	}
	ans, _ = call(t, a2, ToolNeedsYou, MCPArgs{})
	if !strings.Contains(ans.Summary, "1.25") {
		t.Errorf("needs_you must say it too: %q", ans.Summary)
	}
}

// Criterion 45: gtt_needs_you without repo joins every repo of the portfolio.
func TestMCPNeedsYouJoinsThePortfolio(t *testing.T) {
	b1, _ := mcpFake(t, "alpha", "es", true)
	b2, _ := mcpFake(t, "beta", "en", true)
	a := mcpApp(t, b1, b2)
	_, d := call(t, a, ToolNeedsYou, MCPArgs{})
	seen := map[string]int{}
	for _, it := range d["items"].([]any) {
		seen[it.(map[string]any)["repo"].(string)]++
	}
	// per repo: the ASK and the two human findings of definition/governance (the code-plane one is not Sonar's "needs you")
	if seen["alpha"] != 2 || seen["beta"] != 2 {
		t.Errorf("both repos must be joined: %v", seen)
	}
	_, d = call(t, a, ToolNeedsYou, MCPArgs{Repo: "beta"})
	for _, it := range d["items"].([]any) {
		if it.(map[string]any)["repo"] != "beta" {
			t.Errorf("one repo only: %v", it)
		}
	}
	if ans, _ := call(t, a, ToolNeedsYou, MCPArgs{Repo: "nope"}); !strings.Contains(ans.Summary, "alpha") {
		t.Errorf("an unknown repo lists the known ones: %q", ans.Summary)
	}
	// a repo that cannot answer does not stop the others
	b3, _ := mcpFake(t, "gamma", "en", false)
	b3.ops = map[string]ports.Operation{}
	a = mcpApp(t, b1, b3)
	_, d = call(t, a, ToolNeedsYou, MCPArgs{})
	if len(d["items"].([]any)) != 2 || d["notes"] == nil {
		t.Errorf("one repo read, one noted: %v", d)
	}
}

func TestMCPPagesAtFifty(t *testing.T) {
	b, _ := mcpFake(t, "big", "en", true)
	var fs []string
	for i := 0; i < 120; i++ {
		fs = append(fs, fmt.Sprintf(`{"code":"X-%03d","plane":"definition","level":"NOTICE","ref":"STORY-%03d","owner":"agent","text":"t"}`, i, i))
	}
	b.results["findings.list"] = ports.OperationResult{Stdout: `{"findings":[` + strings.Join(fs, ",") + `],"readiness":{},"counts":{}}`}
	a := mcpApp(t, b)
	_, d := call(t, a, ToolFindings, MCPArgs{Limit: 500})
	pg := d["page"].(map[string]any)
	if len(d["findings"].([]any)) != 50 || pg["total"].(float64) != 120 || pg["more"] != true {
		t.Errorf("first page: %v", pg)
	}
	_, d = call(t, a, ToolFindings, MCPArgs{Offset: 100})
	if len(d["findings"].([]any)) != 20 || d["page"].(map[string]any)["more"] != false {
		t.Errorf("last page: %v", d["page"])
	}
}

// operationsDeclared reads what the Bootstrap declares: the real file when it is near, else the fixture copy.
func operationsDeclared(t *testing.T) (map[string]ports.Operation, string) {
	t.Helper()
	var cands []string
	if p := os.Getenv("GTT_TEST_BOOTSTRAP"); p != "" {
		cands = append(cands, filepath.Join(p, ".gtt", "contract", "operations.json"))
	}
	cands = append(cands, filepath.Join("..", "..", "..", "gtt-bootstrap-next", ".gtt", "contract", "operations.json"), filepath.Join("testdata", "operations.json"))
	for _, c := range cands {
		if ops, err := ReadOperationsFile(c); err == nil && len(ops) > 0 {
			return ops, c
		}
	}
	t.Fatal("no operations.json to read the flags from")
	return nil, ""
}

// THE RULE (12.1): walk the table of tools and operations; no tool reaches a human-authority operation, one that
// mutates, or a ratification verb. Adding an action tomorrow cannot open a door to act for the human (I1).
func TestMCPTableReachesNoHumanAuthorityNoMutationNoRatification(t *testing.T) {
	decl, src := operationsDeclared(t)
	table := MCPToolTable()
	if problems := MCPTableProblems(table, decl, false); len(problems) > 0 {
		t.Errorf("against %s:\n%s", src, strings.Join(problems, "\n"))
	}
	var tools []string
	for k := range table {
		tools = append(tools, k)
	}
	sort.Strings(tools)
	want := append([]string(nil), MCPToolOrder...)
	sort.Strings(want)
	if !reflect.DeepEqual(tools, want) || len(tools) != 7 {
		t.Errorf("the table lists %v, the server offers %v", tools, want)
	}
	// the door's list is exactly the union of the tools' rows
	union := map[string]bool{}
	for _, ops := range table {
		for _, op := range ops {
			union[op] = true
		}
	}
	for _, op := range DoorOperations(DoorMCP) {
		if !union[op] {
			t.Errorf("the MCP door lists %s that no tool uses", op)
		}
		delete(union, op)
	}
	if len(union) != 0 {
		t.Errorf("tools reach what the door does not list: %v", union)
	}
	// the walk itself must be able to fail: a table that cheats is caught
	bad := map[string][]string{"t1": {"backlog.set"}, "t2": {"idea.triage"}, "t3": {"epic.approve"}, "t4": {"run.execute"}, "t5": {"promotion.undo"},
		"t6": {"freeze"}, "t7": {"adr.adopt"}, "t8": {"approve"}, "t9": {"no.such.operation"}}
	got := MCPTableProblems(bad, decl, false)
	if len(got) < len(bad) {
		t.Errorf("the walk missed a cheating tool: %v", got)
	}
	for tool := range bad {
		found := false
		for _, p := range got {
			found = found || strings.HasPrefix(p, tool+" ")
		}
		if !found {
			t.Errorf("%s reaches a forbidden operation and the walk did not say so", tool)
		}
	}
}
