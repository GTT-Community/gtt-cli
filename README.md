# gtt-cli

GTT CLI — the operational doorway into [GTT Bootstrap](https://github.com/GTT-Community/gtt-bootstrap).

> **GTT Canonical Governance:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

The Bootstrap owns **what GTT means and how it governs**. This CLI owns **how the Bootstrap is found,
installed, selected, invoked, validated, updated, exported and recovered**. It contains no GTT methodology:
no questionnaire, no ADE catalog, no plan semantics, no validation checks, no exclusion list. It needs no LLM.

```text
GTT Method -> GTT Bootstrap -> versioned contracts -> GTT CLI -> ADE -> Human
```

| | |
|---|---|
| Version | 1.0.0 |
| Works with | GTT Bootstrap 1.x (compatibility is negotiated on every run) |
| Platforms | Linux, macOS |
| License | Apache 2.0 |

## Features

- **One-command setup.** `gtt init` finds, downloads and verifies GTT Bootstrap, asks only the decisions that are
  yours (which ADEs participate and which is Primary, language, Method Plan, initial design documents), installs
  transactionally, validates, and hands off to your Primary ADE.
- **Several ADEs, one governance.** Claude Code, GitHub Copilot, Codex and Kiro side by side. Detected is never
  participating: GTT installs only the integrations you choose.
- **No design document? No problem.** It offers the Bootstrap's Initial Design Questionnaire, which your ADE
  guides you through.
- **Method Plans.** Light, Medium, Hard and Team decide how much GTT does without asking. Destructive operations
  and governed decisions always need you.
- **Agent context kept in sync.** `gtt agents` detects missing or locally changed agent files, syncs them safely,
  and never overwrites your own changes.
- **Initial documents tracked.** Every design document is recorded as read, applied and validated by content hash,
  so nothing is applied twice and changes are detected.
- **Safe by default.** Fail-closed compatibility and integrity checks, optional signed Bootstrap releases,
  dry run before every change, rollback on failure, recovery snapshots, and no shell strings executed.
- **Built for CI.** `gtt validate --ci`, `--json` output, `--no-input`, and stable exit codes.
- **No LLM required.** Every command is deterministic.

## Requirements

- `git`, to download GTT Bootstrap (or pass a local copy with `--bootstrap`).
- `bash` and `python3`: GTT Bootstrap's own scripts use them. The CLI itself needs neither.

## Installation

Prebuilt binaries and package-manager installs are coming with the first tagged release. Until then:

```bash
go install github.com/GTT-Community/gtt-cli/cmd/gtt@latest   # needs Go 1.27 or later
gtt version
```

Or from a clone:

```bash
git clone https://github.com/GTT-Community/gtt-cli.git && cd gtt-cli
make build        # writes ./gtt
```

## Quick start

```bash
cd my-project
gtt init          # interactive: choose ADEs, Primary ADE, language, Method Plan, design documents
gtt status        # where the project stands
```

Then open your Primary ADE. It reads the design documents (or guides you through the questionnaire), drafts the
governed context for your review, and once you confirm it you freeze it with `gtt freeze`. Day to day, `gtt sync`
keeps agent context, document tracking, the index and validation up to date in one run.

## Development

```bash
make build        # go build -o gtt ./cmd/gtt
make test-unit    # unit + architecture tests, a few seconds
make test         # everything, including the integration suite (go test -timeout 30m ./...)
```

The integration tests drive the real CLI against a real Bootstrap catalog: `GTT_TEST_BOOTSTRAP=/path/to/gtt-bootstrap`
(default: a sibling checkout `../gtt-bootstrap`; skipped when there is none).

## Commands

| Command | What it does |
|---|---|
| `gtt init` | Resolve, verify and negotiate a Bootstrap; ask for participating ADEs, the Primary ADE, language, initial sources and the Method Plan; install transactionally; validate; hand off to the Primary ADE |
| `gtt status` | Deterministic project state (`--fast` skips the Bootstrap validation) |
| `gtt inspect` | Topology and installation state |
| `gtt validate [--ci]` | The Bootstrap's validation, unaltered |
| `gtt resume` | Session context derived from actual project state |
| `gtt freeze` | Facade over the Bootstrap freeze; a human act, never unattended. There is no `unfreeze` |
| `gtt doctor` | Operational diagnostics: PASS / WARN / FAIL / CANNOT DETERMINE. Never repairs |
| `gtt audit [--md]` | Operational traceability |
| `gtt update` | Transactional Bootstrap update; project state and locally changed files are preserved |
| `gtt export --clean [dest]` | A clean delivery copy; the project is not modified |
| `gtt clean` | Remove GTT from the project (asks; offers a recovery snapshot) |
| `gtt version` | CLI and installed Bootstrap versions |
| `gtt method [show\|set\|check]` | Method Plan: Light, Medium, Hard, Team (`gtt config methodology` is an alias of `set`) |
| `gtt config primary-ade\|language` | Operational configuration, validated by the Bootstrap |
| `gtt agents [list\|check\|sync\|manifest]` | ADE/agent context discovery and synchronisation (`check --ci` for pipelines); `manifest` lists the paths each agent's adapter manages and their state |
| `gtt sources [list\|apply]` | Initial MD tracking: read, applied, validated, by content hash |
| `gtt sync` | Deterministic upkeep in one run: agent context, protection registry, index, validation, source tracking |
| `gtt snapshot create\|list` | Recovery snapshots of GTT configuration |
| `gtt artifact next-id --kind K` | Next free artifact id, resolved by the Bootstrap |
| `gtt release keygen\|sign` | Create a release key and sign a Bootstrap package (for whoever publishes Bootstrap releases) |

Global flags: `--json`, `--verbose`, `--yes` (an explicit pre-authorisation of confirmations), `--no-input`, `-C dir`,
`--log-level error|warn|info|debug` (internal log on stderr, default `warn`; `debug` lists every process the CLI runs, without argument values).

### Non-interactive init

Every human decision has a flag; a missing one is refused (exit 3), never inferred.

```bash
gtt init --bootstrap ../gtt-bootstrap --ade claude,copilot --primary claude \
         --language es --method light --sources docs/requirements.md --yes
```

Bootstrap resolution order: `--bootstrap PATH`, the source the project recorded, the verified cache
(`~/.gtt/cache/bootstrap/<version>/`), `GTT_BOOTSTRAP_SOURCE`, the canonical repository. `--offline` uses only
a local path or the cache. `GTT_HOME` relocates the cache and the snapshots.

### Release signatures

A Bootstrap package may carry `.gtt/contract/release.sig`: an ed25519 signature over its release version and the
checksum of every other file. Verification follows one rule:

| Trusted keys in `~/.gtt/trusted-keys/` | Package | Result |
|---|---|---|
| none | unsigned or signed | accepted; the signature is reported, not verified |
| one or more | signed by a trusted key, content unchanged | accepted, `Signature: verified` |
| one or more | unsigned, changed after signing, or signed by another key | refused, exit 6, project unmodified |

Publisher side: `gtt release keygen --id NAME --out DIR` creates `NAME.key` (secret) and `NAME.pub`;
`gtt release sign BOOTSTRAP_DIR --key DIR/NAME.key` writes the signature. Users trust a key by copying `NAME.pub`
into `~/.gtt/trusted-keys/`. A package is always a directory tree: an archive is refused, never extracted.

### Team plan policy

Under the Team plan the Bootstrap leaves some values to the team. A team declares them in
`gtt-domain/working-agreements.md`, one agreement per value, applying to `policy.<kind>`:

```text
TA-01 | team | policy.identity_resolution | automatic
TA-02 | team | policy.agent_context_sync | automatic_safe
```

Accepted values: `automatic`, `automatic_safe`, `confirm`, `propose_confirm`. Undeclared values stay
`propose_confirm`. Personal preferences never set team policy, and no declaration removes the confirmation of a
destructive operation or a governed decision. `gtt method show` lists what the team declared.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | validation / operational failure |
| 2 | CLI/Bootstrap incompatibility |
| 3 | invalid invocation (including a missing human decision without a terminal) |
| 4 | user cancellation |
| 5 | project state conflict |
| 6 | Bootstrap integrity failure |
| 7 | external dependency unavailable |

## Architecture

Hexagonal. Dependencies point inward and are enforced by `test/arch`:

```text
cmd/gtt -> internal/compose -> internal/cli -> internal/app -> internal/ports <- adapters
```

| Layer | Packages |
|---|---|
| Kernel | `core` (identity, exit codes, errors), `fsx` (path safety, hashing), `ports` (interfaces) |
| Use cases | `app` |
| Operational services | `ade`, `project`, `methodology`, `tracking`, `agents`, `lifecycle`, `snapshot`, `export` |
| Adapters | `bootstrap` (resolver, verification, negotiation, installer, operation runtime), `execution`, `ui` |
| Driving adapter | `cli` (Cobra), `output` |
| Composition root | `compose` |

The CLI knows one Bootstrap path, the contract entry point, and reaches everything else through operations the
Bootstrap declares. An operation or capability it does not know is refused; nothing is guessed or emulated.

CLI state lives under `.gtt/cli/` (resolution record, install ledger, tracking, sync state, operation log). It is
derived and reconstructible, never an authority.
