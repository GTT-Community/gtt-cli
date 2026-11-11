package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// GTT Sonar: `gtt sonar` generates the read-only page from the Bootstrap's own snapshot, `gtt sonar --all`
// builds the portfolio with each repo's own Bootstrap, and `gtt idea add` hands the text over exactly as given.
// A Bootstrap without sonar.v1 is a one-line answer, never a failure. A Bootstrap without it is skipped.

func sonarProject(t *testing.T) string {
	t.Helper()
	if !exists(pristine, ".gtt/sonar/template.html") {
		t.Skip("this Bootstrap does not bring Sonar")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}
	root := project(t, nil)
	initProject(t, root, pristine, defaults...)
	return root
}

// downgrade makes a project look like one installed with a Bootstrap that has no Sonar.
func downgrade(t *testing.T, root string) {
	t.Helper()
	for _, rel := range []string{".gtt/contract/operations.json", ".gtt/contract/capabilities.json"} {
		data := read(t, root, rel)
		var doc map[string]any
		if err := json.Unmarshal([]byte(data), &doc); err != nil {
			t.Fatal(err)
		}
		if ops, ok := doc["operations"].(map[string]any); ok {
			for k := range ops {
				if strings.HasPrefix(k, "sonar.") || strings.HasPrefix(k, "idea.") {
					delete(ops, k)
				}
			}
		}
		if caps, ok := doc["capabilities"].([]any); ok {
			kept := caps[:0]
			for _, c := range caps {
				if id := c.(map[string]any)["id"]; id != "sonar.v1" && id != "ideas" {
					kept = append(kept, c)
				}
			}
			doc["capabilities"] = kept
		}
		out, _ := json.MarshalIndent(doc, "", "  ")
		write(t, root, rel, string(out)+"\n")
	}
}

var generatedAt = regexp.MustCompile(`"generated_at":\s*"[^"]*"`)

func TestSonarGeneratesTheDerivedPage(t *testing.T) {
	root := sonarProject(t)
	res := gtt(t, root, "sonar", "--no-open", "--json").must(t, 0).json(t)
	page := filepath.Join(root, ".gtt", "local", "sonar.html")
	if res["supported"] != true || res["page"] != page {
		t.Fatalf("the page is generated and its path given: %v", res)
	}
	html := read(t, root, ".gtt/local/sonar.html")
	if !strings.Contains(html, `id="sonar-data"`) || !strings.Contains(html, "payments-api") || !strings.Contains(html, "default-src 'none'") {
		t.Errorf("a page with the snapshot embedded and its CSP:\n%.400s", html)
	}
	before := generatedAt.ReplaceAllString(read(t, root, ".gtt/local/sonar.json"), "")
	for _, f := range []string{"sonar.html", "sonar.json"} {
		os.Remove(filepath.Join(root, ".gtt", "local", f))
	}
	gtt(t, root, "sonar", "--no-open").must(t, 0)
	if after := generatedAt.ReplaceAllString(read(t, root, ".gtt/local/sonar.json"), ""); after != before {
		t.Errorf("deleting the derived files loses nothing: they come back the same")
	}
}

func TestSonarWithABootstrapWithoutItSaysToUpdateWhenTheNewerOneIsNotAvailable(t *testing.T) {
	root := sonarProject(t)
	downgrade(t, root)
	r := gtt(t, root, "sonar", "--no-open", "--offline").must(t, 0)
	if !strings.Contains(r.out, "gtt update") {
		t.Errorf("it says what to do:\n%s", r.out)
	}
	if exists(root, ".gtt/local/sonar.html") {
		t.Error("nothing is generated inside the project without the capability")
	}
}

func TestSonarReadsAnOlderProjectWithTheNewerBootstrapWithoutTouchingIt(t *testing.T) {
	root := sonarProject(t)
	downgrade(t, root)
	t.Setenv("GTT_HOME", filepath.Join(t.TempDir(), "gtt-home"))
	before := treeHash(t, root)
	res := gtt(t, root, "sonar", "--no-open", "--bootstrap", pristine, "--json").must(t, 0).json(t)
	if after := treeHash(t, root); after != before {
		t.Error("reading an older project changes nothing in it")
	}
	page, _ := res["page"].(string)
	html, err := os.ReadFile(page)
	if err != nil {
		t.Fatalf("a page is generated outside the project: %v %v", res, err)
	}
	if !strings.Contains(string(html), `"compat"`) || !strings.Contains(string(html), `"schema": "sonar.v1"`) {
		t.Error("the snapshot says it was read with a newer Bootstrap and what to run")
	}
}

