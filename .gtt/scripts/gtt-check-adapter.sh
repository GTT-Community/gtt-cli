#!/usr/bin/env bash
# GTT - ADE adapter validation gate.
#
# GTT ships a portable core plus one adapter per supported ADE. An installed
# project carries the overlay of every ADE its human chose to have participate,
# recorded in .gtt/ade.json, exactly one of them Primary. This makes that check
# deterministic instead of a visual scan of the file tree.
#
# Two modes:
#
#   gtt-check-adapter.sh
#       Validate every participating ADE against .gtt/ade.json: the state is
#       well formed, the Primary is one participating ADE, every participating
#       ADE has its integration (overlay path, instruction entry point, the
#       files GTT recorded as installed), and nothing is silently ignored. A
#       detected ADE that does not participate is a WARN, never a pass. Exit 2
#       (cannot determine) when the project has no .gtt/ade.json.
#
#   gtt-check-adapter.sh <claude|kiro|codex|copilot|unknown>
#       The single-ADE matrix of projects that predate .gtt/ade.json, kept
#       unchanged: the named ADE's adapter present, every other one absent.
#       Migrate such a project with `gtt-ade.sh adopt`.
#
# Usage:
#   .gtt/scripts/gtt-check-adapter.sh [<claude|kiro|codex|copilot|unknown>]
#
# Exit 0 = ok, 1 = violation, 2 = bad usage or cannot determine.

set -euo pipefail

ADE="${1:-}"

usage() {
  echo "usage: gtt-check-adapter.sh [<claude|kiro|codex|copilot|unknown>]" >&2
  exit 2
}

if [ -z "$ADE" ]; then
  if [ ! -f .gtt/ade.json ]; then
    echo "gtt-check-adapter: cannot determine - no .gtt/ade.json (multi-ADE state). Run 'gtt-ade.sh adopt', or pass an ADE for the single-ADE matrix." >&2
    exit 2
  fi
  if bash "$(dirname "$0")/gtt-ade.sh" validate; then
    echo "gtt-check-adapter: OK - every participating ADE has an intact integration."
    exit 0
  fi
  exit 1
fi

case "$ADE" in
  claude)  EXPECT_CLAUDE=yes; EXPECT_KIRO=no;  EXPECT_COPILOT=no;  EXPECT_AGENTS=yes ;;
  kiro)    EXPECT_CLAUDE=no;  EXPECT_KIRO=yes; EXPECT_COPILOT=no;  EXPECT_AGENTS=yes ;;
  codex)   EXPECT_CLAUDE=no;  EXPECT_KIRO=no;  EXPECT_COPILOT=no;  EXPECT_AGENTS=yes ;;
  copilot) EXPECT_CLAUDE=no;  EXPECT_KIRO=no;  EXPECT_COPILOT=yes; EXPECT_AGENTS=yes ;;
  unknown) EXPECT_CLAUDE=no;  EXPECT_KIRO=no;  EXPECT_COPILOT=no;  EXPECT_AGENTS=skip ;;
  *) usage ;;
esac

FAIL=0

check() {
  label="$1"; path="$2"; expect="$3"

  if [ "$expect" = skip ]; then
    return 0
  fi

  if [ -e "$path" ]; then
    present=yes
  else
    present=no
  fi

  if [ "$present" != "$expect" ]; then
    echo "gtt-check-adapter: FAILED - $label: expected $expect, found $present ($path)" >&2
    FAIL=1
  fi
}

check ".claude/"                        ".claude"                          "$EXPECT_CLAUDE"
check ".kiro/"                           ".kiro"                           "$EXPECT_KIRO"
check ".copilot/copilot-instructions.md"  ".copilot/copilot-instructions.md" "$EXPECT_COPILOT"
check "AGENTS.md"                        "AGENTS.md"                       "$EXPECT_AGENTS"

if [ "$FAIL" -ne 0 ]; then
  cat >&2 <<MSG

gtt-check-adapter: workspace does not match the '$ADE' single-ADE matrix.

This is the matrix of a project with exactly one ADE. A project with several
participating ADEs is validated against .gtt/ade.json instead (run this script
with no argument); a project that predates it is migrated with
'gtt-ade.sh adopt'. See 'ADE adapters' in readme-gtt.md or step 0 of the
gtt-bootstrap skill.
MSG
  exit 1
fi

echo "gtt-check-adapter: OK - workspace matches the '$ADE' adapter matrix."
