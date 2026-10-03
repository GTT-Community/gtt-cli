---
name: gtt-guard
description: Mark a file, class, or method as protected with a @GTTGuard marker, or remove one, then keep .gtt/protection/registry.yaml in sync. Use when the user asks to protect a piece of code from autonomous agent edits, to mark something GTTGuard, to unprotect something, or asks what is currently protected. Not for making the actual change to a protected artifact — that goes through gtt-propose-change (Form 5).
---

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

# Mark or unmark a GTTGuard-protected artifact

GTTGuard is a sibling mechanism to L0/L1 governance, not a copy of it. It
protects arbitrary L3 code the developer opts into — placing the marker
*is* the Solution Designer's proposal ("User proposes → GTT implements →
Agent respects"), so writing the marker itself is an ordinary L3 edit, not
a protected change. Changing something that is *already* protected is a
different situation — see the last section below.

## Adding a marker

Ask what should be protected (a file, a class, or a method) and, if useful,
why (`reason`) and what decision it traces to (`source`, e.g. an ADR
number or ticket) — both optional, but worth prompting for once, since an
unexplained protection flag is a finding waiting to happen.

Insert the marker immediately above the declaration, in the form matching
the file's language:

| Language | Form |
|---|---|
| Java | `@GTTGuard` or `@GTTGuard(reason = "...", source = "ADR-021")` above the class/method |
| Python | `@GTTGuard` or `@GTTGuard(reason="...", source="ADR-021")` above the `class`/`def` |
| C# | `[GTTGuard]` or `[GTTGuard(Reason = "...", Source = "ADR-021")]` above the class/method |
| JS/TS and other comment-based languages | `// @GTTGuard reason="..." source="ADR-021"` on its own line above the declaration |
| Python/shell files, comment form | `# @GTTGuard reason="..." source="ADR-021"` |

The marker always binds to the next class or method declaration it
precedes, wherever it sits in the file. To protect an **entire file**
instead, place the marker where no class/method declaration immediately
follows it (for example, alone at the very top, with a blank line after).

Only Java (`.java`), C# (`.cs`), Python (`.py`), and JS/TS (`.js`/`.jsx`/
`.ts`/`.tsx`) are resolved deterministically today. A marker in an
unsupported file type is not detected at all. If a marker precedes what
looks like a class/method but its body can't be resolved (e.g. a truncated
file), the sync step reports that specific error instead of guessing —
otherwise, a marker that doesn't precede a recognizable declaration falls
back to protecting the whole file rather than erroring, the same fail-safe
default the enforcement hook uses.

## Removing a marker

Delete the `@GTTGuard`/`[GTTGuard]`/comment-marker line. Do not hand-edit
`.gtt/protection/registry.yaml` to remove the entry — it is a derived
artifact and will simply be regenerated with the entry gone once you
re-sync.

## After adding or removing a marker

Run:

```bash
bash .gtt/scripts/gtt-guard-sync.sh
```

This regenerates `.gtt/protection/registry.yaml` from every marker currently
in source. If a marker precedes a class/method whose body can't actually
be resolved (a truncated file, for example), the sync fails with the exact
file and line — fix the underlying code and re-run rather than editing the
registry by hand.

Then, ideally, run `bash .gtt/scripts/gtt-check-protection.sh` — it also
validates that any `source: "ADR-NNN"` actually exists under `gtt-domain/adr/`.

Tell the user in one line what is now protected/unprotected and that the
registry was synced.

## Changing something that is already protected

That is not this skill. A `HUMAN_APPROVAL` artifact/symbol may be read,
analyzed, and reasoned about freely, but an autonomous edit to it is
blocked (real-time, on Claude Code, by `.claude/hooks/protect-guard.py`,
which resolves the exact protected span live from disk — an unprotected
method in the same file is unaffected). Use the `gtt-propose-change` skill,
Form 5, to draft the change instead. See that skill's *Form 5* section for
the full flow, and `AGENTS.md` → *Protected artifacts (GTTGuard)* for why
its promotion model is deliberately lighter than the L0/L1 Human Promotion
Boundary.
