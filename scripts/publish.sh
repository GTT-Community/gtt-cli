#!/usr/bin/env bash
# Publish gtt-cli-dev (private, governed with GTT) to gtt-cli (public).
#
# Usage, from the root of gtt-cli-dev:
#   bash scripts/publish.sh <path-to-public-clone> [version]
#
#   <path-to-public-clone>  a git clone of github.com/GTT-Community/gtt-cli
#   [version]               optional, e.g. 1.0.0: commits and tags v<version>
#
# What it does: checks that the development tree is committed and its unit
# tests pass, exports a clean copy with `gtt export --clean` (no GTT files),
# removes the development-only paths listed in .publishignore, and mirrors
# the result into the public clone. It never pushes: you review and push.
set -euo pipefail

PUBLIC="${1:-}"
VERSION="${2:-}"
if [ -z "$PUBLIC" ]; then
    echo "usage: bash scripts/publish.sh <path-to-public-clone> [version]" >&2
    exit 3
fi
cd "$(dirname "${BASH_SOURCE[0]}")/.."
ROOT="$(pwd)"
PUBLIC="$(cd "$PUBLIC" && pwd)"

if [ "$PUBLIC" = "$ROOT" ]; then
    echo "ERROR: the public clone cannot be this repository." >&2
    exit 3
fi
if [ ! -d "$PUBLIC/.git" ]; then
    echo "ERROR: $PUBLIC is not a git clone of the public repository." >&2
    exit 3
fi
if [ -n "$(git status --porcelain)" ]; then
    echo "ERROR: gtt-cli-dev has uncommitted changes. Commit them first, so the release matches a revision." >&2
    exit 5
fi
if [ -n "$(git -C "$PUBLIC" status --porcelain)" ]; then
    echo "ERROR: the public clone has uncommitted changes. It is generated: discard them first." >&2
    exit 5
fi
if [ -n "$VERSION" ] && ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "ERROR: version must be MAJOR.MINOR.PATCH, got $VERSION" >&2
    exit 3
fi
if [ -n "$VERSION" ] && git -C "$PUBLIC" rev-parse -q --verify "refs/tags/v$VERSION" >/dev/null; then
    echo "ERROR: tag v$VERSION already exists in the public clone." >&2
    exit 5
fi

GO="$(command -v go || echo "$HOME/.local/go/bin/go")"
echo "→ Unit and architecture tests"
"$GO" vet ./...
"$GO" test ./internal/... ./test/arch/ >/dev/null
echo "→ Building gtt"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
"$GO" build -o "$TMP/gtt" ./cmd/gtt

echo "→ Clean export (no GTT files)"
"$TMP/gtt" export --clean "$TMP/export" --yes >/dev/null

echo "→ Removing development-only paths (.publishignore)"
while IFS= read -r pattern; do
    pattern="${pattern%%#*}"
    pattern="$(echo "$pattern" | xargs)"
    [ -z "$pattern" ] && continue
    pattern="${pattern%/}"
    case "$pattern" in
        /*|*..*) echo "ERROR: unsafe pattern in .publishignore: $pattern" >&2; exit 6 ;;
    esac
    (cd "$TMP/export" && for match in $pattern; do
        if [ -e "$match" ]; then rm -rf -- "$match"; fi
    done)
done < .publishignore
find "$TMP/export" -mindepth 1 -type d -empty -delete

for leftover in .gtt gtt-domain AGENTS.md readme-gtt.md readme-gtt.es.md; do
    if [ -e "$TMP/export/$leftover" ]; then
        echo "ERROR: GTT material in the export: $leftover. Nothing was published." >&2
        exit 6
    fi
done

echo "→ Mirroring into $PUBLIC"
rsync -a --delete --exclude .git "$TMP/export/" "$PUBLIC/"

echo ""
git -C "$PUBLIC" status --short
if [ -z "$(git -C "$PUBLIC" status --porcelain)" ]; then
    echo "The public repository is already up to date."
    exit 0
fi

SOURCE="$(git rev-parse --short HEAD)"
if [ -z "$VERSION" ]; then
    echo ""
    echo "The changes are in the public clone, not committed. Review them, then:"
    echo "  git -C \"$PUBLIC\" add -A && git -C \"$PUBLIC\" commit -m \"sync from gtt-cli-dev $SOURCE\" && git -C \"$PUBLIC\" push"
    exit 0
fi

read -r -p "Commit and tag v$VERSION in the public clone? [yes/no]: " answer
if [ "$answer" != "yes" ]; then
    echo "Not committed. The public clone holds the changes for you to review."
    exit 4
fi
git -C "$PUBLIC" add -A
git -C "$PUBLIC" commit -q -m "release v$VERSION (from gtt-cli-dev $SOURCE)"
git -C "$PUBLIC" tag -a "v$VERSION" -m "gtt-cli v$VERSION"
echo ""
echo "Committed and tagged v$VERSION. Nothing was pushed. To publish:"
echo "  git -C \"$PUBLIC\" push && git -C \"$PUBLIC\" push origin v$VERSION"
