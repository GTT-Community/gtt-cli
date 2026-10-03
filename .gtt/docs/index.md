# GTT Index

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

Every file in this kit: what it is, who owns it, and when it enters an agent's
context. If you read one file to orient yourself, read this one.

This is the canonical map of the GTT scaffold — the Engine (`.gtt/`, including GTT's own
documentation in `.gtt/docs/`), the governed domain (`gtt-domain/`), and the ADE overlay.

Every path below is relative to the project root. See *Workspace hygiene* in
`readme-gtt.md` for why the split is drawn where it is, and
`.gtt/scaffold/manifest.yaml` for its declarative definition.

---

## Quick links

- [Change Request](../../gtt-domain/change-request.md) — the only input door
- [Backlog](../../gtt-domain/backlog.md) — Epics, Stories, current focus
- [Completion Report](gtt-completion.md) — durable bootstrap record
- [Context](../../gtt-domain/context/) — L0, governed
- [ADRs](../../gtt-domain/adr/) — L1, accepted decisions
- [Proposals](../../gtt-domain/proposals/) — agent drafts awaiting review
- [Installation](installation.md) — detailed setup procedures
- [Usage](usage.md) — the normal development loop
- [Method Plans](method-plans.md) — Light, Medium, Hard, Team: what each one delegates, asks and requires
- [Docs](./) — human reference, methodology
- [Scripts](../scripts/) — CI gates and freeze
- [AGENTS.md](../../AGENTS.md) — portable core rules (root)
- [readme-gtt.md](../../readme-gtt.md) — setup and tool support (root)

---

## Start here

| I want to... | Go to |
|---|---|
| Populate GTT for the first time in this project | drop your solution doc at the project root, any name (optional), then run the `gtt-bootstrap` skill |
| Ratify a freshly-bootstrapped context so it becomes read-only | `.gtt/scripts/gtt-freeze.sh` |
| Change the stack, architecture, any directive, or an Epic/Story | `gtt-domain/change-request.md` |
| See the development line — Epics, Stories, current focus | `gtt-domain/backlog.md` |
| Check whether an L3 change (infra, a manifest) contradicts ratified architecture | `.claude/skills/gtt-drift-response/SKILL.md`, or wait for the `GTT DRIFT SIGNAL` warning |
| Protect a file, class, or method from autonomous agent edits | `.claude/skills/gtt-guard/SKILL.md` — add a `@GTTGuard` marker |
| Request a change to a GTTGuard-protected artifact | `.claude/skills/gtt-propose-change/SKILL.md` (form 5) |
| See what this system is, in one screen | `gtt-domain/context/stack.md` |
| Understand why GTT works this way | `.gtt/docs/docs.md` (Methodology) |
| Set this up in my project | `readme-gtt.md` |
| Run it on Kiro, Codex, or Copilot | `.gtt/docs/docs.md` (Portability) |
| See which adapter I get for my ADE | `.claude/skills/gtt-bootstrap/SKILL.md` (step 0) or `readme-gtt.md` → *ADE adapters* |
| Resume work / see current state, regardless of which ADE picks it up | `.gtt/scripts/gtt-status.sh` (regenerates `gtt-domain/session.md`) |
| Run every deterministic check before a release | `.gtt/scripts/gtt-validate.sh` |
| Find a concept or section without loading whole documents | `.gtt/scripts/gtt-query.sh` (or the `gtt-retrieve` skill) |
| Moved or renamed an artifact | `.gtt/scripts/gtt-reconcile.sh` (dry-run), then `--apply` |

---

## Governed context — you own it, agents cannot write it once frozen

Pre-freeze (`gtt-domain/.frozen` absent), `gtt-domain/context/` and `gtt-domain/adr/` are the one
exception: `gtt-bootstrap` writes them directly. See `gtt-domain/.frozen` under
*Enforcement* below.

| File | Layer | Contains | Loads |
|---|---|---|---|
| `gtt-domain/change-request.md` | — | Your standing request desk. The only input door | never |
| `SOURCE-BRIEF.*` | — | Your original design document, if one existed. Written once by `gtt-bootstrap`, then locked — not present if the context came entirely from conversation | never |
| `gtt-domain/context/stack.md` | L0 | **The map**: stack table, components, topology, observability, dependency rules, drift signals, change log | on demand |
| `gtt-domain/context/architecture.md` | L0 | Architecture in prose, module responsibilities | on demand |
| `gtt-domain/context/constraints.md` | L0 | Hard limits. Kept short because it is always loaded | **always** |
| `gtt-domain/context/principles.md` | L0 | Design principles in force | on demand |
| `gtt-domain/context/solution-vision.md` | L0 | Purpose and non-goals | on demand |
| `gtt-domain/context/glossary.md` | L0 | Terms whose meaning here differs from the usual | on demand |
| `gtt-domain/adr/ADR-*.md` | L1 | Accepted decisions and their rationale | on demand |
| `gtt-domain/adr/ADR-TEMPLATE.md` | L1 | Blank ADR | never |

