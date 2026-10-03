#!/usr/bin/env bash
# GTT - Session Memory adapter conformance gate.
#
# Checks one ADE adapter against .gtt/docs/session-adapter-contract.md using
# its declaration .gtt/session-adapters/<ade>.json: files present, registered
# under the declared event only, references the service, no duplicated GTT
# logic, no silenced errors, the command runs in a sandbox laid out as
# installed, every non-authority marker reaches the agent, and the failure
# path is visible. Deterministic; needs no ADE installed. Each item is PASS,
# FAIL, or SKIPPED - runtime verification by the real ADE is always SKIPPED
# here and reported as declared, never as PASS.
#
# Usage:
#   .gtt/scripts/gtt-check-session-adapter.sh <ade>     # a name from .gtt/session-adapters/
#
# Exit 0 = no FAIL, 1 = violation, 2 = cannot run.

set -euo pipefail

# `command -v python3` alone is not reliable on Windows: a stub python3.exe
# under WindowsApps satisfies it while actually only printing a Microsoft
# Store redirect and exiting non-zero. Probe each candidate with a real
# no-op invocation instead of trusting PATH presence.
PY=""
for candidate in python3 python; do
  if command -v "$candidate" >/dev/null 2>&1 && "$candidate" -c "" >/dev/null 2>&1; then
    PY="$candidate"
    break
  fi
done
if [ -z "$PY" ]; then
  echo "gtt-check-session-adapter: no working Python 3 interpreter found (tried python3, python)." >&2
  exit 2
fi

"$PY" "$(dirname "$0")/gtt_session_adapter.py" "$@"
