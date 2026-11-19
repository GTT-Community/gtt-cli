package upgrade

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/execution"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// The tests run REAL executables on the real platform (tar.gz and gtt on Unix, zip and gtt.exe on Windows): a
// shell script is not an executable on Windows, and pretending to be Linux would leave the Windows path untested.
const fakeMain = `package main

import (
	"fmt"
	"io"
	"os"
)

var v = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("GTT CLI       %s\n", v)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "wait" { // stays alive until its stdin closes: a running gtt
		io.Copy(io.Discard, os.Stdin)
		return
	}
	os.Exit(2)
}
`

var (
	fakeMu   sync.Mutex
	fakeBins = map[string][]byte{}
)

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// fakeGTT is a small real executable that answers `gtt version` with the given release; built once per version.
func fakeGTT(t *testing.T, version string) []byte {
	t.Helper()
	fakeMu.Lock()
	defer fakeMu.Unlock()
	if b, ok := fakeBins[version]; ok {
		return b
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fakegtt\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(fakeMain), 0o644); err != nil {
		t.Fatal(err)
	}
	goBin := filepath.Join(runtime.GOROOT(), "bin", "go"+exeSuffix())
	if _, err := os.Stat(goBin); err != nil {
		if goBin, err = exec.LookPath("go"); err != nil {
			t.Skip("the go tool is needed to build the stand-in gtt")
		}
	}
	out := filepath.Join(dir, "gtt"+exeSuffix())
	cmd := exec.Command(goBin, "build", "-ldflags", "-X main.v="+version, "-o", out, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOFLAGS=", "GOTOOLCHAIN=local")
	if msg, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the stand-in gtt %s: %v\n%s", version, err, msg)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	fakeBins[version] = b
	return b
}

// archiveFile is the asset name the upgrader asks for on this machine.
func archiveFile() string { return (&Upgrader{}).archiveName() }

func archiveOf(t *testing.T, bin []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	if runtime.GOOS == "windows" {
		zw := zip.NewWriter(&buf)
		w, err := zw.Create("gtt.exe")
		if err != nil {
			t.Fatal(err)
		}
		w.Write(bin)
		zw.Close()
		return buf.Bytes()
	}
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "gtt", Mode: 0o755, Size: int64(len(bin)), Typeflag: tar.TypeReg})
	tw.Write(bin)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func release(t *testing.T, version, sumOverride string) (*httptest.Server, []byte) {
	t.Helper()
	arc := archiveOf(t, fakeGTT(t, version))
	sum := sha256.Sum256(arc)
	hexSum := hex.EncodeToString(sum[:])
	if sumOverride != "" {
		hexSum = sumOverride
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/"+archiveFile(), func(w http.ResponseWriter, _ *http.Request) { w.Write(arc) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(hexSum + "  " + archiveFile() + "\n"))
	})
	mux.HandleFunc("/VERSION", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(version + "\n")) })
	return httptest.NewServer(mux), arc
}

// install puts an "old gtt" (a real 1.0.0 executable) in a fresh directory and returns an Upgrader bound to it and the mirror.
func install(t *testing.T, srv *httptest.Server) (*Upgrader, string) {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "gtt"+exeSuffix())
	if err := os.WriteFile(exe, fakeGTT(t, "1.0.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &Upgrader{Runner: execution.Runner{}, Base: srv.URL, Exe: exe, GoBins: []string{}}, exe
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
	if read(t, exe) != string(fakeGTT(t, "2.0.0")) || read(t, exe+".old") != string(fakeGTT(t, "1.0.0")) {
		t.Fatalf("new binary in place, old one kept as gtt.old")
	}
	if info, _ := os.Stat(exe); runtime.GOOS != "windows" && info.Mode()&0o111 == 0 { // Windows has no execute bit
		t.Fatal("the new binary must be executable")
	}
}

// The CLI replaces the very binary it is running from. On Windows a running .exe can be renamed but not overwritten:
// this is the case `gtt upgrade` meets in real life, with the old gtt still alive while it is moved aside.
func TestReplacesTheBinaryWhileItIsRunning(t *testing.T) {
	srv, _ := release(t, "2.0.0", "")
	defer srv.Close()
	u, exe := install(t, srv)
	running := exec.Command(exe, "wait")
	stdin, err := running.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := running.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stdin.Close(); running.Wait() }) // runs before the temp dir is removed
	if err := u.Apply(context.Background(), u.Plan("2.0.0")); err != nil {
		t.Fatalf("a running gtt must be replaceable: %v", err)
	}
	if read(t, exe) != string(fakeGTT(t, "2.0.0")) {
		t.Fatal("the new binary is not in place")
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
	if len(before) != len(after) || read(t, exe) != string(fakeGTT(t, "1.0.0")) {
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
	if runtime.GOOS == "windows" {
		t.Skip("chmod does not take write permission away on Windows (it uses ACLs)")
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
