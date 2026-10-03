# Source Manifest

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

Declares the design sources a project treats as authorised, with an authority and an
unambiguous precedence, so that which source wins is never decided by reading order,
file order or prompt order. Governed like the rest of L0: it lives at
`gtt-domain/context/sources.md` (written before freeze, or changed through the governed
path after it). Read by `.gtt/scripts/gtt-check-provenance.sh`; the manifest is the
authority on the declaration, never the technical index.

## Rules

- One source per line: `id | path | version | authority | precedence | status`.
- `authority`: `primary`, `secondary`, `reference` or `evidence` (history only).
- `precedence`: an integer, unique across the manifest. It orders sources when interpreting
  a conflict; it never erases one. Two sources that disagree are recorded as
  `[CONFLICTO: id-a vs id-b]` in the governed context until the human decides.
- `status`: `active`, `superseded` or `retired`.
- `policy: provenance=required` makes every technology row of `stack.md` carry a `[FUENTE]`
  or a `Locked by` ADR; `advisory` (the default) only checks what is written.
- After freeze `SOURCE-BRIEF.*` is evidence and history: declare it `evidence`, never
  `primary` or `secondary`. The governed context wins over it.
- Cite a source in governed context as `[FUENTE: id]`, `[FUENTE: id:section]` or
  `[FUENTE: path:line]`.

## Sources

```gtt-sources
# policy: provenance=advisory
# architecture-spec | docs/architecture.md | 2.1 | primary | 1 | active
# requirements | docs/requirements.md | 1.7 | primary | 2 | active
```

---
Governance: L0 once placed at `gtt-domain/context/sources.md`. Read-only for AI agents after freeze.
