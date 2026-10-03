#!/usr/bin/env bash
# GTT - section-level retrieval from the technical index.
#
# Locate a concept/section without loading whole documents. Reads only the
# index to find it, then only the needed lines from the Markdown (the source
# of truth); warns if the file changed since indexing.
#
# Usage:
#   .gtt/scripts/gtt-query.sh <term | ID[#anchor]> [--show] [--deep]
#   .gtt/scripts/gtt-query.sh --governance <open|blocking|resolved|conflicts|sources|agreements>
#       governance questions (gaps, unresolved conflicts, declared sources and their precedence,
#       working agreements), answered from the governed artifacts themselves - never from the index
#
# Exit 0 = ok, 1 = violation / unresolved, 2 = cannot run.

set -euo pipefail

if [ "${1:-}" = "--governance" ]; then
  shift
  HERE="$(cd "$(dirname "$0")" && pwd)"
  exec bash "$HERE/gtt-run-python.sh" "$HERE/gtt_provenance.py" list "$@"
fi

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
  echo "gtt-query: no working Python 3 interpreter found (tried python3, python)." >&2
  exit 2
fi

"$PY" "$(dirname "$0")/gtt_artifacts.py" query "$@"
