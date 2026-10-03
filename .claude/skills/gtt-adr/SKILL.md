---
name: gtt-adr
description: Draft an Architecture Decision Record — and its promotion package — for a change the Solution Designer has already approved. Use when the user says a proposal is approved, asks to record or document a decision, asks to write an ADR, or asks to update the ADR index. Do not use for proposing changes that are not yet approved.
---

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

# Draft an ADR and its promotion package

Only draft an ADR for a decision a human has explicitly approved. If approval is
unclear, ask. An ADR records a decision that was made — it is not a place to
argue for one.

`gtt-domain/adr/` and `gtt-domain/context/` are write-protected. You never write into them
directly. Instead you stage a complete **promotion package** under
`gtt-domain/proposals/` — the ADR draft, the full text of every affected context
file, and an executable script — and the human runs the script. This is the
Human Promotion Boundary (`AGENTS.md`): you prepare, they promote. Generating
this package is not itself approval, and approving one change never carries
over to the next.

If a proposal for this change exists in `gtt-domain/proposals/`, base the ADR on it
rather than restating the reasoning from scratch.

## Numbering

Take the number from `bash .gtt/scripts/gtt-project.sh next-id --kind adr` - deterministic, and it never
reuses a retired id. Do not ask the human which number to use; they see it when they review the package.
Target filename: `ADR-NNN-short-kebab-title.md`. Write the draft itself to
`gtt-domain/proposals/ADR-DRAFT-short-kebab-title.md` — never into `gtt-domain/adr/`.

## Template

```markdown
# ADR-NNN — <Title>

- Status: Accepted
- Date: YYYY-MM-DD
- Approved by: <Solution Designer>
- Supersedes: <ADR-NNN, or none>

## Context

What forced this decision. The constraint, the problem, the trigger. Written so
that someone reading it in a year understands the situation without asking.

## Decision

The decision, stated in one or two sentences, in the active voice.

## Alternatives considered

| Option | Why it lost |
|---|---|

## Consequences

What this makes easy. What this makes hard. What is now locked in.

## Risks

What could make this decision wrong later, and what signal would reveal it.

## Stack map delta

The exact rows this decision changes in `gtt-domain/context/stack.md`, as before/after
pairs. Write `No change to the map` only if that is literally true.

| Section | Row | Before | After |
|---|---|---|---|

Plus the line to append to the map change log:

| YYYY-MM-DD | ADR-NNN | <what changed> |

## Affected context

Which other files under `gtt-domain/context/` this decision changes, and how —
staged as full drafts alongside the ADR (see *Stage the affected context
files* below). The Solution Designer runs the script that applies them; you
do not edit them directly.
```

## After drafting

An ADR without a stack map delta is incomplete. `gtt-domain/context/stack.md` is the
one artifact everyone reads to understand the system; a decision recorded in an
ADR but absent from the map is invisible in practice.

## Stage the affected context files

For every file the ADR's *Affected context* section names — `stack.md` always,
plus any others — write the file's **full new content**, not a diff, to
`gtt-domain/proposals/context-<basename>.md` (for example
`gtt-domain/proposals/context-stack.md`). The promotion script will copy each one
straight over its target, so the staged draft must be the complete file exactly
as it should read after promotion, changelog row included.

## Generate the promotion script

Write `gtt-domain/proposals/apply-ADR-NNN-<slug>.sh`, executable, matching this
shape (fill in the placeholders with the real ADR number and slug everywhere;
keep the review loop, the confirmation, and the fail-fast behavior — do not
weaken any of them). Quote every `mv`/`cp` path as shown, and never leave a
literal `<...>` placeholder in an unquoted command line — bash reads an
unquoted `<` as input redirection, not as a filler you forgot to replace:

