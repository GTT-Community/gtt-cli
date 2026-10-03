#!/usr/bin/env bash
# GTT - repository integrity gate.
#
# Fails on: unreconciled moves, unregistered artifacts, duplicate logical
# identity, broken links, links using an old path, unresolved [[ID]]
# references, and a stale technical index. Git being clean is not enough -
# Git integrity and GTT knowledge integrity are different concerns.
#
# Usage:
#   .gtt/scripts/gtt-check-integrity.sh
#
# Exit 0 = ok, 1 = violation / unresolved, 2 = cannot run.

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
  echo "gtt-check-integrity: no working Python 3 interpreter found (tried python3, python)." >&2
  exit 2
fi

"$PY" "$(dirname "$0")/gtt_artifacts.py" check "$@"
