#!/usr/bin/env python3
"""GTT - L0/L1 context protection (PreToolUse hook).

permissions.deny already blocks the Write/Edit tools for the unconditional
machinery paths. This hook covers what static config can't express: shell
commands (sed -i, tee, redirection, mv, rm, ...) reaching those same
machinery paths and the three protected governance files - AGENTS.md,
change-request.md (in gtt-domain/), and
SOURCE-BRIEF.* (stays at the project root, the one human-facing exception) -
without going through a file tool, and the two-regime condition on
gtt-domain/context/ and gtt-domain/adr/ - writable pre-freeze, denied once
gtt-domain/.frozen exists. A static permissions.deny entry can't test for a
file's existence, so those two paths are deliberately absent from
settings.json and live here instead.

ROOT_FILES matches by filename substring, not by directory prefix, so it
keeps protecting AGENTS.md, change-request.md, and SOURCE-BRIEF.*
correctly regardless of which directory each lives in. This does mean a
nested AGENTS.md some ADEs use to approximate path-scoped rules in an
installed host project (e.g. Codex's src/AGENTS.md convention, documented in
docs/docs.md) would also match here if this hook is copied into that
project - accepted as the same coarse-but-safe tradeoff already made for
change-request.md and SOURCE-BRIEF.*, and irrelevant to this repository,
which has exactly one AGENTS.md, at the root.

Machinery (.claude/settings.json, .claude/settings.local.json,
.claude/hooks/) has no regime exception and no Write/Edit fallback here
either: permissions.deny already blocks the tool path, so this hook's job
for machinery is exclusively the shell path - an agent must never be able
to disarm its own protection via sed/rm/tee/redirection just because that
happens to route through Bash instead of Edit. settings.local.json is
covered for the same reason as settings.json itself: Claude Code merges it
into the effective permissions/hooks configuration, so it is exactly as
capable of disabling this mechanism and must not be reachable by any path
settings.json itself is denied on.

Root-anchored paths: the governed directories moved from the
project root to gtt-domain/context/ and gtt-domain/adr/, and
gtt-domain/proposals/ is the one directory agents may write. A bare
substring test for "context/" would also fire on a host project's own
src/context/ (a very common directory name) and deny legitimate edits, so
paths are RESOLVED instead: every path-like token (a tool's file_path, or a
token of a shell command) is normalised, resolved against the project root,
and only a result that lands at <root>/gtt-domain/context/,
<root>/gtt-domain/adr/ or <root>/gtt-domain/proposals/ counts. This is at least as strict as the old substring
test for everything that resolves into those directories - absolute Windows
paths, Git-Bash /c/... paths, ./ prefixes, backslashes and ../ traversal
all resolve to the same place - and it stops matching unrelated directories
that merely share a name.

Path normalisation (fixed 2026-09-28): Write/Edit/NotebookEdit deliver an
ABSOLUTE WINDOWS PATH (backslashes) as file_path on this platform, and a
PowerShell command may equally use backslash paths. Every pattern below is
written with forward slashes, so without normalising first, an absolute
Windows path to a governed directory silently never matched and the
freeze-regime block never fired for the Write/Edit tools - demonstrated
live: a Write to a Windows-style absolute path under the governed context
after freeze was NOT denied, while the equivalent `rm` via Bash
(forward-slash path) WAS. normalize() is applied to every caller - file-tool
paths and shell command strings alike - without duplicating the fix.

PowerShell coverage (added 2026-09-28): this environment's primary shell is
PowerShell, not Bash, and the hook previously only inspected the Bash tool's
command string - a PowerShell Set-Content/Remove-Item/etc. against a
protected path passed through unchecked. MUTATING_SHELL now also matches
common PowerShell mutating cmdlets and their built-in aliases (case
insensitive - PowerShell itself is), and both Bash and PowerShell tool
calls are inspected the same way.

Exit 2 plus permissionDecision:deny blocks the call deterministically.
Any unexpected input exits 0 so a broken hook never blocks a session.
"""

import json
import os
import posixpath
import re
import sys

FROZEN_MARKER = "gtt-domain/.frozen"

# Directories governed by the two-regime condition, relative to the project
# root. Resolved (see docstring), never substring-matched.
REGIME_DIRS = ("gtt-domain/context", "gtt-domain/adr")
PROPOSALS_DIR = "gtt-domain/proposals"

ROOT_FILES = re.compile(r"AGENTS\.md|change-request\.md|SOURCE-BRIEF\.", re.IGNORECASE)
MACHINERY = re.compile(r"\.claude/settings(\.local)?\.json|\.claude/hooks/", re.IGNORECASE)
MUTATING_SHELL = re.compile(
    r"\b(sed\s+-i|tee|mv|cp|rm|truncate|dd|install"
    # PowerShell cmdlets and their built-in aliases for the same operations.
    r"|del|erase|rd|rmdir|ri|move|copy|ren"
    r"|set-content|add-content|out-file|new-item|remove-item"
    r"|move-item|copy-item|rename-item|clear-content)\b"
    r"|>>?\s*\S*(gtt/|gtt-domain/|change-request\.md|SOURCE-BRIEF\."
    r"|\.claude/settings(\.local)?\.json|\.claude/hooks/)",
    re.IGNORECASE,
)
# An agent may always write drafts into gtt-domain/proposals/ (see is_protected),
# but it must never be the one to execute the promotion script it staged
# there - that is the Human Promotion Boundary (AGENTS.md). Kept as its own
# pattern, checked ahead of the proposals/ exemption in is_protected(),
# so that exemption keeps allowing every other mutating command inside
# proposals/ (mv/rm/cp cleanup of stale drafts). Matches the script name
# under any directory spelling (proposals/, ./proposals/, an absolute path).
PROMOTION_SCRIPT_EXEC = re.compile(
    r"\b(?:bash|sh|\.)\s+\S*proposals/apply-[A-Za-z0-9._-]+\.sh\b",
    re.IGNORECASE,
)

