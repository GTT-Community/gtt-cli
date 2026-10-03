# ADR-002 — GitHub Actions CI, signed Bootstrap releases and declared Team policy

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

- Status: Accepted
- Date: 2026-10-03
- Approved by: Solution Designer
- Supersedes: none

## Context

The bootstrap of this project left four gaps open in the register of `stack.md`, because the design sources did
not determine them [FUENTE: master] [FUENTE: method-plans] [FUENTE: md-tracking]:

- GAP-001: how a Bootstrap package is distributed and signed. The master specification requires that the
  checksum/signature be verified "when required" and lists malicious archives and signature mismatch in its
  security matrix (sections 60.3, 121, 127), but no signature scheme exists.
- GAP-002: which CI/CD provider runs the CLI's own pipeline (section 85 says what CI runs, not where).
- GAP-003: where the Bootstrap keeps the initial-source tracking structure it must prepare.
- GAP-004: how the Team plan applies the values the Bootstrap contract leaves to `team_policy`.

On 2026-10-03 the Solution Designer decided GAP-002 (GitHub), chose how GAP-004 is read, and delegated the
definition of GAP-001 to the implementation. GAP-003 depends on a Bootstrap release; the CLI side is prepared.

## Decision

1. **CI/CD (GAP-002):** GitHub Actions, in `.github/workflows/ci.yml`: formatting, vet, unit and architecture
   tests on Linux and macOS; the integration suite against `GTT-Community/gtt-bootstrap`; and a governance gate
   (`gtt validate --ci`, `gtt agents check --ci`) on this repository.
2. **Bootstrap distribution and signature (GAP-001):** a Bootstrap package is a directory tree (local path,
   verified cache or Git clone); an archive is refused, never extracted. A package may carry a detached ed25519
   signature, `.gtt/contract/release.sig`, over its release version and the checksum of every other file.
   Trusted public keys live in `~/.gtt/trusted-keys/<key-id>.pub`. With no trusted key the signature is reported,
   not required; with one or more, every package must carry a signature that verifies against a trusted key, or
   it is refused with exit 6 and the project is left unmodified. Publishers create keys and sign with
   `gtt release keygen` and `gtt release sign`.
3. **Team policy (GAP-004):** where the selected plan's operating policy says `team_policy`, the CLI applies the
   value the team declares in `gtt-domain/working-agreements.md`, read through the Bootstrap operation
   `query.governance agreements`: a team agreement applying to `policy.<kind>` whose text starts with
   `automatic`, `automatic_safe`, `confirm` or `propose_confirm`. Undeclared values stay `propose_confirm`.
   Personal preferences never set team policy, and no declaration changes the confirmation of a destructive
   operation or a governed decision.
4. **Initial-source tracking (GAP-003, stays OPEN):** the CLI keeps its tracking where the Bootstrap scaffold
   manifest declares it (`tracking.initial_sources.path`), and under `.gtt/cli/` until a Bootstrap declares one.

## Alternatives considered

| Option | Why it lost |
|---|---|
| GitLab CI or another provider | The repository and the Bootstrap it tests against are hosted on GitHub |
| Signatures with Sigstore/cosign | Needs network transparency logs and an external toolchain; the CLI must verify offline with no dependency |
| GPG signatures | Key management and an external binary for every user; heavier than one ed25519 key file |
| Signature always required | Breaks every unsigned Bootstrap until signing exists; trust is an explicit local choice instead |
| Accept and extract archives | Adds an extraction attack surface for a format no distribution channel uses |
| Team policy always `propose_confirm` | Ignores the Bootstrap contract, which says team policy is what the team declares |
| Team policy resolved by the Bootstrap | Blocks the CLI on a Bootstrap release; the declaration is already readable through an existing operation |

## Consequences

- Every push and pull request is tested; the governance gate runs in CI, as the Team plan requires.
- A user or organisation that trusts a release key gets fail-closed verification of every Bootstrap; without one,
  behaviour is unchanged.
- Bootstrap publishers must sign each release once a key is trusted anywhere; a release with changed content needs
  a new signature.
- A team tunes the Team plan in its working agreements, versioned and reviewed, without CLI changes.
- Locked in: ed25519, the signature file path and message format, the `policy.<kind>` agreement convention.

## Risks

- A lost or leaked private key: users must replace the `.pub` file; no revocation list exists. Signal: a release
  signed by a key nobody owns.
- The integration job depends on the Bootstrap's default branch; a breaking Bootstrap change turns CI red without a
  CLI change. Pin the checkout to a release tag once tags are published.
- The agreements report is text: a format change in the Bootstrap would hide team declarations (they fall back to
  `propose_confirm`, the safe side). Signal: `gtt method show` no longer lists declared values.

## Stack map delta

| Section | Row | Before | After |
|---|---|---|---|
| header | Last verified | `2026-10-01` | `2026-10-03` |
| header | Governing ADRs | ADR-001 | ADR-001, ADR-002 |
| 1 | CI/CD | `[VACÍO: GAP-002]` | GitHub Actions, locked by ADR-002 |
| 1 | Release signing | (no row) | ed25519 detached signature, Go standard library, locked by ADR-002 |
| 8 | GAP-001 | OPEN | RESOLVED by ADR-002 |
| 8 | GAP-002 | OPEN | RESOLVED by ADR-002 |
| 8 | GAP-003 | OPEN, scope: tracking under `.gtt/cli/` | OPEN, scope: declared record when the manifest names one, else `.gtt/cli/` |
| 8 | GAP-004 | OPEN | RESOLVED by ADR-002 |

Change log line:

| 2026-10-03 | ADR-002 | CI/CD on GitHub Actions; release signing row; GAP-001, GAP-002, GAP-004 resolved; GAP-003 scope updated |

## Affected context

- `stack.md`: the rows and gap lines above, plus the change log line.
- `architecture.md`:
  - the `agents` module row now also depends on `ade` and `ports` (it builds the AgentAdapter/AgentRegistry from
    the Bootstrap registry), which section 5 of `stack.md` already permits;
  - the integration strategy gains the release signature and the Team policy read;
  - the data ownership row of `gtt-domain/` states that the CLI copies the empty scaffold at install and never
    authors content, and the CLI rows list release keys and the declared tracking record;
  - the known deviations drop the signature (resolved) and update tracking (GAP-003).
