---
name: gtt-propose-change
description: Produce a GTT change proposal instead of applying a change directly. Use when the user says to process the change request, when an architectural change is required, when a governed context file under gtt-domain/context/ is wrong or outdated, when implementation code conflicts with the governed context, when gtt-domain/backlog.md needs an Epic/Story added, removed, or materially changed, or when a change is requested to a GTTGuard-protected file/class/method. Triggers on any request to change architecture, paradigm, module boundaries, frameworks, cloud services, or infrastructure tooling, on any request to change the committed development line, and whenever a permission denial points at gtt-domain/context/, gtt-domain/adr/, gtt-domain/change-request.md, or a GTTGuard-protected artifact.
---

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

# GTT change proposal

Governed context is owned by the Solution Designer. You produce proposals; a
human decides. Do not implement the change, do not partially apply it, and do
not create the ADR yourself.

## Where the request comes from

If the user says "process the change request", read `gtt-domain/change-request.md`
first. It holds the Solution Designer's stated intent. If
the request block is empty or unchanged from the template, say so and stop —
do not invent one.

The request block may be filled in tersely or as a raw paragraph pasted
straight from the Solution Designer's own document — both are valid input.
Read a pasted excerpt for what it actually decides; do not ask them to
compress it into the template's fields first.

Otherwise the request is whatever the user just described.

## Where the output goes

Write the proposal to `gtt-domain/proposals/PROPOSAL-<short-kebab-summary>.md`. That
directory is the only governed place you may write.

Do not edit `gtt-domain/change-request.md` — not to clear it, not to mark it
processed, not to tidy it. It is the Solution Designer's desk.

## Before writing

Read what the change actually touches: `gtt-domain/context/stack.md` always, plus the
relevant context files and any ADR the affected stack rows point to in their
"Locked by" column. A proposal that ignores the decision it overturns is not a
proposal.

If the request is about the development line instead — a new Epic/Story, or
a material change to one's scope or acceptance criteria — read `gtt-domain/backlog.md`
first, and check whether it contradicts governed context or an accepted
ADR before drafting (see backlog *Precedence* in `AGENTS.md`). Routine Story
status updates during already-approved work are not a change request at
all; redirect those back to direct editing, not a proposal.

If the request is a change to a file, class, or method carrying a
`@GTTGuard` marker — check `.gtt/protection/registry.yaml` — use Form 5
below; it is neither an architecture change nor a backlog change.

## Forms

Pick the one that matches the situation.

## 1. Architecture change

Use when the solution needs a different architectural direction, style,
paradigm, module boundary, integration strategy, deployment strategy, framework,
runtime, datastore, or cloud service.

```text
Proposed Architecture Change

Current decision:
Suggested change:
Reason:
Impact:
Risk:
Affected files:
Alternatives considered:

Status: Requires Architect approval
```

## 2. Context change

Use when a file under `gtt-domain/context/` is inaccurate, stale, or contradicts
reality — and the fix is to the document, not the code.

```text
Proposed Context Change

File:
Current statement:
Suggested change:
Reason:
Impact:
Risk:

Status: Requires Solution Designer approval
```

## 3. Context conflict

Use when implementation and governed context disagree and you cannot tell which
one is correct. Do not resolve it yourself and do not pick the code by default.

```text
Context Conflict Detected

Context file:
What the context says:
What the implementation does:
Where they diverge:
Possible resolutions:

Status: Requires human review
```

## 4. Development-line change

Use when `gtt-domain/backlog.md` needs an Epic or Story added, removed, or materially
changed in scope or acceptance criteria. Not for routine status updates
during already-approved work — those are direct edits, not a proposal.

```text
Proposed Backlog Change

Kind: <new Epic / new Story / remove / material scope change>
Epic/Story ID: <EPIC-NNN / STORY-NNN, or "new" if not yet assigned>
Current state: <what gtt-domain/backlog.md says today, or "none" if new>
Suggested change:
Reason:
Contradicts governed context or an ADR?: <no / yes — cite file:line>
Impact:
Risk:

Status: Requires Solution Designer approval
```

If the request would contradict governed context or an accepted ADR, say so
explicitly rather than quietly aligning the Story to the architecture — that
contradiction is exactly what the Solution Designer needs to see and decide.

## 5. Protected artifact change (GTTGuard)

Use when the requested change touches a file, class, or method that
`.gtt/protection/registry.yaml` lists with `protection: HUMAN_APPROVAL`.
This is not an architecture change and not a backlog change — GTTGuard is a
sibling mechanism protecting L3 code the developer opted into protecting,
not `gtt-domain/context/` or `gtt-domain/adr/`. See `AGENTS.md` → *Protected artifacts
(GTTGuard)* before using this form.

```text
Protected Artifact Change

Artifact: <file path>
Symbol: <ClassName, ClassName.method, or "whole file">
Current behavior:
Suggested change:
Reason:
Source (if any, e.g. an ADR or ticket the protection traces to):
Impact:
Risk:
Alternatives considered:

Status: Requires Solution Designer approval
```

**Promotion is lighter here than Forms 1-3.** There is no ADR and no
`apply-*.sh` script — GTTGuard is deliberately not a second, unrelated
approval model; it reuses this proposal mechanism, nothing heavier. Once
the Solution Designer approves
**in conversation**, implement the change directly, update or remove the
`@GTTGuard` marker as appropriate, and run `.gtt/scripts/gtt-guard-sync.sh`
so the registry reflects it. Do not treat this approval as covering any
other protected artifact, and do not confuse this with the L0/L1 Human
Promotion Boundary — that one still requires the Solution Designer to run a
script themselves, this one does not.

## Quality bar

A proposal is only useful if it can be decided without a follow-up question.

- Name the specific decision being changed, not the general area.
- Impact means blast radius: which modules, which interfaces, which deployments.
- Risk means what breaks if this is wrong, and how it would be detected.
- List at least one alternative and say why it loses.
- If you cannot fill a field honestly, say so rather than inventing it.

## After writing

State the file path you wrote and summarize the proposal in two or three lines
in chat, so the decision can be made without opening the file. Then stop.

Once an architecture, context, or conflict proposal (forms 1-3) is approved,
the decision is recorded with the `gtt-adr` skill, which stages the ADR, the
affected context files, and an executable promotion script together — the
Solution Designer reviews and runs it; the agent never applies the change
itself (the Human Promotion Boundary, `AGENTS.md`). A development-line
proposal (form 4) is not an architectural decision — once approved, the
Solution Designer applies it directly to `gtt-domain/backlog.md`, no script
involved; it does not get an ADR unless it also happens to touch governed
context. A protected-artifact proposal (form 5) is approved in conversation
— once the Solution Designer says so, implement it directly, update the
marker, and re-sync the registry; no ADR, no script, and never treat it as
approval for a different protected artifact.
