---
name: gtt-drift-response
description: Use when a GTT DRIFT SIGNAL is emitted, when gtt-audit reports a divergence, or when the user asks whether a change under src/, infra/, or a dependency manifest contradicts ratified architecture. Produces a proposal under gtt-domain/proposals/ plus a promotion script the human runs to ratify it.
---

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

# GTT drift response

Something outside the governed paths may have made the ratified context stale.
Determine whether it did, and if so, prepare the ratification.

This skill is the single response path for drift, whichever way it was found:
the PostToolUse detector, a `gtt-audit` sweep, or a direct question.

## Step 1 — Assess

Read the changed file and compare it against `gtt-domain/context/stack.md`,
`gtt-domain/context/architecture.md`, `gtt-domain/context/constraints.md`, and any ADR under
`gtt-domain/adr/` whose subject covers the change.

Three outcomes:

- **No contradiction.** Say so in one line and stop. Do not write a proposal.
  A proposal for a non-issue is noise that trains the reader to ignore the next
  one.
- **Fast track.** The decision is ratified in principle; only a detail moved —
  a version bump, an added service that fits an existing dependency rule.
- **Full track.** A ratified decision is being replaced. Alternatives and
  rejection reasons matter.

## Step 2 — Draft

Write to `gtt-domain/proposals/PROPOSAL-<short-kebab-summary>.md`. No numbers:
numbering belongs to ADRs, which are the permanent record, and choosing an ADR
number would mean reserving ratified identity for your own draft.

`gtt-domain/proposals/` is the only writable governed path.

**Fast track** — minimum viable delta:

- What changed: file, before, after
- Which L0 statement or ADR it contradicts, quoted with file and line
- What the map should say instead
- The exact change-log row for `stack.md`

**Full track** — follow `gtt-domain/adr/ADR-TEMPLATE.md`: context, decision,
alternatives considered with why each loses, consequences, risks.

Status line: `Status: Requires Architect approval`. Only the human accepts.

## Step 3 — Stage the promotion package, never execute it

If the proposal is accepted (or once you know it will need to be — ask if
unclear), stage the same promotion package the `gtt-adr` skill produces,
using the same shape it defines:

- `gtt-domain/proposals/ADR-DRAFT-<slug>.md` — the ADR, following
  `gtt-domain/adr/ADR-TEMPLATE.md` for the full track, or the fast-track minimum
  delta promoted into the same template shape
- `gtt-domain/proposals/context-<basename>.md` for every affected file (`stack.md`
  at minimum, changelog row included)
- `gtt-domain/proposals/apply-ADR-NNN-<slug>.sh` — the executable promotion script,
  with the required header, the `view`/`yes`/`no` review loop, one `mv`/`cp`
  per staged file, and a final `bash .gtt/scripts/gtt-check-stack.sh`

Take the ADR number from `bash .gtt/scripts/gtt-project.sh next-id --kind adr` - do not ask for it; the
human sees it when reviewing the package. Then deliver the same brief hand-off message
`gtt-adr` uses: where the script is, what it will change, that review comes
first, the exact command to run it, and that you have not promoted anything.

## Hard rule

You draft and stage. The human promotes by running the script themselves.
Running the promotion yourself — or applying its changes by any other means —
would make you the ratifier of your own assertion, which is the exact failure
GTT exists to prevent.
