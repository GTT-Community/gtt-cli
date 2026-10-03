# Architecture

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

The current architecture of this solution.

## Architectural style

Hexagonal (ports and adapters), in one Go module that builds one binary. Dependencies point inward:

```text
cmd/gtt -> internal/compose -> internal/cli -> internal/app -> internal/ports <- adapters
```

`internal/cli` is the driving adapter (Cobra). `internal/app` holds the use cases. Every external
system - the Bootstrap, ADE binaries, Git, the terminal, the process table - is reached through an
interface declared in `internal/ports` and implemented by a driven adapter. `internal/compose` is the
composition root: the only place that wires adapters to use cases. [FUENTE: master]

Three kinds of package sit under the use cases:

- **adapters** talk to an external system and implement a port;
- **operational services** keep the CLI's own derived state on disk or compute over project files;
  they hold no external dependency and are used by the use cases directly;
- **kernel** packages (`core`, `fsx`, `ports`) depend on nothing above them.

## Modules and boundaries

| Module | Kind | Responsibility | May depend on | Must not depend on |
|---|---|---|---|---|
| `cmd/gtt` | entry point | starts the CLI | `compose`, `cli` | everything else |
| `internal/compose` | composition root | wires adapters to use cases | everything | - |
| `internal/cli` | driving adapter | Cobra tree, flags, rendering, exit codes | `app`, `ports`, `core`, `output`, operational services (types only) | adapters, `compose` |
| `internal/app` | use cases | init, status, inspect, validate, resume, freeze, doctor, audit, update, export, clean, agents, method, sources, config, sync | `ports`, `core`, operational services | `cli`, `compose`, adapters, Cobra |
| `internal/ports` | kernel | interfaces and operational data types | standard library | every other internal package |
| `internal/core` | kernel | CLI identity (version, capabilities, schemas), exit codes, typed errors | standard library | every other internal package |
| `internal/fsx` | kernel | path safety, hashing, copy | standard library | every other internal package |
| `internal/bootstrap` | adapter | resolver, cache, verification (checksum, release signature), compatibility negotiation, core installer, operation runtime, release signer | `ports`, `core`, `fsx` | `app`, `cli` |
| `internal/execution` | adapter | structured process execution (executable, argv, env, cwd); no shell | `ports` | `app`, `cli` |
| `internal/ui` | adapter | terminal prompter | `ports`, `core` | `app`, `cli` |
| `internal/ade` | operational service | reads the Bootstrap ADE registry through a port; candidate detection; ADE executors | `ports`, `core` | `app`, `cli`, adapters |
| `internal/project` | operational service | project root, read-only Git, candidate documents | `ports` | `app`, `cli`, adapters |
| `internal/methodology` | operational service | reads the selected plan's operating policy, and the team's declared values where it says team policy, through a port and answers: automatic, confirm or propose | `ports`, `core` | `app`, `cli`, adapters |
| `internal/tracking` | operational service | initial MD tracking by content hash | `fsx`, `lifecycle` | `app`, `cli`, adapters |
| `internal/agents` | operational service | AgentAdapter/AgentRegistry over the Bootstrap ADE registry, agent environment model, managed context manifest and synchronisation state | `ade`, `ports`, `fsx`, `lifecycle` | `app`, `cli`, adapters |
| `internal/lifecycle` | operational service | CLI operational state, init transaction journal, operation log | `core`, `fsx` | `app`, `cli`, adapters |
| `internal/snapshot` | operational service | storage of recovery snapshots produced by the Bootstrap | `fsx` | `app`, `cli`, adapters |
| `internal/export` | operational service | clean export copy driven by the Bootstrap export policy | `fsx` | `app`, `cli`, adapters |
| `internal/output` | rendering | human and JSON rendering | standard library | adapters |

## Integration strategy

- Bootstrap: one entry point, the contract script declared by the Bootstrap. The CLI calls
  `release`, `negotiate`, `capabilities`, `operations`, `show` and `run OPERATION name=value --envelope`.
  Before any `run`, the operation must exist in the registry and its capability must have been
  negotiated; an unknown one is refused. Mutating operations are run as a dry run first.
- Before GTT is installed, the same entry point is used inside the resolved catalog (read only) for
  release identity, negotiation and the ADE registry; candidate detection uses the registry's declared
  signals against the project.
- ADEs: an executor per ADE built from the registry entry (binary, instruction entry). The CLI
  delivers the Bootstrap-defined handoff; it never composes instructions.
- Release signature: a package may carry `.gtt/contract/release.sig` (ed25519 over version and content checksum).
  When `~/.gtt/trusted-keys/` holds any key, a package without a verifying signature is refused (ADR-002).
- Team policy: values the plan leaves to `team_policy` are read from the team's working agreements through
  `query.governance agreements`; undeclared values stay propose and confirm (ADR-002).
- Git: read-only queries through the process runner.
- Output: every operational command renders either human text or versioned JSON from the same result value.

## Data model ownership

| Data | Owner | Written by |
|---|---|---|
| `.gtt/contract/`, manifest, scripts, templates | Bootstrap | package install / update only |
| `.gtt/ade.json`, `.gtt/methodology.json`, `.gtt/selected-sources.json` | Bootstrap | Bootstrap operations |
| `gtt-domain/` | governed domain | humans and ADEs under GTT rules; the CLI only copies the empty scaffold from the package at install and never authors or edits content |
| `.gtt/cli/state.json` | CLI | resolution record and transaction state |
| `.gtt/cli/tracking.json`, or the record the scaffold manifest declares | CLI | initial MD tracking (derived, reconstructible) |
| `.gtt/cli/agents.json`, `.gtt/cli/operations.log` | CLI | synchronisation state and trace (derived) |
| user cache and snapshots under the user's GTT home | CLI | resolver and snapshot store |
| `~/.gtt/trusted-keys/` | the user | trusted Bootstrap release keys; the CLI only reads them |

CLI state is derived and operational: it is never an authority and can be rebuilt from the project.

## Deployment topology

One executable, `gtt`, installed on the developer machine or CI runner. No service, no daemon.

## Known deviations

- The initial MD tracking structure lives under `.gtt/cli/` and is created by the CLI until the Bootstrap
  declares and ships one; the CLI already uses a declared record (GAP-003).

---
Governance: L0. Read-only for AI agents. Changes require an approved ADR.
