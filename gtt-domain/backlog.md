# Project Backlog

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

> The project's living development line — Epics, Stories, and the work
> currently expected to be built. This is a development-planning artifact,
> not architecture: see *Precedence* below. It answers "what exists, what's
> next, what's blocked" — not "how may it be built" or "what decisions are
> authoritative."

**Last verified:** `2026-10-02` (code audited against Story acceptance criteria) — run the `gtt-audit` skill to reconcile against defined requirements.

---

## Development Line

First implementation of the GTT CLI: the seven implementation phases of the master specification
[FUENTE: master] plus the tasks of the three topic documents [FUENTE: ade-sync] [FUENTE: method-plans]
[FUENTE: md-tracking]. The friction rule [FUENTE: friction] is transversal and has no Epic of its own.

## Precedence

```text
Governed Context / L0
        v
ADR / governed decisions
        v
This backlog
        v
Implementation work
```

If a Story appears to contradict governed context or an accepted ADR, that
is a finding, not a resolution. Raise it through `gtt-domain/change-request.md` —
a Story never silently overrides architecture.

---

## Epics

> Accepted by the Solution Designer on 2026-10-02 from `PROPOSAL-initial-backlog.md`. `Completed` means
> implemented with passing tests. Stories accepted on 2026-10-02 from `PROPOSAL-backlog-stories.md`.

### EPIC-001 — Foundation

**Status:** Completed
**Goal:** Go module, Cobra, output, exit codes, process executor, project-root detection.
**Source:** master, phase 1

#### Stories

##### STORY-001 — Go module and Cobra command tree

- **Status:** Done
- **Priority:** High
- **Source:** master 124 phase 1, 126, 127
- **Acceptance Criteria:**
  - Go project builds reproducibly.
  - Cobra tree exists with the v1.0 command surface of master section 35.

##### STORY-002 — Output modes and exit-code contract

- **Status:** Done
- **Priority:** High
- **Source:** master 77, 78
- **Acceptance Criteria:**
  - Human-readable and `--json` output exist for status, inspect, validate, doctor, audit and version.
  - JSON schemas are versioned.
  - Exit codes 0-7 are used consistently as defined in master section 78.

##### STORY-003 — Configuration and logging

- **Status:** Done
- **Priority:** Medium
- **Source:** master 124 phase 1, 80, 81
- **Acceptance Criteria:**
  - Operational configuration is read as master section 81 defines.
  - Logs and state never contain tokens, credentials or environment values.
- **Notes:** Leveled logging to stderr (`--log-level error|warn|info|debug`), values never logged; unit tests and the 44 integration tests pass with it (2026-10-02). Global configuration is environment variables only (`GTT_HOME`, `GTT_BOOTSTRAP_SOURCE`).

##### STORY-004 — Process executor

- **Status:** Done
- **Priority:** High
- **Source:** master 60.4, 73
- **Acceptance Criteria:**
  - The Bootstrap is invoked with structured argv, never a shell string.
  - No arbitrary shell recipe is executed.

##### STORY-005 — Project-root detection

- **Status:** Done
- **Priority:** High
- **Source:** master 12, 127
- **Acceptance Criteria:**
  - Project root detection works.

### EPIC-002 — Bootstrap runtime

**Status:** In Progress
**Goal:** Resolver, release metadata, integrity, manifest, compatibility, capability negotiation, cache, installer, transaction.
**Source:** master, phase 2
**Open:** package signature verification (GAP-001)

#### Stories

##### STORY-006 — Bootstrap resolver and cache

- **Status:** Done
- **Priority:** High
- **Source:** master 8, 9, 63, 127
- **Acceptance Criteria:**
  - Bootstrap 1.0 can be resolved.
  - Bootstrap cache exists.
  - Offline/local Bootstrap resolution works.

##### STORY-007 — Release identity and integrity verification

- **Status:** Done
- **Priority:** High
- **Source:** master 11, 127
- **Acceptance Criteria:**
  - Bootstrap release identity is verified.
  - Bootstrap checksum/signature is verified when required (signature scheme: GAP-001, open).
