#!/usr/bin/env python3
"""GTT - template service (engine).

The GTT Bootstrap owns certain templates (today: the Initial Design
Questionnaire). A CLI must never carry a copy of one, nor its methodology: it
asks this service what exists, whether it fits this scaffold, and where a
working copy goes - then requests that copy. The declarations live in the
`templates:` section of .gtt/scaffold/manifest.yaml; this engine only reads them.

  list                          the templates the Bootstrap declares
  show <id>                     one declaration: path, exists, compatible, target
  materialize <id> [--apply]    copy the template to its declared working
                                location (dry run by default; never overwrites)

A materialized copy is SOURCE MATERIAL for the governed design process. It is
neither governed context nor a decision, and it grants nothing to whoever
fills it in.

Exit 0 = ok, 1 = conflict / violation, 2 = cannot run.
"""

import argparse
import json
import os
import shutil
import subprocess
import sys

import gtt_manifest as gm

SCHEMA = 1


def die(message, code=2):
    print(f"gtt-template: {message}", file=sys.stderr)
    sys.exit(code)


def load(root):
    try:
        manifest = gm.load(os.path.join(root, gm.MANIFEST))
        return manifest, gm.templates(manifest)
    except gm.ManifestError as exc:
        die(str(exc))


def describe(root, manifest, entry):
    layout = gm.layout_version(manifest)
    return dict(entry, exists=os.path.isfile(os.path.join(root, entry["path"])),
                compatible=entry.get("scaffold") in (None, layout), layout=layout)


def pick(templates, tid):
    if tid not in templates:
        die(f"unknown template `{tid}` (declared: {', '.join(sorted(templates)) or 'none'}).")
    return templates[tid]


def cmd_list(args):
    root = args.frm or "."
    manifest, templates = load(root)
    rows = [describe(root, manifest, e) for _, e in sorted(templates.items())]
    if args.json:
        print(json.dumps({"schema": SCHEMA, "templates": rows}, indent=2, sort_keys=True))
        return 0
    print("GTT Bootstrap templates (owned by the Bootstrap; a CLI requests them, never copies them)")
    for row in rows:
        flag = "ok" if row["exists"] and row["compatible"] else "UNAVAILABLE"
        print(f"  {row['id']:32} {flag:12} {row['path']}")
    return 0


def cmd_show(args):
    root = args.frm or "."
    manifest, templates = load(root)
    row = describe(root, manifest, pick(templates, args.id))
    if args.json:
        print(json.dumps(dict(row, schema=SCHEMA), indent=2, sort_keys=True))
        return 0
    for key in ("id", "path", "exists", "compatible", "layout", "use", "when",
                "materialize_to", "becomes", "role"):
        if key in row:
            print(f"{key}: {row[key]}")
    return 0 if row["exists"] and row["compatible"] else 1


def cmd_materialize(args):
    root = args.frm or "."
    manifest, templates = load(root)
    row = describe(root, manifest, pick(templates, args.id))
    if not row["exists"]:
        die(f"template file missing: {os.path.join(root, row['path'])}", 1)
    if not row["compatible"]:
        die(f"template declares scaffold layout {row.get('scaffold')}, this scaffold is {row['layout']}.", 1)
    src, dst = os.path.join(root, row["path"]), row["materialize_to"]
    print(f"materialize {row['id']}: {row['path']} -> {dst}")
    if os.path.exists(dst):
        print(f"CONFLICT  {dst} already exists - never overwritten (a filled questionnaire is work in progress).",
              file=sys.stderr)
        return 1
    if not args.apply:
        print("(dry run - nothing written; re-run with --apply)")
        return 0
    os.makedirs(os.path.dirname(dst) or ".", exist_ok=True)
    shutil.copyfile(src, dst)
    print(f"created {dst}")
    print("It is source material: not governed context, not a decision. The Primary ADE reads its "
          "Operating Contract first; unknowns, conflicts and proposals stay visible until the human decides.")
    index = ".gtt/scripts/gtt-index.sh"
    if os.path.isfile(index):
        done = subprocess.run(["bash", index], capture_output=True, text=True, encoding="utf-8",
                              errors="replace")
        if done.returncode != 0:
            # The working copy is the result of this operation; the index is derived state. In a host project
            # whose index still lists artifacts of overlays that were not installed, registration is refused
            # until `gtt-index.sh` / `gtt-reconcile.sh` is run - say so, but do not fail a materialisation that worked.
            print(done.stdout + done.stderr, file=sys.stderr)
            print(f"WARNING: {dst} was created but is NOT registered in the artifact index; run {index} "
                  "(after gtt-reconcile.sh if it reports missing paths).", file=sys.stderr)
        else:
            print("registered in the artifact index (gtt-index.sh).")
    return 0


def build_parser():
    parser = argparse.ArgumentParser(prog="gtt-template.sh", description=__doc__.split("\n")[0])
    sub = parser.add_subparsers(dest="command", required=True)
    for name, func in (("list", cmd_list), ("show", cmd_show), ("materialize", cmd_materialize)):
        p = sub.add_parser(name)
        p.set_defaults(func=func)
        p.add_argument("--from", dest="frm", metavar="CATALOG",
                       help="root of the GTT Bootstrap catalog (default: this project)")
        if name != "list":
            p.add_argument("id")
        if name != "materialize":
            p.add_argument("--json", action="store_true")
        else:
            p.add_argument("--apply", action="store_true", help="write (default is a dry run)")
    return parser


if __name__ == "__main__":
    parsed = build_parser().parse_args(sys.argv[1:])
    if not hasattr(parsed, "json"):
        parsed.json = False
    sys.exit(parsed.func(parsed))
