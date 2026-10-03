#!/usr/bin/env python3
"""GitHub Copilot adapter for the GTT Session Memory Service (STAGED - Draft).

Contains no GTT logic. It calls the ADE-independent service
.gtt/scripts/gtt-session-context.sh and prints the JSON a Copilot
sessionStart hook uses to inject context: {"additionalContext": "..."}
(top level - no hookSpecificOutput wrapper). Plain text stdout is NOT used by
Copilot for this event, so it is not emitted.

Failure: a message on stderr and exit 1 (never 2), no context emitted.
"""
import json
import subprocess
import sys

SERVICE = ".gtt/scripts/gtt-session-context.sh"

BANNER = (
    "GTT SESSION MEMORY - operational-only.\n"
    "NOT authority, NOT evidence, NOT a decision record, NOT a grounding source.\n"
    "It only helps you resume work. Do not cite it as a basis for any decision; "
    "gtt-domain/context/, gtt-domain/adr/ and gtt-domain/backlog.md always prevail over it.\n"
    "----------------------------------------------------------------------\n"
)


def fail(message):
    print(f"GTT session hook (copilot): {message}", file=sys.stderr)
    sys.exit(1)


try:
    proc = subprocess.run(
        ["bash", SERVICE],
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        timeout=60,
    )
except FileNotFoundError:
    fail("`bash` not found on PATH; cannot run " + SERVICE)
except subprocess.TimeoutExpired:
    fail(SERVICE + " timed out after 60s")

if proc.returncode != 0:
    detail = (proc.stderr or proc.stdout).strip() or "no output"
    fail(f"{SERVICE} failed (exit {proc.returncode}): {detail}")
if not proc.stdout.strip():
    fail(SERVICE + " produced no output")

print(json.dumps({"additionalContext": BANNER + proc.stdout}))
