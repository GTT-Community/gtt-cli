#!/usr/bin/env bash
# Kiro CLI adapter for the GTT Session Memory Service (STAGED - Draft, not a decision).
#
# No GTT logic here. It runs the ADE-independent service and prints its
# stdout. Kiro's trigger reference says agentSpawn stdout is added to the
# agent's context on exit 0; the agent-configuration reference is silent on
# that, so this is DOCUMENTED IN ONE PLACE, NOT RUNTIME VERIFIED (Kiro is not
# installed where this was prepared). Coverage N1, Kiro CLI only.
#
# A failure is reported on stderr with exit 1 (never 2), no partial context.
# Assumes the Kiro session working directory is the project root.

set -u
cd "$(dirname "$0")/.." || { echo "GTT session hook (kiro): cannot enter project root." >&2; exit 1; }

if ! payload="$(bash .gtt/scripts/gtt-session-context.sh)"; then
  echo "GTT session hook (kiro): .gtt/scripts/gtt-session-context.sh failed; no session context was provided." >&2
  exit 1
fi
if [ -z "$payload" ]; then
  echo "GTT session hook (kiro): .gtt/scripts/gtt-session-context.sh produced no output." >&2
  exit 1
fi
printf '%s\n' "$payload"
