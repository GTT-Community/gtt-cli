# GTT — Reference Docs

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

This document is for humans. It is deliberately outside the agent's context
window: an agent does not need to understand GTT to comply with it, and every
token spent explaining the methodology is a token not spent on the problem.

Three topics, one file — methodology, tool portability, and upgrading from v1.
None of it loads automatically in any tool, so merging costs nothing and saves
a file.

- [Methodology](#methodology) — the model itself: layers, enforcement planes, the change flow
- [Portability: Claude Code, Kiro, Codex, Copilot](#portability-claude-code-kiro-codex-copilot) — what each tool enforces and how to adapt
- [Migrating from GTT v1](#migrating-from-gtt-v1) — file mapping and upgrade steps

---

## Methodology

> When context doesn't govern AI, AI governs the solution.

### The premise

AI accelerates implementation. Humans govern context and architecture.

The failure mode GTT addresses is not bad code — agents write reasonable code.
It is **architectural drift**: a sequence of individually defensible changes that
collectively move the solution somewhere nobody decided to go. Drift is invisible
at the commit level and only visible at the architecture level, which is exactly
the level nobody is reviewing.

### Context layers

| Layer | Contents | Policy | Who edits |
|---|---|---|---|
| L0 | `gtt-domain/context/` — the map, vision, architecture, principles, constraints | propose only | Solution Designer |
| L1 | `gtt-domain/adr/` — accepted decisions | propose with review | Solution Designer |
| L2 | `.gtt/docs/` — diagrams, specifications | editable with review | anyone |
| L3 | `src/`, `tests/`, pipelines, infrastructure code | editable | agents and humans |

The layer determines two things: **who may edit** and **when it loads into
context**. Earlier versions of GTT only defined the first, which is what made
the model expensive — every layer loaded on every session regardless of
relevance.

### The three enforcement planes

GTT's guarantees do not come from asking the agent nicely. They come from
putting each concern in the plane that can actually enforce it.

| Plane | Mechanism | Guarantee | Cost per session |
|---|---|---|---|
| Control | `permissions.deny`, PreToolUse hook | Deterministic | Zero context |
| Build | `.gtt/scripts/gtt-check-stack.sh` in CI | Deterministic, after the fact | Zero context |
| Instruction | `AGENTS.md`, `.claude/rules/` | Probabilistic | Tokens |
| Procedural | `.claude/skills/` | On demand | Zero until invoked |

The rule: **anything that can be enforced in the control plane must not be
written as an instruction.** An instruction is a request the model may decline
under pressure; a deny rule is not. Writing "AI must not modify L0 files" into
context is strictly worse than blocking the write — it costs tokens every
session and holds only probabilistically.

Instructions remain necessary for everything that requires judgment: whether a
change is architectural, whether code contradicts context, whether an
abstraction is warranted. No permission rule can decide those.

The drift detector (`detect-drift.py`, see [Drift detection](#drift-detection))
lives in the control plane too — a deterministic hook, zero tokens per
session — but its output is advisory, not a block. It is not a fifth plane; it
extends the control plane's reach from paths to decisions without making L3
governed territory, since L3 stays free by design.

### What lives where

| Concern | Location | Loads |
|---|---|---|
| Non-negotiable behavioral rules | `AGENTS.md`, imported by `.claude/CLAUDE.md` | always |
| Hard project constraints | `gtt-domain/context/constraints.md`, imported by `.claude/CLAUDE.md` | always |
| Rules for one area of the codebase | `.claude/rules/*.md` with `paths:` | when touching matching files |
| Proposal, ADR, audit procedures | `.claude/skills/*/SKILL.md` | when invoked |
| The stack and architecture map | `gtt-domain/context/stack.md` | when the task needs it |
| Architecture prose, vision, principles | `gtt-domain/context/` | when the task needs them |
| Accepted decisions | `gtt-domain/adr/` | when the task needs them |
| Change requests | `gtt-domain/change-request.md` | never |
| Agent drafts awaiting review | `gtt-domain/proposals/` | never |
| GTTGuard protection registry (derived, never hand-edited) | `.gtt/protection/registry.yaml` | never |
| Session state (derived, never hand-edited) | `gtt-domain/session.md` | never |
| This document | `.gtt/docs/` | never |

### The governed domain (`gtt-domain`)

> **Status — applied.** A layout change within v2.1 moved the Engine to `.gtt/` (with GTT's own documentation in
> `.gtt/docs/`) and gathered everything the method governs under `gtt-domain/`. Before it,
> the Engine was formerly `gtt/`, and the governed state and the documentation sat at the project root.


GTT-Method separates **how the method works** from **what the method is applied to**:

| | `.gtt/` — the Engine | `gtt-domain/` — the domain |
|---|---|---|
| Is | GTT-Method's own machinery and its documentation, plus the state it derives — none of it is architectural authority | GTT-Method's definition of the project or domain it governs |
| Holds | `index/`, `protection/`, `scaffold/`, `scripts/`, `session-adapters/`, `.gtt/docs/` | `gtt-domain/context/`, `gtt-domain/adr/`, `gtt-domain/proposals/`, `gtt-domain/backlog.md`, `gtt-domain/change-request.md`, `gtt-domain/session.md`, `gtt-domain/.frozen` |
| Answers | *How do identity, protection, validation and session memory work, and how is GTT used?* | *What is this system, what was decided, and what may change?* |

GTT's own documentation (this file, the installation and usage guides, the evidence and completion
records, the session-adapter contract) belongs to the **Engine**, in `.gtt/docs/`, not to the domain:
it explains the method, it does not define the project.

**`gtt-domain/` is the contextual space over which GTT applies the method.** Context is the source of
truth (ADR-001): `gtt-domain/context/` (L0) says what the system is, `gtt-domain/adr/` (L1) records why it is that way,
`gtt-domain/proposals/` holds what is waiting for a human decision, `gtt-domain/backlog.md` is the development line, and
`gtt-domain/.frozen` marks the moment the domain became governed. The domain is therefore not "documentation
about GTT" — it is the governed subject of the method, the one place an agent is told to read before
acting and forbidden to rewrite once frozen.

Two rules follow, and they are what the split exists to keep true:

- **Architectural authority stays in the domain, and executable GTT logic stays in the Engine.**
  Authority (L0/L1) lives only in `gtt-domain/context/` and `gtt-domain/adr/`; nothing in `.gtt/` is a
  governed decision — its index and protection registry are derived, and its documentation is a
  reference. The domain holds no executable GTT logic *in operation*: `gtt-domain/proposals/` holds
  governed **drafts** — proposals, ADR drafts, staged context files and promotion scripts — that the
  domain never executes and an agent never runs (a human does, under the Human Promotion Boundary).
  `AGENTS.md` and the ADE overlay are instructions to agents, and belong to neither group.
- **The code the project builds (L3: `src/`, `tests/`, infrastructure) belongs to neither.** The domain
  governs it by reference — the architecture map, the dependency rules and the drift signals — and
  never contains it.

For an adopting project the split also reduces GTT's footprint at the project root to the two names
`.gtt/` and `gtt-domain/` (plus the entry points `AGENTS.md`, `readme-gtt*.md`, `SOURCE-BRIEF.*` and the
ADE overlay), so a project's own `docs/`, `context/`, `adr/` or `proposals/` directories no longer collide
with GTT's.

### The change flow

Governance fails when the compliant path is harder than the workaround. GTT
therefore has exactly one entry point for change, and it is a plain markdown
file that is always in the same place.

```
gtt-domain/change-request.md  ->  gtt-domain/proposals/  ->  Designer reviews + runs script  ->  gtt-domain/adr/ + gtt-domain/context/
     Designer states          agent drafts a           bash apply-ADR-NNN-*.sh            applied
     intent, 4 lines          promotion package
```

The asymmetry is the mechanism: `gtt-domain/proposals/` is the only governed directory
an agent can write to. An agent that wants to change the architecture
has exactly one move available — write a reviewable draft. There is no path
where it edits the architecture and no path where it silently skips review,
because the alternative is blocked at the permission layer rather than
discouraged in prose.

The last step is the **Human Promotion Boundary**: the agent's draft, however
complete, is not the change. It becomes one only when the Designer reads the
generated `apply-ADR-NNN-<slug>.sh` and runs it themselves. The agent may
prepare a governed change — analyze it, draft the proposal, draft the ADR,
draft the affected context files, generate the script; it may never promote
one by running that script, editing `gtt-domain/adr/` or `gtt-domain/context/` directly,
or treating its own draft as approval. See `AGENTS.md` → *Human Promotion
Boundary*.

This also removes the friction that kills governance models in practice. The
Designer does not need to remember which of six files to edit, how to format
an ADR, or reconstruct a sequence of commands by hand. They write four lines
in one known location, then review and run one script.

### Governance model

**Human approval required for:** architectural direction, paradigm, module
boundaries, integration strategy, deployment strategy, data model, frameworks,
runtimes, cloud platform and managed services, and any change to L0.

**Agents may:** read and analyze context, detect inconsistencies, propose
changes, draft ADRs and their promotion scripts, and generate implementation
aligned with the governed context.

**The golden rule:** an agent may suggest, analyze, and accelerate. It may not
redefine architecture without explicit approval from the Solution Designer,
and it may not execute a promotion script even after that approval — the
Solution Designer runs it. See *Human Promotion Boundary* in `AGENTS.md`.

**Agent roles and the evidence boundary.** GTT names three responsibilities
that no skill or script may collapse into one — producing output, of any
kind, never grants an agent decision authority:

| Role | Does | Never does | This repo's mechanism |
|---|---|---|---|
| Grounding | Retrieves and exposes governed evidence faithfully | Decide architecture | `gtt-bootstrap`, `gtt-audit` reading `gtt-domain/context/`, `gtt-domain/adr/`, `gtt-domain/backlog.md` |
| Reasoning | Analyzes evidence, drafts gtt-domain/proposals/ADRs/session state | Authorize its own draft | `gtt-propose-change`, `gtt-adr`, `gtt-drift-response` |
| Validation | Runs deterministic structural checks | Make an architectural judgment | `.gtt/scripts/gtt-check-*.sh`, `gtt-validate.sh` |

The evidence boundary is the chain those roles sit on: **Sources → Grounding
→ Evidence Dossier → Reasoning → Proposal → Human Decision → Freeze.** This
is not a new mechanism — it is the change-flow diagram above, named. A
reasoning step (drafting a proposal or an ADR) must never receive raw
sources "for context" in a way that bypasses grounding: it reasons over the
governed context and backlog as read, not over arbitrary source material an
agent decided was relevant. Where the repository cannot enforce this
mechanically (nothing stops a skill's prompt from pasting in extra text),
it is stated here as the operational contract instead — the same honest
gap already documented for the two-regime condition and for GTTGuard on
non-Claude-Code adapters below: an instruction-plane rule, not a claimed
guarantee the tooling doesn't actually have.

### Two regimes

`gtt-domain/context/`, `gtt-domain/adr/`, and `SOURCE-BRIEF.*` are not protected
unconditionally — they are protected only once something has actually been
ratified. Enforcement is split into two regimes, discriminated by
the marker file `gtt-domain/.frozen`:

| Regime | Condition | Those paths |
|---|---|---|
| Pre-freeze | `gtt-domain/.frozen` absent | writable by the agent |
| Governed | `gtt-domain/.frozen` present | denied |

This exists because the alternative — one unconditional deny — is wrong at the
moment a project is created, when the six context files are still template
placeholders and there is nothing ratified to protect yet. `gtt-bootstrap`
writes them directly in this window; the Solution Designer then runs
`.gtt/scripts/gtt-freeze.sh`, which refuses to ratify placeholder content and
writes the marker only once L0 is real. From that point the project is
governed and those paths are read-only for agents again.

Governance machinery — `AGENTS.md`, `.claude/settings.json`, `.claude/hooks/`,
`gtt-domain/change-request.md`, the Kiro equivalents, and the marker itself — has no
regime exception. It is denied in both regimes, always: an agent never drafts
its own directives or its own enforcement.

### The map

`gtt-domain/context/stack.md` is the one page that answers "what is this system" in a
single screen: the stack table, the component map, the deployment topology, the
observability view, the dependency rules, and the change log. It is written in Markdown and Mermaid, so it renders in
GitHub and any IDE without an image to regenerate and without a diagram tool to
keep licensed.

It is governed at L0 and updated in the same change as the ADR that approves the
architectural decision. Three mechanisms hold that line, in descending strength:

1. **CI gate** — `.gtt/scripts/gtt-check-stack.sh` fails the build when an ADR
   changes and the map does not.
2. **ADR procedure** — the `gtt-adr` skill requires a stack map delta section
   with before/after rows; an ADR without one is incomplete.
3. **Audit** — the `gtt-audit` skill checks each map claim against dependency
   manifests, the real import graph, and deployment definitions.

A row in "Stack at a glance" with no ADR in its "Locked by" column is a finding
in its own right: a decision that entered the system without passing through
governance.

### Context freshness

A stale context file is worse than a missing one — it produces confident,
consistent, wrong behavior across every agent in the project.

Context freshness is therefore an operational responsibility, not a
documentation chore. Run the `gtt-audit` skill before releases and after large
merges. When implementation and context diverge, the Solution Designer decides
which one is wrong; the agent is not permitted to assume the code is right.

### Drift detection

The control plane above protects paths, L3 is free by design, and a change
there can contradict a ratified decision without touching any denied path —
editing `docker-compose.yml` can silently override a datastore decision locked
in `stack.md`. Drift detection extends the control plane from paths to
decisions without making L3 governed territory.

**One engine, two triggers.** A `gtt-drift-signals` block declared inside
`gtt-domain/context/stack.md` (L0, human-edited only) names the paths outside the
governed tree that carry architectural weight. A single skill,
`gtt-drift-response`, is the only path from a detected divergence to a
proposal draft — it never runs unratified.

1. **Write-time.** `detect-drift.py`, a `PostToolUse` hook, compares each
   write against the signals block and warns through stderr — advisory, never
   blocking, deduplicated per session so a category warns once per session
   rather than once ever.
2. **Sweep.** `gtt-audit` reads the same signals block and sweeps every
   matching file, not just what changed this session, then hands any
   divergence to `gtt-drift-response` instead of drafting in its own format.

Operates only under the governed regime — pre-freeze there is nothing ratified
yet to contradict. Shell mutations of L3 are not detected by the hook (its
`tool_input` is a command string, not a path); the CI gate is the net for
that case, same as it is for the regime-conditional write block above.

### Backlog governance

`gtt-domain/backlog.md` is the development line: Epics, Stories, current focus,
and next work. It is a development-planning artifact, not architecture, and
it must never become a second source of truth beside `gtt-domain/context/`.

**Precedence:**

```
Governed Context / L0
        v
ADR / governed decisions
        v
gtt-domain/backlog.md
        v
Implementation work
```

A Story that contradicts governed context or an accepted ADR is a finding,
not a resolution. It is surfaced through the normal change process, the same
way architectural drift is — a Story never silently overrides architecture.

**Two kinds of backlog change, two very different bars:**

| Change | Governed how |
|---|---|
| New/removed Epic or Story, or a material scope/acceptance-criteria change | `gtt-domain/change-request.md` → `gtt-domain/proposals/` → Solution Designer decision (`gtt-propose-change`, form 4) |
| Story status, *Current Focus*, *Next Work*, *Blocked* updates during already-approved implementation | Direct edit — routine implementation, not a governed decision |

This mirrors the routine-implementation carve-out GTT already applies to
architecture ("routine implementation does not require a change request") —
extended to development-line tracking instead of invented as a separate
rule. The distinction between "material" and "routine" requires judgment a
hook cannot make deterministically, so — deliberately, unlike `gtt-domain/context/`
and `gtt-domain/adr/` — `gtt-domain/backlog.md` is **not** in `permissions.deny` or
blocked by `protect-l0.py`. Its protection is instruction-plane (`AGENTS.md`,
the `gtt-propose-change` and `gtt-audit` skills) plus one deterministic
backstop: `.gtt/scripts/gtt-check-backlog.sh` fails the build on duplicate
Epic/Story IDs or a status value outside the agreed vocabulary, and warns on
an Epic with no Stories yet. What it cannot check — whether an Epic/Story is
real, current, and actually reflects the work being done, or whether a
structural change actually went through `gtt-domain/change-request.md` — is the
`gtt-audit` *Backlog reconciliation* pass's job, not the script's.

**Reconciling existing Epics/Stories.** When Epics or Stories are already
defined somewhere (a requirements doc, an issue tracker, prior conversation)
but not yet reflected in `gtt-domain/backlog.md`, that is a gap to close through
`gtt-propose-change`, not something to silently ignore or silently rewrite
the backlog to match. When none are defined at all, say so explicitly and
ask whether the development line should be defined — never invent business
Epics/Stories and present them as user-defined requirements; a proposed one
stays labeled `Status: Proposed` until accepted.

**Why not just another ADR-governed file?** An ADR records a decision that,
once made, rarely changes shape again. A Story is expected to move through
statuses constantly as normal work happens — routing every status flip
through `gtt-domain/change-request.md` would make the backlog too expensive to keep
current, and a stale backlog is worse than no backlog (same failure mode as
stale context). The bar is calibrated to what actually needs a human
decision: *what* the project commits to building, not *how far along* it is.

### Provenance, gaps, sources and working agreements

Recovered from CDAD as semantics, not as a parallel system: they reuse the governed artifacts, the
index stays derived and no new authority appears. The rules are in `AGENTS.md` → *Provenance, gaps,
sources and working agreements*. In short: `[FUENTE: ref]` / `[VACÍO: GAP-id]` /
`[CONFLICTO: a vs b]` are allowed in governed context, `[PROPUESTA]` is not; a BLOCKING gap refuses freeze
and an OPEN gap (always scoped) does not and authorises nothing; sources carry authority and an
unambiguous precedence that orders a conflict without erasing it; working agreements sit below governed
context. The one gate is `.gtt/scripts/gtt-check-provenance.sh` (sub-checks: `tags`, `gaps`, `sources`,
`preferences`); `gtt-status.sh` prints the counts and `gtt-query.sh --governance open|blocking|resolved|conflicts|sources|agreements`
lists them from the artifacts. Limits: the checks are lexical and structural — they cannot judge whether a
`[FUENTE]` really supports a claim or whether a scope is wise; that stays a human judgment.

### GTTGuard: protected artifacts

`gtt-domain/context/` and `gtt-domain/adr/` govern architecture. `gtt-domain/backlog.md` governs
the development line. GTTGuard governs neither — it is a **sibling**
mechanism that lets a developer flag an individual file, class, or method
so an agent may read and propose a change to it, but never modify it
autonomously. Conflating it with L0/L1 or with the Human Promotion Boundary
below is a mistake this document exists to prevent.

**The marker.** A developer adds `@GTTGuard` (Java, Python), `[GTTGuard]`
(C#), or `// @GTTGuard` / `# @GTTGuard` (comment-based languages, e.g.
JS/TS) immediately above a declaration — or as the file's first line to
protect the whole file — optionally with `reason="..."` and
`source="..."`. Placing or removing the marker is an ordinary L3 edit: it
*is* the developer's proposal ("User proposes → GTT implements → Agent
respects"), not a protected change in its own right.

**The registry is derived, not authored.** `.gtt/scripts/gtt-guard-sync.sh`
scans source for markers, resolves each one's file/class/method
deterministically — brace-balance matching for Java/C#/JS/TS, indentation
matching for Python, never an LLM's judgment — and regenerates
`.gtt/protection/registry.yaml` in full. This is the same trust model as a
lockfile: the registry is committed for reviewability, but hand-editing it
is self-defeating, because `.gtt/scripts/gtt-check-protection.sh` fails the
build the moment the committed file no longer matches a fresh regeneration
from source. No write-block is needed on the file itself; the drift check
makes tampering with it pointless rather than merely forbidden.

**Real-time enforcement (Claude Code).** `.claude/hooks/protect-guard.py`,
a `PreToolUse` hook registered alongside `protect-l0.py`, blocks an
autonomous edit to a `HUMAN_APPROVAL` entry. A file-scope entry blocks the
whole file, the same way `protect-l0.py` blocks a machinery path. A
class/method-scope entry resolves that symbol's exact current line span
*live against the file on disk* — never a cached value, so it can never go
stale — and blocks only an edit that overlaps it: an unprotected sibling
method in the same file stays freely editable. Whenever that resolution is
ambiguous, the hook fails safe to blocking the whole file rather than
risking a silent bypass. A mutating-looking Bash command naming a
protected file is always blocked outright, for the same reason
`protect-l0.py` takes no chances with shell commands against machinery:
there is no way to inspect a shell command's line-range effect on a file.

**Deterministic validation.** `gtt-check-protection.sh` is the CI gate:
registry-drift, artifact/symbol resolution, protection-value validity, and
`source: "ADR-NNN"` existence are all checked unconditionally; a protected
artifact that changed in the diff must also be accompanied by a change
under `gtt-domain/proposals/` or `gtt-domain/adr/`, or the build fails. This is the one
real enforcement layer on Kiro, Codex, and GitHub Copilot, none of which
has a real-time equivalent — the same honest limitation already documented
for the two-regime `gtt-domain/context/`/`gtt-domain/adr/` condition below. GTTGuard
never claims a guarantee an ADE does not actually provide.

**Changing a protected artifact.** `gtt-propose-change` gets a fifth form
for exactly this. Once the Solution Designer approves it **in
conversation**, the agent implements the change directly, updates or
removes the marker, and re-syncs the registry — no ADR, no `apply-*.sh`
script. This is deliberately lighter than the Human Promotion Boundary:
GTTGuard protects L3 code a developer opted into protecting, not governed
context, and is deliberately not a second, unrelated approval model — it
reuses the existing proposal mechanism, nothing heavier.

### Status and validation

`.gtt/scripts/gtt-status.sh` derives a deterministic snapshot from
repository artifacts — freeze state, Stories currently
`In Progress`/`Blocked`, pending files under `gtt-domain/proposals/`, whether
`gtt-domain/change-request.md` has been filled in, every ADR and its status, and
the GTTGuard protected-artifact count, artifact identity/index health, and
repository resume hints (branch, uncommitted count, recent commits) — and writes it to `gtt-domain/session.md`.
That file is a **derived artifact**, the same trust model as
`.gtt/protection/registry.yaml`: never hand-edited, safe to regenerate at
any time, and never loaded automatically by any agent. It exists so a
project resumes the same way regardless of which ADE's session picks it up
next — session continuity that does not depend on any tool's private
conversation memory. It is operational context only: never architectural
authority, never evidence, never a substitute for an ADR or decision
record, and no agent may invent it from memory instead of running the
script.

`.gtt/scripts/gtt-validate.sh` runs the existing deterministic check scripts
(`gtt-check-backlog.sh`, `gtt-check-adapter.sh`, `gtt-check-protection.sh`,
`gtt-check-stack.sh`, `gtt-check-markdown.sh`, `gtt-check-integrity.sh`) in sequence and reports PASS / FAIL /
CANNOT-DETERMINE per check — it does not reimplement any of their logic,
only aggregates it. The adapter check has two regimes. When `.gtt/ade.json`
exists it validates every participating ADE against it: a missing or
inconsistent integration FAILS, and a detected ADE that does not participate is
a WARN, never a pass. Without it, the check is skipped, not failed, when zero or
more than one adapter is present, since this source repository is the
catalog and legitimately ships every adapter — that is not the "wrong
adapter installed" violation the check exists to catch on an installed
project. `gtt-status.sh` also reports the ADE integration state (Primary,
participating, integration health) in `gtt-domain/session.md`; it records no
"last active ADE", because several ADEs may run at once and no deterministic
source for it exists.

Neither script makes an architectural judgment; both are read-only.

### Artifact identity and the technical index

GTT treats the repository as a set of *identifiable, related, traceable*
artifacts, not a pile of Markdown files. Paths may change; identity,
traceability, and provenance must not be lost.

| File | Role | Authority |
|---|---|---|
| `.gtt/index/artifacts.json` | **Identity manifest**: stable `id`, `type`, current `path`, `history` (former paths), `aliases` (intentional copies) | Authoritative for identity — changed by `gtt-index.sh` / `gtt-reconcile.sh`, reviewed in the diff |
| `.gtt/index/technical-index.json` | **Technical index**: documents, sections (heading, anchor, line span), concepts, references / referenced-by, `supersedes`, provenance, authority, version, content hash | **Derived accelerator.** Rebuildable from the Markdown; never a source of truth; a stale one fails the build |

Ids are path-independent: `ADR-007` is `ADR-007` in `gtt-domain/adr/` or
`gtt/architecture/adr/`. Reference an artifact as `[[ADR-007]]` and the
reference survives any move; relative Markdown links are reconciled
automatically when a move is applied. Code spans and fenced blocks never
count as references.

| Script | Does |
|---|---|
| `gtt-index.sh` | Registers new artifacts, rebuilds the index. Refuses while a registered path is missing (an unreconciled move) so a move is never silently recorded as delete + create |
| `gtt-reconcile.sh [--apply] [--map OLD=NEW] [--retire ID]` | Detects moves (git rename hint → same identity → content similarity), records old path as history, rewrites relative links. Dry-run by default; never rewrites a frozen `gtt-domain/context/`/`gtt-domain/adr/` file, it reports it |
| `gtt-check-integrity.sh` | Fails on unreconciled moves, unregistered artifacts, duplicate logical identity, broken links, links via an old path, unresolved `[[ID]]`, stale index. Part of `gtt-validate.sh` |
| `gtt-query.sh <term \| ID[#anchor]> [--show] [--deep]` | Section-level retrieval: locate via the index, read only the needed lines from the Markdown |

Scope is the GTT kit (the same set `gtt-check-markdown.sh` covers);
`gtt-domain/session.md` is derived and excluded. Git is a rename *hint* only —
correctness never depends on it, and GTT does not replace Git.

An agent never owns identity: it works through these scripts and contracts.
A move of a governed file remains a human/governed act; this tooling only
makes the aftermath deterministic.

### Session memory boundary

`gtt-domain/session.md` (see *Status and validation*) is GTT's session-memory
service. Its boundary:

| Kind | Where | Authority | Agent may write? |
|---|---|---|---|
| Governed project state | `gtt-domain/context/`, `gtt-domain/adr/`, `gtt-domain/backlog.md` | Authoritative | Only via the change flow |
| Grounding / evidence | governed artifacts + technical index (as a locator) | Evidence | No — read faithfully |
| Session memory | `gtt-domain/session.md` | **None** — operational, derived | Regenerated by script only |
| Working preferences | `gtt-domain/working-agreements.md` (team), `.gtt/local/preferences.md` (user, local) | Below governed context | Never self-authored |
| ADE-private memory | the ADE's own store | None to GTT | Out of GTT's scope |

Session memory is reconstructed, never remembered: it is derived from the
freeze marker, backlog, proposals, ADRs, the identity/index state, and Git
(branch, uncommitted count, recent commits) by `gtt-status.sh`. Delete it
and regenerate it — nothing is lost. It is outside the grounding corpus and
`gtt-check-markdown.sh`/the index skip it. The service has one ADE-independent
entry point, and each ADE gets a thin adapter:

```text
GTT Core
   ├── Artifact Identity
   ├── Technical Index
   └── Session Memory Service
          └── .gtt/scripts/gtt-session-context.sh   (ADE-independent; the Core ends here)
                 │
                 └── ADE Adapter Contract          (.gtt/docs/session-adapter-contract.md)
                        ├── Claude Code adapter    installed
                        ├── Codex adapter          staged (Draft)
                        ├── GitHub Copilot adapter staged (Draft)
                        └── Kiro adapter           staged (Draft)
```

> **GTT Core is ADE-agnostic. ADE adapters are integration-specific.**
> Claude Code is one adapter, not part of the Core.

`gtt-session-context.sh` resolves Python (same probe as the other GTT
wrappers), regenerates `gtt-domain/session.md`, verifies it, and prints a payload
headed *operational-only; NOT authority, NOT evidence, NOT a decision
record, NOT a grounding source*. It fails with a real exit code and a
message rather than returning partial context. An adapter only translates
that stdout for its ADE and must not contain GTT logic or hide failures.
The Claude Code adapter is a `SessionStart` hook (warm start on new or
resumed sessions — never `UserPromptSubmit`) that emits the payload as
`additionalContext`; hook commands resolve Python through
`.gtt/scripts/gtt-run-python.sh`, never `python3 X || python X`.

#### Session Memory adapter matrix

**Status: Draft — not a decision.** Full contract:
`.gtt/docs/session-adapter-contract.md`. Each adapter is declared in
`.gtt/session-adapters/<ade>.json`; `gtt-check-session-adapter.sh <ade>` (run by
`gtt-validate.sh`) checks it statically and never runs the ADE. Coverage:
**N1** native session injection · **N2** ADE-loaded session file · **N3**
instruction-based retrieval.

| ADE | Adapter (native path) | Event | Level | Implemented | Static check | Runtime |
|---|---|---|---|---|---|---|
| Claude Code | `.claude/hooks/session-start.py` | `SessionStart` | N1 | installed | PASS | **RUNTIME VERIFIED** — startup and resume both fired the hook (`session_source` observed) and the agent quoted the payload |
| Codex | `.codex/hooks.json` + `.codex/gtt-session-start.sh` | `SessionStart` | N1 | staged | PASS | **NOT VERIFIED** — in a sandbox the hook never executed (documented hook-approval requirement; bypass flag deliberately not used) |
| GitHub Copilot | `.github/hooks/gtt-session.json` + `gtt-session-start.py` | `sessionStart` | N1 (new interactive CLI sessions only) | staged | PASS | **VERIFICATION REQUIRES INTERACTIVE SESSION** |
| Kiro | `.kiro/agents/gtt-session.json` + `.kiro/gtt-session-start.sh` | `agentSpawn` (CLI) | N1 (CLI only, docs disagree) | staged | PASS | **STATICALLY VALIDATED, NOT RUNTIME VERIFIED** — not installed here |

Events per adapter (`verified` = observed at runtime, `documented` = vendor
docs only, `unknown`/`unsupported` as stated by the vendor):

| ADE | startup | resume | clear | compact | fork |
|---|---|---|---|---|---|
| Claude Code | verified | verified (hook fires; delivery not separable from the resumed transcript) | documented | documented | documented |
| Codex | documented | documented | documented | documented | unknown |
| GitHub Copilot | documented | **unsupported** (documented not to fire) | unknown | unknown | unknown |
| Kiro | documented | unknown | unknown | unknown | unknown |

"Static check PASS" means the declaration, registration, service reference,
absence of duplicated logic and silenced errors, the command's output
(markers + declared format) and its visible failure path passed in a sandbox
laid out as installed. It does **not** mean the ADE loads the context.

Known documentation gaps: Kiro's docs disagree on whether `agentSpawn` stdout
reaches the agent and show three hook schemas (this repo's existing
`.kiro/hooks/detect-drift.json` is a fourth shape, unverified). The
compatibility matrix earlier in this document says Copilot has no
programmatic hook; Copilot now documents hooks (`sessionStart`, and per its
reference others), so that row needs re-verification.

### Capability status

Not every mechanism gtt-bootstrap implements is a Canon *requirement* —
some are this tool's particular way of satisfying one. Confusing "exists in
this repo" with "the Canon demands exactly this" is the mistake this table
exists to prevent:

| Mechanism | Status | Note |
|---|---|---|
| Governed context (L0/L1), freeze | CANONICAL | The Canon's core model; not optional |
| Change flow (`gtt-domain/change-request.md` → `gtt-domain/proposals/` → ADR) | CANONICAL | The Canon's one entry point for change |
| Human Promotion Boundary | CANONICAL | Agent prepares, human promotes — non-negotiable |
| Agent roles (Grounding/Reasoning/Validation) | CANONICAL | Named vocabulary from the Canon; mapped onto existing skills here, not a new framework |
| GTTGuard | IMPLEMENTED | A Canon capability; this repo's marker syntax, registry format, and Bash-command heuristic are implementation choices, not canonical requirements |
| Backlog governance (`gtt-domain/backlog.md`) | IMPLEMENTED | Development-line tracking; the Canon requires the precedence rule, not this exact file format |
| Drift detection | IMPLEMENTED | Extends the control plane from paths to decisions; this repo's `gtt-drift-signals` block mechanism is one way to do it |
| Session continuity (`gtt-status.sh` / `gtt-domain/session.md`) | IMPLEMENTED | Satisfies the Canon's ADE-independence requirement for resuming work; the snapshot format is this repo's choice |
| `gtt status` / `gtt validate` | IMPLEMENTED | Deterministic aggregation the Canon asks for; implemented here as bash scripts because that is what this repo already uses, not because the Canon mandates a shell script |
| Artifact identity, integrity & technical index (`.gtt/index/`) | IMPLEMENTED | Canon v2.1 capability; JSON manifest/index, `[[ID]]` reference syntax, and the reconcile heuristics are this repo's choices. The index is derived, never authoritative |
| Working preferences (separate from session state) | IMPLEMENTED | Team file and local user file, both below governed context; `gtt-check-provenance.sh` rejects one that tries to override it. Not runtime-verified outside the rehearsal |
| Provenance, OPEN/BLOCKING gaps, source manifest | IMPLEMENTED | One gate (`gtt-check-provenance.sh`), the gap register in `stack.md`, `gtt-domain/context/sources.md`; wired into `gtt-validate.sh`, `gtt-freeze.sh`, `gtt-status.sh` and `gtt-query.sh --governance`. Optional per project; rehearsed, not run on an installed project |
| Bootstrap 1.0 CLI contracts (`.gtt/contract/`, `gtt-contract.sh`, `gtt-project.sh`) | IMPLEMENTED | Release identity, compatibility negotiation, capability and operation registries, Method Plans (methodology profiles), export policy, recovery, structured status/session/validation; see `.gtt/docs/bootstrap-contract.md`. Rehearsed on disposable copies by `.gtt/tests/bootstrap-acceptance.py`; not run against a real CLI, on Linux/macOS or on an installed project |
| RAG/vector-backed grounding | NOT IMPLEMENTED | Not required by the Canon or this repo's directives; would need its own proposal if ever needed |

### Operational boundary

GTT does not slow implementation down. It prevents accidental architectural
change. Everything under L3 stays fully editable, and the majority of day-to-day
work never touches the governance path at all.

If GTT is producing friction on routine work, the constraints are written too
broadly — narrow them rather than working around them.

---

## Portability: Claude Code, Kiro, Codex, Copilot

GTT v2 separates **content** from **mechanism**. The governed context
(`gtt-domain/context/`, `gtt-domain/adr/`) is plain markdown and is fully portable. What differs per tool is how
that content is loaded and how the L0 protection is enforced.

Nothing in the Engine or the governed domain needs to change to move
between tools. Only the ADE overlay does.
A project installs the adapter of every ADE its human chose to have
participate, with exactly one of them declared Primary — see the ADE adapter
matrix in `README.md` and in `.claude/skills/gtt-bootstrap/SKILL.md` (step 0).
Detection is not participation: an ADE found on the machine or in the repository
is a candidate until the human chooses it. The Primary ADE is a workflow
identifier; it holds no authority over any governed artifact, and instruction
files or overlays (`AGENTS.md`, `.claude/`, `.kiro/`, `.copilot/`, `.codex/`, an
ADE's memory) are integration surfaces, never governance.

### Compatibility matrix

| Capability | Claude Code | Kiro | Codex | GitHub Copilot |
|---|---|---|---|---|
| Always-loaded instructions | `.claude/CLAUDE.md` | `AGENTS.md`, or steering `inclusion: always` | `AGENTS.md` | `AGENTS.md` + `.github/copilot-instructions.md`* |
| Reads `AGENTS.md` natively | no — imports it | yes | yes | yes |
| Path-scoped rules | `.claude/rules/` + `paths:` | `.kiro/steering/` + `inclusion: fileMatch` | nested `AGENTS.md` only | none documented |
| On-demand procedures | Skills | steering `inclusion: manual` / `auto` | prompt or custom command | prompt |
| Declarative file-write blocking | `permissions.deny` | not equivalent | `[permissions.*.filesystem]` globs | not equivalent |
| Programmatic pre-tool block | PreToolUse hook | agent hooks (different model) | hooks / sandbox | none documented |
| Governed context in `gtt-domain/context/` and `gtt-domain/adr/` | works | works | works | works |
| CI gate (`.gtt/scripts/`) | works | works | works | works |
| GTTGuard real-time block | yes — `protect-guard.py` | no — CI gate only | no — CI gate only | no — CI gate only |

\* GTT's Copilot adapter file actually lives at `.copilot/copilot-instructions.md`
(naming consistency with `.claude/`/`.kiro/`), so Copilot does not load it
automatically — see the note under *GitHub Copilot* below.

**Short version:** Claude Code runs everything. Kiro runs everything except the
deterministic write block, which it approximates. Codex runs the content and the
write block, but loses conditional loading — its instruction file is
all-or-nothing. GitHub Copilot is the thinnest adapter: content and the CI
gate work, with neither conditional loading nor a deterministic write block —
GTT does not claim Copilot capabilities beyond what current GitHub
documentation actually supports (`.github/copilot-instructions.md` for
repository-wide instructions, `AGENTS.md` for agent instructions) — including
the fact that GTT's own adapter file does not live at that path by default.

### Claude Code

Native target. `.claude/CLAUDE.md` imports `AGENTS.md` and adds the
Claude-specific layer: skill routing and a note that permission denials are by
design.

Verify with `/context`: only `.claude/CLAUDE.md`, `AGENTS.md`, and
`constraints.md` should appear under memory files.

`.claude/hooks/protect-guard.py` is the only adapter with a real-time
GTTGuard block, resolving each protected symbol's span live against the
file on disk on every `Write`/`Edit`/`NotebookEdit`/`Bash` attempt.

### Kiro

Kiro reads `AGENTS.md` from the workspace root automatically, so the portable
core loads with no adapter at all. Note that `AGENTS.md` in Kiro does not
support inclusion modes — it is always included.

The path-scoped rules are mirrored in `.kiro/steering/` using
`inclusion: fileMatch` with a `fileMatchPattern`. Kiro accepts one pattern per
file, so a rule covering several globs becomes several steering files.

Since Kiro IDE 1.0, declarative permissions exist in `.kiro/permissions.yaml`
(`deny` wins over `ask`/`allow`), covering the unconditional machinery paths —
`AGENTS.md`, `.claude/**`, `.kiro/settings|steering|hooks/**`,
`gtt-domain/.frozen` — the same way `permissions.deny` does for Claude Code.

What `permissions.yaml` cannot express is the two-regime condition on
`gtt-domain/context/` and `gtt-domain/adr/`: whether they are writable depends on whether
`gtt-domain/.frozen` exists, and a static config file has no way to test that. Those
two paths still fall back to the CI gate: `.gtt/scripts/gtt-check-stack.sh`
plus a branch rule requiring review on `gtt-domain/context/**` and `gtt-domain/adr/**` catches what reaches a pull
request. (The drift detector, `detect-drift.py`, is different: it is advisory
rather than a write block, so it mirrors cleanly to Kiro as
`.kiro/hooks/detect-drift.json` — see the Drift detection section.)

Known issue: global steering in `~/.kiro/steering/` has had reports of
`fileMatch` not triggering. Keep GTT steering in the workspace, not global.

GTTGuard has the same gap as the two-regime paths: `permissions.yaml` is
static and cannot evaluate `.gtt/protection/registry.yaml`'s dynamic
contents, so there is no real-time block. `.gtt/scripts/gtt-check-protection.sh`
in CI, plus `.kiro/steering/gtt-guard.md`, is the enforcement.

### Codex

Codex reads `AGENTS.md` from the global config directory, the project root, and
nested directories, with more local files taking priority. The portable core
loads with no adapter.

Two adjustments matter:

**Conditional loading does not exist.** Codex has no `paths:` equivalent. The
closest approximation is nested `AGENTS.md` files that apply when Codex works in
that subtree:

```
AGENTS.md              # portable core
src/AGENTS.md          # implementation rules
infra/AGENTS.md        # infrastructure rules
```

This is coarser than path globs but preserves the principle: rules load near the
code they govern rather than all at once.

**Size cap.** Codex caps project docs at `project_doc_max_bytes`, 32 KiB by
default. GTT v2's core is far under that. GTT v1 was not obviously safe.

**Write protection** is available through filesystem permission globs in
`~/.codex/config.toml`:

```toml
[permissions.gtt.filesystem]
"gtt-domain/context/**" = "deny"
"gtt-domain/adr/**" = "deny"
```

Combine with `sandbox_mode` and `writable_roots` for a harder boundary. Verify
against the current Codex config reference — this surface has been changing
quickly.

GTTGuard has no real-time block on Codex either, for the same reason: its
registry is dynamic content a static filesystem glob cannot evaluate.
`gtt-check-protection.sh` in CI is the enforcement, backed by `AGENTS.md`.

### GitHub Copilot

Copilot reads two files per current GitHub documentation: repository-wide
instructions from `.github/copilot-instructions.md`, and agent instructions
from `AGENTS.md`. GTT's adapter file lives at `.copilot/copilot-instructions.md`
instead — a deliberate naming choice, consistent with `.claude/` and
`.kiro/` being named after the tool rather than the platform — which means
Copilot does **not** load it automatically at that path; mirror it to
`.github/copilot-instructions.md` too if automatic loading matters for the
project. The portable core loads through `AGENTS.md` with no other adapter
machinery beyond that single pointer file.

**Conditional loading does not exist.** `.github/copilot-instructions.md` is
repository-wide only — there is no documented path-scoped equivalent to
`paths:` or `inclusion: fileMatch`.

**Write protection** has no declarative or programmatic equivalent in the
Copilot adapter today. `.gtt/scripts/gtt-check-stack.sh` plus a required
review on `gtt-domain/context/**` and `gtt-domain/adr/**` is the only backstop, the same fallback Kiro uses for the
two regime-conditional paths.

The adapter file itself must stay thin: it points at `AGENTS.md` and the governed directories
(`gtt-domain/context/`, `gtt-domain/adr/`) as the canonical source rather than restating GTT methodology, so there is
never a second copy of the rules to drift out of sync with the first.

GTTGuard is no exception to that thinness: Copilot gets `gtt-check-protection.sh`
in CI as its only real enforcement, and one pointer line in the adapter
file back to `AGENTS.md` — never a restatement of the mechanism.

### If your team uses more than one ADE

Bootstrap installs the adapter of every ADE the human chose — see the ADE
adapter matrix. If the team uses Claude Code and Copilot side by side, both are
declared participating (one of them Primary) and both get their overlay:
`gtt-ade.sh install --participating claude,copilot --primary claude`. A later
run adds an ADE only when told to, never drops one, never reintroduces one that
was excluded and never changes the Primary on its own. Real-time blocking exists
only on Claude Code; every other participating ADE is governed by its
instructions plus the CI gate, and the registry's `enforcement:` field says which.

Keep `AGENTS.md` as the single source for the core rules. Never restate a rule
in `.claude/CLAUDE.md` or `.copilot/copilot-instructions.md` that already lives
in `AGENTS.md` — that duplication is exactly the defect v2 was built to
remove.

The path-scoped rule files are the one place duplication is unavoidable,
since `.claude/rules/` and `.kiro/steering/` use incompatible front matter. They
are short and change rarely.

### If you use only one

Bootstrap installs only the adapters of the ADEs the human chose — there is
nothing to prune. If the project moves to a different ADE later, see
*Switching ADE later* in the README for the exact `gtt-ade.sh` commands and the
caveats: removing `.claude/` removes the deterministic enforcement layer, and
Codex/Copilot need nested `AGENTS.md` files to approximate path-scoped rules.

`AGENTS.md` is never deleted — it is the core every tool reads.

---

## Migrating from GTT v1

### What changed and why

v1 was correct as a methodology and expensive as an implementation. Every file
loaded on every session, whether relevant or not, and the same rules were
restated across four files.

| Problem in v1 | Fix in v2 |
|---|---|
| `AGENTS.md` ordered the agent to read 9 files at session start | Nothing is read at startup; content loads when relevant |
| L0 file list repeated 6 times across 4 files | Stated once, in `.claude/CLAUDE.md` |
| `Proposed Architecture Change` template duplicated in 3 files | One copy, in the `gtt-propose-change` skill |
| L0–L3 table in both `governance.md` and `project-context.md` | One copy, in the Methodology section above |
| 518 lines of methodology vs 263 lines of actual project context | Methodology moved out of the context window entirely |
| L0 protection written as prose the model may ignore | `permissions.deny` plus a PreToolUse hook |

Always-loaded context drops from roughly 826 lines to roughly 80.

### File mapping

| v1 | v2 |
|---|---|
| `gtt/AGENTS.md` | `.claude/CLAUDE.md` (rules) + this document's Methodology section (rationale) |
| `gtt/ai-rules.md` | `.claude/CLAUDE.md` + `.claude/skills/gtt-propose-change/` |
| `gtt/governance.md` | Methodology section above |
| `gtt/guardrails.md` | `.claude/settings.json` + `.claude/rules/` |
| `gtt/project-context.md` | Methodology section above |
| `gtt/context/*` | `gtt-domain/context/*` — unchanged in purpose; trimmed and marked read-only (in `gtt-domain/` in the current layout) |
| *(new)* | `gtt-domain/context/stack.md` — the visual stack and architecture map |
| *(new)* | `gtt-domain/change-request.md` — the single entry point for changes (root in v2, under `gtt/` in v2.1, project root after the scaffold restructure, `gtt-domain/` in the current layout) |
| *(new)* | `gtt-domain/proposals/` — agent-writable staging area |
| *(new)* | `.gtt/docs/index.md` — map of every file (root in v2, under `gtt/` in v2.1, `docs/index.md` after the scaffold restructure, `.gtt/docs/index.md` in the current layout) |
| *(new)* | `SOURCE-BRIEF.*` (project root) — the original design document, preserved by `gtt-bootstrap` |
| *(new, v2.1)* | `gtt-domain/backlog.md` — the development line: Epics, Stories, current focus |
| *(new, v2.1)* | `.gtt/docs/gtt-completion.md` — durable bootstrap completion record |
| `gtt-domain/adr/*` | unchanged; template added |

### Scaffold restructure (layout change within v2.1)

The v2.1 scaffold was reorganised so that its layers are separate, and then (below)
gathered into two homes: the Engine in `.gtt/` and the governed domain in `gtt-domain/`. Nothing about
governance semantics changed in either step; only locations (and lowercase names) did. The table lists where
each artifact lives now and where it lived in the v2.1 layout.

| Now | Formerly (before the scaffold restructure) |
|---|---|
| `gtt-domain/context/`, `gtt-domain/adr/`, `gtt-domain/proposals/` | `gtt/context/`, `gtt/adr/`, `gtt/proposals/` | <!-- legacy-layout -->
| `gtt-domain/backlog.md`, `gtt-domain/change-request.md` | `gtt/backlog.md`, `gtt/CHANGE-REQUEST.md` | <!-- legacy-layout -->
| `gtt-domain/session.md`, `gtt-domain/.frozen` | `gtt/SESSION.md`, `gtt/.frozen` | <!-- legacy-layout -->
| `.gtt/docs/index.md`, `.gtt/docs/installation*.md`, `.gtt/docs/usage*.md` | `gtt/INDEX.md`, `gtt/INSTALLATION*.md`, `gtt/USAGE*.md` | <!-- legacy-layout -->
| `.gtt/docs/gtt-completion.md`, `.gtt/docs/evidence.md` | `gtt/GTT-COMPLETION.md`, `gtt/EVIDENCE.md` | <!-- legacy-layout -->
| `.gtt/docs/docs.md`, `.gtt/docs/session-adapter-contract.md` | `gtt/docs/DOCS.md`, `gtt/docs/SESSION-ADAPTER-CONTRACT.md` | <!-- legacy-layout -->
| `readme-gtt.md`, `readme-gtt.es.md` | `README-GTT.md`, `README-GTT.es.md` | <!-- legacy-layout -->
| `.gtt/scaffold/manifest.yaml` | *(new)* the declarative scaffold definition |

`.gtt/scripts/`, `.gtt/index/`, `.gtt/protection/`, and `.gtt/session-adapters/` moved to `.gtt/`
in the layout migration below (formerly `gtt/`). Artifact ids are unchanged — identity follows the artifact, not the path.

### Layout migration to `.gtt/` and `gtt-domain/`

This migration renamed the Engine directory, moved GTT's documentation into it, and gathered the governed
state under one directory. Only locations changed; the freeze marker moved byte for byte.

| Now | Before this migration (the scaffold-restructure layout) |
|---|---|
| `.gtt/index/`, `.gtt/protection/`, `.gtt/scaffold/`, `.gtt/scripts/`, `.gtt/session-adapters/`, `.gtt/README.md` | `gtt/index/`, `gtt/protection/`, `gtt/scaffold/`, `gtt/scripts/`, `gtt/session-adapters/`, `gtt/README.md` | <!-- legacy-layout -->
| `.gtt/docs/` | `docs/` | <!-- legacy-layout -->
| `gtt-domain/context/`, `gtt-domain/adr/`, `gtt-domain/proposals/` | `context/`, `adr/`, `proposals/` | <!-- legacy-layout -->
| `gtt-domain/backlog.md`, `gtt-domain/change-request.md`, `gtt-domain/session.md`, `gtt-domain/.frozen` | `backlog.md`, `change-request.md`, `session.md`, `.frozen` | <!-- legacy-layout -->
| `.gtt/scaffold/manifest.yaml` (layout version 2) | `gtt/scaffold/manifest.yaml` (layout version 1) | <!-- legacy-layout -->

### Steps

1. Copy `.claude/` (includes `.claude/CLAUDE.md`), `.gtt/` (the Engine, including `.gtt/docs/`), and the
   governed-domain skeleton (`gtt-domain/context/`, `gtt-domain/adr/`, `gtt-domain/proposals/`, `gtt-domain/backlog.md`,
   `gtt-domain/change-request.md`) into your project root.
2. Port your real content into the files under `gtt-domain/context/`. Copy from your
   v1 files; the governed content itself did not change.
3. Fill in `gtt-domain/context/stack.md`. This file is new in v2 and has no v1
   equivalent. Set the "Locked by" column to the ADR that made each decision;
   blanks are findings, not omissions.
4. Delete `gtt/AGENTS.md`, `ai-rules.md`, `governance.md`, `guardrails.md`, and
   `project-context.md`.
5. Adjust the `paths:` globs in `.claude/rules/` to match your folder layout.
   They ship with `src/`, `tests/`, `infra/`, `deploy/`.
6. Start a session and run `/context`. Confirm `.claude/CLAUDE.md` appears under
   memory files and that nothing from `.gtt/docs/` or `.claude/skills/` is
   loaded.
7. Wire `.gtt/scripts/gtt-check-stack.sh` into CI against your default branch.
8. Verify the protection holds: ask the agent to edit
   `gtt-domain/context/architecture.md`. It must be blocked, not merely reluctant.

### If you use other agents too

v2 ships this way already: `AGENTS.md` holds the portable core and
`.claude/CLAUDE.md` imports it. Kiro and Codex read `AGENTS.md` natively, so
they pick up the core with no adapter. See the Portability section above for
what each tool does and does not enforce.

If you only ever use Claude Code, you can inline `AGENTS.md` into
`.claude/CLAUDE.md` and delete it — but the indirection costs nothing and keeps
the door open.

### Verifying the token saving

Run `/context` before and after. The `Memory files` section shows what loaded
and what it costs. If you still see files from `gtt-domain/context/`, `gtt-domain/adr/`, or `.gtt/docs/` other than
`constraints.md`, something is importing more than it should.

