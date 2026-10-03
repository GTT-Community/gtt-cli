#!/usr/bin/env bash
# GTT - project-facing Bootstrap contracts (ADE-independent, CLI-facing).
#
# Structured, deterministic views and small per-project state, derived from the actual project:
#   detect                      project detection (GTT state, layout, freeze, design-source candidates)
#   status [--with-validation]  structured status: bootstrap, ade, methodology, sources, governance, freeze, validation, session
#   session                     structured session context (operational-only, never authority)
#   validation                  gtt-validate.sh as one structured result
#   next-id --kind adr|epic|story   the next free id; deterministic, never reuses a retired id
#   profile get|set             Method Plan light|medium|hard|team and language (meaning: .gtt/contract/profiles.json)
#   source select|list          initial sources; a selected source is never a governed authority
#   export-policy | clean-plan  clean-export policy / what `clean` would remove (removes nothing)
#   recovery snapshot|restore   preserve / re-apply GTT configuration
#
# Add --json for machine-readable output. Mutating commands are dry runs unless --apply, never overwrite state.
#
# Usage (from the project root):
#   .gtt/scripts/gtt-project.sh <command> [options]
#
# Exit 0 = ok, 1 = violation / refused, 2 = cannot run.

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
exec bash "$HERE/gtt-run-python.sh" "$HERE/gtt_project.py" "$@"
