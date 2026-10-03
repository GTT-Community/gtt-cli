#!/usr/bin/env bash
# GTT - provenance gate (ONE script, several sub-checks).
#
# Deterministic checks over the governed artifacts; makes no architectural
# judgment and is never authority. Sub-checks, all optional for a project that
# has not adopted them:
#   tags         [FUENTE] traceable, [VACIO] classified, [CONFLICTO] naming its sources,
#                [PROPUESTA] never inside governed context
#   gaps         OPEN has a scope and is never an authorisation; BLOCKING pending is a
#                violation before/after freeze; RESOLVED cites an existing ADR
#   sources      source manifest: authority, unambiguous precedence, existing paths,
#                SOURCE-BRIEF is evidence after freeze
#   preferences  working agreements never override governed context; user-level stays local
#
# Usage:
#   .gtt/scripts/gtt-check-provenance.sh [--pre-freeze] [--only tags|gaps|sources|preferences]
#
# Exit 0 = ok (WARN allowed), 1 = violation, 2 = cannot run.

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
exec bash "$HERE/gtt-run-python.sh" "$HERE/gtt_provenance.py" check "$@"
