#!/usr/bin/env bash
# GTT - PROMOTION SCRIPT (human execution required)
#
# Status: Draft - not a decision. Prepared by an agent; never run by one.
# Running it does NOT ratify anything: adopting an ADE adapter changes the
# adapter matrix (AGENTS.md, gtt-check-adapter.sh, the gtt-bootstrap skill) and
# needs Proposal -> Epic/Story -> ADR -> human ratification. This only copies
# staged files to the ADE's native paths and flips the declaration to
# "installed" so you can try them.
#
# Installs the staged Session Memory adapters you name:
#   codex    -> .codex/hooks.json, .codex/gtt-session-start.sh
#   copilot  -> .github/hooks/gtt-session.json, .github/hooks/gtt-session-start.py
#   kiro     -> .kiro/agents/gtt-session.json, .kiro/gtt-session-start.sh
# It refuses to overwrite an existing file. On any failure it removes what it
# copied and restores the declarations.
#
# Usage (from the project root):
#   bash gtt-domain/proposals/apply-session-adapters.sh <codex|copilot|kiro>...

set -euo pipefail

[ "$#" -ge 1 ] || { echo "usage: apply-session-adapters.sh <codex|copilot|kiro>..." >&2; exit 2; }
for ade in "$@"; do
  case "$ade" in codex|copilot|kiro) ;; *) echo "unknown adapter '$ade' (claude is already installed)" >&2; exit 2 ;; esac
  [ -f ".gtt/session-adapters/$ade.json" ] || { echo "run from the project root: .gtt/session-adapters/$ade.json not found" >&2; exit 2; }
  grep -q '"adapter_status": "staged"' ".gtt/session-adapters/$ade.json" \
    || { echo "$ade is not staged (already installed?)" >&2; exit 2; }
done

STAGE_ROOT="gtt-domain/proposals/session-adapters"
COPIED=()
FLIPPED=()

echo "=================================================================="
echo " GTT promotion: Session Memory adapters   (HUMAN EXECUTION)"
echo "=================================================================="
for ade in "$@"; do
  echo
  echo "--- $ade: files that would be created ---"
  ( cd "$STAGE_ROOT/$ade" && find . -type f | sed 's|^\./||' | sort ) | while read -r f; do
    if [ -e "$f" ]; then echo "  CONFLICT (exists, will abort): $f"; else echo "  $f"; fi
  done
done
echo
printf "Type 'apply' to install these adapters (anything else aborts): "
read -r ANSWER
[ "$ANSWER" = "apply" ] || { echo "aborted - nothing was written."; exit 1; }

rollback() {
  echo "failed - rolling back" >&2
  for f in "${COPIED[@]:-}"; do [ -n "$f" ] && rm -f "$f"; done
  for m in "${FLIPPED[@]:-}"; do
    [ -n "$m" ] && sed -i 's/"adapter_status": "installed"/"adapter_status": "staged"/' "$m"
  done
  exit 1
}

for ade in "$@"; do
  while read -r f; do
    [ -e "$f" ] && { echo "refusing to overwrite existing $f" >&2; rollback; }
    mkdir -p "$(dirname "$f")"
    cp "$STAGE_ROOT/$ade/$f" "$f" || rollback
    COPIED+=("$f")
  done < <( cd "$STAGE_ROOT/$ade" && find . -type f | sed 's|^\./||' | sort )
  sed -i 's/"adapter_status": "staged"/"adapter_status": "installed"/' ".gtt/session-adapters/$ade.json"
  FLIPPED+=(".gtt/session-adapters/$ade.json")
done

for ade in "$@"; do
  bash .gtt/scripts/gtt-check-session-adapter.sh "$ade" || rollback
done
bash .gtt/scripts/gtt-validate.sh || rollback

# One ownership contract for clean / export --clean: attribute the files just
# installed to their ADE in the .gtt/ade.json ledger. Only meaningful when that ADE
# participates; otherwise say so - never guess, never fail the install over it.
if [ -f .gtt/ade.json ] && [ -f .gtt/scripts/gtt-ade.sh ]; then
  for ade in "$@"; do
    RECORD=()
    while IFS= read -r p; do [ -n "$p" ] && RECORD+=("$p"); done < <(grep -oE '"install_path": *"[^"]+"' ".gtt/session-adapters/$ade.json" | sed -E 's/.*: *"([^"]+)"/\1/')
    if bash .gtt/scripts/gtt-ade.sh record "$ade" "${RECORD[@]}" --apply; then
      echo "recorded $ade's adapter files in .gtt/ade.json"
    else
      echo "WARNING: $ade's files were NOT recorded (is $ade a participating ADE?)." >&2
      echo "         clean / export --clean will keep them until you run: gtt-ade.sh record $ade <paths> --apply" >&2
    fi
  done
else
  echo "note: no .gtt/ade.json here (catalog or pre-multi-ADE project); nothing recorded."
fi

echo
echo "Installed: $*. Runtime verification is still whatever the declaration says -"
echo "see .gtt/session-adapters/<ade>.json. Review with 'git status'; commit when satisfied."
