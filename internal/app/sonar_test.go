package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestVersionLess(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{{"1.18.0", "1.19.0", true}, {"1.19.0", "1.18.0", false}, {"1.9.0", "1.10.0", true}, {"1.19.0", "1.19.0", false}, {"", "1.0.0", true}, {"2.0.0-rc1", "2.0.1", true}} {
		if got := versionLess(c.a, c.b); got != c.want {
			t.Errorf("versionLess(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestPortfolioListIsPlainConfigAndRepoIsNeverTouched(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{Factory: fakeFactory{}}
	if _, err := a.SonarAdd(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SonarAdd(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	c, err := readSonarConfig()
	if err != nil || len(c.Repos) != 1 {
		t.Fatalf("adding twice keeps one entry: %v %v", c, err)
	}
	if SonarConfigPath() != filepath.Join(cfg, "gtt", "sonar.json") {
		t.Errorf("the list lives in the user's config: %s", SonarConfigPath())
	}
	if entries, _ := os.ReadDir(repo); len(entries) != 1 {
		t.Errorf("adding a repo does not touch it: %v", entries)
	}
	if _, err := a.SonarRemove(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SonarRemove(context.Background(), repo); err == nil {
		t.Error("removing what is not listed is refused")
	}
}
