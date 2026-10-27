package conformance

import (
	"reflect"
	"regexp"
	"testing"
)

func TestPathsDropAnnotationsAndPlaceholders(t *testing.T) {
	got := Paths("src/features/* (another slice), `src/legacy/**`, -")
	if want := []string{"src/features/*", "src/legacy/**"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestPatternMatchesTheGlob(t *testing.T) {
	cases := []struct {
		glob, path string
		match      bool
	}{
		{"src/features/*", "src/features/cart/index.ts", true},
		{"src/features/*", "src/features", false},
		{"src/shared/**", "src/shared/a/b/c.ts", true},
		{"src/shared/**", "src/sharedx/c.ts", false},
		{"src/domain", "src/domain/order.ts", true},
		{"src/domain", "src/domainx.ts", false},
		{"src/contexts/*/internal", "src/contexts/billing/internal/model.ts", true},
		{"src/contexts/*/internal", "src/contexts/billing/api/x.ts", false},
	}
	for _, c := range cases {
		if got := regexp.MustCompile(Pattern(c.glob, false)).MatchString(c.path); got != c.match {
			t.Errorf("%s ~ %s: got %v", c.glob, c.path, got)
		}
	}
}

func TestSiblingRefersToTheCapturedModule(t *testing.T) {
	if got := Sibling("src/features/*"); got != `^src/features/$1(/|$)` {
		t.Errorf("got %s", got)
	}
	if got := Pattern("src/features/*", true); got != `^src/features/([^/]+)(/|$)` {
		t.Errorf("got %s", got)
	}
}

func TestAModuleIsNeverForbiddenFromItself(t *testing.T) {
	for _, c := range []struct {
		from, to string
		same     bool
	}{
		{"src/features/*", "src/features/*", true},
		{"src/contexts/*", "src/contexts/*/internal", true},
		{"src/shared/**", "src/features/**", false},
		{"src/domain", "src/domain", false}, // no wildcard: a plain rule
	} {
		if got := sameKind(c.from, c.to); got != c.same {
			t.Errorf("%s vs %s: got %v", c.from, c.to, got)
		}
	}
}

func TestConfigHasOneRulePerRowAndGlob(t *testing.T) {
	rows := []Row{
		{Rule: "VS-1", Module: "src/features/*", MustNot: "src/features/* (another slice)"},
		{Rule: "HX-1", Module: "src/domain", MustNot: "src/adapters, src/infra"},
	}
	cfg := Config(t.TempDir(), rows)
	rules := cfg["forbidden"].([]cruiseRule)
	if len(rules) != 3 {
		t.Fatalf("got %d rules", len(rules))
	}
	if rules[0].Name != "VS-1#1" || rules[0].To["pathNot"] != `^src/features/$1(/|$)` || rules[0].From["path"] != `^src/features/([^/]+)(/|$)` {
		t.Errorf("a slice rule excludes the slice itself: %+v", rules[0])
	}
	if rules[1].Name != "HX-1#2" || rules[1].To["path"] != `^src/adapters(/|$)` || rules[1].To["pathNot"] != "" {
		t.Errorf("a plain rule forbids the path: %+v", rules[1])
	}
}

func TestRootsAreTheTopFolders(t *testing.T) {
	got := Roots([]Row{{Module: "src/features/*", MustNot: "lib/x"}, {Module: "src/shared/**", MustNot: "-"}})
	if want := []string{"lib", "src"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}
