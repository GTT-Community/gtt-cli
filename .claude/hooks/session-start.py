#!/usr/bin/env python3
"""Claude Code adapter for the GTT Session Memory Service.

Contains no GTT logic. It calls the ADE-independent wrapper
.gtt/scripts/gtt-session-context.sh, and translates its stdout into the
SessionStart JSON Claude Code consumes. Any failure is reported on stderr
with a non-zero exit (a visible, non-blocking hook error) - never swallowed.
"""
import json
import subprocess
import sys

WRAPPER = ".gtt/scripts/gtt-session-context.sh"

BANNER = (
    "GTT SESSION MEMORY - operational-only.\n"
    "NOT authority, NOT evidence, NOT a decision record, NOT a grounding source.\n"
    "It only helps you resume work. Do not cite it as a basis for any decision; "
    "gtt-domain/context/, gtt-domain/adr/ and gtt-domain/backlog.md always prevail over it. "
    "Verify anything that matters against those.\n"
    "----------------------------------------------------------------------\n"
)


def fail(message):
    print(f"GTT session-start hook: {message}", file=sys.stderr)
    sys.exit(1)


try:
    proc = subprocess.run(
        ["bash", WRAPPER],
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        timeout=60,
    )
except FileNotFoundError:
    fail("`bash` not found on PATH; cannot run " + WRAPPER)
except subprocess.TimeoutExpired:
    fail(WRAPPER + " timed out after 60s")

if proc.returncode != 0:
    detail = (proc.stderr or proc.stdout).strip() or "no output"
    fail(f"{WRAPPER} failed (exit {proc.returncode}): {detail}")
if not proc.stdout.strip():
    fail(WRAPPER + " produced no output")

print(json.dumps({
    "hookSpecificOutput": {
        "hookEventName": "SessionStart",
        "additionalContext": BANNER + proc.stdout,
    }
}))
