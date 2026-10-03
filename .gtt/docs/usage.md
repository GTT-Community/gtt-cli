# GTT Bootstrap — Usage Guide

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

GTT governs the context that guides AI-assisted development. The normal development loop is:

```text
Context → Decision → Proposal → Approval → Implementation → Verification
```

## The normal workflow

### 1. Start from governed context

Before making an implementation decision, the agent should read the applicable governed context.

At minimum, this includes the core rules and relevant context files.

The goal is not to load every document into every session. GTT deliberately separates always-needed constraints from context that is required only for a specific task.

### 2. Work on implementation

Routine implementation belongs to the implementation layer.

The agent may modify source code, tests, pipelines, and infrastructure according to the project's rules.

Routine implementation does not require a `gtt-domain/change-request.md`.

### 3. Detect an architectural change

If a requested change affects an architectural decision, technology choice, dependency rule, deployment topology, observability design, or another governed decision, do not silently modify the governed map.

Use:

```text
gtt-domain/change-request.md
```

### 4. Propose the change

The agent creates a reviewable proposal under:

```text
gtt-domain/proposals/
```

The proposal should explain:

- current decision
- requested change
- reason
- trigger
- scope
- impact
- risk
- alternatives
- affected architecture-map rows

### 5. Approve

The human reviews the proposal.

Approval is a governance decision, not an implementation detail.

### 6. Record the decision

The approved change becomes a **promotion package**, staged under:

```text
gtt-domain/proposals/
```

— the ADR draft, the full text of every affected file under `gtt-domain/context/`
(including the updated `gtt-domain/context/stack.md`), and an executable script:

```text
gtt-domain/proposals/apply-ADR-NNN-<slug>.sh
```

The agent never runs it. Review the package, then run it yourself from the
project root:

```bash
bash gtt-domain/proposals/apply-ADR-NNN-<slug>.sh
```

It asks for a final confirmation, applies every affected file together, and
fails clearly rather than leaving the map half-updated. Only this explicit
human execution actually writes to `gtt-domain/adr/` and `gtt-domain/context/stack.md` —
see the Human Promotion Boundary in `AGENTS.md`.

### 7. Verify

Run the relevant audit/check mechanisms.

The CI gate:

```text
.gtt/scripts/gtt-check-stack.sh
```

must fail when the governed architecture and its map become inconsistent.

---

## The architecture map

`gtt-domain/context/stack.md` provides seven views:

1. stack
2. components
3. deployment
4. observability
5. dependency rules
6. map change log
7. drift signals — paths outside the GTT-owned directories that carry architectural weight, watched by `detect-drift.py`

Use the map as the first architectural orientation point.

If a stack entry has no ADR in its `Locked by` field, investigate it as an ungoverned decision.

---

## The backlog

`gtt-domain/backlog.md` is the development line: Epics, Stories, current focus, and
next work. It is a planning artifact, not architecture — precedence is
governed context → ADR → backlog → implementation, and a Story never
overrides an architectural decision.

Before development work, establish the applicable Epic/Story from the
backlog. Adding, removing, or materially changing one goes through
`gtt-domain/change-request.md`, same as an architecture change. Updating a Story's
status or the Current Focus / Next Work / Blocked lists during
already-approved work is a direct edit, not a change request.

Run `.gtt/scripts/gtt-check-backlog.sh` for structural integrity (unique
IDs, valid status values); run `gtt-audit` to reconcile the backlog against
what is actually defined and actually done.

---

## Protected artifacts (GTTGuard)

Some L3 code deserves a narrower rule than "freely editable": a file,
class, or method a developer has explicitly marked so an agent may read and
propose changes to it, but never modify it autonomously. That is GTTGuard —
a sibling to L0/L1 governance, not part of it.

