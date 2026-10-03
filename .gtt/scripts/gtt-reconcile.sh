#!/usr/bin/env bash
# GTT - move/rename reconciliation.
#
# Detects artifacts whose path changed (git rename hints, same identity, or
# content similarity), then - with --apply - updates identity (old path kept
# as history) and rewrites relative links. Dry-run by default. Frozen
# governed files are never rewritten; they are reported instead.
#
# Usage:
#   .gtt/scripts/gtt-reconcile.sh [--apply] [--map OLD=NEW] [--retire ID] [--retire-missing]
#
# --retire-missing retires EVERY registered artifact whose path is gone (never a detected move).
# It is for a fresh host project whose identity manifest still lists artifacts of overlays/documents
# that were not installed; use it deliberately, dry run first.
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
  echo "gtt-reconcile: no working Python 3 interpreter found (tried python3, python)." >&2
  exit 2
fi

"$PY" "$(dirname "$0")/gtt_artifacts.py" reconcile "$@"
