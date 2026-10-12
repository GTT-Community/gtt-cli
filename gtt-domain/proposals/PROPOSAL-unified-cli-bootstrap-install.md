# PROPOSAL — Unified CLI / Bootstrap installation

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

Source of the request: `TASK-gtt-cli-bootstrap-unified-install.md` (project root, 2026-10-12; not yet a
declared source in `gtt-domain/context/sources.md`). The Solution Designer asked for it to be implemented
("avancemos") before this proposal existed; the implementation is in the working tree, uncommitted. This
proposal is what makes that work traceable. It is not approved by being written.

```text
Proposed Backlog Change

Kind: new Story
Epic/Story ID: STORY-048, under EPIC-002 — Bootstrap runtime (next free id from gtt-project.sh next-id)
Current state: none. STORY-006 (resolver and cache) and STORY-009 (installer and transaction) are Done and
  say nothing about which Bootstrap release a CLI release installs.
Suggested change:

  ##### STORY-048 — Unified CLI and Bootstrap installation

  - Status: Proposed
  - Priority: High
  - Source: TASK-gtt-cli-bootstrap-unified-install.md
  - Acceptance Criteria:
    - Each CLI release pins one exact Bootstrap release; `gtt version` reports both.
    - `gtt init` always ends with the required release installed and verified, or fails with a distinct message.
    - A release is downloaded from its tag and checked against the commit the tag names; never a branch head.
    - A matching installed release is reused: nothing is downloaded or written, however often init runs.
    - Another installed release is reported (installed vs required) and changed only on a human confirmation
      in a terminal; `--yes` does not give it; a project is never downgraded.
    - A locally modified Bootstrap file is shown as a diff and replaced only on its own confirmation.
    - A failed or interrupted fetch leaves nothing in the cache or the project.
    - Governed context, ADRs, SOURCE-BRIEF.*, change-request.md and the freeze marker stay byte-identical.
    - No install channel (install.sh, install.ps1, uv, npm, go) places Bootstrap files.

Reason: the CLI and the Bootstrap are inseparable at project level (TASK, section 3.1), and today a CLI
  release can install whatever the Bootstrap default branch holds.
Contradicts governed context or an ADR?: no governed file is contradicted. Two things the Solution Designer
  should see:
  - Declared source `master`, section 83 ("CLI and Bootstrap versions are independent... The CLI must not
    assume equal version numbers", GTT_BOOTSTRAP_1.0_CLI_CONTRACT_IMPLEMENTATION_TASK.md:3532). The pin does
    not assume equal numbers and negotiation is unchanged, but "one version" (TASK 3.2) narrows that section.
    Which document prevails on this topic is a precedence decision for `sources.md`, not mine.
  - ADR-002 decision 2 (a package is a directory tree: local path, verified cache or Git clone) is kept: the
    fetch is still a Git clone, now of a release tag only.
Impact: `internal/core` (the pin), `internal/bootstrap` (resolver, manifest, installer error), `internal/app`
  (init on an existing project, update, version), `internal/cli` (rendering), `internal/fsx` (file diff),
  `internal/lifecycle` and `internal/ports` (one field each), tests, README. No new package, no new dependency,
  no release-pipeline change. `gtt update` with no flag now targets the pinned release instead of the branch.
Risk:
  - The pinned release must exist as a tag in GTT-Community/gtt-bootstrap. Today that repository publishes no
    tag at all; until `v1.5.1` is pushed there, a plain `gtt init` fails with "is not published" (exit 7) and
    only `--bootstrap PATH` works. Detected by the first `gtt init` after release; avoided by tagging first.
  - The pin is a constant: every Bootstrap release needs a CLI release to reach users.
Alternatives considered:
  - Package the Bootstrap into the CLI release: works offline, but PyPI wheels and the npm wrapper carry only
    the binary and `go install` sees no release asset, so it would not cover the five channels without
    embedding Bootstrap content in the CLI module - which duplicates it. Loses.
  - Keep resolving the default branch and only record the version: violates TASK 3.2. Loses.

Status: Requires Solution Designer approval
```

## Architectural decision carried by this Story (for an ADR, if approved)

"A CLI release pins exactly one Bootstrap release and resolves it only from that release's tag" is a
decision about Bootstrap distribution, the area ADR-002 locks. If the Story is approved, it should be
recorded with the `gtt-adr` skill (a new ADR refining ADR-002 decision 2, plus the `stack.md` row it locks).
Nothing under `gtt-domain/context/` or `gtt-domain/adr/` was changed for this work.

## Decisions taken as defaults during implementation — each is the Solution Designer's to confirm or change

| Decision | Default taken | Where to change it |
|---|---|---|
| Pinned release | `1.5.1` (confirmed by the Solution Designer on 2026-10-12: 1.5.0 plus the Windows bash correction) | `internal/core/identity.go`, `BootstrapPin` |
| Fetch or package | Fetch the tag at `init`; verified cache afterwards | `internal/bootstrap/resolver.go` |
| `--bootstrap PATH` / `--bootstrap-version` naming another release | Allowed as an explicit human choice, and reported as such | `internal/app/resolve.go`, `ensure.go` |
| Project on a newer release than required | Refused (exit 2), never downgraded | `internal/app/ensure.go` |
| Locally modified file on update | Kept unless confirmed in a terminal; `--yes` does not confirm | `internal/app/distribution.go` |
