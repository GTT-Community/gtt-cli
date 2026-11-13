package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/execution"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

func archiveOf(t *testing.T, script string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "gtt", Mode: 0o755, Size: int64(len(script)), Typeflag: tar.TypeReg})
	tw.Write([]byte(script))
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func release(t *testing.T, version, sumOverride string) (*httptest.Server, []byte) {
	t.Helper()
	arc := archiveOf(t, "#!/bin/sh\necho \"GTT CLI       "+version+"\"\n")
	sum := sha256.Sum256(arc)
	hexSum := hex.EncodeToString(sum[:])
	if sumOverride != "" {
		hexSum = sumOverride
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/gtt_linux_amd64.tar.gz", func(w http.ResponseWriter, _ *http.Request) { w.Write(arc) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(hexSum + "  gtt_linux_amd64.tar.gz\n"))
	})
	mux.HandleFunc("/VERSION", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(version + "\n")) })
	return httptest.NewServer(mux), arc
}

// install puts an "old gtt" in a fresh directory and returns an Upgrader bound to it and the mirror.
func install(t *testing.T, srv *httptest.Server) (*Upgrader, string) {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "gtt")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\necho \"GTT CLI       1.0.0\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &Upgrader{Runner: execution.Runner{}, Base: srv.URL, Exe: exe, GOOS: "linux", GOARCH: "amd64", GoBins: []string{}}, exe
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// AC-2: the replacement works against a local mirror and leaves gtt.old.
func TestReplacesTheBinaryFromAMirrorAndKeepsTheOldOne(t *testing.T) {
	srv, _ := release(t, "2.0.0", "")
	defer srv.Close()
	u, exe := install(t, srv)
	plan := u.Plan("2.0.0")
	if plan.Manual || plan.Channel != "direct" {
		t.Fatalf("a writable direct install is replaced by the CLI: %+v", plan)
	}
	if v, err := u.Latest(context.Background()); err != nil || v != "2.0.0" {
		t.Fatalf("the mirror says its release: %q %v", v, err)
	}
	if err := u.Apply(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read(t, exe), "2.0.0") || !strings.Contains(read(t, exe+".old"), "1.0.0") {
		t.Fatalf("new binary in place, old one kept as gtt.old")
	}
	if info, _ := os.Stat(exe); info.Mode()&0o111 == 0 {
		t.Fatal("the new binary must be executable")
	}
}

// AC-3: a checksum that does not match aborts before the binary is touched.
func TestAChecksumMismatchChangesNothing(t *testing.T) {
	srv, _ := release(t, "2.0.0", strings.Repeat("0", 64))
	defer srv.Close()
	u, exe := install(t, srv)
	before := read(t, exe)
	err := u.Apply(context.Background(), u.Plan("2.0.0"))
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("a bad checksum must be refused: %v", err)
	}
	if read(t, exe) != before {
		t.Fatal("the binary changed")
	}
	if _, err := os.Stat(exe + ".old"); err == nil {
		t.Fatal("no gtt.old should exist when nothing was replaced")
	}
	if entries, _ := os.ReadDir(filepath.Dir(exe)); len(entries) != 1 {
		t.Fatalf("no temporary file may stay behind: %v", entries)
	}
}

// AC-4: a new binary that does not answer as the target is rolled back.
func TestANewBinaryThatDoesNotStartIsRolledBack(t *testing.T) {
	srv, _ := release(t, "9.9.9", "") // the archive says 9.9.9, we ask for 2.0.0
	defer srv.Close()
	u, exe := install(t, srv)
	before := read(t, exe)
	err := u.Apply(context.Background(), u.Plan("2.0.0"))
	if err == nil || !strings.Contains(err.Error(), "restored") {
		t.Fatalf("expected a restore: %v", err)
	}
	if read(t, exe) != before {
		t.Fatal("the previous binary must be back")
	}
}

