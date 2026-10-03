#!/usr/bin/env bash
# GTT - ADE integration service (Multi-ADE). ADE-independent entry point.
#
# One GTT governance model, several ADE integration surfaces, one Primary ADE.
# The Bootstrap - not the CLI - knows which ADEs exist, where each integrates,
# which the human chose, and exactly which files GTT installed for them.
#
#   list                  the ADE integration registry (manifest `overlays:`)
#   detect                candidate ADEs (observation only, never authorization)
#   state                 primary / participating / integration state
#   validate              every participating integration is intact (exit 1 if not)
#   owned                 the GTT-installed ADE surfaces: what `clean` and
#                         `export --clean` may remove, and nothing else
#   install               copy the chosen overlays and record them
#   adopt                 record an integration that predates .gtt/ade.json
#   set-primary <ade>     change the Primary ADE (a workflow identifier only)
#   record <ade> <path>   add files to an ADE's install ledger
#   remove <ade>...|--all remove GTT-installed integration
#   update                evaluate and migrate every participating overlay
#
# Every mutating command is a dry run unless --apply is given, never
# overwrites a file it did not install, and rolls back on failure. The
# per-project state is .gtt/ade.json; never hand-edit it. Nothing here grants
# any ADE authority over a governed artifact.
#
# Usage (from the project root):
#   .gtt/scripts/gtt-ade.sh <command> [options]      # see: gtt-ade.sh <command> --help
#
# Exit 0 = ok, 1 = violation / conflict, 2 = cannot run.

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
exec bash "$HERE/gtt-run-python.sh" "$HERE/gtt_ade.py" "$@"
