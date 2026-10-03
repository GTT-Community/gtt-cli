#!/usr/bin/env python3
"""Build the PyPI wheels of gtt-cli from the GoReleaser binaries.

One wheel per platform, each carrying the native `gtt` binary as a script, so
`uv tool install gtt-cli`, `uvx gtt-cli` and `pipx install gtt-cli` put `gtt`
on the PATH with no Python code in between (the same layout ruff and uv use).

Usage: build_wheels.py --dist dist --version 1.2.3 --out wheels
"""
import argparse
import base64
import hashlib
import json
import os
import sys
import zipfile

NAME = "gtt-cli"
DIST_NAME = "gtt_cli"

# Go target -> wheel platform tags. The binary is static, so one Linux wheel
# serves both glibc (manylinux) and musl (musllinux) systems.
PLATFORMS = {
    ("linux", "amd64"): "manylinux_2_17_x86_64.manylinux2014_x86_64.musllinux_1_1_x86_64",
    ("linux", "arm64"): "manylinux_2_17_aarch64.manylinux2014_aarch64.musllinux_1_1_aarch64",
    ("darwin", "amd64"): "macosx_10_12_x86_64",
    ("darwin", "arm64"): "macosx_11_0_arm64",
    ("windows", "amd64"): "win_amd64",
    ("windows", "arm64"): "win_arm64",
}

SUMMARY = "GTT CLI - the operational doorway into GTT Bootstrap"


def record_line(path, data):
    digest = base64.urlsafe_b64encode(hashlib.sha256(data).digest()).rstrip(b"=").decode()
    return f"{path},sha256={digest},{len(data)}"


def metadata(version, readme):
    return "\n".join([
        "Metadata-Version: 2.1",
        f"Name: {NAME}",
        f"Version: {version}",
        f"Summary: {SUMMARY}",
        "Home-page: https://github.com/GTT-Community/gtt-cli",
        "License: Apache-2.0",
        "Project-URL: Source, https://github.com/GTT-Community/gtt-cli",
        "Project-URL: Releases, https://github.com/GTT-Community/gtt-cli/releases",
        "Classifier: License :: OSI Approved :: Apache Software License",
        "Classifier: Environment :: Console",
        "Classifier: Operating System :: POSIX :: Linux",
        "Classifier: Operating System :: MacOS",
        "Classifier: Operating System :: Microsoft :: Windows",
        "Requires-Python: >=3.8",
        "Description-Content-Type: text/markdown",
        "",
        readme,
    ])


def build(binary, goos, goarch, version, readme, out):
    tags = PLATFORMS[(goos, goarch)]
    wheel = os.path.join(out, f"{DIST_NAME}-{version}-py3-none-{tags}.whl")
    exe = "gtt.exe" if goos == "windows" else "gtt"
    data_dir = f"{DIST_NAME}-{version}.data"
    info_dir = f"{DIST_NAME}-{version}.dist-info"
    with open(binary, "rb") as f:
        payload = f.read()
    files = [
        (f"{DIST_NAME}/__init__.py", f'"""gtt-cli {version}: the native gtt binary is installed as a script."""\n__version__ = "{version}"\n'.encode(), 0o644),
        (f"{data_dir}/scripts/{exe}", payload, 0o755),
        (f"{info_dir}/METADATA", metadata(version, readme).encode(), 0o644),
        (f"{info_dir}/WHEEL", "\n".join(
            ["Wheel-Version: 1.0", "Generator: gtt-cli build_wheels.py", "Root-Is-Purelib: false"]
            + [f"Tag: py3-none-{t}" for t in tags.split(".")]).encode() + b"\n", 0o644),
    ]
    record = [record_line(p, d) for p, d, _ in files] + [f"{info_dir}/RECORD,,"]
    files.append((f"{info_dir}/RECORD", ("\n".join(record) + "\n").encode(), 0o644))
    with zipfile.ZipFile(wheel, "w", zipfile.ZIP_DEFLATED) as z:
        for path, data, mode in files:
            info = zipfile.ZipInfo(path, date_time=(2020, 1, 1, 0, 0, 0))
            info.external_attr = (0o100000 | mode) << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            z.writestr(info, data)
    return wheel


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--dist", required=True, help="GoReleaser dist directory (holds artifacts.json)")
    ap.add_argument("--version", required=True, help="release version, e.g. 1.2.3")
    ap.add_argument("--out", required=True, help="directory for the wheels")
    ap.add_argument("--readme", default=os.path.join(os.path.dirname(__file__), "README.md"))
    args = ap.parse_args()

    version = args.version.lstrip("v")
    with open(os.path.join(args.dist, "artifacts.json")) as f:
        artifacts = json.load(f)
    with open(args.readme) as f:
        readme = f.read()
    os.makedirs(args.out, exist_ok=True)
    built = []
    for a in artifacts:
        if a.get("type") != "Binary":
            continue
        key = (a.get("goos"), a.get("goarch"))
        if key not in PLATFORMS:
            continue
        built.append(build(a["path"], key[0], key[1], version, readme, args.out))
    missing = set(PLATFORMS) - {(a.get("goos"), a.get("goarch")) for a in artifacts if a.get("type") == "Binary"}
    if missing:
        print(f"build_wheels: no binary for {sorted(missing)}", file=sys.stderr)
        return 1
    for w in sorted(built):
        print(w)
    return 0


if __name__ == "__main__":
    sys.exit(main())
