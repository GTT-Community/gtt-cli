# Working Agreements

> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

How this team prefers to work. Versioned and reviewed like any other file, but **below governed
context**: a working agreement never overrides a governed decision, an ADR or a constraint, and
when one appears to, the governed context wins — that is not an architectural conflict. It is not
Session Memory (operational continuity) and not evidence. A person's own preferences belong in
`.gtt/local/preferences.md`, which is local and never committed. No agent authors its own
preferences: a human writes or approves each entry.

Read by `.gtt/scripts/gtt-check-provenance.sh`, which rejects an entry that tries to override
governed context.

## Format

One agreement per line: `id | scope | applies-to | text`. `scope` is `team` in this file
(and `user` in `.gtt/local/preferences.md`).

## Agreements

```gtt-preferences
# PREF-001 | team | pull requests | Keep pull requests small and describe the why in the first line.
```
