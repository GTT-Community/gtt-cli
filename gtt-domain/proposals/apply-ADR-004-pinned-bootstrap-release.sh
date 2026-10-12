#!/usr/bin/env bash
set -euo pipefail

# GTT GOVERNED CHANGE
# Proposal: PROPOSAL-unified-cli-bootstrap-install
# ADR: ADR-004-pinned-bootstrap-release
# Human execution required: YES
#
# Review the proposal, the ADR draft, and this script before running it.
# This script must not be executed automatically by an AI agent.

cd "$(dirname "${BASH_SOURCE[0]}")/../.." # run from the project root regardless of cwd

ADR_DRAFT="gtt-domain/proposals/ADR-DRAFT-pinned-bootstrap-release.md"
ADR_FINAL="gtt-domain/adr/ADR-004-pinned-bootstrap-release.md"
STACK_DRAFT="gtt-domain/proposals/context-stack-ADR-004.md"
STACK_FINAL="gtt-domain/context/stack.md"
SOURCES_DRAFT="gtt-domain/proposals/context-sources-ADR-004.md"
SOURCES_FINAL="gtt-domain/context/sources.md"

# Preconditions: fail before touching anything if the expected state is absent.
for f in "$ADR_DRAFT" "$STACK_DRAFT" "$SOURCES_DRAFT" "$STACK_FINAL" "$SOURCES_FINAL"; do
    if [ ! -f "$f" ]; then
        echo "ERROR: expected file not found: $f. Nothing was changed." >&2
        exit 1
    fi
done
if [ -e "$ADR_FINAL" ]; then
    echo "ERROR: $ADR_FINAL already exists. Nothing was changed." >&2
    exit 1
fi
if [ ! -f "TASK-gtt-cli-bootstrap-unified-install.md" ]; then
    echo "ERROR: the source document TASK-gtt-cli-bootstrap-unified-install.md is not at the project root. Nothing was changed." >&2
    exit 1
fi

echo "GTT governed change"
echo "Proposal: PROPOSAL-unified-cli-bootstrap-install"
echo "ADR: ADR-004-pinned-bootstrap-release"
echo ""
echo "This will apply:"
echo "  $ADR_DRAFT  ->  $ADR_FINAL   (new file)"
echo "  $STACK_DRAFT  ->  $STACK_FINAL   (overwrite)"
echo "  $SOURCES_DRAFT  ->  $SOURCES_FINAL   (overwrite)"
echo ""

while true; do
    read -r -p "Review the change, or approve it? [view/yes/no]: " choice
    case "$choice" in
        view|v)
            echo ""
            echo "===== $ADR_DRAFT (new ADR) ====="
            cat "$ADR_DRAFT"
            echo ""
            echo "===== $STACK_FINAL: current -> staged ====="
            diff -u "$STACK_FINAL" "$STACK_DRAFT" || true
            echo ""
            echo "===== $SOURCES_FINAL: current -> staged ====="
            diff -u "$SOURCES_FINAL" "$SOURCES_DRAFT" || true
            echo ""
            ;;
        yes|y)
            break
            ;;
        no|n)
            echo "Change not promoted."
            exit 1
            ;;
        *)
            echo "Please answer view, yes, or no."
            ;;
    esac
done

mv "$ADR_DRAFT" "$ADR_FINAL"
cp "$STACK_DRAFT" "$STACK_FINAL"
cp "$SOURCES_DRAFT" "$SOURCES_FINAL"

bash .gtt/scripts/gtt-check-stack.sh

rm -f "$STACK_DRAFT" "$SOURCES_DRAFT"

echo ""
echo "Promoted. ADR-004-pinned-bootstrap-release is now in gtt-domain/adr/ and gtt-domain/context/ reflects it."
echo "The staged drafts for this change were removed from gtt-domain/proposals/."