## Development line — governed like architecture, but not L0

`gtt-domain/backlog.md` is Epics, Stories, current focus, and next work. Structural
changes (new/removed Epic or Story, material scope or acceptance-criteria
change) go through `gtt-domain/change-request.md` like an architectural decision. Story
status and focus updates during already-approved implementation are direct
edits — see *Backlog governance* in `AGENTS.md`. Precedence: L0 → ADR →
`gtt-domain/backlog.md` → implementation; a Story never overrides governed context.

## Staging — agents write here

| File | Contains | Loads |
|---|---|---|
| `gtt-domain/proposals/` | Agent drafts, and approved changes' promotion packages (ADR draft + affected context files + `apply-*.sh`), awaiting your review. Delete when resolved | never |

## Protection registry — derived, self-correcting

GTTGuard (see `AGENTS.md` → *Protected artifacts*) is a sibling to L0/L1,
not part of it — it protects L3 code you opt into protecting, not
`gtt-domain/context/`/`gtt-domain/adr/`.

| File | Contains | Loads |
|---|---|---|
| `.gtt/protection/registry.yaml` | Every `@GTTGuard`-marked artifact, regenerated from source markers by `gtt-guard-sync.sh`. Derived, like a lockfile — never hand-edited; `gtt-check-protection.sh` fails the build if it drifts from a fresh regeneration | never |

## Session state — derived, on demand

Operational context for resuming work across ADEs, not architectural
authority, evidence, or a substitute for an ADR — see `AGENTS.md` →
*Session continuity*.

| File | Contains | Loads |
|---|---|---|
| `gtt-domain/session.md` | Snapshot of freeze/backlog-focus/proposals/change-request/ADRs/GTTGuard state, regenerated by `gtt-status.sh`. Derived — never hand-edited | never |

## Artifact identity and technical index — derived accelerator, identity manifest

See `.gtt/docs/docs.md` → *Artifact identity and the technical index*. Ids survive
moves; reference an artifact as `[[ID]]`.

| File | Contains | Loads |
|---|---|---|
| `.gtt/index/artifacts.json` | Identity manifest: id, type, path, former paths, aliases. Authoritative for identity; changed only by `gtt-index.sh` / `gtt-reconcile.sh` | never |
| `.gtt/index/technical-index.json` | Documents, sections, concepts, relationships, provenance, authority, version. Derived — rebuildable, never authoritative; `gtt-check-integrity.sh` fails on a stale one | never (query it via `gtt-query.sh`) |

## Scaffold definition — declarative, canonical

| File | Contains | Loads |
|---|---|---|
| `.gtt/scaffold/manifest.yaml` | The one canonical, declarative definition of the scaffold: Engine components (including documentation), the governed domain, ADE overlays, and which are required or optional. Data only — no logic; the repository is the source of truth it must agree with | never |

## Instructions — how agents behave

A project has the adapter of every ADE its human chose to have participate, one of
them Primary — see `.claude/skills/gtt-bootstrap/SKILL.md` (step 0) and `.gtt/ade.json`.
The rows below marked *(Claude Code adapter)* or *(Kiro adapter)* are present in an
installed project only for participating ADEs; all are shown here because this
source repository is the catalog, not an installed project.

| File | Contains | Loads |
|---|---|---|
| `AGENTS.md` | Portable core rules. Read natively by Kiro, Codex, and Copilot | **always** |
| `.claude/CLAUDE.md` *(Claude Code adapter)* | Imports `AGENTS.md`, adds skill routing | **always** |
| `.claude/rules/implementation.md` *(Claude Code adapter)* | Rules for `src/`, `tests/`, `lib/` | on matching files |
| `.claude/rules/infrastructure.md` *(Claude Code adapter)* | Rules for `infra/`, `deploy/`, CI | on matching files |
| `.kiro/steering/gtt-implementation.md` *(Kiro adapter)* | Kiro mirror of the above | on matching files |
| `.kiro/steering/gtt-infrastructure.md` *(Kiro adapter)* | Kiro mirror of the above | on matching files |
| `.copilot/copilot-instructions.md` *(Copilot adapter)* | Points Copilot at `AGENTS.md` and the governed directories (`gtt-domain/context/`, `gtt-domain/adr/`) as the canonical source; no duplicated methodology. Not auto-loaded by Copilot at this path — see the note in `readme-gtt.md` → *ADE adapters* | never (manual reference only, unless also mirrored to `.github/copilot-instructions.md`) |

