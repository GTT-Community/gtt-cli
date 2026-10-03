#!/usr/bin/env python3
"""GTT - scaffold manifest reader (engine).

Reads .gtt/scaffold/manifest.yaml, the single declarative definition of the
scaffold. The manifest is DATA ONLY: a few top-level sections whose entries are
single-line flow mappings, e.g.

    - { id: example, name: Example ADE, path: .example/, owned: [.example/], role: prose }

This is deliberately NOT a YAML parser. It understands exactly that subset
(scalars, `[a, b]` lists, booleans, prose after `role:`) and nothing else, so
the Core needs no third-party package. If the manifest ever stops fitting that
subset, this reader fails loudly instead of guessing.

The module holds no GTT decision: it turns lines into dictionaries. What an
entry means belongs to the engine that consumes it (gtt_ade.py, gtt_template.py).
"""

import os
import re

MANIFEST = ".gtt/scaffold/manifest.yaml"

SECTION_RE = re.compile(r"^([a-z_]+):\s*(?:#.*)?$")
ENTRY_RE = re.compile(r"^\s+-\s*\{(.*)\}\s*$")
SCALAR_RE = re.compile(r"^\s+([a-z_]+):\s*(.*?)\s*$")
KEY_RE = re.compile(r"(?:^|,\s*)([a-z_]+):\s+")


class ManifestError(Exception):
    pass


def _scalar(text):
    text = text.strip()
    if len(text) >= 2 and text[0] == text[-1] and text[0] in "\"'":
        return text[1:-1]
    if text == "true":
        return True
    if text == "false":
        return False
    if re.fullmatch(r"-?\d+", text):
        return int(text)
    return text


def _value(text):
    text = text.strip()
    if text.startswith("[") and text.endswith("]"):
        inner = text[1:-1].strip()
        return [_scalar(x) for x in inner.split(",") if x.strip()] if inner else []
    return _scalar(text)


def parse_entry(body):
    """`k: v, k: [a, b], role: prose, with commas` -> dict. Keys are lowercase
    identifiers followed by a colon and a space; anything between two keys is
    the first key's value."""
    body = body.strip()
    hits = list(KEY_RE.finditer(body))
    if not hits or hits[0].start() != 0:
        raise ManifestError(f"entry does not start with `key: value`: {{{body}}}")
    out = {}
    for i, hit in enumerate(hits):
        end = hits[i + 1].start() if i + 1 < len(hits) else len(body)
        key = hit.group(1)
        if key in out:
            raise ManifestError(f"duplicate key `{key}` in entry: {{{body}}}")
        out[key] = _value(body[hit.end():end])
    return out


def load(path=MANIFEST):
    """{'entries': {section: [dict, ...]}, 'scalars': {section: {key: value}}}"""
    if not os.path.isfile(path):
        raise ManifestError(f"{path} not found")
    entries, scalars, section = {}, {}, None
    with open(path, encoding="utf-8") as handle:
        for number, raw in enumerate(handle, 1):
            line = raw.rstrip("\n").rstrip("\r")
            if not line.strip() or line.lstrip().startswith("#"):
                continue
            match = SECTION_RE.match(line)
            if match:
                section = match.group(1)
                continue
            if section is None:
                continue
            match = ENTRY_RE.match(line)
            if match:
                try:
                    entries.setdefault(section, []).append(parse_entry(match.group(1)))
                except ManifestError as exc:
                    raise ManifestError(f"{path}:{number}: {exc}")
                continue
            match = SCALAR_RE.match(line)
            if match:
                value = re.sub(r"\s+#.*$", "", match.group(2))
                scalars.setdefault(section, {})[match.group(1)] = _scalar(value)
    return {"entries": entries, "scalars": scalars}


def layout_version(manifest):
    version = manifest["scalars"].get("scaffold", {}).get("version")
    if not isinstance(version, int):
        raise ManifestError("scaffold.version is missing or not an integer")
    return version


def overlays(manifest):
    """id -> normalised overlay entry (the ADE integration registry)."""
    out = {}
    for entry in manifest["entries"].get("overlays", []):
        ade = entry.get("id")
        if not ade:
            raise ManifestError("overlay entry without id")
        if ade in out:
            raise ManifestError(f"duplicate overlay id `{ade}`")
        for key in ("path", "entry"):
            if not isinstance(entry.get(key), str):
                raise ManifestError(f"overlay `{ade}`: `{key}` is required")
        for key in ("owned", "detect"):
            if not isinstance(entry.get(key, []), list):
                raise ManifestError(f"overlay `{ade}`: `{key}` must be a list")
        out[ade] = {
            "id": ade,
            "name": str(entry.get("name", ade)),
            "path": entry["path"],
            "entry": entry["entry"],
            "owned": list(entry.get("owned", [])),
            "detect": list(entry.get("detect", [])),
            "enforcement": str(entry.get("enforcement", "ci-gate")),
            "scaffold": entry.get("scaffold"),
            "version": entry.get("version", 1),
            "role": str(entry.get("role", "")),
        }
    return out


def templates(manifest):
    """id -> template entry (Bootstrap-owned templates a CLI may request)."""
    out = {}
    for entry in manifest["entries"].get("templates", []):
        tid = entry.get("id")
        if not tid:
            raise ManifestError("template entry without id")
        if tid in out:
            raise ManifestError(f"duplicate template id `{tid}`")
        for key in ("path", "materialize_to"):
            if not isinstance(entry.get(key), str):
                raise ManifestError(f"template `{tid}`: `{key}` is required")
        out[tid] = dict(entry)
    return out