- **Notes:** Optional detached ed25519 release signature (`.gtt/contract/release.sig`), required as soon as a key is trusted in `~/.gtt/trusted-keys/`; publishers use `gtt release keygen|sign` (2026-10-03). Unit and integration tests pass. Recorded by ADR-002 once promoted.

##### STORY-008 — Manifest, compatibility and capability negotiation

- **Status:** Done
- **Priority:** High
- **Source:** master 10, 68, 69, 127
- **Acceptance Criteria:**
  - Bootstrap manifest is parsed.
  - CLI/Bootstrap compatibility is enforced.
  - Capability negotiation exists.
  - An unknown or unsupported capability is refused and the project is left unmodified.

##### STORY-009 — Installer and transaction engine

- **Status:** Done
- **Priority:** High
- **Source:** master 61, 88, 127
- **Acceptance Criteria:**
  - Installation is transactional.
  - A failed installation leaves the project unmodified.

### EPIC-003 — Project initialization

**Status:** Completed
**Goal:** Existing GTT, ADE candidates, sources, language, Method Plan, questionnaire fallback.
**Source:** master, phase 3

#### Stories

##### STORY-010 — Project, existing-GTT and conflict detection

- **Status:** Done
- **Priority:** High
- **Source:** master 12, 13, 14, 127
- **Acceptance Criteria:**
  - Existing GTT is detected.
  - Conflicts with existing host files are reported, never silently overwritten.

##### STORY-011 — Initial source discovery and selection

- **Status:** Done
- **Priority:** High
- **Source:** master 21-24, 27, 115, 127
- **Acceptance Criteria:**
  - Initial sources can be selected.
  - One document, multiple documents and no document are each handled as master section 115 defines.

##### STORY-012 — Questionnaire fallback

- **Status:** Done
- **Priority:** High
- **Source:** master 24-26, 127
- **Acceptance Criteria:**
  - No-document questionnaire path works.
  - Questionnaire is never embedded in CLI source.
  - A user who declines the questionnaire is handled as master section 115 defines.

##### STORY-013 — Language selection

- **Status:** Done
- **Priority:** Medium
- **Source:** master 20, 127
- **Acceptance Criteria:**
  - Language can be selected.

##### STORY-014 — Method Plan selection at init

- **Status:** Done
- **Priority:** High
- **Source:** master 28-30, method-plans, 127
- **Acceptance Criteria:**
  - Methodology profile can be selected.
  - Profile is passed to Bootstrap.
  - Profile semantics are not duplicated in CLI.

##### STORY-015 — `gtt init` transaction and resume

- **Status:** Done
- **Priority:** High
- **Source:** master 32, 36, 88, 89, 105
- **Acceptance Criteria:**
  - `gtt init` asks for the human confirmations of master section 105.
  - An interrupted init can be resumed.

### EPIC-004 — ADE

**Status:** Completed
**Goal:** Participation, Primary, secondaries, installation, validation, invocation, handoff.
**Source:** master, phase 4

#### Stories

##### STORY-016 — ADE candidate detection

- **Status:** Done
- **Priority:** High
- **Source:** master 15, 127
- **Acceptance Criteria:**
  - ADE candidates are detected.
  - A detected ADE is never treated as participating.

##### STORY-017 — Participation and Primary ADE selection

- **Status:** Done
- **Priority:** High
- **Source:** master 16, 19, 91, 127
- **Acceptance Criteria:**
  - Participating ADEs can be selected.
  - Exactly one Primary ADE can be selected.
  - The Primary ADE can be changed through `gtt config primary-ade`.

##### STORY-018 — ADE installation and validation

- **Status:** Done
- **Priority:** High
- **Source:** master 17, 18, 114, 127
- **Acceptance Criteria:**
  - Multiple ADEs can be installed.
  - Bootstrap validation is invoked.
  - Multi-ADE integration tests pass.

##### STORY-019 — ADE invocation and handoff