## Procedures — load only when invoked

| File | Invoked when |
|---|---|
| `.claude/skills/gtt-bootstrap/SKILL.md` | first time populating `gtt-domain/context/`, pre-freeze, right after cloning the kit |
| `.claude/skills/gtt-propose-change/SKILL.md` | processing a change request — architecture, context, conflict, a development-line (Epic/Story) change (form 4), or a change to a GTTGuard-protected artifact (form 5) |
| `.claude/skills/gtt-adr/SKILL.md` | a change was approved and needs recording — drafts the ADR, the affected context files, and the `apply-ADR-NNN-<slug>.sh` promotion script together (architecture/context changes only — a backlog-only or GTTGuard-only change does not get an ADR) |
| `.claude/skills/gtt-audit/SKILL.md` | checking whether context still matches the code, or whether `gtt-domain/backlog.md` is reconciled with defined Epics/Stories — also the scheduled sweep counterpart to the drift detector below |
| `.claude/skills/gtt-drift-response/SKILL.md` | a `GTT DRIFT SIGNAL` fired, `gtt-audit` found a divergence, or you're asking whether an L3 change contradicts ratified architecture — stages the same promotion-script package as `gtt-adr` |
| `.claude/skills/gtt-retrieve/SKILL.md` | locating a concept/section or an artifact's relationships, or reconciling a move — through the technical index instead of scanning Markdown |
| `.claude/skills/gtt-guard/SKILL.md` | marking or unmarking a `@GTTGuard`-protected file/class/method, then syncing the registry |

## Enforcement — costs zero context

