package core

import (
	"strings"
	"testing"
)

func spanishSpeaker() *Language {
	l := &Language{}
	l.Set(LangExplicit, "es")
	return l
}

func TestLanguageChoiceOrder(t *testing.T) {
	var none *Language
	if none.Code() != "en" || (&Language{}).Code() != "en" {
		t.Error("without a choice the language is English")
	}
	l := &Language{}
	l.Set(LangEnv, "es_CL.UTF-8")
	if l.Code() != "es" {
		t.Errorf("GTT_LANG applies when nothing else is chosen: %s", l.Code())
	}
	l.Set(LangProject, "en")
	if l.Code() != "en" {
		t.Errorf("the project's language wins over GTT_LANG: %s", l.Code())
	}
	l.Set(LangExplicit, "es")
	if l.Code() != "es" {
		t.Errorf("an explicit choice wins over the project's: %s", l.Code())
	}
	l.Set(LangExplicit, "")
	if l.Code() != "en" {
		t.Errorf("clearing the explicit choice falls back to the project's: %s", l.Code())
	}
}

func TestTranslationKeepsArgumentsAndFallsBackToEnglish(t *testing.T) {
	es := spanishSpeaker()
	if got := es.Sprintf("Validating..."); got != "Validando..." {
		t.Errorf("exact entry: %q", got)
	}
	if got := es.Sprintf("✓ Design reset: %d file(s) back to the templates, %d removed.", 3, 1); got != "✓ Diseño reiniciado: 3 archivo(s) de vuelta a las plantillas, 1 eliminado(s)." {
		t.Errorf("the format is translated before its arguments are applied: %q", got)
	}
	if got := es.Sprintf("Primary: %s", "none"); got != "Principal: ninguno" {
		t.Errorf("a string argument that is a known message is translated too: %q", got)
	}
	if got := es.Sprintf("Something nobody translated: %d", 7); got != "Something nobody translated: 7" {
		t.Errorf("an unknown string stays English: %q", got)
	}
	if got := es.T("GTT is not initialized in this project (/tmp/x)."); got != "GTT no está inicializado en este proyecto (/tmp/x)." {
		t.Errorf("a message built by concatenation is matched by its slots: %q", got)
	}
	if got := es.T("The language must be chosen by a human."); got != "El idioma debe elegirlo una persona." {
		t.Errorf("a slot that is itself a message is translated: %q", got)
	}
	if got := es.T("\nNext:\nsomething\n"); got != "\nSiguiente:\nsomething\n" {
		t.Errorf("surrounding blank space is kept: %q", got)
	}
	en := &Language{}
	if got := en.Sprintf("Primary: %s", "none"); got != "Primary: none" {
		t.Errorf("English is never translated: %q", got)
	}
}

func TestEveryTranslationKeepsTheSlotsOfItsKey(t *testing.T) {
	for key, val := range spanish {
		k, v := verb.FindAllString(key, -1), verb.FindAllString(val, -1)
		if len(k) != len(v) {
			t.Errorf("%q: %d slots, translation has %d", key, len(k), len(v))
			continue
		}
		for i := range k {
			if !strings.Contains(v[i], "[") && k[i][len(k[i])-1] != v[i][len(v[i])-1] {
				t.Errorf("%q: slot %d is %s in English and %s in the translation", key, i, k[i], v[i])
			}
		}
	}
}

func TestErrorFrameIsTranslated(t *testing.T) {
	e := &Error{What: "The project is already frozen.", Why: "frozen since today", Next: "Run: gtt status", Unmodified: true}
	got := e.Render(spanishSpeaker())
	for _, want := range []string{"El proyecto ya está en freeze.", "Motivo:\nfrozen since today", "No se modificó ningún archivo del proyecto.", "Siguiente paso:\nEjecuta: gtt status"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if e.Error() != e.Render(nil) || !strings.Contains(e.Error(), "No project files were modified.") {
		t.Errorf("Error() is the English rendering:\n%s", e.Error())
	}
}