- **Status:** Done
- **Priority:** Medium
- **Source:** master 33, 34, 65, 86, 127
- **Acceptance Criteria:**
  - ADE handoff works.
  - Primary ADE invocation works where supported.
- **Notes:** Handoff covered by the init integration tests; the launch path by unit tests in `internal/ade` (2026-10-02). No test launches a real ADE binary.

### EPIC-005 — Lifecycle

**Status:** Completed
**Goal:** status, inspect, validate, resume, freeze, doctor, audit.
**Source:** master, phase 5

#### Stories

##### STORY-020 — `gtt status` and `gtt inspect`

- **Status:** Done
- **Priority:** High
- **Source:** master 37, 38, 127
- **Acceptance Criteria:**
  - `status` works.
  - `inspect` works.

##### STORY-021 — `gtt validate` and CI mode

- **Status:** Done
- **Priority:** High
- **Source:** master 39, 40, 127
- **Acceptance Criteria:**
  - `validate` works.
  - CI mode works.

##### STORY-022 — `gtt resume`

- **Status:** Done
- **Priority:** Medium
- **Source:** master 41, 127
- **Acceptance Criteria:**
  - `resume` works.

##### STORY-023 — `gtt freeze`

- **Status:** Done
- **Priority:** High
- **Source:** master 42, 99, 127
- **Acceptance Criteria:**
  - `freeze` delegates to Bootstrap.

##### STORY-024 — `gtt doctor` and `gtt audit`

- **Status:** Done
- **Priority:** Medium
- **Source:** master 43, 44, 127
- **Acceptance Criteria:**
  - `doctor` works.
  - `audit` works.

### EPIC-006 — Distribution

**Status:** Completed
**Goal:** update, snapshot, restore, export --clean, clean.
**Source:** master, phase 6

#### Stories

##### STORY-025 — `gtt update`

- **Status:** Done
- **Priority:** High
- **Source:** master 45, 46, 116, 127
- **Acceptance Criteria:**
  - `update` works with strict compatibility.
  - Update preserves what master section 46 requires.

##### STORY-026 — Recovery snapshot and restore

- **Status:** Done
- **Priority:** High
- **Source:** master 49, 50, 117, 127
- **Acceptance Criteria:**
  - Recovery snapshot works.
  - Recovery is tested.

##### STORY-027 — `gtt export --clean`

- **Status:** Done
- **Priority:** High
- **Source:** master 47, 51, 118, 127
- **Acceptance Criteria:**
  - `export --clean` works.
  - A clean export does not destroy the development project.

##### STORY-028 — `gtt clean`

- **Status:** Done
- **Priority:** High
- **Source:** master 48, 95, 127
- **Acceptance Criteria:**
  - `clean` is protected: it requires explicit human confirmation.
  - Only files GTT installed are removed.

### EPIC-007 — Hardening

**Status:** In Progress
**Goal:** Security, transactionality, CI, JSON, integration tests, E2E, release, documentation.
**Source:** master, phase 7
**Open:** CI pipeline (GAP-002), release packaging, localisation of CLI messages

#### Stories

##### STORY-029 — Path, symlink and archive safety

- **Status:** Done
- **Priority:** High
- **Source:** master 60, 121
- **Acceptance Criteria:**
  - Paths from a package, snapshot or flag are validated against traversal and symlink escape.
  - The security test matrix of master section 121 passes.
- **Notes:** Traversal, absolute path and symlink escape tested; signature mismatch, untrusted key and unsigned package are refused with exit 6; archives are not a distribution format and are refused, never extracted (2026-10-03).

##### STORY-030 — Idempotency and transactionality

- **Status:** Done
- **Priority:** High
- **Source:** master 61, 62
- **Acceptance Criteria:**
  - Every mutating flow is dry run, apply, validate.
  - Repeating an operation yields a stable result.

##### STORY-031 — Layered tests: unit, integration, end-to-end

- **Status:** Done
- **Priority:** High
- **Source:** master 113-122
- **Acceptance Criteria:**
  - Unit tests cover the list in master section 113.
  - Integration tests use real Bootstrap fixtures.
  - End-to-end tests run complete workflows.
  - The no-LLM test and the Bootstrap evolution test pass.
