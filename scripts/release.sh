#!/usr/bin/env bash
# One command to ship gtt-cli: push the development history to gtt-cli-dev
# (private), publish the clean product to gtt-cli (public) and, with a
# version, tag it so GitHub builds and releases the binaries.
#
# Usage, from the root of gtt-cli-dev, with everything committed:
#   bash scripts/release.sh 1.0.1     # code + tag v1.0.1 -> binaries in Releases
#   bash scripts/release.sh           # code only, no new binaries
#
# The public clone lives next to this repository (../gtt-cli-public, cloned on
# first use; override with GTT_PUBLIC_CLONE). It asks once before publishing.
set -euo pipefail

VERSION="${1:-}"
PUBLIC_URL="https://github.com/GTT-Community/gtt-cli.git"
cd "$(dirname "${BASH_SOURCE[0]}")/.."
PUBLIC="${GTT_PUBLIC_CLONE:-$(cd .. && pwd)/gtt-cli-public}"

if [ -n "$(git status --porcelain)" ]; then
    echo "ERROR: there are uncommitted changes. Commit them first." >&2
    exit 5
fi
if git remote get-url origin 2>/dev/null | grep -q '/gtt-cli\.git$'; then
    echo "ERROR: origin points to the public repository. It must be gtt-cli-dev." >&2
    exit 5
fi

echo "→ Pushing the development history to gtt-cli-dev"
git push origin HEAD

if [ ! -d "$PUBLIC/.git" ]; then
    echo "→ Cloning the public repository into $PUBLIC"
    git clone -q "$PUBLIC_URL" "$PUBLIC"
fi
echo "→ Updating the public clone"
git -C "$PUBLIC" checkout -q main
git -C "$PUBLIC" pull -q --ff-only origin main
git -C "$PUBLIC" fetch -q --tags origin

if [ -z "$VERSION" ]; then
    bash scripts/publish.sh "$PUBLIC"
    if [ -z "$(git -C "$PUBLIC" status --porcelain)" ]; then
        exit 0
    fi
    read -r -p "Publish these changes to the public repository (no new version)? [yes/no]: " answer
    if [ "$answer" != "yes" ]; then
        git -C "$PUBLIC" checkout -q -- . && git -C "$PUBLIC" clean -qfd
        echo "Not published. The public clone was reset."
        exit 4
    fi
    git -C "$PUBLIC" add -A
    git -C "$PUBLIC" commit -q -m "sync from gtt-cli-dev $(git rev-parse --short HEAD)"
    git -C "$PUBLIC" push -q origin main
    echo "✓ Code published. No new binaries (no version given)."
    exit 0
fi

bash scripts/publish.sh "$PUBLIC" "$VERSION"
echo "→ Pushing the public commit and tag v$VERSION"
git -C "$PUBLIC" push -q origin main
git -C "$PUBLIC" push -q origin "v$VERSION"
echo ""
echo "✓ v$VERSION published. GitHub is building the binaries (about two minutes):"
echo "  https://github.com/GTT-Community/gtt-cli/releases/tag/v$VERSION"
