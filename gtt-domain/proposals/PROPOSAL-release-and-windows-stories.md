# PROPOSAL — Stories for release automation and Windows support

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

**Status:** awaiting Solution Designer review. This is a draft, not a decision.
**Form:** development-line change (adds two Stories to EPIC-007 in `gtt-domain/backlog.md`).
**Applied by:** a direct edit of `gtt-domain/backlog.md` after approval. No ADR, no script.

## Requested change

Add two Stories to EPIC-007 (Hardening) that record work the Solution Designer asked for on 2026-10-03 and that
was implemented before it had a Story. That order is the finding this proposal corrects: development must start
from a Story.

## Proposed Stories

##### STORY-046 — Release binaries and one-command publishing

- **Status:** Done
- **Priority:** High
- **Source:** master 82, 85, 124 phase 7; Solution Designer request of 2026-10-03
- **Acceptance Criteria:**
  - A v* tag on the public repository builds the single `gtt` binary for every supported platform and publishes it
    with SHA-256 checksums to GitHub Releases.
  - Download names carry no version, so `releases/latest/download/gtt_<os>_<arch>` always points at the newest
    release and the install instructions never change.
  - One command (`scripts/release.sh [version]`) pushes the development history to the private repository,
    publishes the clean product (no GTT, no development material) to the public one and, with a version, tags it.
  - Nothing reaches the public repository without one explicit human confirmation.
- **Notes:** v1.0.0 released 2026-10-03 for Linux and macOS and verified by download.

##### STORY-047 — Windows support

- **Status:** In Progress
- **Priority:** High
- **Source:** Solution Designer request of 2026-10-03
- **Acceptance Criteria:**
  - Windows binaries (amd64, arm64) are released as zip archives.
  - The CLI runs the Bootstrap with Git for Windows' bash (or `GTT_BASH`) and never with the WSL `bash.exe`.
  - A Bootstrap downloaded on Windows keeps LF line endings whatever the user's Git configuration.
  - Paths that are rooted or carry a drive on Windows (`/x`, `\x`, `C:\x`, `C:x`) are refused as project paths, and
    a symlink whose target leaves the project is refused even when the target does not exist.
  - Unit and integration tests pass on Windows in CI.
  - The README documents the Windows requirements (Git for Windows, Python 3) and installation.
- **Notes:** implemented 2026-10-03; Done after the Windows CI jobs pass and v1.0.1 publishes the Windows binaries.

## Impact

Only `gtt-domain/backlog.md`. The architectural side (GoReleaser, supported platforms) is recorded separately in
ADR-003.

## Risk

Low: both Stories describe work already in the repository.

## Alternatives

Fold the work into STORY-034 and STORY-029. Rejected: it would change their accepted scope without a record.