```bash
#!/usr/bin/env bash
set -euo pipefail

# GTT GOVERNED CHANGE
# Proposal: PROPOSAL-<slug>
# ADR: ADR-NNN-<slug>
# Human execution required: YES
#
# Review the proposal, the ADR draft, and this script before running it.
# This script must not be executed automatically by an AI agent.

cd "$(dirname "$0")/../.." # run from the project root regardless of cwd

echo "GTT governed change"
echo "Proposal: PROPOSAL-<slug>"
echo "ADR: ADR-NNN-<slug>"
echo ""
echo "This will apply:"
echo "  gtt-domain/proposals/ADR-DRAFT-<slug>.md  ->  gtt-domain/adr/ADR-NNN-<slug>.md   (new file)"
echo "  gtt-domain/proposals/context-stack.md     ->  gtt-domain/context/stack.md       (overwrite)"
# ... one line per additional affected context file, noting new file / overwrite
echo ""

while true; do
    read -r -p "Review the change, or approve it? [view/yes/no]: " choice
    case "$choice" in
        view|v)
            echo ""
            echo "===== gtt-domain/proposals/ADR-DRAFT-<slug>.md (new ADR) ====="
            cat "gtt-domain/proposals/ADR-DRAFT-<slug>.md"
            echo ""
            echo "===== gtt-domain/context/stack.md: current -> staged ====="
            diff -u "gtt-domain/context/stack.md" "gtt-domain/proposals/context-stack.md" || true
            # ... one "cat" for the new ADR is enough; one "diff -u" per
            # additional affected context file, same files listed above
            echo ""
            ;;
        yes|y)
            break
            ;;
        no|n)
            echo "Change not promoted."
            exit 1
            ;;
        *)
            echo "Please answer view, yes, or no."
            ;;
    esac
done

mv "gtt-domain/proposals/ADR-DRAFT-<slug>.md" "gtt-domain/adr/ADR-NNN-<slug>.md"  # <slug> replaced with the real value, no literal brackets
cp "gtt-domain/proposals/context-stack.md" "gtt-domain/context/stack.md"
# ... one quoted cp per additional affected context file, same order as above

bash .gtt/scripts/gtt-check-stack.sh

echo ""
echo "Promoted. ADR-NNN-<slug> is now in gtt-domain/adr/ and gtt-domain/context/ reflects it."
echo "You can now delete the staged drafts in gtt-domain/proposals/ for this change."
```

`set -euo pipefail` is the fail-fast mechanism: if any `mv`/`cp` or the check
script fails, the script stops immediately with a visible error instead of
silently continuing with a half-applied change. List every affected file from
the ADR's *Affected context* section as its own `mv`/`cp` line, applied in one
run — do not ask the user to execute several unrelated commands by hand.

The `view`/`yes`/`no` loop is not optional polish: a `[yes/no]` prompt alone
asks the user to approve a change they can only read about in your chat
message. `view` lets them inspect the actual ADR text and the exact diff
against the current `gtt-domain/context/` files, from inside the script itself,
as many times as they want, before deciding — and it loops back to the same
prompt afterward instead of assuming they now mean yes.

## Tell the user

State, in chat, briefly - what is ready, that nothing was applied, the one command, and that running it is
their decision:

> ⚠ `ADR-NNN-<slug>` is ready and needs your authorization: it changes governed
> context (<the files it touches>). **Nothing has been applied.**
>
> Review the proposal, the ADR draft and the script, then run it from the project root:
>
> ```bash
> bash gtt-domain/proposals/apply-ADR-NNN-<slug>.sh
> ```
>
> It shows the ADR text and the exact diff (`view`) before it asks `yes` or `no`.
> Running it is your decision; I have not promoted anything. Once you have run it, I
> validate the result and continue.

Then stop: this is a genuine authorization, not a precaution. When the human says it has run, do not ask
what to do next - run `bash .gtt/scripts/gtt-maintain.sh`, report the result in a few lines and continue
with the planned work. Do not execute the script yourself under any circumstance — not to
save the user a step, not because the change looks obviously correct, and not
because a prior change was approved. Each governed promotion needs its own
explicit human decision.