- **Notes:** Unit tests for every package including `app` (fake Bootstrap); integration suite passes (2026-10-03). The ADE matrix rows without Claude Code are skipped in `TestADEMatrix` until the Bootstrap fix in `reports/BUG-bootstrap-session-adapter-non-participating.md` ships.

##### STORY-032 — Architecture boundary: no methodology in Go

- **Status:** Done
- **Priority:** High
- **Source:** master 123, 127
- **Acceptance Criteria:**
  - No GTT methodology is duplicated in Go.
  - Forbidden package names are absent under `internal/`.

##### STORY-033 — CI pipeline

- **Status:** Done
- **Priority:** Medium
- **Source:** master 85
- **Acceptance Criteria:**
  - A CI pipeline runs the tests (provider: GAP-002, open).
- **Notes:** GitHub Actions (ADR-002). First run green on the public repository, 2026-10-03: unit and architecture tests on Linux and macOS, integration against GTT-Community/gtt-bootstrap (run 37098138258). The governance gate runs only where GTT is installed (the private development repository).

##### STORY-034 — Release packaging and documentation

- **Status:** Done
- **Priority:** Medium
- **Source:** master 82, 124 phase 7
- **Acceptance Criteria:**
  - Release artifacts are produced for the declared channels.
  - User documentation exists for the command surface.
- **Notes:** GoReleaser builds the single static `gtt` binary for Linux and macOS (amd64, arm64) on a v* tag of the public repository and publishes it with SHA-256 checksums to GitHub Releases; v1.0.0 released 2026-10-03 (run 37099001640) and verified by downloading and running it. README covers installation, version and features. Package-manager channels (uv, npm, Homebrew) are not part of this Story.

### EPIC-008 — Agent/ADE context discovery and synchronisation

**Status:** Completed
**Goal:** `gtt agents`, `gtt agents check`, `gtt agents sync`, on top of the Bootstrap ADE registry and ledger.
**Source:** ade-sync, section 40

#### Stories

##### STORY-035 — Agent adapters, registry and context manifest

- **Status:** Done
- **Priority:** High
- **Source:** ade-sync 5, 16, 32, 40 (tasks 1-4, 12, 13)
- **Acceptance Criteria:**
  - `AgentAdapter` and `AgentRegistry` exist.
  - The mechanism is extensible through adapters (RF-10).
  - The paths each adapter manages are documented.
- **Notes:** `AgentAdapter` and `AgentRegistry` (ports) with a data-driven implementation over the Bootstrap ADE registry, a managed context manifest (`gtt agents manifest`: synchronized, changed, missing, unexpected) and a per-agent sync trace, added 2026-10-02; unit and integration tests pass. Not implemented because the Bootstrap registry does not declare them: user-level context locations, the active agent (ade-sync 26) and the `outdated` state.

##### STORY-036 — `gtt agents` and `gtt agents check`

- **Status:** Done
- **Priority:** High
- **Source:** ade-sync 19, 20, 39, 40
- **Acceptance Criteria:**
  - Supported ADEs/agents are detected and compared with those declared (RF-01 to RF-04).
  - An installed but undeclared agent is reported and the project is not modified (case 3).
  - State can be validated without an LLM (RF-11).

##### STORY-037 — `gtt agents sync`

- **Status:** Done
- **Priority:** High
- **Source:** ade-sync 21, 39, 40
- **Acceptance Criteria:**
  - Missing managed context is synchronised after confirmation (case 2).
  - Locally modified managed context is detected and never silently overwritten (case 4, RF-07).
  - Running sync several times gives a stable result with no duplication (case 6, RF-09).
  - Credentials and private sessions are never synchronised (RF-08).

##### STORY-038 — Integration with `gtt status`, traceability and CI

- **Status:** Done
- **Priority:** Medium
- **Source:** ade-sync 31, 34, 35, 40
- **Acceptance Criteria:**
  - `gtt status` reports the agent context check.
  - Synchronisation is traceable.
  - The check can run in CI/CD.

