#!/usr/bin/env python3
"""GTT - GTTGuard protected-artifact enforcement (PreToolUse hook).

Sibling to protect-l0.py, but for a different layer: protect-l0.py guards
gtt-domain/context/, gtt-domain/adr/, and the two governance root files; this hook guards
whatever L3 code a developer has opted into protecting with a @GTTGuard
marker (GTT Method canonical spec v2.1, sections 13-17 "GTTGuard"), recorded in
.gtt/protection/registry.yaml by .gtt/scripts/gtt-guard-sync.sh.

Granularity: a file-scope entry (empty symbol) blocks the whole file, same
as protect-l0.py. A class/method-scope entry resolves that symbol's current
line span live from disk (via gtt_guard.resolve_symbol_span - never cached,
so it can never go stale) and blocks any Edit whose old_string OVERLAPS
that span - by line range, not text containment, so an edit that merely
starts or ends inside the boundary (e.g. a reformat that also touches the
next line) is still caught - while an unprotected sibling method in the
same file stays freely editable, matching the proposal's own worked
example. Any ambiguity - the symbol can't be resolved, or the tool is
Write/NotebookEdit (which replace the whole file) - fails SAFE to blocking
the entire file rather than risking a silent bypass.

Bash and PowerShell (added 2026-09-28, alongside the same fix to
protect-l0.py) are both blocked outright for any mutating-looking command
that names a protected file, for the same reason protect-l0.py takes the
same stance for machinery: there is no way to inspect a shell command's
line-range effect on a file, so no attempt is made to be more precise than
that. This environment's primary shell is PowerShell, not Bash - before
this fix, only the Bash tool's command string was inspected, so a
PowerShell Remove-Item/Set-Content/etc. against a protected artifact would
have passed through unchecked. (File-tool paths were already normalised
for backslashes here - see matches_target - only the shell-command side
needed the PowerShell tool and its cmdlets added.)

Exit 2 plus permissionDecision:deny blocks the call deterministically. Any
unexpected input, or an empty/missing registry, exits 0 so a broken hook or
an ungoverned project never blocks a session.
"""

import json
import os
import re
import sys

sys.path.insert(0, os.path.join(".gtt", "scripts"))
import gtt_guard  # noqa: E402  (path must be set up first)

REGISTRY = os.path.join(".gtt", "protection", "registry.yaml")
MUTATING_SHELL = re.compile(
    r"\b(sed\s+-i|tee|mv|rm|truncate|dd|install"
    r"|del|erase|rd|rmdir|ri|move|copy|ren"
    r"|set-content|add-content|out-file|new-item|remove-item"
    r"|move-item|copy-item|rename-item|clear-content)\b|>>?",
    re.IGNORECASE,
)

PROPOSE_HINT = "Use the gtt-propose-change skill (Form 5) instead of editing it directly."


def load_registry(root):
    try:
        with open(os.path.join(root, REGISTRY), "r", encoding="utf-8") as f:
            text = f.read()
    except OSError:
        return []
    return [e for e in gtt_guard.parse_registry(text) if e.get("protection") == "HUMAN_APPROVAL"]


def matches_target(artifact, target):
    artifact = artifact.replace("\\", "/")
    target = target.replace("\\", "/")
    return target == artifact or target.endswith("/" + artifact)


def find_hits(entries, target):
    return [e for e in entries if matches_target(e["artifact"], target)]


def blocked_reason(root, hits, tool, tool_input, target):
    if tool in ("Bash", "PowerShell"):
        symbols = ", ".join(h["symbol"] or "whole file" for h in hits)
        return (
            f"GTTGuard: {target} is a protected artifact ({symbols}). Shell "
            f"commands that could mutate a protected file are always blocked, "
            f"regardless of which symbol they touch. {PROPOSE_HINT}"
        )

    file_scope = [h for h in hits if not h["symbol"]]
    if file_scope:
        return f"GTTGuard: {target} is protected (HUMAN_APPROVAL) - the whole file is protected. {PROPOSE_HINT}"

    if tool in ("Write", "NotebookEdit"):
        return (
            f"GTTGuard: {target} contains a protected symbol. A {tool} replaces "
            f"the whole file, which would also overwrite it. {PROPOSE_HINT}"
        )

    old_string = tool_input.get("old_string", "")
    for h in hits:
        span = gtt_guard.resolve_symbol_span(root, h["artifact"], h["symbol"])
        if span is None:
            return (
                f"GTTGuard: {target} declares protected symbol '{h['symbol']}' "
                f"that could not be resolved deterministically. Failing safe: "
                f"treating the whole file as protected until this is fixed. "
                f"Run .gtt/scripts/gtt-check-protection.sh for details."
            )
        if not old_string:
            return f"GTTGuard: {h['symbol']} in {target} is protected (HUMAN_APPROVAL). {PROPOSE_HINT}"
        start, end = span
        with open(os.path.join(root, h["artifact"]), "r", encoding="utf-8", errors="replace") as f:
            file_text = f.read()
        # A substring-of-the-protected-block check only catches edits fully
        # CONTAINED within [start, end]. An edit that starts, ends, or spans
        # across that boundary - e.g. reformatting a protected method plus a
        # neighboring unprotected line - is a real overlap that must also be
        # blocked, so this locates old_string in the full file and compares
        # LINE RANGES instead of doing text containment.
        idx = file_text.find(old_string)
        if idx == -1:
            continue
        edit_start_line = file_text.count("\n", 0, idx)
        edit_end_line = edit_start_line + old_string.count("\n")
        if edit_start_line <= end and edit_end_line >= start:
            return f"GTTGuard: {h['symbol']} in {target} is protected (HUMAN_APPROVAL). {PROPOSE_HINT}"

    return None


def main() -> int:
    try:
        event = json.load(sys.stdin)
    except Exception:
        return 0

    root = os.getcwd()
    entries = load_registry(root)
    if not entries:
        return 0

    tool = event.get("tool_name", "")
    tool_input = event.get("tool_input") or {}
    target = ""

    if tool in ("Write", "Edit", "NotebookEdit"):
        raw = tool_input.get("file_path") or tool_input.get("notebook_path") or ""
        target = os.path.relpath(raw, root) if os.path.isabs(raw) else raw
        target = target.replace("\\", "/")
    elif tool in ("Bash", "PowerShell"):
        command = tool_input.get("command", "")
        if MUTATING_SHELL.search(command):
            for e in entries:
                if e["artifact"] in command:
                    target = e["artifact"]
                    break

    if not target:
        return 0

    hits = find_hits(entries, target)
    if not hits:
        return 0

    reason = blocked_reason(root, hits, tool, tool_input, target)
    if reason:
        print(
            json.dumps(
                {
                    "hookSpecificOutput": {
                        "hookEventName": "PreToolUse",
                        "permissionDecision": "deny",
                        "permissionDecisionReason": reason,
                    }
                }
            )
        )
        return 2

    return 0


if __name__ == "__main__":
    sys.exit(main())
