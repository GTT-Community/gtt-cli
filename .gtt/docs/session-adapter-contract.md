# GTT Session Memory — ADE Adapter Contract

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

**Status: Draft — not a decision.** A working specification. No ADR ratifies
it and no Epic/Story in `gtt-domain/backlog.md` covers it yet. See *Governance*.

> **GTT Core is ADE-agnostic. ADE adapters are integration-specific.**

## Layers

```text
GTT Core
   ├── Artifact Identity
   ├── Technical Index
   └── Session Memory Service
          │
          ▼
   .gtt/scripts/gtt-session-context.sh        ← the only entry point
          │   (the Core ends here: it names no ADE, event, or format)
          ▼
   ADE Adapter Contract                      ← this document
          │
          ▼
   ADE-specific adapters
          ├── Claude Code
          ├── Codex
          ├── GitHub Copilot
          └── Kiro
```

The Session Memory Service chain — `gtt-session-context.sh`,
`gtt-run-python.sh`, `gtt-status.sh` — and the `gtt-domain/session.md` it produces
describe GTT's operational state only. They name no ADE, hook event, or
config format. Everything specific to an ADE lives in exactly three places:
its adapter files, its declaration in `.gtt/session-adapters/<ade>.json`, and
this document.

| Layer | Knows about ADEs? | Files |
|---|---|---|
| Core service | **No** | `gtt-session-context.sh`, `gtt-run-python.sh`, `gtt-status.sh`, `gtt-domain/session.md` |
| Contract | Only as *data* (declarations), never as code paths | this file, `.gtt/session-adapters/<ade>.json`, `gtt-check-session-adapter.sh` |
| Adapter | **Yes — the only layer that does** | the ADE's native hook/config files |

`gtt-domain/session.md` is session memory: operational-only, **NOT authority, NOT
evidence, NOT a decision record, NOT a grounding source.** It never outranks
`gtt-domain/context/`, `gtt-domain/adr/`, or `gtt-domain/backlog.md`.

## The contract

Every adapter MUST:

1. **Use only the service.** Invoke `.gtt/scripts/gtt-session-context.sh`. It
   must not regenerate `gtt-domain/session.md`, run `gtt-status.sh`, read repository or
   Git state, read the index/manifest, or alter the payload. (Exception: an
   N2 adapter may name `gtt-domain/session.md` as the file the ADE loads — nothing more.)
2. **Preserve the authority boundary.** What reaches the agent must contain
   the markers `operational-only`, `NOT authority`, `NOT evidence`,
   `NOT a decision record`, `NOT a grounding source`. The service payload and
   `gtt-domain/session.md` header already carry them; an adapter may add its own banner
   but never remove them.
3. **Fail visibly.** No `|| true`, no `2>/dev/null` on the service call, no
   bare `except`. On failure: a message on stderr, no partial context, and a
   non-zero exit that is **never 2** — exit 2 blocks session start in Claude
   Code and its effect is undocumented elsewhere. Respect the ADE's own
   non-blocking error channel.
4. **Declare coverage** — exactly one level, never a level not verified:

   | Level | Meaning |
   |---|---|
   | **N1** Native Session Injection | ADE runs a start hook; the hook's output enters the agent's context |
   | **N2** ADE-loaded Session File | ADE loads `gtt-domain/session.md` itself as a resource; freshness is not guaranteed by the adapter |
   | **N3** Instruction-based Retrieval | An instruction file tells the agent to run the service; depends on the agent obeying |

5. **Declare events**, per event, without assuming equal semantics across
   ADEs: `startup`, `resume`, `clear`, `compact`, `fork`. Each is one of
   `verified` (observed at runtime), `documented` (vendor docs say so, not
   observed), `unsupported`, `unknown`.
6. **Declare verification honestly** (below).

## Declaration file

Each adapter is described by `.gtt/session-adapters/<ade>.json` — data, so the
Core and the check contain no per-ADE code:

| Field | Meaning |
|---|---|
| `ade`, `display_name` | Identifier and name |
| `adapter_status` | `installed` (files at native paths) or `staged` (under `gtt-domain/proposals/`, awaiting promotion) |
| `coverage` | `N1` / `N2` / `N3` |
| `files[]` | `{role, install_path, staged_path}` for each adapter file |
| `registration` | `{file, event}` — the config file/event that runs the adapter |
| `command` | The command the ADE runs, from the project root |
| `output` | `text`, `json-hookSpecificOutput`, or `json-additionalContext` |
| `events` | Per-event status (see above) |
| `verification` | `{static, runtime, notes}` |
| `ade_binary` | Executable name, used only to report whether it is installed |
| `limitations[]` | Known gaps, stated plainly |

A declaration is configuration and capability *data*, never implementation:
it holds no GTT logic and no scripts. Its `command` is only the invocation the
ADE runs, and the check rejects any key outside the schema above. Behavior
lives in the adapter files and in the service, not here.

`verification.runtime` is one of `runtime-verified`, `not-verified`,
`requires-interactive-session`, `not-installed`. An event may be `verified`
only if `runtime` is `runtime-verified`. The declaration is a claim, not
evidence: the runtime value must be backed by an actual run.

## Conformance check

`.gtt/scripts/gtt-check-session-adapter.sh <ade>` — deterministic, needs no
ADE installed. It verifies: manifest well-formed; adapter files present
(installed or staged); registration under the declared event only; the
service is referenced; no duplicated GTT logic; no silenced errors; the
command runs in a sandbox copy laid out as installed and its output carries
every marker in the declared format; the failure path (service broken) yields
non-zero, non-2, a stderr message, and no context; declared events do not
overclaim. Each item is `PASS`, `FAIL`, or `SKIPPED`; `SKIPPED` is never
counted as `PASS`. Whether the ADE actually loads the context is **not**
checkable this way — that is `verification.runtime`.

`gtt-validate.sh` runs it for every declared adapter.

## Governance

Adopting an adapter changes the adapter matrix in `AGENTS.md`,
`gtt-check-adapter.sh`, and the `gtt-bootstrap` skill, and puts files at
native ADE paths (e.g. `.codex/`, `.github/hooks/`). That path is:

```text
Proposal → Epic/Story → ADR → Human ratification → (Freeze)
```

Until then adapters other than the Claude Code one live **staged** under
`gtt-domain/proposals/session-adapters/` and are promoted by a human running
`gtt-domain/proposals/apply-session-adapters.sh`. Nothing here is ratified.
