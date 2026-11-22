package app

import (
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/core"
)

// What a person pasted when `gtt update` failed on a protection hook: the script's own report, in English.
const adeReport = `GTT ADE update - evaluating every participating overlay against the catalog
  MERGE      claude   .claude/CLAUDE.md   (kept as yours; the catalog's version is saved as .claude/CLAUDE.md.gtt-new to merge by hand)
  KEEP       claude   .claude/hooks/pending-watch.py   (modified locally; catalog unchanged)

FAIL   claude    protection hook changed since install: .claude/hooks/pending-watch.py - the real-time protection may be off. Put it back with ` + "`git checkout -- .claude/hooks/pending-watch.py`" + ` (or copy it from the Bootstrap release); a change you made on purpose is recorded with ` + "`gtt-ade.sh record claude .claude/hooks/pending-watch.py --apply`" + `
gtt-ade: post-change validation failed; every change was rolled back.`

func TestAdeFailureIsSaidInTheProjectsLanguage(t *testing.T) {
	es := &core.Language{}
	es.Set(core.LangExplicit, "es")
	text := (&App{Lang: es}).adeFailureText(adeReport)
	for _, want := range []string{"claude: el hook de protección .claude/hooks/pending-watch.py no es el que GTT instaló", "Si cambiaste .claude/hooks/pending-watch.py a propósito, regístralo con: gtt-ade.sh record claude .claude/hooks/pending-watch.py --apply", "git checkout -- .claude/hooks/pending-watch.py, y corre: gtt update"} {
		if !strings.Contains(text, want) {
			t.Errorf("the Spanish text lacks %q:\n%s", want, text)
		}
	}
	for _, english := range []string{"protection hook", "real-time", "Put it back", "rolled back", "evaluating", "catalog"} {
		if strings.Contains(text, english) {
			t.Errorf("the Spanish text still carries English (%q):\n%s", english, text)
		}
	}
	en := (&App{}).adeFailureText(adeReport)
	if !strings.Contains(en, "the protection hook .claude/hooks/pending-watch.py is not the one GTT installed") {
		t.Errorf("English:\n%s", en)
	}
	if got := (&App{}).adeFailureText("something unexpected\nin two lines"); got != "" {
		t.Errorf("a report with no known line is shown as it came, so nothing is hidden: %q", got)
	}
	conflict := (&App{Lang: es}).adeFailureText("  CONFLICT   claude   .claude/settings.json   (exists but was not installed by GTT)\n  PROBLEM    kiro: no longer in the catalog registry\n")
	if !strings.Contains(conflict, ".claude/settings.json existe en el proyecto pero GTT no lo instaló") || !strings.Contains(conflict, "No se cambió nada") {
		t.Errorf("conflict:\n%s", conflict)
	}
}