// AC-1: planning and asking for the latest write nothing.
func TestPlanAndLatestWriteNothing(t *testing.T) {
	srv, _ := release(t, "2.0.0", "")
	defer srv.Close()
	u, exe := install(t, srv)
	dir := filepath.Dir(exe)
	before, _ := os.ReadDir(dir)
	u.Latest(context.Background())
	u.Plan("2.0.0")
	after, _ := os.ReadDir(dir)
	if len(before) != len(after) || read(t, exe) != "#!/bin/sh\necho \"GTT CLI       1.0.0\"\n" {
		t.Fatal("--check must leave the binary and its directory identical")
	}
}

// AC-5: each channel picks the right action.
func TestEachChannelPicksItsAction(t *testing.T) {
	bins := []string{"/home/u/go/bin"}
	cases := map[string]string{
		"/usr/lib/node_modules/@gtt-community/gtt-cli/vendor/gtt":   "npm",
		"/home/u/.local/share/uv/tools/gtt-cli/bin/gtt":             "uv",
		"/home/u/.local/pipx/venvs/gtt-cli/bin/gtt":                 "pipx",
		"/home/u/go/bin/gtt":                                        "go",
		"/usr/local/bin/gtt":                                        "direct",
		"/home/u/.nvm/versions/node/v20/lib/node_modules/other/gtt": "direct",
	}
	for exe, want := range cases {
		if got := Channel(exe, bins); got != want {
			t.Errorf("%s: %s, want %s", exe, got, want)
		}
	}
	u := &Upgrader{Runner: noTools{}, Exe: "/usr/lib/node_modules/@gtt-community/gtt-cli/vendor/gtt", GoBins: []string{}}
	if p := u.Plan("2.0.0"); !p.Manual || !strings.Contains(p.Hint, "npm install -g @gtt-community/gtt-cli@2.0.0") {
		t.Errorf("without npm the exact command is printed: %+v", p)
	}
	u = &Upgrader{Runner: hasTools{}, Exe: "/usr/lib/node_modules/@gtt-community/gtt-cli/vendor/gtt", GoBins: []string{}}
	if p := u.Plan("2.0.0"); p.Manual || strings.Join(p.Command, " ") != "npm install -g @gtt-community/gtt-cli@2.0.0" {
		t.Errorf("npm channel: %+v", p)
	}
	u.Exe = "/home/u/.local/share/uv/tools/gtt-cli/bin/gtt"
	if p := u.Plan("2.0.0"); strings.Join(p.Command, " ") != "uv tool install --force gtt-cli==2.0.0" {
		t.Errorf("uv channel: %+v", p)
	}
	u.Exe = "/home/u/go/bin/gtt"
	u.GoBins = bins
	if p := u.Plan("2.0.0"); !p.Manual || p.Hint != "go install github.com/GTT-Community/gtt-cli/cmd/gtt@v2.0.0" {
		t.Errorf("go channel is printed, never run: %+v", p)
	}
}

func TestWithoutWritePermissionTheCommandIsPrintedAndNothingChanges(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes anywhere")
	}
	srv, _ := release(t, "2.0.0", "")
	defer srv.Close()
	u, exe := install(t, srv)
	dir := filepath.Dir(exe)
	os.Chmod(dir, 0o555)
	defer os.Chmod(dir, 0o755)
	p := u.Plan("2.0.0")
	if !p.Manual || !strings.Contains(p.Hint, "GTT_VERSION=2.0.0") {
		t.Fatalf("expected the installer command: %+v", p)
	}
}

func TestOnlyHTTPSOrLocalMirrors(t *testing.T) {
	for raw, ok := range map[string]bool{
		"https://github.com/x": true, "http://127.0.0.1:8080/a": true, "http://localhost/a": true,
		"http://example.com/a": false, "file:///tmp/x": false, "ftp://x/y": false,
	} {
		if (checkURL(raw) == nil) != ok {
			t.Errorf("%s: allowed=%v, want %v", raw, !ok, ok)
		}
	}
}

type noTools struct{ execution.Runner }

func (noTools) LookPath(string) (string, error) { return "", os.ErrNotExist }

type hasTools struct{ execution.Runner }

func (hasTools) LookPath(n string) (string, error) { return "/usr/bin/" + n, nil }

var _ ports.Runner = noTools{}
