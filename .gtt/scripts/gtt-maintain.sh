#!/usr/bin/env bash
# GTT - maintain. The deterministic work that follows any operation, in one run
# and with a brief report: GTTGuard registry sync, artifact index rebuild, and
# gtt-validate.sh. Those three are derived state or read-only checks - automatic
# in every Method Plan - so nothing here asks a question. It reuses the existing
# scripts; it reimplements none of them and makes no architectural judgment.
#
# One step depends on the selected plan (.gtt/contract/profiles.json ->
# plan.policy.automation.reference_updates): when artifacts were moved or renamed,
# rewriting the references to them is a relevant change.
#   Light                  reference updates are automatic: the unambiguous moves
#                          are reconciled here, and reported.
#   Medium, Hard, Team,    reference updates are confirmed: it stops and gives the
#   or no plan selected    exact command; nothing is rewritten.
# In every plan an ambiguous move or a missing artifact is a decision, never
# resolved here.
#
# What it never does: touch governed context, run a promotion script, or freeze.
#
# Usage (from the project root):
#   .gtt/scripts/gtt-maintain.sh             brief report (default)
#   .gtt/scripts/gtt-maintain.sh --verbose   the full output of every step
#
# Exit 0 = everything in order, 1 = a step failed or a human action is pending,
# 2 = cannot run.

set -uo pipefail

VERBOSE=0
for arg in "$@"; do
  case "$arg" in
    --verbose) VERBOSE=1 ;;
    *) echo "usage: gtt-maintain.sh [--verbose]" >&2; exit 2 ;;
  esac
done
[ -f .gtt/scripts/gtt-validate.sh ] || { echo "gtt-maintain: run from the project root (.gtt/scripts/ not found)" >&2; exit 2; }

OK="✓"; WARN="⚠"; BAD="✗"; NEXT="→"
case "${LC_ALL:-${LC_CTYPE:-${LANG:-}}}" in
  *[Uu][Tt][Ff]*) ;;
  *) OK="ok"; WARN="!!"; BAD="FAIL"; NEXT="->" ;;
esac

PLAN="$(bash .gtt/scripts/gtt-project.sh interaction --key plan 2>/dev/null || echo medium)"
[ "$(bash .gtt/scripts/gtt-project.sh interaction --key selected 2>/dev/null)" = "true" ] || PLAN="no plan selected"
RC=0
detail() { [ "$VERBOSE" -eq 1 ] && printf '%s\n' "$1" | sed 's/^/    /'; return 0; }
failed() {
  # $1=summary $2=full output $3=command to see it again
  echo "$BAD $1"
  printf '%s\n' "$2" | sed 's/^/    /'
  echo "$NEXT $3"
  RC=1
}

out="$(bash .gtt/scripts/gtt-guard-sync.sh 2>&1)"
if [ $? -eq 0 ]; then
  echo "$OK Protection registry in sync."
  detail "$out"
else
  failed "Protection registry sync failed." "$out" "bash .gtt/scripts/gtt-guard-sync.sh"
fi

out="$(bash .gtt/scripts/gtt-reconcile.sh 2>&1)"
if printf '%s' "$out" | grep -q 'nothing to reconcile'; then
  out="$(bash .gtt/scripts/gtt-index.sh 2>&1)"
  if [ $? -eq 0 ]; then
    echo "$OK Index rebuilt$(printf '%s\n' "$out" | sed -n 's/^gtt-index: OK - \(.*\) indexed.*/ (\1)/p')."
    detail "$out"
  else
    failed "Index rebuild failed." "$out" "bash .gtt/scripts/gtt-index.sh"
  fi
elif [ "$(bash .gtt/scripts/gtt-project.sh interaction --key maintain_reconciles_moves 2>/dev/null)" = "true" ]; then
  out="$(bash .gtt/scripts/gtt-reconcile.sh --apply 2>&1)"
  if [ $? -eq 0 ]; then
    echo "$OK Reconciled $(printf '%s\n' "$out" | grep -c '^MOVE ') moved artifact(s) and rebuilt the index ($PLAN: reference updates are automatic)."
    printf '%s\n' "$out" | grep -E '^(MOVE|rewrote|SKIPPED) ' | sed 's/^/    /'
  else
    failed "Some moved or missing artifacts need your decision; the index was not rebuilt." "$out" "bash .gtt/scripts/gtt-reconcile.sh"
  fi
else
  echo "$WARN Moved or renamed artifacts need reconciling; the index was not rebuilt ($PLAN: reference updates are confirmed)."
  detail "$out"
  echo "$NEXT bash .gtt/scripts/gtt-reconcile.sh          (shows the plan; add --apply to write it)"
  RC=1
fi

out="$(bash .gtt/scripts/gtt-validate.sh 2>&1)"
vrc=$?
count() { printf '%s\n' "$out" | grep -c "^$1 " || true; }
summary="$(count PASS) passed"
[ "$(count SKIPPED)" -gt 0 ] && summary="$summary, $(count SKIPPED) skipped"
[ "$(count CANNOT-DETERMINE)" -gt 0 ] && summary="$summary, $(count CANNOT-DETERMINE) cannot determine"
if [ "$vrc" -eq 0 ]; then
  echo "$OK Validation: $summary."
  detail "$out"
else
  if [ "$VERBOSE" -eq 1 ]; then shown="$out"; else shown="$(printf '%s\n' "$out" | grep -vE '^(PASS|SKIPPED) |^$|^gtt-validate: ')"; fi
  failed "Validation: $(count FAIL) failed, $summary." "$shown" "bash .gtt/scripts/gtt-validate.sh"
fi

[ "$RC" -eq 0 ] && [ "$VERBOSE" -eq 0 ] && echo "  (details: .gtt/scripts/gtt-maintain.sh --verbose)"
exit "$RC"
