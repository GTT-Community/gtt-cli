# ADR-003 — Release packaging with GoReleaser for Linux, macOS and Windows

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

- Status: Accepted
- Date: 2026-10-03
- Approved by: Solution Designer
- Supersedes: none

## Context

The CLI is open source and installed by end users as a single binary [FUENTE: master] (sections 74, 82, 85). On
2026-10-03 the Solution Designer decided to publish binaries in GitHub Releases, to develop in a private repository
(`gtt-cli-dev`, governed with GTT) and publish the clean product to the public one (`gtt-cli`), and to support
Windows "in any case". The stack map named no release tooling and no supported platforms.

## Decision

1. Releases are built by GoReleaser v2 from a `v*` tag on the public repository and published to GitHub Releases:
   one static binary per platform (Linux, macOS, Windows; amd64, arm64), tar.gz (zip on Windows), with SHA-256
   checksums. Archive names carry no version, so `releases/latest/download/...` always serves the newest release.
2. Development happens in the private repository; `scripts/release.sh` publishes the result of `gtt export --clean`,
   minus the development-only paths in `.publishignore`, after one human confirmation.
3. On Windows the CLI runs the Bootstrap with Git for Windows' bash (or `GTT_BASH`), never with the WSL launcher,
   and downloads the Bootstrap with LF line endings.

## Alternatives considered

| Option | Why it lost |
|---|---|
| Hand-written build script and manual uploads | Repeats what GoReleaser does, with more room for error per release |
| Binaries committed to the repository | Bloats history; GitHub Releases is the standard place for downloads |
| One public repository with a private branch | GitHub visibility is per repository; a branch of a public repository is public |
| Windows only through WSL | The Solution Designer asked for native Windows binaries |
| Use whatever `bash` is first on Windows' PATH | It is often the WSL launcher, which cannot run the Bootstrap by its Windows path |

## Consequences

- A release is one command and one confirmation; GitHub builds every platform in about two minutes.
- Windows users need Git for Windows and Python 3, as the Bootstrap scripts require bash and Python.
- Locked in: GoReleaser v2, GitHub Releases, the archive naming `gtt_<os>_<arch>`, the two-repository flow.

## Risks

- macOS binaries are not notarized: a browser download may be quarantined by Gatekeeper. Signal: user reports;
  remedy is notarization or a Homebrew tap.
- Windows behaviour depends on Git for Windows' bash; another bash (MSYS2, Cygwin) is accepted from PATH but not
  tested. Signal: Windows-only failures in the field.
- The public repository's CI is the only place Windows is tested; a change that breaks Windows is caught after it is
  published as code, but before any release is tagged.

## Stack map delta

| Section | Row | Before | After |
|---|---|---|---|
| header | Governing ADRs | ADR-001, ADR-002 | ADR-001, ADR-002, ADR-003 |
| 1 | Runtime | Native static binary | Native static binary: Linux, macOS, Windows (amd64, arm64), locked by ADR-003 |
| 1 | Release packaging | (no row) | GoReleaser to GitHub Releases, v2, locked by ADR-003 |

Change log line:

| 2026-10-03 | ADR-003 | Supported platforms on the Runtime row; Release packaging row (GoReleaser, GitHub Releases) |

## Affected context

- `stack.md`: the rows above, the governing ADRs line and the change log line.
