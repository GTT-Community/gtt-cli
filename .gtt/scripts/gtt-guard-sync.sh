#!/usr/bin/env bash
# GTT - GTTGuard registry sync.
#
# Regenerates .gtt/protection/registry.yaml from the @GTTGuard markers found
# in source. The registry is a derived artifact, like a lockfile - never
# hand-edit it; edit the marker in source and re-run this script instead.
# .gtt/scripts/gtt-check-protection.sh fails the build if the committed
# registry has drifted from what this script would produce.
#
# Usage:
#   .gtt/scripts/gtt-guard-sync.sh
#
# Exit 0 = synced, 1 = a marker could not be resolved, 2 = cannot run.

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
  echo "gtt-guard-sync: no working Python 3 interpreter found (tried python3, python)." >&2
  exit 2
fi

"$PY" "$(dirname "$0")/gtt_guard.py" sync
