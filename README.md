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
| Version | see [Releases](https://github.com/GTT-Community/gtt-cli/releases/latest) |
| Works with | GTT Bootstrap 1.x (compatibility is negotiated on every run) |
| Platforms | Linux, macOS, Windows |
| License | Apache 2.0 |

## Features

- **One-command setup.** `gtt init` finds, downloads and verifies GTT Bootstrap, asks only the decisions that are
  yours (which ADEs participate and which is Primary, language, Autonomy Profile, initial design documents), installs
  transactionally, validates, and hands off to your Primary ADE.
- **Several ADEs, one governance.** Claude Code, GitHub Copilot, Codex, Kiro, Cursor, Google Antigravity and OpenHands side by side. Detected is never
  participating: GTT installs only the integrations you choose.
- **No design document? No problem.** It offers the Bootstrap's Initial Design Questionnaire, which your ADE
  guides you through.
- **Autonomy Profiles.** Light, Medium, Hard and Team decide how much GTT does without asking. Destructive operations
  and governed decisions always need you.
- **Development method.** `gtt init` asks how you want to develop (Vertical Slice, DDD, Hexagonal, Layered,
  Event-First) and how to start (Walking Skeleton or straight to the main functionality), with the texts of the
  Bootstrap's Method Packs. "Help me choose" decides nothing and nothing is applied by default. `gtt check`
  verifies that the code respects the method's dependency rules with the language's own tool.
- **Change during development.** `gtt change` records a change request and has your ADE process it right away;
  `gtt proposals`, `gtt promote` and `gtt undo` follow it to the end - and one step back when needed.
- **Agent context kept in sync.** `gtt agents` detects missing or locally changed agent files, syncs them safely,
  and never overwrites your own changes.
- **Initial documents tracked.** Every design document is recorded as read, applied and validated by content hash,
  so nothing is applied twice and changes are detected.
- **Safe by default.** Fail-closed compatibility and integrity checks, optional signed Bootstrap releases,
  dry run before every change, rollback on failure, recovery snapshots, and no shell strings executed.
- **Built for CI.** `gtt validate --ci`, `--json` output, `--no-input`, and stable exit codes.
- **No LLM required.** Every command is deterministic.

## Requirements

- `git`, to download GTT Bootstrap the first time `gtt init` runs (or pass a local copy with `--bootstrap`).
- `bash` and `python3`: GTT Bootstrap's own scripts use them. The CLI itself needs neither.
- On Windows: [Git for Windows](https://git-scm.com/download/win), which provides bash, and
  [Python 3](https://www.python.org/downloads/windows/). The CLI finds Git Bash on its own and never uses the WSL
  `bash.exe`; set `GTT_BASH` to point it at another bash.
  Note: GTT Bootstrap up to 1.5.0 starts its own scripts with `bash` by name, which on Windows is the WSL `bash.exe`,
  so `gtt init` and other operations fail there. From Bootstrap 1.5.1 on - this CLI ships with 1.9.1 - they run with
  the bash the CLI hands it (`GTT_BASH`).

## Installation

One command, on any platform. Every option installs the same single `gtt` binary (Linux, macOS, Windows; amd64 and
arm64) from the latest [release](https://github.com/GTT-Community/gtt-cli/releases/latest), verified against its
SHA-256 checksum.

| Method | Command |
|---|---|
| **curl** (Linux, macOS) | `curl -fsSL https://raw.githubusercontent.com/GTT-Community/gtt-cli/main/install.sh \| sh` |
| **PowerShell** (Windows) | `irm https://raw.githubusercontent.com/GTT-Community/gtt-cli/main/install.ps1 \| iex` |
| **uv** (Python) | `uv tool install gtt-cli` |
| **npm** (Node.js 18+) | `npm install -g @gtt-community/gtt-cli` |

Then check it with `gtt version`.

- Run it once without installing: `uvx --from gtt-cli gtt version` or `npx @gtt-community/gtt-cli version`.
- Upgrade: run the installer again, `uv tool upgrade gtt-cli`, or `npm update -g @gtt-community/gtt-cli`.
- A specific version: `GTT_VERSION=1.9.1` with the curl or PowerShell installer, `uv tool install gtt-cli==1.9.1`, or
  `npm install -g @gtt-community/gtt-cli@1.9.1`.
- Install location for curl and PowerShell: `GTT_INSTALL_DIR` (default `/usr/local/bin` or `~/.local/bin`; on Windows
  `%LOCALAPPDATA%\Programs\gtt`, added to your PATH).
- `pipx install gtt-cli` works as well, and every archive can be downloaded by hand from
  [Releases](https://github.com/GTT-Community/gtt-cli/releases/latest).

With Go 1.27 or later:

```bash
go install github.com/GTT-Community/gtt-cli/cmd/gtt@latest
```

Or from a clone:

```bash
git clone https://github.com/GTT-Community/gtt-cli.git && cd gtt-cli
make build        # writes ./gtt
```

## Quick start

```bash
cd my-project
gtt init          # interactive: choose ADEs, Primary ADE, language, Autonomy Profile, design documents
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
| `gtt init` | Resolve, verify and negotiate a Bootstrap; ask for participating ADEs, the Primary ADE, language, initial sources, the Autonomy Profile and the development method; install transactionally; validate; hand off to the Primary ADE |
| `gtt status` | Deterministic project state (`--fast` skips the Bootstrap validation) |
| `gtt inspect` | Topology and installation state |
| `gtt validate [--ci]` | The Bootstrap's validation, unaltered |
| `gtt resume` | Session context derived from actual project state |
| `gtt freeze` | Facade over the Bootstrap freeze; a human act, never unattended. There is no `unfreeze` |
| `gtt doctor` | Operational diagnostics: PASS / WARN / FAIL / CANNOT DETERMINE. Never repairs |
| `gtt audit [--md]` | Operational traceability |
| `gtt update` | Transactional Bootstrap update; project state is preserved; a locally changed file is shown and replaced only on your confirmation |
| `gtt export --clean [dest]` | A clean delivery copy; the project is not modified |
| `gtt clean` | Remove GTT from the project (asks; offers a recovery snapshot) |
| `gtt version` | CLI version, the Bootstrap release it ships with, and the Bootstrap installed in the project |
| `gtt autonomy [show\|set\|check]` (alias: `gtt method`) | Autonomy Profile: Light, Medium, Hard, Team (`gtt config methodology` is an alias of `set`) |
| `gtt dev-method [show\|list]` | The development method (Method Governance), read-only: the method in force, its status (ratified / open), the ADR that locks it and its verification level; `list` shows the Method Packs with their short text. It changes only through a change request |
| `gtt check` | Method Conformance: the code against the ratified method's dependency rules, through the language's own tool (see *Method Conformance* below). A violation is a conflict, never corrected; without an adapter for the language the method is reported `declared-only` |
| `gtt change ["what"] [--reason ...]` | Request a governed change in the middle of the work: it is written to `gtt-domain/change-request.md` through the Bootstrap with an id, the agent is told to process it into a proposal (the session's Next action), and the CLI offers to open the Primary ADE. `gtt change show` follows it |
| `gtt proposals [list\|show\|withdraw]` | The proposals, ADR drafts and staged promotion sets, with their state and the request they answer; `withdraw` archives one (never deletes it) |
| `gtt promote <set>` | Apply a staged promotion set after reviewing its diff (a human act) |
| `gtt undo` | One step back: put back what the last promotion replaced, while its files are as it left it. A frozen promotion is re-frozen over the earlier state - never an unfreeze. Further back is a change request |
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

Every human decision has a flag; a missing one is refused (exit 3), never inferred. The development method is the
exception that confirms the rule: without `--dev-method` / `--start` nothing is recorded and it stays OPEN - it is
never defaulted. `--dev-method help` leaves it for the agent to propose during THINK.

```bash
gtt init --bootstrap ../gtt-bootstrap --ade claude,copilot --primary claude \
         --language es --autonomy light --sources docs/requirements.md \
         --dev-method ddd,vertical-slice --start walking-skeleton --yes
```

Bootstrap resolution order: `--bootstrap PATH`, the source the project recorded, the verified cache
(`~/.gtt/cache/bootstrap/<version>/`), `GTT_BOOTSTRAP_SOURCE`, the canonical repository. `--offline` uses only
a local path or the cache. `GTT_HOME` relocates the cache and the snapshots.

### The CLI and the Bootstrap go together

Installing the CLI puts one binary on the machine and nothing else: there is no project yet. The Bootstrap arrives
with `gtt init`, which always leaves the project with the Bootstrap release this CLI ships with, installed and
verified - or fails and says why.

- **One version.** Each CLI release pins one exact Bootstrap release; `gtt version` shows both. Pinning the CLI
  (`GTT_VERSION=x.y.z`, `gtt-cli==x.y.z`, `gtt-cli@x.y.z`) pins the Bootstrap. The release is downloaded from its
  tag (`v<version>`) and checked against the commit that tag names; a branch head is never installed. Another
  release is used only when you name it: `--bootstrap PATH` or `--bootstrap-version X.Y.Z`.
- **Missing** - it is downloaded once, staged, verified and moved into place; later projects reuse the verified cache.
- **Present and matching** - it is reused. Nothing is downloaded and nothing is written, however often you run it.
- **Present, another release** - `gtt init` reports the installed and the required release and changes nothing
  until you confirm it in a terminal. `--yes` does not give that confirmation; without a terminal it exits 5, and
  `gtt update` is the explicit command. A project on a newer release is never downgraded (exit 2).
- **Files you changed** - a Bootstrap file modified in the project is never replaced silently: `gtt update` shows
  its difference and replaces it only if you say so, in a terminal.
- **Governed state is not package content.** `gtt-domain/context/`, `gtt-domain/adr/`, `SOURCE-BRIEF.*`,
  `gtt-domain/change-request.md` and the freeze marker are never created, modified or deleted by an installation,
  a reinstallation or an update.
- **Failures are distinct.** The source cannot be reached (exit 7), the release is not published (exit 7), the
  download is not the release (exit 6), the cache or the project cannot be written (exit 1), the versions differ
  (exit 5 or 2). A failed or interrupted download leaves nothing in the cache; an interrupted `gtt init` is
  detected from its recorded state (`gtt init --resume` or `--rollback`).

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
destructive operation or a governed decision. `gtt autonomy show` lists what the team declared.

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

`gtt check`: 0 the code respects the method (or nothing to check, or `declared-only`), 1 a forbidden dependency,
7 the result cannot be determined (the language's tool is not installed).

### Method Conformance

`gtt check` has no import analyser of its own: it writes the rules into the configuration of the language's own
tool, runs it, and reports what the tool finds. The language is detected from the project's manifest. The tool is
never installed for you; when it is missing, the result cannot be determined (exit 7) and the install command is shown.

- Go (`go.mod`): the `go` toolchain itself (`go list`); nothing else to install.
- Java/Kotlin (`pom.xml`, `build.gradle`, `build.gradle.kts`): `jdeps` from a JDK 11+; it reads compiled classes, so build first (`mvn compile` or `gradle classes`).
- PHP (`composer.json`): deptrac, in `vendor/bin` or on the PATH (`composer require --dev deptrac/deptrac`).
- Python (`pyproject.toml`, `setup.py`, `setup.cfg`, `requirements*.txt`): import-linter, in `.venv`/`venv` or on the PATH (`pip install import-linter`).
- JavaScript/TypeScript (`package.json`, `tsconfig.json`): dependency-cruiser, in `node_modules` or on the PATH (`npm install --save-dev dependency-cruiser`).

Other languages, .NET included, have no adapter: the method is reported `declared-only`.

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
