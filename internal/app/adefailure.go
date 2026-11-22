package app

import (
	"regexp"
	"strings"
)

var (
	adeHookRE     = regexp.MustCompile("^FAIL\\s+(\\S+)\\s+protection hook changed since install: (\\S+) - .*recorded with `([^`]+)`")
	adeFailRE     = regexp.MustCompile(`^FAIL\s+(\S+)\s+(.+)$`)
	adeConflictRE = regexp.MustCompile(`^\s*CONFLICT\s+(\S+)\s+(\S+)`)
	adeProblemRE  = regexp.MustCompile(`^\s*PROBLEM\s+(.+)$`)
)

// adeFailureText says in the person's language why the update of the agent layers (ADE) failed: what is wrong and what to do, not the
// script's list of actions. The Bootstrap script speaks English; the failures it can report are few and known. An unknown line is shown
// as it came, so nothing is hidden. It returns "" when the output holds none of those lines.
func (a *App) adeFailureText(out string) string {
	var lines []string
	type hook struct{ file, record string }
	var hooks []hook
	for _, raw := range strings.Split(out, "\n") {
		line := strings.TrimRight(raw, " \r")
		switch {
		case adeHookRE.MatchString(line):
			m := adeHookRE.FindStringSubmatch(line)
			hooks = append(hooks, hook{m[2], m[3]}) // the validator says how a change made on purpose is recorded: the CLI does not name it
			lines = append(lines, a.Lang.Sprintf("  - %s: the protection hook %s is not the one GTT installed, so the real-time protection may be off.", m[1], m[2]))
		case adeFailRE.MatchString(line):
			m := adeFailRE.FindStringSubmatch(line)
			lines = append(lines, a.Lang.Sprintf("  - %s: %s", m[1], m[2]))
		case adeConflictRE.MatchString(line):
			m := adeConflictRE.FindStringSubmatch(line)
			lines = append(lines, a.Lang.Sprintf("  - %s: %s exists in the project but GTT did not install it.", m[1], m[2]))
		case adeProblemRE.MatchString(line):
			lines = append(lines, a.Lang.Sprintf("  - %s", adeProblemRE.FindStringSubmatch(line)[1]))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	text := strings.Join(lines, "\n")
	if len(hooks) > 0 {
		for _, h := range hooks {
			text += "\n\n" + a.Lang.Sprintf("If you changed %s on purpose, record it with: %s. If not, put it back with: git checkout -- %s, and run: gtt update", h.file, h.record, h.file)
		}
	} else {
		text += "\n\n" + a.Lang.Sprintf("Nothing was changed. Fix what is listed and run: gtt update")
	}
	return text
}
