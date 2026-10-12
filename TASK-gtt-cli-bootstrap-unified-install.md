# TASK: Unify GTT CLI installation with GTT Bootstrap

## 1. Objective

The GTT CLI and GTT Bootstrap always go together. There is no supported
configuration where a GTT project has one without the other.

Two different installs exist, at two different scopes:

| Install | Scope | Mechanism |
|---|---|---|
| GTT CLI | the machine (global binary) | `install.sh`, `install.ps1`, `uv tool`, `npm -g`, `go install` |
| GTT Bootstrap | a project (files in the workspace) | placed by the CLI |

At CLI install time there is no project, so the CLI installers must **not**
try to place Bootstrap anywhere. The unification point is the project
command: `gtt init` (and any other CLI command that prepares a project).

Modify the existing `gtt init` flow so that a successful `init` **always**
leaves the Bootstrap release that matches this CLI version installed and
verified in the project. A single entry point, a single truthful result.

The five CLI install channels (`install.sh`, `install.ps1`, uv, npm, go) and
the try-without-installing paths (`uvx`, `npx`) stay exactly as they are and
must keep working identically.

Do this without duplicating Bootstrap governance logic inside the CLI, and
without replacing the existing installation mechanism.

## 2. Repositories and authority

| Repository | Role |
|---|---|
| `https://github.com/GTT-Community/gtt-cli` | GTT CLI — where the change is implemented |
| `https://github.com/GTT-Community/gtt-bootstrap` | GTT Bootstrap — authoritative source of its own artifacts and behavior |
| `https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md` | GTT Canonical — sole canonical reference for methodology rules. Read-only |

**The CLI does all the work. Only `gtt-cli` is modified.** `gtt-bootstrap`
and `gtt-method` are read-only inputs: inspect them, never change them. If the
task seems to require a change outside `gtt-cli`, report it as
`DECISION REQUIRED` instead of making it.

Inspect the actual repositories, code, documentation, and installation flow
before making changes. Do not assume any capability that you have not verified
in the source code.

## 3. Decisions already made (do not reopen)

1. **CLI and Bootstrap are inseparable at project level.** `gtt init` always
   ensures Bootstrap in the project. This is not conditional on any detected
   setting. No package-manager install hook (npm `postinstall`, etc.) is used:
   uv and go have no equivalent, and the machine install has no project.
2. **One version.** Each CLI release declares the exact Bootstrap release (tag)
   it ships with. Pinning the CLI (`GTT_VERSION=x.y.z`, `gtt-cli==x.y.z`,
   `gtt-cli@x.y.z`) therefore pins Bootstrap. Never resolve Bootstrap from
   `main` or any branch head. `gtt version` reports both versions.
3. **The CLI orchestrates; Bootstrap defines.** The CLI places and verifies
   Bootstrap. It never rewrites or reimplements Bootstrap content or rules.
   How the CLI obtains the pinned release (fetched from the Bootstrap repo at
   `init`, or packaged unmodified into the CLI release at build time) is
   decided in Phase 1 — see Section 4, item 6.
4. **Governed state is never touched by installation.** Installing, reinstalling,
   or updating Bootstrap must never create, modify, or delete: the governed
   context directory, ADRs, `SOURCE-BRIEF.*`, `CHANGE-REQUEST.md`, or the freeze
   marker.
5. **No silent overwrite of existing machinery.** If a Bootstrap artifact
   already exists in the project and differs from the pinned release (e.g. the
   user customized path globs), the CLI shows the difference and requires
   explicit human confirmation before replacing it. No `--force` / `--yes`
   bypass.

If the source code contradicts any of these, stop and report it as
`DECISION REQUIRED` — do not adapt the decision to match the code.

## 4. Phase 1 — Investigation (report before implementing)

Determine and report, with file references:

1. The CLI's current install and `init` entry points and their flow.
2. How Bootstrap is currently installed, located, versioned, and updated —
   including whether the CLI already references it.
3. Any existing version pinning, compatibility check, integrity check, or
   update policy.
4. Any existing recovery mechanism for interrupted operations.
5. The exact files that must change, and the smallest change that satisfies
   Section 3.
