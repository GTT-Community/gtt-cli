#!/usr/bin/env bash
# GTT - Bootstrap template service. ADE-independent entry point.
#
# The Bootstrap owns its templates (today: the Initial Design Questionnaire).
# A CLI asks here what exists, whether it fits this scaffold and where a
# working copy goes, then requests the copy - it never carries the template or
# its methodology itself.
#
#   list                          the templates the Bootstrap declares
#   show <id>                     path, availability, compatibility, target
#   materialize <id> [--apply]    copy to the declared working location
#                                 (dry run by default; never overwrites)
#
# A materialized copy is source material for the governed design process, not
# governed context and not a decision.
#
# Usage (from the project root):
#   .gtt/scripts/gtt-template.sh <list|show|materialize> [id] [--from CATALOG] [--json|--apply]
#
# Exit 0 = ok, 1 = conflict / unavailable, 2 = cannot run.

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
exec bash "$HERE/gtt-run-python.sh" "$HERE/gtt_template.py" "$@"
