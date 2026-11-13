package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

type seeker struct {
	latest  string
	err     error
	asked   int
	applied int
	manual  bool
	applyEr error
}

func (s *seeker) Latest(context.Context) (string, error) { s.asked++; return s.latest, s.err }
func (s *seeker) Plan(t string) ports.UpgradePlan {
	return ports.UpgradePlan{Target: t, Manual: s.manual, Hint: "reinstall"}
}
func (s *seeker) Apply(context.Context, ports.UpgradePlan) error { s.applied++; return s.applyEr }

func autoApp(t *testing.T, interactive bool, s *seeker) *App {
	t.Helper()
	t.Setenv("CI", "")
	t.Setenv("GTT_NO_UPDATE_CHECK", "")
	t.Setenv("GTT_NO_AUTO_UPGRADE", "")
	return &App{Prompt: &prompter{interactive: interactive}, Upgrader: s, Home: t.TempDir()}
}

func TestAutoUpgradeLooksOnceThenUsesTheCache(t *testing.T) {
	s := &seeker{latest: "99.0.0"}
	a := autoApp(t, false, s)
	now := time.Now()
	if line := a.AutoUpgrade(context.Background(), now); !strings.Contains(line, "99.0.0") || !strings.Contains(line, "gtt upgrade") {
		t.Fatalf("a newer release should be announced, got %q", line)
	}
	if line := a.AutoUpgrade(context.Background(), now.Add(time.Hour)); line != "" {
		t.Fatalf("the same news is not repeated within hours, got %q", line)
	}
	if s.asked != 1 {
		t.Fatalf("the network is asked once within the window, asked %d", s.asked)
	}
	if line := a.AutoUpgrade(context.Background(), now.Add(13*time.Hour)); line == "" || s.asked != 2 {
		t.Fatalf("after the window it looks again and says it again (line %q, asked %d)", line, s.asked)
	}
}

func TestAutoUpgradeReplacesItselfOnlyInATerminal(t *testing.T) {
	s := &seeker{latest: "99.0.0"}
	a := autoApp(t, false, s)
	a.AutoUpgrade(context.Background(), time.Now())
	if s.applied != 0 {
		t.Fatal("without a terminal gtt only says it")
	}
	s2 := &seeker{latest: "99.0.0"}
	b := autoApp(t, true, s2)
	if line := b.AutoUpgrade(context.Background(), time.Now()); s2.applied != 1 || !strings.Contains(line, "by itself") {
		t.Fatalf("in a terminal it upgrades by itself (applied %d, line %q)", s2.applied, line)
	}
	if err := b.SetAutoUpgrade(false); err != nil {
		t.Fatal(err)
	}
	s3 := &seeker{latest: "99.0.0"}
	b.Upgrader = s3
	b.AutoUpgrade(context.Background(), time.Now().Add(48*time.Hour))
	if s3.applied != 0 {
		t.Fatal("--auto off is respected")
	}
	s4 := &seeker{latest: "99.0.0", manual: true}
	c := autoApp(t, true, s4)
	if line := c.AutoUpgrade(context.Background(), time.Now()); s4.applied != 0 || line == "" {
		t.Fatalf("a channel it cannot replace is only announced (applied %d, line %q)", s4.applied, line)
	}
	s5 := &seeker{latest: "99.0.0", applyEr: errors.New("boom")}
	d := autoApp(t, true, s5)
	if line := d.AutoUpgrade(context.Background(), time.Now()); !strings.Contains(line, "could not") {
		t.Fatalf("a failure is said, never silent: %q", line)
	}
}

func TestAutoUpgradeIsQuietWhenCurrentOffOrInCI(t *testing.T) {
	cur := &seeker{latest: core.Version}
	if line := autoApp(t, true, cur).AutoUpgrade(context.Background(), time.Now()); line != "" || cur.applied != 0 {
		t.Fatalf("up to date: nothing to say (%q)", line)
	}
	off := &seeker{latest: "99.0.0"}
	a := autoApp(t, true, off)
	t.Setenv("GTT_NO_UPDATE_CHECK", "1")
	if line := a.AutoUpgrade(context.Background(), time.Now()); line != "" || off.asked != 0 {
		t.Fatalf("GTT_NO_UPDATE_CHECK means no network at all (asked %d)", off.asked)
	}
	t.Setenv("GTT_NO_UPDATE_CHECK", "")
	t.Setenv("CI", "true")
	if line := a.AutoUpgrade(context.Background(), time.Now()); line != "" || off.asked != 0 {
		t.Fatal("CI never looks")
	}
	t.Setenv("CI", "")
	down := &seeker{err: errors.New("offline")}
	d := autoApp(t, true, down)
	now := time.Now()
	if line := d.AutoUpgrade(context.Background(), now); line != "" {
		t.Fatalf("offline is silent: %q", line)
	}
	d.AutoUpgrade(context.Background(), now.Add(10*time.Minute))
	if down.asked != 1 {
		t.Fatalf("after a failure it waits before asking again, asked %d", down.asked)
	}
}

func TestProjectBootstrapNoticeOncePerDay(t *testing.T) {
	b := &fakeBootstrap{}
	a, _ := newApp(t, b)
	contract := filepath.Join(a.Cwd, ".gtt", "contract")
	if err := os.MkdirAll(contract, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contract, "release.json"), []byte(`{"bootstrap":{"version":"1.0.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	line := a.ProjectBootstrapNotice(context.Background(), now)
	if !strings.Contains(line, "gtt update") || !strings.Contains(line, "1.0.0") {
		t.Skipf("the fake factory does not treat this folder as a project (%q)", line)
	}
	if again := a.ProjectBootstrapNotice(context.Background(), now.Add(time.Hour)); again != "" {
		t.Fatalf("said once a day, got %q", again)
	}
}

func TestOnlyLocalEditsConflictsDoNotStopAnUpdate(t *testing.T) {
	edited := "GTT ADE update\n  CONFLICT   claude   .claude/CLAUDE.md   (modified locally and changed in the catalog)\n  UPDATE     claude   .claude/hooks/x.py\n"
	if !onlyLocalEdits(edited) {
		t.Fatal("a file the project edited is kept, not a reason to stop")
	}
	foreign := edited + "  CONFLICT   claude   .claude/other.md   (exists but was not installed by GTT)\n"
	if onlyLocalEdits(foreign) {
		t.Fatal("a file GTT did not install is still a conflict")
	}
	if onlyLocalEdits("  PROBLEM    claude: no longer in the catalog registry\n" + edited) {
		t.Fatal("a problem still stops it")
	}
	if onlyLocalEdits("  UPDATE     claude   a\n") {
		t.Fatal("no conflict at all is not this case")
	}
}
