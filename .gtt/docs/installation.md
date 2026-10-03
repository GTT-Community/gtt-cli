# GTT Bootstrap — Installation Guide

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

This document contains the detailed installation procedures for GTT Bootstrap.

The repository README remains the canonical entry point. This guide expands the installation details without removing the Quick Start from the README.

## Installation modes

GTT supports two initial installation modes:

1. **Manual installation** — a human copies and integrates the GTT bootstrap into an existing project.
2. **Agent-assisted installation** — an ADE/AI coding agent reads the GTT documentation and performs the bootstrap under explicit rules.

In both modes, the final objective is the same: establish the GTT workspace contract, populate governed context, obtain human confirmation, and freeze the context before normal governed development.

---

## Prerequisites

- A project repository.
- Git.
- Bash for the CI gate and shell scripts.
- Python 3 for the protection hook.
- Claude Code, Kiro, Codex, GitHub Copilot, or another ADE capable of following the GTT bootstrap procedure.
- A completed design/source document is recommended but not mandatory.

The design document may be Markdown, text, Word, PDF, or another common format.

---

## Manual installation

### 1. Inspect the host project

Before copying GTT, inspect the project root.

Identify:

- existing `AGENTS.md`
- existing `readme-gtt.md` / `readme-gtt.es.md`
- existing `.gtt/` (the Engine)
- existing `gtt-domain/` (the governed domain: `context/`, `adr/`, `proposals/`, `backlog.md`, `change-request.md`, `session.md`, `.frozen`)
- existing `.claude/`
- existing `.kiro/`
- existing `.copilot/copilot-instructions.md`
- existing `.gitignore`
- source/design documents
- files or directories with names that GTT requires

**Do not overwrite existing files silently.**

If a required GTT filename already exists, stop and resolve the conflict deliberately.

### 2. Install the GTT scaffolding

The resulting workspace must contain:

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

The Engine lives in `.gtt/` (its documentation, including this file, in
`.gtt/docs/`) and the governed domain in `gtt-domain/` — see *Workspace hygiene* in
`readme-gtt.md`. Each artifact is generated directly where the scaffold
(`.gtt/scaffold/manifest.yaml`) places it, never elsewhere and moved afterward.
Both directory names are namespaced, so they do not collide with a project's own `docs/`, `context/` or `adr/`:
if the project already has a `.gtt/` or a `gtt-domain/`, stop and report the conflict instead of merging.


Install the ADE-specific adapter of every ADE you actually use in the project — `.claude/` for Claude Code, `.kiro/` for Kiro, `.copilot/copilot-instructions.md` for GitHub Copilot (Codex takes no extra adapter file) — and declare exactly one of them as the Primary ADE (the principal environment of your workflow; it holds no authority). Do not copy the others "just in case"; the source repository ships every adapter as a catalog, not as a package to install whole. The supported way is `gtt-ade.sh`, which copies the overlays, records what it installed in `.gtt/ade.json` and validates the result; run it without `--apply` first:

```bash
bash .gtt/scripts/gtt-ade.sh detect                       # candidates only - detection is not participation
bash .gtt/scripts/gtt-ade.sh install --from <bootstrap catalog> --participating claude,copilot --primary claude
bash .gtt/scripts/gtt-ade.sh install --from <bootstrap catalog> --participating claude,copilot --primary claude --apply
```

See the ADE adapter matrix in the README.

### 3. Preserve the source design

If the project contains a design document and it is the source used for bootstrap, preserve it as:

```text
SOURCE-BRIEF.*
```

Do not silently modify the original source.

The purpose of `SOURCE-BRIEF.*` is to retain the original human-authored design that was used to populate the governed context.

### 4. Merge `.gitignore`

If the host project already has a `.gitignore`, merge GTT's required entries into it.

Do not replace the host project's `.gitignore`.

The protection hook may generate `__pycache__/`; ensure the relevant generated files remain ignored.

### 5. Populate the context

Preferred method:

```text
bootstrap GTT
```

or:

```text
set up GTT
```

Do not begin by manually inventing the six context files if the bootstrap procedure is available.

If you deliberately choose manual authoring, start with:

```text
gtt-domain/context/stack.md
```

and explicitly mark unknown decisions rather than guessing.

### 6. Review

The human owner reviews:

- architecture
- technology stack
- requirements
- constraints
- principles
- solution vision
- glossary
- stack map

The context is not ratified merely because files were generated.

### 7. Freeze

When the context is complete and confirmed:

```bash
./.gtt/scripts/gtt-freeze.sh
```

The freeze operation establishes the governed regime and creates:

```text
gtt-domain/.frozen
```

From that point, governed context is protected against direct agent writes according to the installed enforcement adapters.