REASON = (
    "GTT governance: AGENTS.md, gtt-domain/context/, gtt-domain/adr/, "
    "gtt-domain/change-request.md, SOURCE-BRIEF.*, and the .claude/ governance "
    "machinery itself are owned by the Solution Designer. Write your draft to "
    "gtt-domain/proposals/ instead - that directory is yours. Use the "
    "gtt-propose-change or gtt-bootstrap skill."
)

# Characters that separate path-like tokens inside a shell command string.
TOKEN_SPLIT = re.compile(r"[\s\"'=(<>|;&,`]+")


def is_frozen() -> bool:
    return os.path.exists(FROZEN_MARKER)


def normalize(path: str) -> str:
    """Every pattern in this module is written with forward slashes; every
    caller's raw input (an absolute Windows path, or a shell command that
    may contain one) is normalised here, before matching."""
    return path.replace("\\", "/")


def project_root() -> str:
    root = os.environ.get("CLAUDE_PROJECT_DIR") or os.getcwd()
    return normalize(os.path.abspath(root)).rstrip("/")


def relative_to_root(token: str, root: str):
    """The token as a path relative to the project root, or None when it
    resolves outside the project (or is not path-like)."""
    t = normalize(token).strip()
    if not t or t.startswith("-"):
        return None
    # Git-Bash / MSYS drive spelling: /c/foo -> C:/foo. Windows only: on a POSIX
    # system /x/proj is an ordinary path under a one-letter top-level directory, and
    # rewriting it to X:/proj would resolve it OUTSIDE the project root (unprotected).
    if os.name == "nt":
        m = re.match(r"^/([A-Za-z])/(.*)$", t)
        if m:
            t = f"{m.group(1).upper()}:/{m.group(2)}"
    absolute = t if (t.startswith("/") or re.match(r"^[A-Za-z]:/", t)) else f"{root}/{t}"
    resolved = posixpath.normpath(absolute)
    # Windows file systems are case-insensitive; compare accordingly.
    if resolved.lower() == root.lower():
        return ""
    prefix = root.lower() + "/"
    if resolved.lower().startswith(prefix):
        return resolved[len(prefix):]
    return None


def path_tokens(text: str):
    return [t for t in TOKEN_SPLIT.split(normalize(text)) if t]


def under(rel: str, directory: str) -> bool:
    r = rel.lower()
    return r == directory or r.startswith(directory + "/")


def classify(target: str):
    """(in_regime_dir, in_proposals) over every path-like token of target."""
    root = project_root()
    regime = proposals = False
    for token in path_tokens(target):
        rel = relative_to_root(token, root)
        if rel is None:
            continue
        if any(under(rel, d) for d in REGIME_DIRS):
            regime = True
        if under(rel, PROPOSALS_DIR):
            proposals = True
    return regime, proposals


def is_protected(target: str) -> bool:
    target = normalize(target)
    if MACHINERY.search(target):
        return True
    if PROMOTION_SCRIPT_EXEC.search(target):
        return True
    regime, proposals = classify(target)
    # gtt-domain/proposals/ is the one directory an agent may always write to -
    # SOURCE-BRIEF.* and change-request.md are only protected outside it
    # (e.g. gtt-bootstrap staging gtt-domain/proposals/bootstrap/SOURCE-BRIEF.md).
    if proposals:
        return False
    if regime:
        return is_frozen()
    return bool(ROOT_FILES.search(target))


def main() -> int:
    try:
        event = json.load(sys.stdin)
    except Exception:
        return 0

    tool = event.get("tool_name", "")
    tool_input = event.get("tool_input") or {}
    target = ""

    if tool in ("Write", "Edit", "NotebookEdit"):
        target = tool_input.get("file_path") or tool_input.get("notebook_path") or ""
    elif tool in ("Bash", "PowerShell"):
        command = tool_input.get("command", "")
        # Only flag commands that could mutate, or that execute a staged
        # promotion script. Reads stay allowed.
        if MUTATING_SHELL.search(command) or PROMOTION_SCRIPT_EXEC.search(command):
            target = command

    if target and is_protected(target):
        print(
            json.dumps(
                {
                    "hookSpecificOutput": {
                        "hookEventName": "PreToolUse",
                        "permissionDecision": "deny",
                        "permissionDecisionReason": REASON,
                    }
                }
            )
        )
        return 2

    return 0


if __name__ == "__main__":
    sys.exit(main())
