#!/usr/bin/env bash
# GTT - artifact identity + technical index (rebuild).
#
# Registers new artifacts in .gtt/index/artifacts.json and regenerates
# .gtt/index/technical-index.json. The index is DERIVED from the Markdown -
# never hand-edit it, never treat it as authority; delete and rebuild at
# will. Refuses to run while a registered path is missing (an unreconciled
# move) - run gtt-reconcile.sh first.
#
# Usage:
#   .gtt/scripts/gtt-index.sh
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
  echo "gtt-index: no working Python 3 interpreter found (tried python3, python)." >&2
  exit 2
fi

"$PY" "$(dirname "$0")/gtt_artifacts.py" index "$@"