### 8. Verify the guardrail

Ask the agent to modify:

```text
gtt-domain/context/stack.md
```

The applicable enforcement mechanism should block the write.

A model saying “I should not do that” is not equivalent to deterministic enforcement.

### 9. Install the CI gate

Wire:

```text
.gtt/scripts/gtt-check-stack.sh
.gtt/scripts/gtt-check-backlog.sh
.gtt/scripts/gtt-check-protection.sh
```

into CI against the project's default branch.

The goal is to ensure that governed architectural changes and the architecture map remain synchronized, and that `.gtt/protection/registry.yaml` (GTTGuard) always matches the `@GTTGuard` markers actually in source.

---

## Agent-assisted installation

An agent should treat this repository as an executable documentation contract, not as a collection of files to copy blindly.

### Agent procedure

1. Read `readme-gtt.md`.
2. Read `AGENTS.md`.
3. Detect the candidate ADEs (`gtt-ade.sh detect`: an observation, never a choice) and show them to the human. The human chooses the participating ADEs and exactly one Primary; if that can't be established with confidence, stop and ask — never guess, and never install an adapter nobody chose.
4. Inspect the host project.
5. Identify the host project's design/source document.
6. If there is no document, offer the Initial Design Questionnaire (`gtt-template.sh materialize initial-design-questionnaire`) and proceed through the ADE-guided conversation it structures; never improvise a second questionnaire.
7. If there are multiple candidates, ask the user.
8. Never guess which source document is authoritative.
9. Never overwrite an existing same-name file silently.
10. Create the GTT workspace contract: the portable core plus only the overlays of the participating ADEs, explicitly excluding the others.
11. Map the confirmed source into the governed context.
12. Check for defined Epics/Stories (a requirements doc, issue tracker, or prior conversation). If found, reconcile them into `gtt-domain/backlog.md`; if none exist, say so explicitly rather than inventing them.
13. Ask the user to confirm the generated context.
14. Preserve the source as `SOURCE-BRIEF.*`.
15. Freeze only after explicit human confirmation.
16. Verify protection.
17. Report the final state.

### Required agent report

After installation, the agent should report:

- detected ADEs, participating ADEs and the Primary ADE
- adapters explicitly excluded
- whether native adapter support exists for this ADE
- files created
- files preserved
- conflicts found
- files intentionally skipped
- source document used
- whether context was confirmed
- whether `gtt-domain/backlog.md` is defined and reconciled with any known Epics/Stories
- whether freeze was executed
- whether protection was verified
- whether CI gate was connected
- any remaining human action

---

## Existing projects and upgrades

If GTT is being added to an existing project, preserve the host architecture and source tree.

GTT is not a license to reorganize the host project.

If a host file conflicts with a GTT-required file:

1. identify the conflict;
2. explain the role of both files;
3. ask for a decision;
4. resolve explicitly;
5. record the resolution when it affects governed architecture.

### Two-regime upgrade

For projects created before the two-regime freeze model:

```bash
./.gtt/scripts/gtt-freeze.sh
```

Run this after verifying that `gtt-domain/context/` contains real context and no template placeholders.

---

## Installation checklist

- [ ] Host project inspected.
- [ ] Candidate ADEs detected; participating ADEs and exactly one Primary chosen by the human (asked, not guessed, if unclear).
- [ ] Only the participating ADEs' adapters installed and recorded in `.gtt/ade.json`; the others explicitly excluded.
- [ ] `.gtt/scripts/gtt-check-adapter.sh` passes for every participating ADE.
- [ ] Required GTT files identified.
- [ ] Existing files protected from silent overwrite.
- [ ] GTT scaffolding created.
- [ ] `SOURCE-BRIEF.*` preserved when applicable.
- [ ] `.gitignore` merged.
- [ ] Context populated.
- [ ] `gtt-domain/backlog.md` present; known Epics/Stories reconciled or explicitly absent.
- [ ] Human review completed.
- [ ] Context explicitly confirmed.
- [ ] `gtt-domain/.frozen` created.
- [ ] Protected write tested.
- [ ] CI gate connected.
- [ ] Installation reported.

---

## Next step

After installation, continue with [usage.md](usage.md).

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

Every participating ADE's adapter — `.claude/` (Claude Code), `.kiro/` (Kiro), or `.copilot/copilot-instructions.md` (GitHub Copilot) — remains at the host-project root. Only the adapters of participating ADEs are installed; the choice is recorded in `.gtt/ade.json`, written only by `gtt-ade.sh`.

The agent must preserve existing host-project files, must not silently overwrite conflicts, and must not run the freeze step automatically. Human review and confirmation precede freezing.