6. **Fetch vs. package — report as `DECISION REQUIRED`.** Compare, with
   evidence from the code and release pipeline:
   - **Fetch at `init`:** the CLI downloads the pinned Bootstrap tag. Needs
     network at `init`; requires a fetch, integrity check, and error path.
   - **Package at build:** the CLI release pipeline includes the pinned
     Bootstrap tag unmodified as a release asset. `init` works offline;
     consistent across all five channels; requires a release-pipeline change
     in `gtt-cli`.
   State which one the existing architecture already supports or leans
   toward. Do not implement either until the decision is confirmed.

**Stop after Phase 1** and present the findings and the proposed change plan.
Continue to Phase 2 only after confirmation. If Phase 1 shows the behavior
already exists, report what is missing and implement only that.

## 5. Phase 2 — Implementation requirements

### 5.1 Unified flow

- Bootstrap missing → install the pinned release through the established
  mechanism.
- Bootstrap present and matching the pinned release → reuse it; install nothing.
- Bootstrap present but a different release → report the installed vs. required
  version and apply Section 3.5 (show diff, require confirmation). Never
  downgrade or upgrade silently.
- The CLI reports success only when Bootstrap is present and verified. If
  Bootstrap fails, the overall result is failure, with a clear message.

### 5.2 Reliability

- **Idempotent:** running install/init N times produces the same state as once.
- **Atomic placement:** stage into a temporary location and move into place
  only after the fetch and verification succeed, so an interruption never
  leaves a half-installed Bootstrap. Reuse an existing mechanism if one exists.
- **Integrity:** verify the fetched release against the pinned tag (and a
  checksum if the existing tooling supports it).
- **Clear errors:** network failure, missing release, permission error, and
  version mismatch each produce a distinct, actionable message.
- **ADE compatibility:** existing ADE integrations (Claude Code, Kiro, Codex,
  etc.) keep working unchanged.

Do not introduce a new package manager, installer framework, versioning
convention, or repository structure.

## 6. Governance boundaries

You are the implementation executor, not the authority that defines GTT.

Do not:

- Redesign GTT Method or modify the Canonical document.
- Invent governance rules, conventions, or versioning policies beyond Section 3.
- Duplicate Bootstrap content inside the CLI.
- Modify unrelated ADE integrations or CLI functionality.
- Refactor opportunistically or add unrelated features.
- Treat an observed repository convention as authorization to create a new one.
- Make commits. Leave all changes uncommitted in the working tree.

If a necessary decision is not established by Section 3, the source code, or
authorized documentation, stop the affected part and report:

`DECISION REQUIRED` — the unresolved decision, the evidence, and the minimum
decision needed to proceed. Do not silently choose an alternative.

## 7. Testing and verification

Tests must be hermetic: no network access. Serve Bootstrap releases from a local
fixture or mock the fetch.

Cover at minimum:

1. Install with Bootstrap missing → installed, success.
2. Install with the pinned Bootstrap already present → reused, nothing rewritten.
3. Repeated install → no duplicate artifacts, identical state.
4. Bootstrap fetch failure → overall failure, clear message, no partial files.
5. Different Bootstrap release present → mismatch reported, nothing replaced
   without confirmation.
6. Interrupted install → no half-installed Bootstrap left behind.
7. Customized Bootstrap artifact present → diff shown, not overwritten silently.
8. Project with populated governed context and freeze marker → install leaves
   all governed files and the marker byte-identical.
9. Existing CLI functionality and ADE integrations → existing tests still pass.
10. `gtt version` reports the CLI version and the Bootstrap release it pins.
11. The five install channels and `uvx`/`npx` are unchanged: none of them
    writes Bootstrap files at machine-install time.

Run the available tests, build checks, and linters. Report any check that could
not be executed. Do not claim a scenario is verified without a test or
reproducible check behind it.

## 8. Final report

1. The installation flow discovered.
2. The existing Bootstrap mechanism.
3. Files changed in `gtt-cli` (no other repository is modified).
4. Implemented behavior, mapped to Section 5.
5. Test commands and results.
6. Compatibility and failure-handling verification (scenarios 1–9).
7. Remaining limitations and every `DECISION REQUIRED` raised.

**Acceptance criterion:** running `gtt init` in a project, with the CLI
installed through any of its established channels, always leaves the
Bootstrap release pinned to that CLI version available — idempotently, atomically, truthfully reported — without duplicating
governance logic, touching governed state, or bypassing GTT's authority
boundaries.
