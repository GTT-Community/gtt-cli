# Proposals — staging area

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

Agent-writable. This is the only governed place an agent may create files.

A proposal here is a **draft, not a decision**. Nothing in this directory
governs anything. It exists so the agent can hand you a complete, reviewable
artifact without ever touching `gtt-domain/context/` or `gtt-domain/adr/`.

## Flow

```
gtt-domain/change-request.md  →  gtt-domain/proposals/  →  you review, then `bash apply-*.sh`  →  gtt-domain/adr/ + gtt-domain/context/
                                                or a direct edit (dev-line changes)     or gtt-domain/backlog.md
   you write intent         agent drafts             the Human Promotion Boundary
   (always writable)        (agent writable)
```

For an architecture/context/conflict change (`gtt-propose-change` forms 1-3,
or a `gtt-drift-response` track), once you approve the proposal the agent
stages a full **promotion package** here: the ADR draft, the full text of
every affected `gtt-domain/context/` file, and `apply-ADR-NNN-<slug>.sh` — an
executable script that applies all of them together. Review it, then run it
yourself. The agent never runs it — see `AGENTS.md` → *Human Promotion
Boundary*.

A development-line proposal (new/removed Epic or Story, or a material scope
change — `gtt-propose-change` form 4) is applied to `gtt-domain/backlog.md`
directly by you after approval, not via a script — it does not get an ADR
unless it also happens to touch governed context. Routine Story status
updates never pass through here at all — they're direct edits.

## Lifecycle

| State | Meaning |
|---|---|
| `PROPOSAL-<slug>.md` | awaiting your review |
| `ADR-DRAFT-<slug>.md` + any `context-<file>.md` drafts + `apply-ADR-NNN-<slug>.sh` | an approved change's promotion package — review the drafts, then run the script |
| `bootstrap/` | the `gtt-bootstrap` skill's one-time draft of all six `gtt-domain/context/` files, plus `SOURCE-BRIEF.*` if a source document existed — the one case where a batch of files lands here instead of a single proposal |
| deleted | rejected, or promoted — the script's own output tells you it is safe to delete its staged files once it has applied them |

Delete proposals once resolved. A directory full of stale drafts is the same
failure as stale context: it makes the current state ambiguous.

## Naming

`PROPOSAL-<short-kebab-summary>.md` — no numbers. Numbering belongs to ADRs,
which are the permanent record. A proposal that never gets accepted should not
consume a number.

Once approved, the promotion package takes the ADR's reserved number:
`ADR-DRAFT-<slug>.md` and `apply-ADR-NNN-<slug>.sh`, matching the
`gtt-domain/adr/ADR-NNN-<slug>.md` filename the script will create.
