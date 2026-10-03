#!/usr/bin/env bash
# GTT - Python launcher for ADE hooks.
#
# The single place a hook command resolves a working Python 3. It replaces
# the fragile `python3 -c '' && python3 X || python X` shell idiom, in which
# a failure of the script itself ALSO triggers the `python X` fallback, and
# bare `python3 X`, which on Windows can hit the Microsoft Store stub.
#
# Probes each candidate with a real no-op invocation (same strategy as the
# other GTT wrappers), then execs the chosen interpreter so the hook script's
# own exit code and output pass through untouched. No fallback runs after the
# script starts.
#
# Exit codes: the script's own; or 1 (never 2) when no interpreter is found -
# exit 2 is a blocking status for some hook runners, and a missing Python
# should be a visible error, not a silent block.
#
# Usage:
#   .gtt/scripts/gtt-run-python.sh <script.py> [args...]

set -u

if [ "$#" -lt 1 ]; then
  echo "gtt-run-python: usage: gtt-run-python.sh <script.py> [args...]" >&2
  exit 1
fi

PY=""
for candidate in python3 python; do
  if command -v "$candidate" >/dev/null 2>&1 && "$candidate" -c "" >/dev/null 2>&1; then
    PY="$candidate"
    break
  fi
done
if [ -z "$PY" ]; then
  echo "gtt-run-python: no working Python 3 interpreter found (tried python3, python); cannot run $1." >&2
  exit 1
fi

exec "$PY" "$@"
