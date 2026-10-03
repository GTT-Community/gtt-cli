# Constraints

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

Hard limits this solution must respect. This file is loaded into every AI
session, so keep it short and concrete: only constraints that would change a
decision.

## Platform

- Cloud provider: none - the CLI is a local binary; nothing is hosted.
- Compute model: single statically linked executable run on the developer machine or a CI runner.
- IaC tooling: none.

## Technical

- Runtime and language version: Go 1.27, command tree built with Cobra. The CLI itself requires no
  Python, no Node.js and no LLM. (Bash and Python 3 are requirements of the Bootstrap scripts it invokes.)
- Datastore: none. CLI operational state is JSON under `.gtt/cli/` and must be reconstructible; the
  Bootstrap state files (`.gtt/ade.json`, `.gtt/methodology.json`, `.gtt/selected-sources.json`) are
  written only through Bootstrap operations, never by the CLI directly.
- Communication style: local process execution only. The Bootstrap is reached through its contract entry
  point with a declared operation and `name=value` arguments: structured argv, never a shell string.
  Network is used only to resolve a Bootstrap release (init, update, cache miss).
- The CLI contains no GTT methodology. Packages named `governance`, `architecture`, `adr`, `reasoning`,
  `think` or `gtt-rules` are forbidden under `internal/`.
- Fail closed: an incompatible, unverifiable or unknown Bootstrap, capability or operation is refused and
  the project is left unmodified.
- Every mutating flow is: dry run, then apply, then validate. Destructive operations always require
  explicit human confirmation, in every Method Plan.
- Exit codes are a contract: 0 success, 1 validation or operational failure, 2 incompatibility,
  3 invalid invocation, 4 user cancellation, 5 project state conflict, 6 Bootstrap integrity failure,
  7 external dependency unavailable.
- Paths coming from a package, a snapshot or a flag are validated against traversal and symlink escape
  before use.

## Regulatory and organizational

- Data residency: not applicable - no data leaves the machine.
- Compliance regime: none declared.
- Secrets: logs and state never contain tokens, credentials or environment values.

## Explicitly out of scope

- Anything listed under Non-goals in `solution-vision.md`.
- Hosting a Bootstrap registry or signing infrastructure (see the gap register in `stack.md`).

---
Governance: L0. Read-only for AI agents. Changes require Solution Designer
approval via the `gtt-propose-change` skill.
