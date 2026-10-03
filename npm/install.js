#!/usr/bin/env node
// Downloads the native gtt binary that matches this package version from the
// GitHub release, verifies its SHA-256 against the release checksums and
// unpacks it into vendor/. Run on install and, if that was skipped (for
// example with --ignore-scripts), on the first `gtt` call.
"use strict";

const crypto = require("crypto");
const fs = require("fs");
const os = require("os");
const path = require("path");
const { execFileSync } = require("child_process");

const REPO = "GTT-Community/gtt-cli";
const pkg = require("./package.json");

const PLATFORMS = { linux: "linux", darwin: "darwin", win32: "windows" };
const ARCHS = { x64: "amd64", arm64: "arm64" };

function binaryPath() {
  return path.join(__dirname, "vendor", process.platform === "win32" ? "gtt.exe" : "gtt");
}

function asset() {
  const goos = PLATFORMS[process.platform];
  const goarch = ARCHS[process.arch];
  if (!goos || !goarch) {
    throw new Error(`gtt does not support ${process.platform}/${process.arch}`);
  }
  return `gtt_${goos}_${goarch}.${goos === "windows" ? "zip" : "tar.gz"}`;
}

async function download(url) {
  const res = await fetch(url, { redirect: "follow" });
  if (!res.ok) {
    throw new Error(`download failed (${res.status}): ${url}`);
  }
  return Buffer.from(await res.arrayBuffer());
}

async function install() {
  const target = binaryPath();
  if (fs.existsSync(target)) {
    return target;
  }
  const name = asset();
  const base = process.env.GTT_DOWNLOAD_BASE || `https://github.com/${REPO}/releases/download/v${pkg.version}`;
  const [archive, sums] = await Promise.all([download(`${base}/${name}`), download(`${base}/checksums.txt`)]);

  const line = sums.toString("utf8").split(/\r?\n/).find((l) => l.trim().endsWith(` ${name}`));
  if (!line) {
    throw new Error(`${name} is not listed in checksums.txt`);
  }
  const expected = line.trim().split(/\s+/)[0].toLowerCase();
  const actual = crypto.createHash("sha256").update(archive).digest("hex");
  if (expected !== actual) {
    throw new Error(`checksum mismatch for ${name}: expected ${expected}, got ${actual}`);
  }

  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "gtt-npm-"));
  try {
    const file = path.join(tmp, name);
    fs.writeFileSync(file, archive);
    // tar ships with Linux, macOS and Windows 10+ (bsdtar reads zip as well).
    execFileSync("tar", ["-xf", file, "-C", tmp], { stdio: "ignore" });
    const unpacked = path.join(tmp, path.basename(target));
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.copyFileSync(unpacked, target);
    fs.chmodSync(target, 0o755);
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
  return target;
}

module.exports = { install, binaryPath };

if (require.main === module) {
  install().then(
    (p) => console.log(`gtt ${pkg.version} installed: ${p}`),
    (err) => {
      // A failed postinstall must not break `npm install`; the first `gtt`
      // call retries and reports the error.
      console.warn(`gtt: binary not installed yet (${err.message}); it will be fetched on first use.`);
    },
  );
}
