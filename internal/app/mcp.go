package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/lifecycle"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// The data side of `gtt mcp`: seven read-only tools that answer from the same declared operations as Sonar
// (sonar.snapshot, findings.list, code.map), each through `query` and never `mutate`. The protocol itself (the
// SDK, stdio) lives in internal/cli; here a tool is a function from arguments to an answer, so a test can call it
// without a transport. Which operation each tool may reach is the table in doors.go (mcpToolOps).

const (
	// MCPPageMax is the most items a tool answers in one page.
	MCPPageMax = 50
	// MCPNeedsFindings and MCPNeedsCode are the Bootstrap releases that bring what the tools read.
	MCPNeedsFindings = "1.25"
	MCPNeedsCode     = "1.26"
	findingsCap      = "findings.v1"
	codeCap          = "code.v1"
)

// MCPArgs is what any tool may be asked; each tool reads the fields it declares.
type MCPArgs struct {
	Repo   string // a repo of the portfolio: its folder name or its path
	Plane  string
	Owner  string
	Level  string
	Ref    string
	Epic   string
	Story  string
	Offset int
	Limit  int
}

// MCPAnswer is one tool's answer: ONE summary line in the project's language and compact JSON.
type MCPAnswer struct {
	Summary string
	Data    map[string]any
}

// JSON is the compact JSON of the answer.
func (m MCPAnswer) JSON() string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(m.Data)
	return strings.TrimSpace(buf.String())
}

// ---------------------------------------------------------------------------------------------- language

func tr(lang, en, es string) string {
	if lang == "es" {
		return es
	}
	return en
}

// langOf is the language recorded for a project, English when it has none.
func langOf(root string) string {
	if st, err := (lifecycle.Store{Root: root}).Load(); err == nil && st.Language != "" {
		return strings.ToLower(strings.SplitN(st.Language, "-", 2)[0])
	}
	return "en"
}

// ---------------------------------------------------------------------------------------------- portfolio