func TestPortfolioShowsTheOldRepoAsUpdateAndTheOtherComplete(t *testing.T) {
	home := t.TempDir()
	a, b := sonarProject(t), sonarProject(t)
	downgrade(t, b)
	t.Setenv("GTT_HOME", filepath.Join(home, "gtt-home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	gtt(t, a, "sonar", "add", a).must(t, 0)
	gtt(t, a, "sonar", "add", a).must(t, 0) // adding twice is the same list
	gtt(t, a, "sonar", "add", b).must(t, 0)
	list := gtt(t, a, "sonar", "list", "--json").must(t, 0).json(t)
	repos := list["repos"].([]any)
	if len(repos) != 2 || repos[0].(map[string]any)["state"] != "ok" || repos[1].(map[string]any)["state"] != "update" {
		t.Fatalf("the list says which repo needs an update: %v", repos)
	}
	res := gtt(t, a, "sonar", "--all", "--no-open", "--json").must(t, 0).json(t)
	page := filepath.Join(home, "gtt-home", "sonar", "index.html")
	if res["page"] != page {
		t.Fatalf("the portfolio page is in the GTT home: %v", res["page"])
	}
	html, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	text := string(html)
	data := regexp.MustCompile(`(?s)id="sonar-data">(.*?)</script>`).FindStringSubmatch(text)
	if data == nil {
		t.Fatalf("the page embeds its data:\n%.400s", text)
	}
	var payload struct {
		Repos []map[string]any `json:"repos"`
	}
	if err := json.Unmarshal([]byte(data[1]), &payload); err != nil || len(payload.Repos) != 2 {
		t.Fatalf("two entries in the page: %v %d", err, len(payload.Repos))
	}
	snap, _ := payload.Repos[0]["snapshot"].(map[string]any)
	stale, _ := payload.Repos[1]["stale"].(map[string]any)
	if snap == nil || snap["schema"] != "sonar.v1" || stale == nil || !strings.Contains(stale["reason"].(string), "has no sonar.v1") {
		t.Errorf("the repo with Sonar is complete and the other is a row that says to update GTT there: %v", payload.Repos)
	}
	if _, err := os.Stat(filepath.Join(home, "gtt-home", "sonar", "fonts", "Geist-Variable.woff2")); err != nil {
		t.Error("the portfolio page carries its fonts next to it")
	}
	if exists(b, ".gtt/local/sonar.html") || exists(b, ".gtt/local/sonar.json") {
		t.Error("the old repo was not touched")
	}
	gtt(t, a, "sonar", "remove", b).must(t, 0)
	if n := len(gtt(t, a, "sonar", "list", "--json").must(t, 0).json(t)["repos"].([]any)); n != 1 {
		t.Errorf("remove takes the repo out of the list, %d left", n)
	}
	if gtt(t, a, "sonar", "remove", b).code == 0 {
		t.Error("removing what is not there is refused")
	}
}

func TestIdeaAddKeepsTheTextExactlyAndRefreshesThePage(t *testing.T) {
	root := sonarProject(t)
	text := "Sincronizar parcelas\nsin señal con 'comillas', $HOME y `date` ✓"
	gtt(t, root, "idea", "add", text).must(t, 0)
	body := read(t, root, "docs/ideas/IDEA-001.md")
	if !strings.HasSuffix(body, "\n\n"+text+"\n") || !strings.HasPrefix(body, "# IDEA-001 · Sincronizar parcelas") {
		t.Errorf("the idea is stored exactly as written:\n%s", body)
	}
	if !exists(root, ".gtt/local/sonar.json") || !strings.Contains(read(t, root, ".gtt/local/sonar.json"), "IDEA-001") {
		t.Error("the page is regenerated at the end of the act and shows the idea")
	}
	list := gtt(t, root, "idea", "list", "--json").must(t, 0).json(t)
	if list["new"] != float64(1) {
		t.Errorf("one new idea: %v", list)
	}
	gtt(t, root, "idea", "park", "IDEA-001").must(t, 0)
	gtt(t, root, "idea", "drop", "IDEA-001").must(t, 0)
	if gtt(t, root, "idea", "park", "IDEA-001").code == 0 {
		t.Error("a dropped idea is closed")
	}
	if gtt(t, root, "idea", "add", "   ").code == 0 {
		t.Error("an empty idea is refused")
	}
}

func TestIdeaWithABootstrapWithoutItAsksForAnUpdate(t *testing.T) {
	root := sonarProject(t)
	downgrade(t, root)
	r := gtt(t, root, "idea", "add", "x")
	if r.code == 0 || !strings.Contains(r.err, "gtt update") || exists(root, "docs/ideas") {
		t.Errorf("one clear line and nothing written: code %d\n%s", r.code, r.err)
	}
}

func TestSonarInAFolderOfProjectsShowsThemAllWithoutAList(t *testing.T) {
	home := t.TempDir()
	a, b := sonarProject(t), sonarProject(t)
	downgrade(t, b)
	t.Setenv("GTT_HOME", filepath.Join(home, "gtt-home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config")) // no portfolio list at all
	parent := t.TempDir()
	for name, src := range map[string]string{"alpha": a, "deep/beta": b} {
		dst := filepath.Join(parent, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(src, dst); err != nil {
			t.Fatal(err)
		}
	}
	os.MkdirAll(filepath.Join(parent, "node_modules", "pkg"), 0o755)
	res := gtt(t, parent, "sonar", "--no-open", "--json").must(t, 0).json(t)
	repos, _ := res["repos"].([]any)
	if res["page"] != filepath.Join(home, "gtt-home", "sonar", "index.html") || len(repos) != 2 {
		t.Fatalf("both projects are found under the folder, to two levels: %v", res)
	}
	html, _ := os.ReadFile(filepath.Join(home, "gtt-home", "sonar", "index.html"))
	text := string(html)
	if !strings.Contains(text, `"schema": "sonar.v1"`) || !strings.Contains(text, "gtt update") {
		t.Errorf("the project with Sonar is complete and the old one says how to update it")
	}
	// inside a project it is that project, not a scan
	res = gtt(t, filepath.Join(parent, "alpha"), "sonar", "--no-open", "--json").must(t, 0).json(t)
	if res["page"] != filepath.Join(parent, "alpha", ".gtt", "local", "sonar.html") {
		t.Errorf("inside a project it shows that project: %v", res["page"])
	}
	// a folder with nothing
	empty := gtt(t, t.TempDir(), "sonar", "--no-open").must(t, 0)
	if !strings.Contains(empty.out, "No GTT project") {
		t.Errorf("one line when there is nothing to show:\n%s", empty.out)
	}
}
