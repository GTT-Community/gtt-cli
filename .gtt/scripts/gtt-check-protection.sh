#!/usr/bin/env bash
# GTT - GTTGuard protection registry gate.
#
# Deterministic validation of .gtt/protection/registry.yaml (see the GTT
# Method canonical spec v2.1, sections 13-17 "GTTGuard"): the registry must match a
# fresh regeneration from source markers, every artifact/symbol must
# resolve, every protection value must be valid, every ADR-shaped `source`
# must exist, and a protected artifact that changed in the diff must be
# accompanied by a change under gtt-domain/proposals/ or gtt-domain/adr/.
#
# Usage:
#   .gtt/scripts/gtt-check-protection.sh              # compare against origin/main
#   .gtt/scripts/gtt-check-protection.sh <base-ref>   # compare against another ref
#
# Exit 0 = pass, 1 = violation, 2 = cannot determine.

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
  echo "gtt-check-protection: no working Python 3 interpreter found (tried python3, python)." >&2
  exit 2
fi

"$PY" "$(dirname "$0")/gtt_guard.py" check "${1:-origin/main}"