// mcpPortfolio is every repo the tools may read: the list of `gtt sonar list` and the project this server was
// started in. A tool never reads a folder that is not one of them.
func (a *App) mcpPortfolio(ctx context.Context) []string {
	var paths []string
	seen := map[string]bool{}
	add := func(p string) {
		p = filepath.Clean(p)
		if !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	if c, err := readSonarConfig(); err == nil {
		for _, r := range c.Repos {
			add(r.Path)
		}
	}
	if det := a.detect(ctx); a.Factory.Installed(det.Root) {
		add(det.Root)
	}
	return paths
}

func names(paths []string) string {
	var n []string
	for _, p := range paths {
		n = append(n, filepath.Base(p))
	}
	return strings.Join(n, ", ")
}

// mcpPick resolves the repo argument. Empty: every repo when the tool joins them (all), else the project this
// server runs in, else the only repo there is. A name that does not match says which ones exist.
func (a *App) mcpPick(ctx context.Context, repo string, all bool) ([]string, string) {
	port := a.mcpPortfolio(ctx)
	if len(port) == 0 {
		return nil, "No GTT project is known here. Start `gtt mcp` inside a project, or add repos with `gtt sonar add <folder>`."
	}
	if repo == "" {
		if all || len(port) == 1 {
			return port, ""
		}
		if det := a.detect(ctx); a.Factory.Installed(det.Root) {
			return []string{filepath.Clean(det.Root)}, ""
		}
		return nil, "Which repo? Pass `repo` with one of: " + names(port) + "."
	}
	var hit []string
	for _, p := range port {
		if p == filepath.Clean(repo) || strings.EqualFold(filepath.Base(p), repo) {
			hit = append(hit, p)
		}
	}
	switch len(hit) {
	case 0:
		return nil, fmt.Sprintf("No repo %q in the portfolio. Known repos: %s.", repo, names(port))
	case 1:
		return hit, ""
	}
	return nil, fmt.Sprintf("%q matches more than one repo (%s): pass the full path.", repo, strings.Join(hit, ", "))
}

// ---------------------------------------------------------------------------------------------- reading

// mcpRun runs ONE declared operation for a tool, through query. The table in doors.go decides whether the tool
// may; nothing else in this file reaches a Bootstrap.
func (a *App) mcpRun(ctx context.Context, b ports.Bootstrap, tool, op string, args map[string]string, v any) error {
	if !mcpToolAllows(tool, op) {
		return fmt.Errorf("the tool %s may not reach %s", tool, op)
	}
	res, err := a.query(ctx, b, op, args, v)
	if err != nil {
		return err
	}
	if !res.OK() {
		msg := strings.TrimSpace(res.Stderr)
		if msg == "" {
			msg = strings.TrimSpace(res.Stdout)
		}
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return fmt.Errorf("%s failed: %s", op, msg)
	}
	return nil
}

// hasCapability: the Bootstrap declares the capability and the operations it names.
func hasCapability(ctx context.Context, b ports.Bootstrap, id string, ops ...string) bool {
	caps, err := b.Capabilities(ctx)
	if err != nil {
		return false
	}
	found := false
	for _, c := range caps {
		found = found || c.ID == id
	}
	if !found {
		return false
	}
	for _, op := range ops {
		if !supports(ctx, b, op) {
			return false
		}
	}
	return true
}

type mcpSnap = map[string]any

// mcpRepo is one repo opened for a tool: its Bootstrap, and what it said.
type mcpRepo struct {
	path    string
	name    string
	version string
	lang    string
	b       ports.Bootstrap
	snap    mcpSnap
	// problem is why the repo cannot answer; it is data for the tool, never a failure.
	problem map[string]any
	line    string
}

// mcpOpen probes a repo and, when it offers Sonar, reads its snapshot (when withSnap).
func (a *App) mcpOpen(ctx context.Context, tool, path string, withSnap bool) *mcpRepo {
	r := &mcpRepo{path: path, name: filepath.Base(path), lang: langOf(path)}
	row := a.sonarProbe(ctx, path)
	r.version = row.Version
	switch row.State {
	case "ok":
	case "update":
		r.fail(tool, "needs", MCPNeedsFindings, tr(r.lang,
			fmt.Sprintf("%s: its Bootstrap %s has no Sonar snapshot; this needs %s or newer. Run `gtt update` there.", r.name, orUnknown(row.Version), MCPNeedsFindings),
			fmt.Sprintf("%s: su Bootstrap %s no tiene el snapshot de Sonar; hace falta la %s o una más nueva. Corre `gtt update` ahí.", r.name, orUnknown(row.Version), MCPNeedsFindings)))
		return r
	default:
		r.fail(tool, "unavailable", "", fmt.Sprintf("%s: %s.", r.name, row.Reason))
		return r
	}
	r.b = row.b
	if withSnap {
		var snap mcpSnap
		if err := a.mcpRun(ctx, r.b, tool, OpSonarSnapshot, nil, &snap); err != nil || snap == nil {
			r.fail(tool, "unreadable", "", fmt.Sprintf("%s: its snapshot could not be read.", r.name))
			return r
		}
		r.snap = snap
		if l, _ := snap["lang"].(string); l != "" {
			r.lang = strings.ToLower(l)
		}
	}
	return r
}

func (r *mcpRepo) fail(_, state, needs, line string) {
	r.problem = map[string]any{"repo": r.name, "path": r.path, "state": state, "bootstrap": r.version}
	if needs != "" {
		r.problem["needs_bootstrap"] = needs
	}
	r.problem["message"] = line
	r.line = line
}

// need answers a repo whose Bootstrap lacks a capability: a normal result that says which release is needed.
func (r *mcpRepo) need(release, what string) *mcpRepo {
	r.fail("", "needs", release, tr(r.lang,
		fmt.Sprintf("%s: its Bootstrap %s has no %s; this needs %s or newer. Run `gtt update` there.", r.name, orUnknown(r.version), what, release),
		fmt.Sprintf("%s: su Bootstrap %s no tiene %s; hace falta la %s o una más nueva. Corre `gtt update` ahí.", r.name, orUnknown(r.version), what, release)))
	return r
}

func problemAnswer(r *mcpRepo) MCPAnswer {
	return MCPAnswer{Summary: r.line, Data: map[string]any{"supported": false, "repo": r.name, "state": r.problem["state"],
		"needs_bootstrap": r.problem["needs_bootstrap"], "bootstrap": r.version, "message": r.line}}
}

// ---------------------------------------------------------------------------------------------- helpers

func mm(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func ms(v any) []map[string]any {
	var out []map[string]any
	if l, ok := v.([]any); ok {
		for _, x := range l {
			if m, ok := x.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	return out
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// pick copies the named fields that exist.
func pick(m map[string]any, keys ...string) map[string]any {
	out := map[string]any{}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	return out
}

// page cuts a list to one page of at most 50 and says where it stands.
func page[T any](items []T, a MCPArgs) ([]T, map[string]any) {
	off, lim := a.Offset, a.Limit
	if off < 0 {
		off = 0
	}
	if lim <= 0 || lim > MCPPageMax {
		lim = MCPPageMax
	}
	if off > len(items) {
		off = len(items)
	}
	end := off + lim
	if end > len(items) {
		end = len(items)
	}
	return items[off:end], map[string]any{"total": len(items), "offset": off, "limit": lim, "more": end < len(items)}
}

func (a MCPArgs) args(keys map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range keys {
		if v != "" {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func oneOf(v string, allowed ...string) bool {
	if v == "" {
		return true
	}
	for _, x := range allowed {
		if strings.EqualFold(v, x) {
			return true
		}
	}
	return false
}

// MCPCall answers one tool. It does not fail for what a repo cannot give (an older Bootstrap, an unreadable
// snapshot): that is a normal answer that says so. An error is a misuse of the tool.
func (a *App) MCPCall(ctx context.Context, tool string, args MCPArgs) (MCPAnswer, error) {
	switch tool {
	case ToolProjects:
		return a.mcpProjects(ctx, args), nil
	case ToolFindings:
		return a.mcpFindings(ctx, args)
	case ToolReady:
		return a.mcpReady(ctx, args)
	case ToolNeedsYou:
		return a.mcpNeedsYou(ctx, args), nil
	case ToolEpic:
		return a.mcpEpic(ctx, args), nil
	case ToolActivity:
		return a.mcpActivity(ctx, args), nil
	case ToolCode:
		return a.mcpCode(ctx, args)
	}
	return MCPAnswer{}, fmt.Errorf("unknown tool %q", tool)
}

// one resolves the single repo a tool answers about, or the answer that says why not.
func (a *App) one(ctx context.Context, tool string, args MCPArgs, withSnap bool) (*mcpRepo, *MCPAnswer) {
	paths, msg := a.mcpPick(ctx, args.Repo, false)
	if msg != "" {
		return nil, &MCPAnswer{Summary: msg, Data: map[string]any{"supported": true, "message": msg}}
	}
	r := a.mcpOpen(ctx, tool, paths[0], withSnap)
	if r.problem != nil {
		ans := problemAnswer(r)
		return nil, &ans
	}
	return r, nil
}

// ---------------------------------------------------------------------------------------------- gtt_projects

func (a *App) mcpProjects(ctx context.Context, args MCPArgs) MCPAnswer {
	port := a.mcpPortfolio(ctx)
	if len(port) == 0 {
		msg := "No GTT project is known here. Start `gtt mcp` inside a project, or add repos with `gtt sonar add <folder>`."
		return MCPAnswer{Summary: msg, Data: map[string]any{"projects": []any{}, "message": msg}}
	}
	rows := make([]map[string]any, 0, len(port))
	lang, blocked, mine := "", 0, 0
	for _, p := range port {
		r := a.mcpOpen(ctx, ToolProjects, p, true)
		row := map[string]any{"name": r.name, "path": r.path, "bootstrap": r.version}
		if r.problem != nil {
			row["state"] = r.problem["state"]
			row["message"] = r.problem["message"]
			if n, ok := r.problem["needs_bootstrap"]; ok {
				row["needs_bootstrap"] = n
			}
			rows = append(rows, row)
			continue
		}
		if lang == "" {
			lang = r.lang
		}
		row["state"] = "ok"
		row["lang"] = r.lang
		row["phase"] = mm(r.snap["project"])["phase"]
		row["suite_version"] = r.snap["suite_version"]
		row["counts"] = mm(r.snap["readiness"])["counts"]
		sum := mm(r.snap["summary"])
		row["epics"] = sum["epics"]
		row["stories"] = map[string]any{"done": sum["stories_done"], "total": sum["stories_total"], "blocked": sum["stories_blocked"], "in_progress": sum["in_progress"]}
		row["progress_pct"] = sum["progress_pct"]
		c := mm(mm(r.snap["readiness"])["counts"])
		if f, ok := c["blocked"].(float64); ok {
			blocked += int(f)
		}
		if f, ok := c["mine"].(float64); ok {
			mine += int(f)
		}
		rows = append(rows, row)
	}
	if lang == "" {
		lang = "en"
	}
	cut, pg := page(rows, args)
	return MCPAnswer{
		Summary: tr(lang, fmt.Sprintf("%d projects; Sonar counts %d as yours, %d stories blocked.", len(rows), mine, blocked),
			fmt.Sprintf("%d proyectos; Sonar cuenta %d como tuyas, %d historias bloqueadas.", len(rows), mine, blocked)),
		Data: map[string]any{"projects": cut, "page": pg},
	}
}

// ---------------------------------------------------------------------------------------------- gtt_findings

type findingsList struct {
	Findings  []map[string]any `json:"findings"`
	Readiness map[string]any   `json:"readiness"`
	Counts    map[string]any   `json:"counts"`
}

func findingFields(f map[string]any) map[string]any {
	return pick(f, "code", "plane", "level", "ref", "epic", "where", "owner", "for", "source", "text", "fix")
}

func (a *App) findingsFor(ctx context.Context, tool string, r *mcpRepo, filters map[string]string) (*findingsList, error) {
	if !hasCapability(ctx, r.b, findingsCap, OpFindingsList) {
		return nil, nil
	}
	var fl findingsList
	if err := a.mcpRun(ctx, r.b, tool, OpFindingsList, filters, &fl); err != nil {
		return nil, err
	}
	return &fl, nil
}

func (a *App) mcpFindings(ctx context.Context, args MCPArgs) (MCPAnswer, error) {
	if !oneOf(args.Plane, "definition", "governance", "delivery", "code") {
		return MCPAnswer{}, errors.New("plane must be definition, governance, delivery or code")
	}
	if !oneOf(args.Owner, "human", "agent") {
		return MCPAnswer{}, errors.New("owner must be human or agent")
	}
	if !oneOf(args.Level, "notice", "warning", "blocking") {
		return MCPAnswer{}, errors.New("level must be notice, warning or blocking")
	}
	r, bad := a.one(ctx, ToolFindings, args, false)
	if bad != nil {
		return *bad, nil
	}
	fl, err := a.findingsFor(ctx, ToolFindings, r, args.args(map[string]string{"plane": strings.ToLower(args.Plane), "owner": strings.ToLower(args.Owner),
		"level": strings.ToLower(args.Level), "ref": args.Ref}))
	if err != nil {
		r.fail(ToolFindings, "unreadable", "", fmt.Sprintf("%s: its findings could not be read.", r.name))
		return problemAnswer(r), nil
	}
	if fl == nil {
		return problemAnswer(r.need(MCPNeedsFindings, "findings")), nil
	}
	items := make([]map[string]any, 0, len(fl.Findings))
	human, agent, blocking := 0, 0, 0
	for _, f := range fl.Findings {
		items = append(items, findingFields(f))
		switch str(f["owner"]) {
		case "human":
			human++
		case "agent":
			agent++
		}
		if strings.EqualFold(str(f["level"]), "blocking") {
			blocking++
		}
	}
	cut, pg := page(items, args)
	return MCPAnswer{
		Summary: tr(r.lang, fmt.Sprintf("%s: %d findings (%d up to you, %d to the agent, %d blocking).", r.name, len(items), human, agent, blocking),
			fmt.Sprintf("%s: %d hallazgos (%d te tocan a ti, %d al agente, %d bloqueantes).", r.name, len(items), human, agent, blocking)),
		Data: map[string]any{"repo": r.name, "findings": cut, "page": pg, "counts": fl.Counts},
	}, nil
}

// ---------------------------------------------------------------------------------------------- gtt_ready

func (a *App) mcpReady(ctx context.Context, args MCPArgs) (MCPAnswer, error) {
	if args.Ref == "" {
		return MCPAnswer{}, errors.New("ref is required: an Epic or a Story, for example EPIC-001 or STORY-003")
	}
	r, bad := a.one(ctx, ToolReady, args, false)
	if bad != nil {
		return *bad, nil
	}
	fl, err := a.findingsFor(ctx, ToolReady, r, map[string]string{"ref": args.Ref})
	if err != nil {
		r.fail(ToolReady, "unreadable", "", fmt.Sprintf("%s: its findings could not be read.", r.name))
		return problemAnswer(r), nil
	}
	if fl == nil {
		return problemAnswer(r.need(MCPNeedsFindings, "findings")), nil
	}
	own := mm(fl.Readiness[args.Ref])
	if own == nil && len(fl.Findings) == 0 {
		msg := tr(r.lang, fmt.Sprintf("%s: no %s in this backlog.", r.name, args.Ref), fmt.Sprintf("%s: no hay %s en este backlog.", r.name, args.Ref))
		return MCPAnswer{Summary: msg, Data: map[string]any{"repo": r.name, "ref": args.Ref, "found": false}}, nil
	}
	lacks := make([]map[string]any, 0, len(fl.Findings))
	human, agent := 0, 0
	for _, f := range fl.Findings {
		lacks = append(lacks, findingFields(f))
		switch str(f["owner"]) {
		case "human":
			human++
		case "agent":
			agent++
		}
	}
	data := map[string]any{"repo": r.name, "ref": args.Ref, "found": own != nil, "situation": own["situation"], "kind": own["kind"],
		"status": own["status"], "up_to": map[string]any{"human": human, "agent": agent}}
	if str(own["kind"]) == "epic" {
		var stories []map[string]any
		for ref, v := range fl.Readiness {
			if s := mm(v); str(s["kind"]) == "story" && str(s["epic"]) == args.Ref {
				stories = append(stories, map[string]any{"ref": ref, "status": s["status"], "situation": s["situation"]})
			}
		}
		sort.Slice(stories, func(i, j int) bool { return str(stories[i]["ref"]) < str(stories[j]["ref"]) })
		cut, pg := page(stories, args)
		data["stories"], data["stories_page"] = cut, pg
	}
	cut, pg := page(lacks, args)
	data["lacks"], data["page"] = cut, pg
	sit := str(own["situation"])
	return MCPAnswer{
		Summary: tr(r.lang, fmt.Sprintf("%s %s: %s; %d findings (%d up to you, %d to the agent).", r.name, args.Ref, orUnknown(sit), len(lacks), human, agent),
			fmt.Sprintf("%s %s: %s; %d hallazgos (%d te tocan a ti, %d al agente).", r.name, args.Ref, orUnknown(sit), len(lacks), human, agent)),
		Data: data,
	}, nil
}

// ---------------------------------------------------------------------------------------------- gtt_needs_you

func (a *App) mcpNeedsYou(ctx context.Context, args MCPArgs) MCPAnswer {
	paths, msg := a.mcpPick(ctx, args.Repo, true)
	if msg != "" {
		return MCPAnswer{Summary: msg, Data: map[string]any{"items": []any{}, "message": msg}}
	}
	var items []map[string]any
	var notes []map[string]any
	lang := ""
	for _, p := range paths {
		r := a.mcpOpen(ctx, ToolNeedsYou, p, true)
		if r.problem != nil {
			notes = append(notes, r.problem)
			continue
		}
		if lang == "" {
			lang = r.lang
		}
		for _, n := range ms(r.snap["needs_you"]) {
			it := pick(n, "kind", "ref", "title", "how", "command", "choices", "unblocks", "weight")
			it["repo"] = r.name
			it["from"] = "needs_you"
			items = append(items, it)
		}
		for _, f := range ms(r.snap["findings"]) {
			plane := str(f["plane"])
			if str(f["owner"]) != "human" || (plane != "definition" && plane != "governance") {
				continue
			}
			it := pick(f, "code", "plane", "level", "ref", "epic", "where", "for", "text", "fix")
			it["repo"] = r.name
			it["from"] = "finding"
			items = append(items, it)
		}
	}
	if lang == "" {
		lang = "en"
	}
	cut, pg := page(items, args)
	data := map[string]any{"items": cut, "page": pg}
	if len(notes) > 0 {
		data["notes"] = notes
	}
	line := tr(lang, fmt.Sprintf("%d things need you in %d repos.", len(items), len(paths)-len(notes)),
		fmt.Sprintf("%d cosas te necesitan en %d repos.", len(items), len(paths)-len(notes)))
	if len(notes) > 0 {
		line += " " + str(notes[0]["message"])
		if len(notes) > 1 {
			line += tr(lang, fmt.Sprintf(" (+%d more repos not read)", len(notes)-1), fmt.Sprintf(" (+%d repos más sin leer)", len(notes)-1))
		}
	}
	return MCPAnswer{Summary: line, Data: data}
}

// ---------------------------------------------------------------------------------------------- gtt_epic

func (a *App) mcpEpic(ctx context.Context, args MCPArgs) MCPAnswer {
	r, bad := a.one(ctx, ToolEpic, args, true)
	if bad != nil {
		return *bad
	}
	epicFields := []string{"ref", "title", "raw_status", "derived", "ready", "pct", "estimate_h", "active_h", "real_h", "start", "end", "eta", "eta_range", "owner", "lane", "group", "flags", "codes", "stories"}
	epics := ms(r.snap["epics"])
	if args.Epic == "" {
		rows := make([]map[string]any, 0, len(epics))
		for _, e := range epics {
			rows = append(rows, pick(e, epicFields...))
		}
		cut, pg := page(rows, args)
		return MCPAnswer{
			Summary: tr(r.lang, fmt.Sprintf("%s: %d Epics.", r.name, len(rows)), fmt.Sprintf("%s: %d Épicas.", r.name, len(rows))),
			Data:    map[string]any{"repo": r.name, "epics": cut, "page": pg},
		}
	}
	for _, e := range epics {
		if !strings.EqualFold(str(e["ref"]), args.Epic) {
			continue
		}
		ref := str(e["ref"])
		var rows []map[string]any
		for _, s := range ms(r.snap["stories"]) {
			if str(s["epic"]) == ref {
				rows = append(rows, pick(s, "ref", "title", "status", "ready", "estimate_h", "active_h", "real_h", "elapsed_h", "owner", "blocked_by", "next", "closed"))
			}
		}
		cut, pg := page(rows, args)
		est, act := e["estimate_h"], e["active_h"]
		when := e["eta"]
		if rng, ok := e["eta_range"].([]any); ok && len(rng) > 1 {
			when = fmt.Sprintf("%v to %v", rng[0], rng[len(rng)-1])
		}
		pct, _ := e["pct"].(float64)
		return MCPAnswer{
			Summary: tr(r.lang, fmt.Sprintf("%s %s: %s, %d%%, %vh estimated, %vh active, forecast %v, %d stories.", r.name, ref, orUnknown(str(e["raw_status"])), int(pct), est, act, orNone(when), len(rows)),
				fmt.Sprintf("%s %s: %s, %d%%, %vh estimadas, %vh activas, pronóstico %v, %d historias.", r.name, ref, orUnknown(str(e["raw_status"])), int(pct), est, act, orNone(when), len(rows))),
			Data: map[string]any{"repo": r.name, "epic": pick(e, epicFields...), "forecast": when, "stories": cut, "page": pg},
		}
	}
	var refs []string
	for _, e := range epics {
		refs = append(refs, str(e["ref"]))
	}
	msg := tr(r.lang, fmt.Sprintf("%s: no Epic %s. Epics: %s.", r.name, args.Epic, strings.Join(refs, ", ")),
		fmt.Sprintf("%s: no hay la Épica %s. Épicas: %s.", r.name, args.Epic, strings.Join(refs, ", ")))
	return MCPAnswer{Summary: msg, Data: map[string]any{"repo": r.name, "found": false, "epics": refs}}
}

func orNone(v any) any {
	if v == nil {
		return "-"
	}
	return v
}

// ---------------------------------------------------------------------------------------------- gtt_activity

func (a *App) mcpActivity(ctx context.Context, args MCPArgs) MCPAnswer {
	r, bad := a.one(ctx, ToolActivity, args, true)
	if bad != nil {
		return *bad
	}
	act := mm(r.snap["activity"])
	last := ms(act["last"])
	cut, pg := page(last, args)
	now := mm(act["now"])
	data := map[string]any{"repo": r.name, "now": act["now"], "working_now": r.snap["working_now"], "last": cut, "page": pg,
		"journal": act["journal"], "sources": act["sources"], "runs": act["runs"]}
	var line string
	if story := str(now["story"]); story != "" {
		line = tr(r.lang, fmt.Sprintf("%s: working on %s (%vh active of %vh estimated); %d recent events.", r.name, story, orNone(now["active_h"]), orNone(now["estimate_h"]), len(last)),
			fmt.Sprintf("%s: trabajando en %s (%vh activas de %vh estimadas); %d eventos recientes.", r.name, story, orNone(now["active_h"]), orNone(now["estimate_h"]), len(last)))
	} else {
		line = tr(r.lang, fmt.Sprintf("%s: nothing in progress right now; %d recent events.", r.name, len(last)),
			fmt.Sprintf("%s: nada en curso ahora; %d eventos recientes.", r.name, len(last)))
	}
	return MCPAnswer{Summary: line, Data: data}
}

// ---------------------------------------------------------------------------------------------- gtt_code

func (a *App) mcpCode(ctx context.Context, args MCPArgs) (MCPAnswer, error) {
	r, bad := a.one(ctx, ToolCode, args, false)
	if bad != nil {
		return *bad, nil
	}
	if !hasCapability(ctx, r.b, codeCap, OpCodeMap) {
		return problemAnswer(r.need(MCPNeedsCode, "code map")), nil
	}
	var cm map[string]any
	if err := a.mcpRun(ctx, r.b, ToolCode, OpCodeMap, args.args(map[string]string{"epic": args.Epic, "story": args.Story}), &cm); err != nil || cm == nil {
		r.fail(ToolCode, "unreadable", "", fmt.Sprintf("%s: its code map could not be read.", r.name))
		return problemAnswer(r), nil
	}
	var mods []map[string]any
	for _, m := range ms(cm["modules"]) {
		mods = append(mods, pick(m, "id", "path", "layer", "files", "new", "changed", "touched", "stories", "may", "must_not", "empty"))
	}
	var bannedEdges []map[string]any
	for _, e := range ms(cm["edges"]) {
		if ok, has := e["allowed"].(bool); has && !ok {
			bannedEdges = append(bannedEdges, pick(e, "from", "to", "count", "example"))
		}
	}
	cutMods, modPg := page(mods, args)
	cutEdges, edgePg := page(bannedEdges, args)
	d2c := mm(cm["design_to_code"])
	if args.Epic != "" {
		only := map[string]any{}
		for k, v := range d2c {
			if strings.EqualFold(k, args.Epic) {
				only[k] = v
			}
		}
		d2c = only
	}
	var epicKeys []string
	for k := range d2c {
		epicKeys = append(epicKeys, k)
	}
	sort.Strings(epicKeys)
	design := map[string]any{}
	built, total := 0, 0
	for _, k := range epicKeys {
		items := ms(d2c[k])
		rows := make([]map[string]any, 0, len(items))
		for _, it := range items {
			rows = append(rows, pick(it, "item", "text", "stories", "files", "evidence", "closed", "built"))
			total++
			if b, _ := it["built"].(bool); b {
				built++
			}
		}
		cut, pg := page(rows, args)
		design[k] = map[string]any{"items": cut, "page": pg}
	}
	data := map[string]any{"repo": r.name, "at": cm["at"], "modules": cutMods, "modules_page": modPg,
		"disallowed_dependencies": cutEdges, "disallowed_page": edgePg, "outside": cm["outside"], "design_to_code": design}
	return MCPAnswer{
		Summary: tr(r.lang, fmt.Sprintf("%s: %d modules, %d dependencies not allowed, %d of %d design items built.", r.name, len(mods), len(bannedEdges), built, total),
			fmt.Sprintf("%s: %d módulos, %d dependencias no permitidas, %d de %d ítems del diseño construidos.", r.name, len(mods), len(bannedEdges), built, total)),
		Data: data,
	}, nil
}

// ---------------------------------------------------------------------------------------------- the rule

// MCPTableProblems walks a table of tools and operations against what the Bootstrap declares and returns what
// breaks the rule of the order (12.1): a tool may reach no operation that has human authority, none that
// mutates, and no ratification verb. An operation the Bootstrap does not declare is a problem too, unless the
// caller says declarations are partial (an older Bootstrap).
func MCPTableProblems(table map[string][]string, decl map[string]ports.Operation, partial bool) []string {
	var out []string
	tools := make([]string, 0, len(table))
	for t := range table {
		tools = append(tools, t)
	}
	sort.Strings(tools)
	for _, tool := range tools {
		if len(table[tool]) == 0 {
			out = append(out, tool+" reaches no operation")
		}
		for _, op := range table[tool] {
			d, ok := decl[op]
			switch {
			case RatificationVerb(op) || strings.HasPrefix(op, "approve") || strings.HasPrefix(op, "promote") || strings.HasPrefix(op, "undo") ||
				strings.HasPrefix(op, "freeze") || strings.HasPrefix(op, "run.execute") || strings.HasPrefix(op, "adr.adopt"):
				out = append(out, fmt.Sprintf("%s reaches %s, a ratification verb", tool, op))
			case !ok && !partial:
				out = append(out, fmt.Sprintf("%s reaches %s, which the Bootstrap does not declare", tool, op))
			case ok && d.HumanAuthority:
				out = append(out, fmt.Sprintf("%s reaches %s, which has human_authority", tool, op))
			case ok && d.Mutates:
				out = append(out, fmt.Sprintf("%s reaches %s, which mutates", tool, op))
			}
		}
	}
	return out
}

// MCPToolTable is a copy of the table, for the tests that walk it.
func MCPToolTable() map[string][]string {
	out := map[string][]string{}
	for k, v := range mcpToolOps {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// ReadOperationsFile reads the declared operations of a Bootstrap's operations.json.
func ReadOperationsFile(path string) (map[string]ports.Operation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Operations map[string]ports.Operation `json:"operations"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc.Operations, nil
}
