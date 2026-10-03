#!/usr/bin/env bash
# GTT - Bootstrap 1.0 contract entry point (ADE-independent, CLI-facing).
#
# What the Bootstrap offers and how to invoke it, as machine-readable contracts (.gtt/contract/):
#   release [--json]                      Bootstrap release identity (distinct from the scaffold version)
#   negotiate --cli-version V --cli-capabilities a,b --cli-schemas 1 [--json]
#                                         COMPATIBLE, or REFUSE (exit 3) with reasons - never modifies anything
#   capabilities | operations [--json]    the capability and operation registries
#   show <profiles|export-policy|recovery|elicitation|ade-registry|initial-design>
#   run <operation> [name=value ...] [--envelope]
#                                         execute ONE declared operation: typed arguments, fixed argv, no shell,
#                                         implementation confined to .gtt/scripts/
#   check [--json]                        integrity of the contracts themselves
#
# Exit 0 ok, 1 violation, 2 usage, 3 REFUSED (incompatible), 4 human authority required, 5 invalid / undeclared.
#
# Usage (from the project root):
#   .gtt/scripts/gtt-contract.sh <command> [options]

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
exec bash "$HERE/gtt-run-python.sh" "$HERE/gtt_contract.py" "$@"
