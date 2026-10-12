# ADR-004 — One pinned Bootstrap release per CLI release

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

- Status: Accepted
- Date: 2026-10-12
- Approved by: Solution Designer
- Supersedes: none (refines ADR-002, decision 2)

## Context

The CLI and the Bootstrap always go together in a project, but nothing said which Bootstrap a CLI release
installs: with no flag, the resolver cloned the default branch of the distribution source, so two runs of the same
CLI could install different Bootstraps, and an unreleased change could reach every user
[FUENTE: unified-install] (sections 1, 3). `gtt init` in a project that already held a Bootstrap answered
"already initialized" without checking which release was there or whether it was intact. ADR-002 decided how a
package is distributed and verified (a directory tree: local path, verified cache or Git clone; optional ed25519
signature); it did not decide which release.

On 2026-10-12 the Solution Designer approved `PROPOSAL-unified-cli-bootstrap-install` and confirmed the pinned
release, 1.5.1 (1.5.0 plus the correction that makes the Bootstrap run its scripts with the bash the CLI resolves,
needed on Windows).

## Decision

1. Each GTT CLI release pins exactly one GTT Bootstrap release, declared as a constant in `internal/core`
   (`BootstrapPin`, currently 1.5.1). Pinning the CLI pins the Bootstrap. The two version numbers need not be
   equal, and compatibility is still negotiated.
2. A Bootstrap is downloaded only from a release tag (`v<version>`) of the distribution source, and what was
   fetched must be the commit that tag names. A branch head is never installed, also not as a fallback.
3. `gtt init` always ends with the required release installed and verified in the project, or fails. The matching
   release is reused and nothing is written. Another installed release is reported and changed only after a human
   confirms it in a terminal; `--yes` does not give that confirmation; a project is never downgraded.
4. Another release is used only when the human names it explicitly (`--bootstrap PATH`, `--bootstrap-version`),
   and the CLI says that it is not the pinned one.
5. A Bootstrap file modified in the project is never replaced silently: its difference is shown and it is
   replaced only on its own confirmation in a terminal.
6. The CLI install channels (install.sh, install.ps1, uv, npm, go) deliver the binary only; none places Bootstrap
   files. The release pipeline (ADR-003) is unchanged.

## Alternatives considered

| Option | Why it lost |
|---|---|
| Package the Bootstrap into the CLI release | PyPI wheels and the npm wrapper carry only the binary and `go install` sees no release asset; covering the five channels would mean embedding Bootstrap content in the CLI module, which duplicates it |
| Keep resolving the default branch and only record the version | Two runs of one CLI release could install different Bootstraps; contradicts [FUENTE: unified-install] section 3.2 |
| Refuse every Bootstrap other than the pinned one | Developing and testing the pair needs a local catalog; an explicit, reported human choice is not a silent deviation |

## Consequences

- Reproducible: a CLI version determines the Bootstrap a project gets. After the first download a machine works
  from its verified cache, offline.
- Every Bootstrap release needs a CLI release to reach users, and the Bootstrap tag must be published before the
  CLI release that pins it.
- `gtt update` with no flag now targets the pinned release.
- A Bootstrap file the Bootstrap itself rewrites (declared `derived` or `state` in its manifest, or seen to be
  rewritten by its own operations) is not treated as a local customization.

## Risks

- The pinned tag is missing or is moved in the distribution source: `gtt init` fails with "is not published"
  (exit 7) or with a commit mismatch (exit 6). Signal: the first `gtt init` after a release; the CI integration job.
- A tag can be re-pointed by whoever controls the source. The commit check ties the download to the tag, not to a
  content checksum fixed in the CLI; release signatures (ADR-002) remain the protection against that.

## Stack map delta

| Section | Row | Before | After |
|---|---|---|---|
| 1. Stack at a glance | Bootstrap release | (no row) | One GTT Bootstrap release pinned per CLI release, fetched from its release tag; 1.5.1; locked by ADR-004 |
| 2. Component map | Adapters to Dist | `git clone over HTTPS` | `git clone of the pinned release tag over HTTPS` |
| Header | Governing ADRs | ADR-001, ADR-002, ADR-003 | ADR-001, ADR-002, ADR-003, ADR-004 |

| 2026-10-12 | ADR-004 | Bootstrap release row (one pinned release per CLI release, 1.5.1); the distribution edge fetches a release tag |

## Affected context

- `gtt-domain/context/stack.md`: the delta above (staged in full as `gtt-domain/proposals/context-stack-ADR-004.md`).
- `gtt-domain/context/sources.md`: `TASK-gtt-cli-bootstrap-unified-install.md` is declared as source
  `unified-install` with precedence 5; `master` moves to 6; the conflict with `master` section 83 is recorded as
  resolved by precedence, not erased (staged in full as `gtt-domain/proposals/context-sources-ADR-004.md`).
