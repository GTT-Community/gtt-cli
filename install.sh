#!/bin/sh
# Install the gtt CLI (Linux, macOS):
#   curl -fsSL https://raw.githubusercontent.com/GTT-Community/gtt-cli/main/install.sh | sh
#
# Options (environment variables):
#   GTT_VERSION=1.2.3       a specific release (default: the latest)
#   GTT_INSTALL_DIR=/path   where to put gtt (default: /usr/local/bin if writable, else ~/.local/bin)
#   GTT_DOWNLOAD_BASE=URL   a mirror holding the release assets
set -eu

REPO="GTT-Community/gtt-cli"

fail() { echo "gtt install: $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || fail "$1 is required"; }

need uname
need tar
if command -v curl >/dev/null 2>&1; then
    fetch() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
    fetch() { wget -q "$1" -O "$2"; }
else
    fail "curl or wget is required"
fi

case "$(uname -s)" in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) fail "unsupported system $(uname -s); on Windows use install.ps1 (PowerShell)" ;;
esac
case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *) fail "unsupported architecture $(uname -m)" ;;
esac

if [ -n "${GTT_VERSION:-}" ]; then
    base="https://github.com/$REPO/releases/download/v${GTT_VERSION#v}"
else
    base="https://github.com/$REPO/releases/latest/download"
fi
base="${GTT_DOWNLOAD_BASE:-$base}"   # a mirror of the release assets, if any
archive="gtt_${os}_${arch}.tar.gz"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

echo "Downloading $archive ..."
fetch "$base/$archive" "$tmp/$archive" || fail "download failed: $base/$archive"
fetch "$base/checksums.txt" "$tmp/checksums.txt" || fail "download failed: $base/checksums.txt"

expected="$(grep " $archive\$" "$tmp/checksums.txt" | cut -d' ' -f1)"
[ -n "$expected" ] || fail "$archive is not listed in checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$tmp/$archive" | cut -d' ' -f1)"
elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$tmp/$archive" | cut -d' ' -f1)"
else
    fail "sha256sum or shasum is required to verify the download"
fi
[ "$expected" = "$actual" ] || fail "checksum mismatch for $archive: expected $expected, got $actual"

tar -xzf "$tmp/$archive" -C "$tmp" gtt

dir="${GTT_INSTALL_DIR:-}"
if [ -z "$dir" ]; then
    if [ -w /usr/local/bin ]; then dir=/usr/local/bin; else dir="$HOME/.local/bin"; fi
fi
mkdir -p "$dir"
install -m 0755 "$tmp/gtt" "$dir/gtt" 2>/dev/null || { cp "$tmp/gtt" "$dir/gtt" && chmod 0755 "$dir/gtt"; }

echo "Installed: $("$dir/gtt" version 2>/dev/null | head -n 1) -> $dir/gtt"
case ":$PATH:" in
    *":$dir:"*) ;;
    *) echo "Add $dir to your PATH, for example: echo 'export PATH=\"$dir:\$PATH\"' >> ~/.bashrc" ;;
esac
echo "GTT Bootstrap also needs git, bash and python3."
