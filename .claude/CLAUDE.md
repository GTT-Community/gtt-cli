> **Canonical reference:** https://github.com/GTT-Community/gtt-method/blob/main/GTT-CANONICAL-v2.1.md

@../AGENTS.md

## Claude Code specifics

Use these skills instead of improvising the format:

| Situation | Skill |
|---|---|
| First time populating `gtt-domain/context/`, or `gtt-domain/context/` still holds template placeholders | `gtt-bootstrap` |
| "process the change request" | `gtt-propose-change` |
| Architectural or context change needed | `gtt-propose-change` |
| A change was approved and needs recording | `gtt-adr` |
| Verify the governed context still matches the code | `gtt-audit` |
| `gtt-domain/backlog.md` needs an Epic/Story added, removed, or materially changed | `gtt-propose-change` (form 4) |
| Reconcile `gtt-domain/backlog.md` against defined Epics/Stories | `gtt-audit` |
| Mark/unmark a file, class, or method as GTTGuard-protected | `gtt-guard` |
| A change is requested to a GTTGuard-protected artifact | `gtt-propose-change` (form 5) |

`gtt-domain/context/`, `gtt-domain/adr/`, `gtt-domain/change-request.md`, and `SOURCE-BRIEF.*`
are blocked at the permission layer and by a PreToolUse hook. A denial there
is the system working as designed — write to `gtt-domain/proposals/` instead, and
never look for another way to reach a blocked path. `gtt-domain/backlog.md` is not
blocked the same way — routine Story status/focus updates are direct edits;
only structural changes go through the flow above (see `AGENTS.md` →
*Backlog governance*).

A GTTGuard-protected file/class/method is a separate, live-resolved block by
a different PreToolUse hook (`protect-guard.py`), driven by
`.gtt/protection/registry.yaml` rather than a fixed path list. A denial there
is the same kind of signal — use `gtt-propose-change` (form 5), never look
for another way in. `.claude/hooks/` and `.claude/settings.json` are
themselves machinery and permission-denied like any other governance file;
if a change needs to land there, stage it under `gtt-domain/proposals/` and tell
the Solution Designer to move it into place.

Hard constraints, always in context:

@../gtt-domain/context/constraints.md
