#!/usr/bin/env bash
set -euo pipefail

# GTT GOVERNED CHANGE
# Proposal: decisions of the Solution Designer on 2026-10-03 (binaries in GitHub Releases,
#           private development repository, native Windows support)
# ADR: ADR-003-release-packaging-platforms
# Human execution required: YES
#
# Review the ADR draft, the staged stack.md and this script before running it.
# This script must not be executed automatically by an AI agent.

cd "$(dirname "${BASH_SOURCE[0]}")/../.." # run from the project root regardless of cwd

ADR_DRAFT="gtt-domain/proposals/ADR-DRAFT-release-packaging-platforms.md"
ADR_TARGET="gtt-domain/adr/ADR-003-release-packaging-platforms.md"
STACK_DRAFT="gtt-domain/proposals/context-stack-ADR-003.md"
STACK_TARGET="gtt-domain/context/stack.md"

for f in "$ADR_DRAFT" "$STACK_DRAFT" "$STACK_TARGET"; do
    if [ ! -f "$f" ]; then
        echo "ERROR: expected file is missing: $f" >&2
        exit 1
    fi
done
if ls gtt-domain/adr/ADR-003-*.md >/dev/null 2>&1; then
    echo "ERROR: ADR-003 already exists in gtt-domain/adr/. Nothing was changed." >&2
    exit 1
fi

echo "GTT governed change"
echo "ADR: ADR-003-release-packaging-platforms"
echo ""
echo "This will apply:"
echo "  $ADR_DRAFT  ->  $ADR_TARGET   (new file)"
echo "  $STACK_DRAFT  ->  $STACK_TARGET   (overwrite)"
echo ""
echo "It records the supported platforms (Linux, macOS, Windows) and GoReleaser /"
echo "GitHub Releases as release packaging in the stack map."
echo ""

while true; do
    read -r -p "Review the change, or approve it? [view/yes/no]: " choice
    case "$choice" in
        view|v)
            echo ""
            echo "===== $ADR_DRAFT (new ADR) ====="
            cat "$ADR_DRAFT"
            echo ""
            echo "===== $STACK_TARGET: current -> staged ====="
            diff -u "$STACK_TARGET" "$STACK_DRAFT" || true
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

mv "$ADR_DRAFT" "$ADR_TARGET"
cp "$STACK_DRAFT" "$STACK_TARGET"

bash .gtt/scripts/gtt-check-stack.sh

echo ""
echo "Promoted. ADR-003-release-packaging-platforms is now in gtt-domain/adr/ and stack.md reflects it."
echo "You can now delete: $STACK_DRAFT and gtt-domain/proposals/apply-ADR-003-release-packaging-platforms.sh"