### EPIC-009 — Method Plans and automation policy

**Status:** In Progress
**Goal:** `gtt method show`, `set`, `check`; identity allocator.
**Source:** method-plans
**Open:** Team policy (GAP-004)

#### Stories

##### STORY-039 — `gtt method show` and `gtt method set`

- **Status:** Done
- **Priority:** High
- **Source:** method-plans 14, 15
- **Acceptance Criteria:**
  - `gtt method`, `gtt method show` and `gtt method set light|medium|hard|team` exist.
  - A plan change is recorded.
  - `gtt config methodology` remains as an alias.

##### STORY-040 — `gtt method check`

- **Status:** Done
- **Priority:** Medium
- **Source:** method-plans 16
- **Acceptance Criteria:**
  - `gtt method check` validates plan validity, coherent configuration, automation rules, compatible policies, required artifacts and integrity.

##### STORY-041 — Artifact identity allocator

- **Status:** Done
- **Priority:** Medium
- **Source:** method-plans 10
- **Acceptance Criteria:**
  - An occupied id resolves to the next valid id, references are updated, the result is validated and the operation is recorded.
  - It works independently of the agent in use.
- **Notes:** `gtt artifact next-id` resolves the next id per plan and records the allocation in the operation log (2026-10-02); integration test passes. Updating references is left to the Bootstrap (`reconcile`): the CLI never writes the governed domain.

##### STORY-042 — Team plan operating policy

- **Status:** Done
- **Priority:** Low
- **Source:** method-plans 7
- **Acceptance Criteria:**
  - Values a team declares are applied (GAP-004, open).
- **Notes:** Values the plan leaves to `team_policy` follow the team's working agreements (`policy.<kind>`), read through the Bootstrap; undeclared stays propose and confirm (2026-10-03). Unit and integration tests pass.

### EPIC-010 — Initial MD tracking

**Status:** In Progress
**Goal:** Read, applied, validated; no reprocessing; change detection.
**Source:** md-tracking
**Open:** Bootstrap-side structure (GAP-003)

#### Stories

##### STORY-043 — Tracking states READ, APPLIED, VALIDATED

- **Status:** Done
- **Priority:** High
- **Source:** md-tracking 5, 14
- **Acceptance Criteria:**
  - A successfully applied directive is reported as applied and validated without a second command.
  - A failed application is reported and never left as APPLIED.

##### STORY-044 — Reprocessing avoidance and change detection

- **Status:** Done
- **Priority:** High
- **Source:** md-tracking 6, 14, 15
- **Acceptance Criteria:**
  - An already applied MD is not reprocessed.
  - A relevant change in the MD is detected.

##### STORY-045 — Bootstrap-side tracking structure

- **Status:** Blocked
- **Priority:** Low
- **Source:** md-tracking 12
- **Acceptance Criteria:**
  - The Bootstrap provides the tracking structure (GAP-003, open).
- **Notes:** CLI side ready: it uses the record the scaffold manifest declares (`tracking.initial_sources.path`) and falls back to `.gtt/cli/` (2026-10-03). Waiting on the Bootstrap: `reports/REQ-bootstrap-initial-source-tracking.md` (GAP-003).

---

## General Development Work

Work tracked here that is not tied to a specific Story.

- None

## Current Focus

What is actively being worked on right now.

- None

## Next Work

What comes after Current Focus.

- None

## Blocked

- STORY-045 — Bootstrap-side tracking structure (GAP-003): `reports/REQ-bootstrap-initial-source-tracking.md`

---
Governance: adding or removing an Epic/Story, or materially changing its
scope or acceptance criteria, goes through `gtt-domain/change-request.md` ->
`gtt-domain/proposals/` -> Solution Designer decision — the same funnel as an
architecture change. Updating a Story's status, or the *Current Focus* /
*Next Work* / *Blocked* lists, as part of already-approved implementation
work does not need a change request. See `AGENTS.md` → *Backlog governance*
for the full rule and precedence relative to L0/ADRs.