| File | Does |
|---|---|
| `gtt-domain/.frozen` | The regime marker. Absent = pre-freeze, `gtt-domain/context/`/`gtt-domain/adr/` are agent-writable. Present = governed, they're denied. Human-written only, via `gtt-freeze.sh`; versioned, not ignored |
| `.gtt/scripts/gtt-freeze.sh` | Run by the Solution Designer to ratify: validates L0 has real content, then writes `gtt-domain/.frozen` |
| `.claude/settings.json` | Denies writes to governance machinery unconditionally; registers all hooks below |
| `.claude/hooks/protect-l0.py` | `PreToolUse`. Regime-aware: blocks `gtt-domain/context/`/`gtt-domain/adr/`/`gtt-domain/change-request.md`/`SOURCE-BRIEF.*` (the last two unconditionally) once frozen; blocks machinery paths always, via shell too. Exit 2 |
| `.claude/hooks/protect-guard.py` | `PreToolUse`. Blocks an autonomous edit to a `HUMAN_APPROVAL` entry in `.gtt/protection/registry.yaml` in real time — file-scope blocks the whole file, symbol-scope resolves the exact span live from disk and fails safe to whole-file if resolution is ambiguous. Exit 2 |
| `.claude/hooks/detect-drift.py` | `PostToolUse`, governed regime only. Warns (never blocks) when a write matches the `gtt-drift-signals` block in `stack.md`. Deduplicated per session |
| `.claude/hooks/notify-change-request.py` | `UserPromptSubmit`. Advisory only: notices a filled-in, unprocessed `gtt-domain/change-request.md` and surfaces it in context — never analyzes or drafts. Deduplicated per session/content |
| `.kiro/permissions.yaml` | Kiro's declarative equivalent of the machinery-path deny (1.0+) |
| `.kiro/hooks/detect-drift.json` | Kiro mirror of `detect-drift.py` — same script, different trigger wiring |
| `.gtt/scripts/gtt-check-stack.sh` | CI gate: an ADR without a map update fails the build; also checks referential integrity of ADR citations and warns if the drift-signals block is missing |
| `.gtt/scripts/gtt-check-adapter.sh` | Deterministic validation of the ADE integrations: with no argument, every participating ADE against `.gtt/ade.json` (FAIL if a participating integration is missing or inconsistent, WARN for a detected ADE that does not participate); given a target ADE, the single-ADE matrix of projects that predate `.gtt/ade.json` |
| `.gtt/scripts/gtt-ade.sh` / `.gtt/scripts/gtt_ade.py` / `.gtt/scripts/gtt_manifest.py` | The ADE integration service (Multi-ADE): registry (`list`), candidates (`detect`), `state`, `validate`, GTT-installed surfaces (`owned`), and the mutating `install` / `adopt` / `set-primary` / `record` / `remove` / `update` (dry run until `--apply`). ADE-independent; the Primary ADE holds no authority |
| `.gtt/scripts/gtt-template.sh` / `.gtt/scripts/gtt_template.py` | The Bootstrap's template service: `list`, `show`, `materialize` (dry run until `--apply`, never overwrites) — how a CLI requests the Initial Design Questionnaire instead of carrying a copy |
| `.gtt/ade.json` | Per-project ADE state: participating, primary and excluded ADEs and the ledger (path → sha256) of files GTT installed for them. Written only by `gtt-ade.sh`; not governed context, not authority. Absent in the catalog |
| `.gtt/scaffold/templates/gtt-initial-design-questionnaire.md` | The Initial Design Questionnaire: the ADE-guided elicitation instrument for a project without a sufficient design document. Owned by the Bootstrap; a filled copy is source material, never governed architecture |
| `.gtt/scripts/gtt-check-backlog.sh` | CI gate: fails on duplicate Epic/Story IDs or a status value outside the agreed vocabulary in `gtt-domain/backlog.md`; warns on an Epic with no Stories yet |
| `.gtt/scripts/gtt-check-protection.sh` | CI gate: fails if `.gtt/protection/registry.yaml` drifts from a fresh regeneration, if an artifact/symbol/source fails to resolve, or if a protected artifact changed with no accompanying `gtt-domain/proposals/`/`gtt-domain/adr/` change (the one real-time backstop on Kiro, Codex, and Copilot) |
| `.gtt/scripts/gtt-guard-sync.sh` / `.gtt/scripts/gtt_guard.py` | Regenerates `.gtt/protection/registry.yaml` from `@GTTGuard` markers in source; the shared, deterministic engine both this script and `protect-guard.py` import |
| `.gtt/scripts/gtt-index.sh` / `.gtt/scripts/gtt-reconcile.sh` / `.gtt/scripts/gtt-query.sh` / `.gtt/scripts/gtt_artifacts.py` | Register artifacts and rebuild the derived index / reconcile moves and rewrite links / section-level retrieval; the shared deterministic engine behind all three and the check below |
| `.gtt/scripts/gtt-check-integrity.sh` | CI gate: fails on unreconciled moves, unregistered artifacts, duplicate logical identity, broken/old-path links, unresolved `[[ID]]`, or a stale technical index |
| `.gtt/scripts/gtt-session-context.sh` | Session Memory Service entry point (ADE-independent): runs `gtt-status.sh`, verifies `gtt-domain/session.md`, prints an operational-only payload for an ADE adapter; fails loudly. See `.gtt/docs/docs.md` → *Session memory boundary* |
| `.gtt/scripts/gtt-check-session-adapter.sh` / `.gtt/scripts/gtt_session_adapter.py` | Static conformance of a Session Memory ADE adapter against `.gtt/docs/session-adapter-contract.md`, driven by `.gtt/session-adapters/<ade>.json`; needs no ADE; PASS/FAIL/SKIPPED per item, runtime verification always SKIPPED and reported as declared. Run per adapter by `gtt-validate.sh` |
| `.gtt/session-adapters/<ade>.json` | Declaration of one adapter (coverage N1/N2/N3, events, registration, verification, limitations). Adapter layer, not Core |
| `.gtt/scripts/gtt-run-python.sh` | Launcher for ADE hooks: one Python 3 probe, then `exec`; a missing interpreter is a visible error, never a fallback after the script started |
| `.gtt/scripts/gtt-check-provenance.sh` / `.gtt/scripts/gtt_provenance.py` | The single provenance gate (sub-checks `tags`, `gaps`, `sources`, `preferences`; `--pre-freeze`) and the governance view behind `gtt-status.sh` and `gtt-query.sh --governance`. Deterministic, never authority |
| `gtt-domain/context/sources.md` *(optional, L0)* | Source manifest: declared sources with authority, precedence, version, status and the provenance policy |
| `gtt-domain/working-agreements.md` *(optional)* / `.gtt/local/preferences.md` *(optional, local)* | Team working agreements / a person's own preferences: below governed context, never authority, never session memory |
| `.gtt/scaffold/templates/gtt-sources-manifest.md`, `gtt-working-agreements.md` | Bootstrap-owned templates for the two files above, requested through `gtt-template.sh` |
| `.gtt/scripts/gtt-contract.sh` / `.gtt/scripts/gtt_contract.py` | The CLI-facing Bootstrap 1.0 contract entry point: `release`, `negotiate` (COMPATIBLE or REFUSE, never modifies), `capabilities`, `operations`, `show`, `run <operation>` (declared operations only, typed arguments, no shell), `check`. See `.gtt/docs/bootstrap-contract.md` |
| `.gtt/scripts/gtt-project.sh` / `.gtt/scripts/gtt_project.py` | Structured project contracts: `detect`, `status`, `session`, `validation`, `next-id` (next free ADR/Epic/Story id, never reusing a retired one), `interaction` (what the selected plan does without asking and what it asks for), `profile get/set`, `source select/list`, `export-policy`, `clean-plan`, `recovery snapshot/restore` (dry run until `--apply`; derived from the project, never authority) |
| `.gtt/scripts/gtt-maintain.sh` | The deterministic work after any operation in one run: GTTGuard registry sync, index rebuild, `gtt-validate.sh`. Brief report by default, `--verbose` for the full output. Under Light it also reconciles unambiguous moves; under the other plans it stops with the command. Never touches governed context |
| `.gtt/scripts/gtt-check-contract.sh` | Gate: the contracts are consistent, versioned and fail closed (no unfreeze operation, no shell in argv, implementations confined to `.gtt/scripts/`, invariants not relaxable) |
| `.gtt/contract/*.json` | The Bootstrap 1.0 contracts as data: `release`, `capabilities`, `operations`, `profiles`, `export-policy`, `recovery`, `elicitation` |
| `.gtt/methodology.json` / `.gtt/selected-sources.json` | Per-project state written only by `gtt-project.sh`: the Method Plan the human selected (absent = not selected) and language / initial sources selected (a selection is never an authority) |
| `.gtt/tests/bootstrap-acceptance.py` | Bootstrap 1.0 acceptance tests: drives the contracts as a CLI would on disposable copies; never modifies the project |
| `.gtt/scripts/gtt-status.sh` | Deterministic snapshot of current/governed/pending/proposed/blocked/frozen state, derived from repository artifacts; regenerates `gtt-domain/session.md` |
| `.gtt/scripts/gtt-validate.sh` | Runs every `gtt-check-*.sh` above in sequence and reports pass/fail/cannot-determine; validates every participating ADE against `.gtt/ade.json` when it exists; otherwise skips (not fails) the adapter check when this repo's own multi-adapter catalog state is detected |

