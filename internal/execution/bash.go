package execution

import (
	"errors"
	"os"
	"runtime"
	"strings"
)

// bashCandidates are the usual places of Git for Windows' bash, which runs
// the Bootstrap scripts natively on Windows.
func bashCandidates(getenv func(string) string) []string {
	var out []string
	for _, base := range []string{getenv("ProgramFiles"), getenv("ProgramW6432"), getenv("ProgramFiles(x86)")} {
		if base != "" {
			out = append(out, winJoin(base, `Git\bin\bash.exe`))
		}
	}
	if local := getenv("LOCALAPPDATA"); local != "" {
		out = append(out, winJoin(local, `Programs\Git\bin\bash.exe`))
	}
	return out
}

// winJoin joins Windows path segments the same way on every build platform.
func winJoin(base, rest string) string { return strings.TrimRight(base, `\/`) + `\` + rest }

// isWSLLauncher reports the bash.exe of Windows' system directory: it starts
// a Linux distribution under WSL, which cannot run a script by its Windows
// path, so it is never used for the Bootstrap.
func isWSLLauncher(path string) bool {
	p := strings.ToLower(strings.ReplaceAll(path, `\`, "/"))
	return strings.HasSuffix(p, "/windows/system32/bash.exe") || strings.HasSuffix(p, "/windows/sysnative/bash.exe") ||
		strings.Contains(p, "/microsoft/windowsapps/")
}

// lookBash finds bash. Everywhere GTT_BASH wins when set. On Windows the
// Git for Windows installation is preferred, and the WSL launcher is never
// accepted from PATH; elsewhere PATH decides.
func lookBash(goos string, getenv func(string) string, lookPath func(string) (string, error), exists func(string) bool) (string, error) {
	if explicit := getenv("GTT_BASH"); explicit != "" {
		if exists(explicit) {
			return explicit, nil
		}
		return "", errors.New("GTT_BASH points to a missing file: " + explicit)
	}
	if goos != "windows" {
		return lookPath("bash")
	}
	if found, err := lookPath("bash"); err == nil && !isWSLLauncher(found) {
		return found, nil
	}
	for _, c := range bashCandidates(getenv) {
		if exists(c) {
			return c, nil
		}
	}
	return "", errors.New("bash not found: install Git for Windows (it provides bash), or set GTT_BASH to bash.exe")
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// systemBash is lookBash for the running system.
func systemBash(lookPath func(string) (string, error)) (string, error) {
	return lookBash(runtime.GOOS, os.Getenv, lookPath, fileExists)
}
