#!/usr/bin/env bash
# GTT - Bootstrap contract gate.
#
# Deterministic consistency of .gtt/contract/: release identity and versions, capability and operation
# registries (implementations exist under .gtt/scripts/, no shell syntax, typed arguments, mutating
# operations dry-run by default, no unfreeze operation), the four Method Plans and their
# invariants (none relaxable, only Light may relax anything), export policy, recovery, elicitation
# references into the questionnaire, ADE integration versions. Makes no architectural judgment.
#
# Usage:
#   .gtt/scripts/gtt-check-contract.sh
#
# Exit 0 = ok, 1 = violation, 2 = cannot run.

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
exec bash "$HERE/gtt-run-python.sh" "$HERE/gtt_contract.py" check "$@"