## Human documentation — never loaded by any agent

| File | Contains |
|---|---|
| `.gtt/docs/index.md` | This file |
| `readme-gtt.md` / `readme-gtt.es.md` | What GTT is, setup, tool support — the canonical entry point |
| `.gtt/docs/bootstrap-contract.md` | The Bootstrap 1.0 contract for the CLI: what each contract is, where it lives, how it is invoked, honest limits |
| `.gtt/docs/session-adapter-contract.md` | Draft contract every Session Memory ADE adapter must meet; GTT Core is ADE-agnostic, adapters are integration-specific |
| `.gtt/docs/installation.md` / `.gtt/docs/installation.es.md` | Detailed installation procedures, manual and agent-assisted |
| `.gtt/docs/usage.md` / `.gtt/docs/usage.es.md` | The normal development loop, change requests, freeze model |
| `.gtt/docs/method-plans.md` / `.gtt/docs/method-plans.es.md` | The four Method Plans in plain words: what GTT does without asking, what the human is asked, what each plan requires, and where each rule is enforced |
| `.gtt/docs/docs.md` | Governance model, layers, enforcement planes, Claude Code vs Kiro vs Codex vs Copilot, upgrading from GTT v1 |

---

## The one flow that matters

```
gtt-domain/change-request.md  ->  gtt-domain/proposals/  ->  your review + `bash apply-*.sh`  ->  gtt-domain/adr/ + gtt-domain/context/stack.md
   you state intent      agent drafts        the Human Promotion Boundary       blocked for agents
   always writable       agent writable      your terminal, your decision
```

The promotion script the
agent stages in `gtt-domain/proposals/` is never run by the agent — see `AGENTS.md` →
*Human Promotion Boundary*. Everything else in this kit exists to make that
flow cheap to run and hard to skip.

---

## What loads on every single session

Only these. Everything else is on demand or never.

- `.claude/CLAUDE.md` (or `AGENTS.md` on Kiro, Codex, and Copilot)
- `AGENTS.md`
- `gtt-domain/context/constraints.md`

If `/context` shows anything else from `gtt-domain/context/`, `gtt-domain/adr/`, or `.gtt/docs/`, something
is importing more than it should.
