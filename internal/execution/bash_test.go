package execution

import (
	"errors"
	"testing"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func path(found string) func(string) (string, error) {
	return func(string) (string, error) {
		if found == "" {
			return "", errors.New("not found")
		}
		return found, nil
	}
}

func existing(paths ...string) func(string) bool {
	return func(p string) bool {
		for _, x := range paths {
			if x == p {
				return true
			}
		}
		return false
	}
}

func TestLookBashOutsideWindowsUsesPath(t *testing.T) {
	if got, err := lookBash("linux", env(nil), path("/usr/bin/bash"), existing()); err != nil || got != "/usr/bin/bash" {
		t.Errorf("got %q, %v", got, err)
	}
	if _, err := lookBash("darwin", env(nil), path(""), existing()); err == nil {
		t.Error("no bash on PATH is an error")
	}
}

func TestLookBashHonoursGTTBash(t *testing.T) {
	e := env(map[string]string{"GTT_BASH": "/opt/bash"})
	if got, err := lookBash("windows", e, path("/usr/bin/bash"), existing("/opt/bash")); err != nil || got != "/opt/bash" {
		t.Errorf("GTT_BASH wins: %q, %v", got, err)
	}
	if _, err := lookBash("linux", e, path("/usr/bin/bash"), existing()); err == nil {
		t.Error("a GTT_BASH that does not exist is an error, never silently replaced")
	}
}

func TestLookBashOnWindowsNeverUsesTheWSLLauncher(t *testing.T) {
	gitBash := `C:\Program Files\Git\bin\bash.exe`
	e := env(map[string]string{"ProgramFiles": `C:\Program Files`})
	for _, wsl := range []string{`C:\Windows\System32\bash.exe`, `C:\Users\me\AppData\Local\Microsoft\WindowsApps\bash.exe`} {
		if got, err := lookBash("windows", e, path(wsl), existing(gitBash)); err != nil || got != gitBash {
			t.Errorf("PATH gives %s: got %q, %v; want Git Bash", wsl, got, err)
		}
	}
	if _, err := lookBash("windows", e, path(`C:\Windows\System32\bash.exe`), existing()); err == nil {
		t.Error("only the WSL launcher available: refuse with an actionable error")
	}
	if got, _ := lookBash("windows", e, path(`D:\tools\msys64\usr\bin\bash.exe`), existing(gitBash)); got != `D:\tools\msys64\usr\bin\bash.exe` {
		t.Errorf("a non-WSL bash on PATH is used as is: %q", got)
	}
}
