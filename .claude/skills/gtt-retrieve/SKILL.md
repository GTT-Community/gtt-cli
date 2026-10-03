---
name: gtt-retrieve
description: Locate a concept, section, or artifact in the GTT/project Markdown through the technical index instead of scanning or loading whole documents, and reconcile artifacts that were moved or renamed. Use when the user asks where something is documented, which document is authoritative for a concept, what references or supersedes an artifact, or after files under .gtt/ or gtt-domain/ were moved/renamed/reorganized. Not for changing governed context — that goes through gtt-propose-change.
---

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

# Retrieve by identity, reconcile moves

The technical index (`.gtt/index/technical-index.json`) is a **derived
locator**, never authority. It tells you *where* to read; the Markdown it
points at is the evidence. Never quote the index as if it were the decision.

## Retrieval

1. `bash .gtt/scripts/gtt-query.sh "<term>"` — ranked hits as
   `ID#anchor  path:lines  [authority]  breadcrumb`. Add `--deep` to also
   search section bodies.
2. `bash .gtt/scripts/gtt-query.sh ADR-007` — outline, authority, and
   `referenced-by` for one artifact; `ADR-007#decision --show` prints only
   that section.
3. Read only the returned line span from the file. Prefer the higher
   authority: `L0` (governed context) > `L1` (ADR) > `development-line` >
   `instruction`/`human-reference` > `draft`.
4. If the query warns a file changed since indexing, run
   `bash .gtt/scripts/gtt-index.sh` (regenerating a derived file is always
   safe) and query again.

## References you write

Reference another artifact as `[[ADR-007]]` (path-independent) rather than a
relative path where you can. Relative links still work and are reconciled on
a move.

## Moves and renames

If `gtt-check-integrity.sh` (or `gtt-validate.sh`) reports an unreconciled
path, or you were told files were moved:

1. `bash .gtt/scripts/gtt-reconcile.sh` — dry run. Read what it matched and
   how (git rename, same identity, content similarity).
2. `AMBIGUOUS`/`MISSING` items need a human answer: `--map OLD=NEW` for a
   move it could not pair, `--retire ID` for a genuine deletion. Ask; never
   guess.
3. `bash .gtt/scripts/gtt-reconcile.sh --apply` — updates identity (old path
   kept as history), rewrites relative links, rebuilds the index.
4. `bash .gtt/scripts/gtt-check-integrity.sh` must pass.

Frozen `gtt-domain/context/` and `gtt-domain/adr/` files are **never** rewritten by the
tool, nor by you: the tool reports them as `SKIPPED`; fixing them is a
governed change (`gtt-propose-change`). Moving those files at all is a
governed/human act — do not perform the move yourself.

## Duplicate identity

Two files deriving the same id (`ADR-007` in two directories) is a
*duplicate logical identity*, not two artifacts. Ask whether one is a moved
copy (reconcile) or an intentional alias (`aliases` in the manifest, a
human edit). Never resolve it by deleting one.
