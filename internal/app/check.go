package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/conformance"
	"github.com/GTT-Community/gtt-cli/internal/core"
)

// CheckResult is `gtt check`: Method Conformance of the code.
type CheckResult struct {
	Schema int `json:"schema"`
	// Status: pass | conflict | declared-only | open | not-declared | cannot-determine.
	Status       string                 `json:"status"`
	Packs        []string               `json:"packs"`
	Declared     string                 `json:"declared_verification,omitempty"` // what the Method Lock declares
	Verification string                 `json:"verification"`                    // what this run actually verified
	Language     string                 `json:"language,omitempty"`
	Adapter      string                 `json:"adapter,omitempty"`
	Supported    []string               `json:"supported_languages"`
	Checked      []string               `json:"checked_rules"`
	NotInMap     []string               `json:"rules_not_in_map,omitempty"`
	AuditOnly    []string               `json:"audit_only_rules,omitempty"`
	Conflicts    []conformance.Conflict `json:"conflicts"`
	Note         string                 `json:"note,omitempty"`
}

type methodRules struct {
	Status       string   `json:"status"`
	Packs        []string `json:"packs"`
	Verification string   `json:"verification"`
	Rules        []struct {
		ID           string            `json:"id"`
		Check        string            `json:"check"`
		From         string            `json:"from"`
		MustNot      string            `json:"must_not_depend_on"`
		Materialized []conformance.Row `json:"materialized"`
	} `json:"rules"`
}

// Check verifies the code against the dependency rules of the ratified
// development method, with the language's own tool. It never corrects code
// or context: a forbidden dependency is reported as a conflict (exit 1).
// Without an adapter for the language the method is reported declared-only;
// with an adapter whose tool is not installed, the result cannot be
// determined (exit 7) - it is never reported as a pass.
func (a *App) Check(ctx context.Context) (CheckResult, error) {
	res := CheckResult{Schema: core.OutputSchema, Supported: conformance.Supported(), Checked: []string{}, Conflicts: []conformance.Conflict{}}
	p, err := a.open(ctx)
	if err != nil {
		return res, err
	}
	if !devMethodSupported(ctx, p.B) {
		return res, &core.Error{Code: core.ExitIncompatible, Unmodified: true,
			What: "The installed GTT Bootstrap " + p.Release.Bootstrap.Version + " does not offer Method Governance.", Next: "Run: gtt update"}
	}
	var mr methodRules
	if _, err := a.query(ctx, p.B, "method.rules", nil, &mr); err != nil {
		return res, err
	}
	res.Packs, res.Declared = mr.Packs, mr.Verification
	switch mr.Status {
	case "ratified":
	case "open":
		res.Status, res.Verification = "open", "none"
		res.Note = "The development method is OPEN: no rule applies until the human decides it."
		return res, nil
	default:
		res.Status, res.Verification = "not-declared", "none"
		res.Note = "No development method is declared in the governed context: nothing to check."
		return res, nil
	}
	var rows []conformance.Row
	for _, r := range mr.Rules {
		if r.Check != "dependency" {
			res.AuditOnly = append(res.AuditOnly, r.ID)
			continue
		}
		if len(r.Materialized) == 0 {
			res.NotInMap = append(res.NotInMap, r.ID)
			continue
		}
		res.Checked = append(res.Checked, r.ID)
		for _, row := range r.Materialized {
			row.Rule = r.ID
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		res.Status, res.Verification = "pass", "none"
		res.Note = "The method declares no dependency rule to verify in the code."
		return res, nil
	}
	adapter := conformance.For(p.Root)
	if adapter == nil {
		res.Status, res.Verification = "declared-only", "declared-only"
		res.Note = "No adapter for this project's language: the method's rules are declared, not verified in the code. " +
			"Languages with an adapter: " + strings.Join(res.Supported, ", ") + "."
		return res, nil
	}
	res.Language, res.Adapter = adapter.Language(), adapter.ID()
	tool := adapter.Tool(p.Root, a.Runner)
	if tool == "" {
		res.Status, res.Verification = "cannot-determine", "none"
		return res, &core.Error{Code: core.ExitUnavailable, Unmodified: true,
			What: "Method Conformance cannot be determined: " + adapter.ID() + " is not installed in this project.",
			Why:  "GTT verifies the code with the language's own tool and has no analyser of its own.",
			Next: "Install it (" + adapter.Install() + ") and run: gtt check"}
	}
	out, err := adapter.Check(ctx, p.Root, tool, rows, a.Runner)
	if err != nil {
		res.Status, res.Verification = "cannot-determine", "none"
		return res, &core.Error{Code: core.ExitUnavailable, Unmodified: true, What: "Method Conformance cannot be determined.", Why: err.Error()}
	}
	res.Conflicts = out.Conflicts
	res.Verification = "verified"
	if len(res.NotInMap) > 0 {
		res.Verification = "partial"
	}
	a.log(p.Root, "check", map[bool]string{true: "pass", false: "conflict"}[len(out.Conflicts) == 0],
		map[string]any{"packs": mr.Packs, "conflicts": len(out.Conflicts)})
	if len(out.Conflicts) > 0 {
		res.Status = "conflict"
		return res, &core.Error{Code: core.ExitFailure, Unmodified: true,
			What: fmt.Sprintf("Method Conformance: %d forbidden dependenc%s (reported, not corrected).", len(out.Conflicts), map[bool]string{true: "y", false: "ies"}[len(out.Conflicts) == 1]),
			Next: "Fix the code, or - if the design must change - raise a change request."}
	}
	res.Status = "pass"
	return res, nil
}
