#!/usr/bin/env bash
# GTT - validate. Deterministic aggregator over the existing CI gates:
# backlog structural integrity, ADE-adapter matrix, GTTGuard registry, the
# stack map, the markdown canonical-reference gate, and artifact identity /
# repository integrity (broken references, duplicate identity, stale index),
# and Session Memory adapter conformance (one line per declared adapter). Reuses each check
# script's own logic rather than reimplementing it - this script only runs
# them and summarizes.
#
# Adapter check, two regimes. When .gtt/ade.json exists (an installed project
# with declared participating ADEs) every participating ADE's integration is
# validated against it - a missing or inconsistent integration FAILS, a detected
# ADE that does not participate is a WARN. Without it (a project that predates
# multi-ADE, or this source repository, the catalog) the check is skipped, not
# failed, when zero or more than one adapter is present: the catalog legitimately
# ships every adapter, which is not the "wrong adapter installed" violation
# gtt-check-adapter.sh exists to catch. Run `.gtt/scripts/gtt-check-adapter.sh
# <ade>` by hand there if needed, or `gtt-ade.sh adopt` to declare the state.
#
# Usage:
#   .gtt/scripts/gtt-validate.sh
#
# Exit 0 = every check passed or was skipped, 1 = at least one check failed.

set -uo pipefail

FAIL=0
report() {
  # $1=label $2=exit-code (2 means "cannot determine", not a failure)
  case "$2" in
    0) echo "PASS               $1" ;;
    2) echo "CANNOT-DETERMINE   $1" ;;
    *) echo "FAIL               $1"; FAIL=1 ;;
  esac
}

bash .gtt/scripts/gtt-check-backlog.sh >/tmp/gtt-validate-backlog.$$ 2>&1
BACKLOG_RC=$?
report "gtt-check-backlog.sh" "$BACKLOG_RC"
[ "$BACKLOG_RC" -ne 0 ] && cat /tmp/gtt-validate-backlog.$$
rm -f /tmp/gtt-validate-backlog.$$

# The Core does not enumerate ADE names or paths itself. It discovers the
# set of known ADEs from .gtt/session-adapters/*.json - the same declaration
# registry the Session Memory loop below already uses - and asks
# gtt-check-adapter.sh (the one script whose contract is to know per-ADE
# paths) whether each one matches this workspace. Adding
# .gtt/session-adapters/<new-ade>.json tomorrow is picked up here with no
# edit to this file.
known_ades=()
for manifest in .gtt/session-adapters/*.json; do
  [ -f "$manifest" ] || continue
  known_ades+=("$(basename "$manifest" .json)")
done

if [ -f .gtt/ade.json ]; then
  adapter_out="$(bash .gtt/scripts/gtt-check-adapter.sh 2>&1)"
  adapter_rc=$?
  report "gtt-check-adapter.sh (participating ADEs, .gtt/ade.json)" "$adapter_rc"
  printf '%s
' "$adapter_out" | sed 's/^/                   /'
else
  matched_ades=()
  for ade in "${known_ades[@]}"; do
    bash .gtt/scripts/gtt-check-adapter.sh "$ade" >/dev/null 2>&1 && matched_ades+=("$ade")
  done

  if [ "${#matched_ades[@]}" -eq 1 ]; then
    report "gtt-check-adapter.sh ${matched_ades[0]}" 0
  else
    echo "SKIPPED            gtt-check-adapter.sh (${#matched_ades[@]} of ${#known_ades[@]} declared adapters match this workspace and there is no .gtt/ade.json - catalog repo or ambiguous; run 'gtt-ade.sh adopt' to declare the participating ADEs)"
  fi
fi

bash .gtt/scripts/gtt-check-protection.sh >/tmp/gtt-validate-protection.$$ 2>&1
PROTECTION_RC=$?
report "gtt-check-protection.sh" "$PROTECTION_RC"
[ "$PROTECTION_RC" -ne 0 ] && cat /tmp/gtt-validate-protection.$$
rm -f /tmp/gtt-validate-protection.$$

bash .gtt/scripts/gtt-check-stack.sh >/tmp/gtt-validate-stack.$$ 2>&1
STACK_RC=$?
report "gtt-check-stack.sh" "$STACK_RC"
[ "$STACK_RC" -ne 0 ] && cat /tmp/gtt-validate-stack.$$
rm -f /tmp/gtt-validate-stack.$$

bash .gtt/scripts/gtt-check-markdown.sh >/tmp/gtt-validate-markdown.$$ 2>&1
MARKDOWN_RC=$?
report "gtt-check-markdown.sh" "$MARKDOWN_RC"
[ "$MARKDOWN_RC" -ne 0 ] && cat /tmp/gtt-validate-markdown.$$
rm -f /tmp/gtt-validate-markdown.$$

if [ -f .gtt/scripts/gtt-check-provenance.sh ]; then
  bash .gtt/scripts/gtt-check-provenance.sh >/tmp/gtt-validate-provenance.$$ 2>&1
  PROVENANCE_RC=$?
  report "gtt-check-provenance.sh" "$PROVENANCE_RC"
  [ "$PROVENANCE_RC" -ne 0 ] && cat /tmp/gtt-validate-provenance.$$
  rm -f /tmp/gtt-validate-provenance.$$
fi

if [ -f .gtt/scripts/gtt-check-contract.sh ]; then
  bash .gtt/scripts/gtt-check-contract.sh >/tmp/gtt-validate-contract.$$ 2>&1
  CONTRACT_RC=$?
  report "gtt-check-contract.sh" "$CONTRACT_RC"
  [ "$CONTRACT_RC" -ne 0 ] && cat /tmp/gtt-validate-contract.$$
  rm -f /tmp/gtt-validate-contract.$$
fi

bash .gtt/scripts/gtt-check-integrity.sh >/tmp/gtt-validate-integrity.$$ 2>&1
INTEGRITY_RC=$?
report "gtt-check-integrity.sh" "$INTEGRITY_RC"
[ "$INTEGRITY_RC" -ne 0 ] && cat /tmp/gtt-validate-integrity.$$
rm -f /tmp/gtt-validate-integrity.$$

# Session Memory adapter conformance: one result per declared adapter. The
# check is static and needs no ADE; runtime verification by the real ADE is
# never run here, so each line carries the DECLARED runtime status instead of
# implying it was proven. A staged (not yet promoted) adapter is labelled so.
for manifest in .gtt/session-adapters/*.json; do
  [ -f "$manifest" ] || continue
  ade="$(basename "$manifest" .json)"
  out="$(bash .gtt/scripts/gtt-check-session-adapter.sh "$ade" 2>&1)"
  rc=$?
  summary="$(printf '%s
' "$out" | grep '^SUMMARY ' | sed 's/^SUMMARY //')"
  if [ "$rc" -eq 0 ]; then
    echo "PASS               gtt-check-session-adapter.sh $ade  [$summary]"
  elif [ "$rc" -eq 2 ]; then
    echo "CANNOT-DETERMINE   gtt-check-session-adapter.sh $ade"
    printf '%s
' "$out"
  else
    echo "FAIL               gtt-check-session-adapter.sh $ade"
    FAIL=1
    printf '%s
' "$out" | grep -E '^(FAIL|SUMMARY)'
  fi
done

echo
if [ "$FAIL" -ne 0 ]; then
  echo "gtt-validate: FAILED - see the FAIL line(s) above."
  exit 1
fi
echo "gtt-validate: OK"
exit 0
