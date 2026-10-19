package core

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSemver(t *testing.T) {
	a, ok := ParseSemver("1.2.10")
	if !ok || a != [3]int{1, 2, 10} {
		t.Fatalf("got %v %v", a, ok)
	}
	for _, bad := range []string{"", "1.2", "v1.2.3", "1.2.3-rc1", "a.b.c"} {
		if _, ok := ParseSemver(bad); ok {
			t.Errorf("%q must not parse", bad)
		}
	}
	b, _ := ParseSemver("1.10.0")
	if CompareSemver(a, b) != -1 || CompareSemver(b, a) != 1 || CompareSemver(a, a) != 0 {
		t.Error("comparison is numeric, not lexical")
	}
}

func TestExitCodeContract(t *testing.T) {
	want := map[ExitCode]int{ExitOK: 0, ExitFailure: 1, ExitIncompatible: 2, ExitUsage: 3, ExitCancelled: 4,
		ExitConflict: 5, ExitIntegrity: 6, ExitUnavailable: 7}
	for code, n := range want {
		if int(code) != n {
			t.Errorf("exit code %d changed to %d", n, code)
		}
	}
	if CodeOf(nil) != ExitOK || CodeOf(errors.New("x")) != ExitFailure {
		t.Error("default mapping")
	}
	wrapped := fmt.Errorf("ctx: %w", &Error{Code: ExitIntegrity, What: "bad"})
	if CodeOf(wrapped) != ExitIntegrity {
		t.Error("a wrapped Error keeps its code")
	}
}

func TestErrorAnswersWhatWhyNext(t *testing.T) {
	msg := (&Error{What: "Refused.", Why: "Because.", Next: "Upgrade.", Unmodified: true}).Error()
	for _, part := range []string{"Refused.", "Reason:\nBecause.", "No project files were modified.", "Next step:\nUpgrade."} {
		if !strings.Contains(msg, part) {
			t.Errorf("missing %q in %q", part, msg)
		}
	}
}

// TestSuiteVersion holds ADR-005: the CLI, its source-build version and the
// Bootstrap it pins carry one suite number.
func TestSuiteVersion(t *testing.T) {
	if baseVersion != BootstrapPin {
		t.Fatalf("suite version mismatch: baseVersion %s, BootstrapPin %s (ADR-005: they are one number)", baseVersion, BootstrapPin)
	}
	if _, ok := ParseSemver(baseVersion); !ok {
		t.Fatalf("baseVersion %q is not MAJOR.MINOR.PATCH", baseVersion)
	}
}
