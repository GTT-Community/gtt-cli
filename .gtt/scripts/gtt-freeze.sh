#!/usr/bin/env bash
# GTT - freeze. The act of ratification.
#
# Run by the Solution Designer, never by an agent. Validates that L0 holds
# real content, then writes the marker that switches the project from the
# pre-freeze regime to the governed regime.
#
# Exit 0 = frozen, 1 = validation failed, 2 = already frozen.

set -euo pipefail

MARKER="gtt-domain/.frozen"
CTX="gtt-domain/context"
REQUIRED=(stack.md architecture.md solution-vision.md principles.md constraints.md glossary.md)

if [ -f "$MARKER" ]; then
  echo "gtt-freeze: already frozen ($MARKER exists, written $(cat "$MARKER" | head -1))." >&2
  echo "             Freeze is not idempotent by accident. Nothing to do." >&2
  exit 2
fi

fail=0
for f in "${REQUIRED[@]}"; do
  path="$CTX/$f"
  if [ ! -f "$path" ]; then
    echo "gtt-freeze: missing $path" >&2
    fail=1
    continue
  fi
  lines="$(grep -cve '^\s*$' "$path" || true)"
  if [ "$lines" -le 5 ]; then
    echo "gtt-freeze: $path has $lines non-empty lines - still effectively empty" >&2
    fail=1
  fi
  if grep -qE 'TODO|PLACEHOLDER|REPLACE ME|<[a-z][^>]*>' "$path"; then
    echo "gtt-freeze: $path still contains template placeholders" >&2
    fail=1
  fi
done

if ! ls gtt-domain/adr/ADR-001*.md >/dev/null 2>&1; then
  echo "gtt-freeze: ADR-001 not found under gtt-domain/adr/" >&2
  fail=1
fi

# Provenance / gap / source rules: a pending BLOCKING gap, an unresolved
# [CONFLICTO], a [PROPUESTA] inside governed context or a malformed source manifest
# refuse the freeze. A valid OPEN gap does not: it is known, scoped and visible.
if [ -f .gtt/scripts/gtt-check-provenance.sh ]; then
  if ! bash .gtt/scripts/gtt-check-provenance.sh --pre-freeze >&2; then
    echo "gtt-freeze: provenance / gap / source rules do not hold (see above)" >&2
    fail=1
  fi
fi

if [ "$fail" -ne 0 ]; then
  cat >&2 <<'MSG'

gtt-freeze: FAILED

Freezing is a ratification, not a file move. Context that is still template
text would be ratified as if it were a decision. Fill in the files above, or
run the gtt-bootstrap skill, then try again.
MSG
  exit 1
fi

{
  date -u +"%Y-%m-%dT%H:%M:%SZ"
  echo "Frozen by: ${USER:-unknown}"
} > "$MARKER"

if [ -d "gtt-domain/proposals/bootstrap" ]; then
  rm -rf gtt-domain/proposals/bootstrap
  echo "gtt-freeze: removed gtt-domain/proposals/bootstrap (superseded by ratified context)."
fi

cat <<'MSG'
gtt-freeze: DONE

gtt-domain/context/ and gtt-domain/adr/ are now read-only for agents. From here, changes
go through gtt-domain/change-request.md -> gtt-domain/proposals/ -> you apply.

Commit gtt-domain/.frozen. It is versioned on purpose: a clone of a frozen project
stays frozen.
MSG
