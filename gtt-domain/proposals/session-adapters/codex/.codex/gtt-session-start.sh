#!/usr/bin/env bash
# Codex adapter for the GTT Session Memory Service (STAGED - Draft, not a decision).
#
# No GTT logic here. It runs the ADE-independent service and hands its stdout
# to Codex, which adds plain SessionStart stdout as developer context.
# Coverage N1. A failure is reported on stderr with exit 1 (never 2), and no
# partial context is emitted.
#
# Assumes the Codex session working directory is the project root, where
# .codex/ lives.

set -u
cd "$(dirname "$0")/.." || { echo "GTT session hook (codex): cannot enter project root." >&2; exit 1; }

if ! payload="$(bash .gtt/scripts/gtt-session-context.sh)"; then
  echo "GTT session hook (codex): .gtt/scripts/gtt-session-context.sh failed; no session context was provided." >&2
  exit 1
fi
if [ -z "$payload" ]; then
  echo "GTT session hook (codex): .gtt/scripts/gtt-session-context.sh produced no output." >&2
  exit 1
fi
printf '%s\n' "$payload"
