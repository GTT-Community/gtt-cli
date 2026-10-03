# Solution Vision

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

## Problem

GTT Bootstrap owns what GTT means and how it governs, and exposes that as versioned contracts
(`.gtt/contract/`) plus deterministic scripts. Operating it today means invoking those scripts by hand
or trusting that an ADE remembers to do it: finding and verifying a compatible Bootstrap, installing only
the chosen ADE overlays, recording the Method Plan and the initial sources, validating, updating,
exporting a clean delivery and removing GTT safely. That operational burden falls on the developer, and
whatever depends on an agent's memory is not deterministic. [FUENTE: master]

## Users

- An individual developer who wants a governed project from one command and almost no mechanical work.
- A team that needs the same operations to be reproducible across people and ADEs.
- CI pipelines that need non-interactive validation with stable exit codes and JSON output.
- ADEs (Claude Code, Kiro, GitHub Copilot, Codex), which receive a Bootstrap-generated handoff, never CLI prompts.

## What success looks like

- `gtt init` takes a project from nothing to an installed, validated GTT workspace: Bootstrap resolved and
  verified, compatibility negotiated, ADEs and exactly one Primary chosen by the human, sources recorded,
  Method Plan recorded, handoff ready.
- `status`, `inspect`, `validate`, `doctor`, `audit`, `update`, `export --clean`, `clean` and `version`
  work with no LLM.
- A Bootstrap change that stays inside the existing capability surface (new template, new questionnaire
  text, new ADE built from the existing primitives, new plan semantics) needs no CLI source change.
- A Bootstrap that requires something the CLI lacks is refused before any project file is modified.
- The developer is interrupted only for decisions that are theirs; deterministic work is done and reported
  in a few lines. [FUENTE: friction]

## Non-goals

- Not a second GTT engine: no methodology, governance rules, evidence handling, proposal or freeze
  semantics, ADR logic or architectural judgment in Go.
- No copy of the questionnaire, of the ADE catalog, of the export exclusion list or of the validation checks.
- No interpretation of what Light, Medium, Hard or Team mean beyond executing the operating policy the
  Bootstrap declares for the selected plan.
- No retrieval / RAG / reasoning system of its own.
- No `unfreeze`, and no shortcut around a human decision.
- No dependency on a specific recipe document, and no execution of shell strings supplied by a project file.
- No synchronisation of credentials, tokens or provider session state.

---
Governance: L0. Read-only for AI agents.
