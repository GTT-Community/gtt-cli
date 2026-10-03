#!/usr/bin/env node
// npm entry point: runs the native gtt binary, fetching it first if the
// install step was skipped. Arguments, streams and the exit code pass through.
"use strict";

const { spawnSync } = require("child_process");
const { install } = require("../install.js");

install().then(
  (binary) => {
    const res = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
    if (res.error) {
      console.error(`gtt: ${res.error.message}`);
      process.exit(7);
    }
    process.exit(res.status === null ? 1 : res.status);
  },
  (err) => {
    console.error(`gtt: could not install the gtt binary: ${err.message}`);
    process.exit(7);
  },
);
