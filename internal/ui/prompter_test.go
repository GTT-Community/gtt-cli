package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

func TestSelectHasNoDefault(t *testing.T) {
	p := Scripted(strings.NewReader("\n9\n2\n"), &bytes.Buffer{})
	i, err := p.Select("t", []ports.Option{{Label: "a"}, {Label: "b"}})
	if err != nil || i != 1 {
		t.Fatalf("got %d %v: empty and out-of-range answers must be asked again", i, err)
	}
}

func TestMultiSelectAndConfirm(t *testing.T) {
	p := Scripted(strings.NewReader("1, 3,1\n\ny\n"), &bytes.Buffer{})
	got, err := p.MultiSelect("t", []ports.Option{{Label: "a"}, {Label: "b"}, {Label: "c"}})
	if err != nil || len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("got %v %v", got, err)
	}
	if ok, _ := p.Confirm("q", false); ok {
		t.Error("empty answer takes the default (no)")
	}
	if ok, _ := p.Confirm("q", false); !ok {
		t.Error("y is yes")
	}
}

func TestEndOfInputIsCancellation(t *testing.T) {
	p := Scripted(strings.NewReader(""), &bytes.Buffer{})
	if _, err := p.Confirm("destroy?", true); !errors.Is(err, core.ErrCancelled) {
		t.Errorf("no answer must never become yes: %v", err)
	}
}

func TestNonTerminalIsNotInteractive(t *testing.T) {
	if New(strings.NewReader("y\n"), &bytes.Buffer{}, true).Interactive() {
		t.Error("--no-input disables prompting")
	}
}