**Marking something protected** is a normal source edit, not a governed
change: add `@GTTGuard` (Java/Python), `[GTTGuard]` (C#), or `// @GTTGuard`
/ `# @GTTGuard` (comment-based languages), optionally with
`reason="..."`/`source="..."`, immediately above the declaration — or as
the file's first line to protect the whole file. Use the `gtt-guard` skill
for this, then run:

```bash
bash .gtt/scripts/gtt-guard-sync.sh
```

to regenerate `.gtt/protection/registry.yaml` from every marker in source.
The registry is derived, never hand-edited — `gtt-check-protection.sh`
fails the build if it drifts from what the markers actually declare.

**Changing something already protected** goes through `gtt-propose-change`
form 5, not a direct edit. Once you approve the proposal **in
conversation**, the agent implements the change directly and re-syncs the
registry — no ADR, no promotion script. This is deliberately lighter than
the Human Promotion Boundary above: GTTGuard protects L3 code you opted
into protecting, not `gtt-domain/context/` or `gtt-domain/adr/`, and the two promotion
models must never be conflated.

On Claude Code, `.claude/hooks/protect-guard.py` blocks a protected edit in
real time, resolving each protected symbol's current line span live from
disk so an unprotected method next to a protected one stays editable. Kiro,
Codex, and GitHub Copilot have no equivalent real-time block; enforcement
there is `.gtt/scripts/gtt-check-protection.sh` in CI plus an
instruction-plane note, the same honest fallback the two-regime
`gtt-domain/context/`/`gtt-domain/adr/` condition already uses.

---

## Change request example

A request should communicate intent rather than prescribe an implementation blindly.

Example:

```text
What needs to change?
Replace the current cache technology.

Why?
The current technology no longer meets the agreed operational constraints.

Trigger:
New deployment requirements.

Scope:
Caching layer and related observability.

Impact:
Architecture, deployment, configuration and operational documentation.

Risk:
Migration compatibility and cache invalidation behavior.

Priority:
High.
```

The agent should turn this into a proposal rather than directly editing the architecture map.

---

## Context layers

### L0 — governed context

```text
gtt-domain/context/
```

Contains the current governed understanding of the solution.

### L1 — architecture decisions

```text
gtt-domain/adr/
```

Contains accepted decisions and their rationale.

### L2 — documentation

```text
.gtt/docs/
```

Contains human reference material and methodology documentation.

### L3 — implementation

```text
src/
tests/
pipelines/
IaC/
```

Contains the implementation governed by the upper layers.

---

## Freeze and the two-regime model

GTT distinguishes between:

### Bootstrap regime

Before freeze:

- context can be populated by the bootstrap procedure;
- the design is still being confirmed;
- the context is not yet ratified.

### Governed regime

After:

```text
gtt-domain/.frozen
```

the governed context is protected.

Architectural changes must follow the change-request/proposal/ADR process.

---

## Keeping context useful

Keep the always-loaded context small.

`constraints.md` should contain only constraints that genuinely need to be available continuously.

Put detailed explanations, methodology, migration material, and reference documentation under `.gtt/docs/`.

Do not turn every instruction into a permanently loaded rule.

---

## Tool-specific adapters

GTT provides one adapter per supported ADE. A project installs the adapter of
every ADE its human chose to have participate, with exactly one of them declared
Primary (a workflow identifier that carries no authority) — chosen at bootstrap and
recorded in `.gtt/ade.json`, not by copying files around afterward. Several ADEs may
work in the project at once: the same governed artifacts and the same Human Promotion
Boundary apply to all of them.

- Claude Code uses `.claude/`.
- Kiro uses `.kiro/`.
- Codex uses `AGENTS.md` and applicable configuration, no extra adapter file.
- GitHub Copilot uses `.copilot/copilot-instructions.md` plus `AGENTS.md`.

Manage the set with `.gtt/scripts/gtt-ade.sh` (dry run until `--apply`): `state` shows the
Primary and participating ADEs and the health of each integration, `validate` fails when a
participating integration is broken, `install`/`remove`/`set-primary`/`update` change the set,
and `owned` lists exactly the files GTT installed for the ADEs — what a clean or an
`export --clean` may remove, and nothing else. `gtt-domain/session.md` carries the same state,
so any ADE that resumes the project sees it.

**Never remove `AGENTS.md`.**

---

## Operational checklist

Before implementation:

- [ ] Read applicable governed context.
- [ ] Establish the applicable Epic/Story from `gtt-domain/backlog.md`, if one exists.
- [ ] Determine whether the task is routine or architectural.
- [ ] If architectural, or a new/changed Epic/Story, create/process a change request.

During implementation:

- [ ] Keep implementation aligned with governed context.
- [ ] Do not silently modify governed decisions.
- [ ] If a file/class/method carries a `@GTTGuard` marker, use `gtt-propose-change` (form 5) instead of editing it directly.
- [ ] Update the Story's status and Current Focus as work actually progresses.
- [ ] Preserve host-project structure.

Before merge:

- [ ] Confirm required ADRs exist for architectural changes.
- [ ] Confirm the architecture map reflects accepted decisions.
- [ ] Run the stack check.
- [ ] Review the resulting diff.

---

## Guiding principle

> **Context is the Source of Truth.**

GTT does not attempt to make AI incapable of changing software. It establishes a governed boundary around the decisions that define what the software is supposed to be.

### Deployment target: bootstrap repository vs. host project

`readme-gtt.md`, `readme-gtt.es.md`, and `AGENTS.md` stay at the
**project root** — the canonical entry points, read before anything else.
Everything else follows the same scaffold in the bootstrap repository and in any
host project it installs into: the Engine (with Documentation, including this
file) in `.gtt/`, and the governed domain in `gtt-domain/`.


When an agent deploys GTT into a host project, it MUST reorganize the installed workspace so that it matches the GTT scaffold:

```text
/
├── AGENTS.md                     # portable agent contract (ADE discovery file)
├── readme-gtt.md
├── readme-gtt.es.md
├── SOURCE-BRIEF.*                # if a source document existed
│
├── .gtt/                         # GTT-METHOD ENGINE (machinery, derived state, GTT's own documentation)
│   ├── README.md
│   ├── index/
│   ├── protection/
│   ├── ade.json                  #   per-project ADE state: participating, primary, install ledger (installed projects)
│   ├── scaffold/manifest.yaml    #   also the ADE registry (overlays:) and the templates: the Bootstrap owns
│   ├── scaffold/templates/       #   Bootstrap-owned templates (the Initial Design Questionnaire)
│   ├── scripts/
│   ├── session-adapters/
│   └── docs/                     #   GTT's own documentation: index, installation, usage, docs, evidence,
│                                 #   gtt-completion, session-adapter-contract
│
├── gtt-domain/                   # THE DOMAIN GOVERNED BY GTT-METHOD
│   ├── context/                  #   L0 governed context
│   ├── adr/                      #   L1 accepted decisions
│   ├── proposals/                #   governed drafts awaiting a human decision
│   ├── backlog.md
│   ├── change-request.md
│   ├── session.md                #   derived operational state (never authority)
│   └── .frozen                   #   freeze marker, written by gtt-freeze.sh
│
└── .claude/ | .kiro/ | .copilot/ # ADE OVERLAYS - one per participating ADE; exactly one ADE is Primary
```

Every participating ADE's adapter — `.claude/`, `.kiro/`, or `.copilot/copilot-instructions.md` — remains at the host-project root. Only the adapters of participating ADEs are installed.

The agent must preserve existing host-project files, must not silently overwrite conflicts, and must not run the freeze step automatically. Human review and confirmation precede freezing.
