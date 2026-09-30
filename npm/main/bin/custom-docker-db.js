#!/usr/bin/env node
// Launcher: finds the platform package that ships the Go binary and runs it.
"use strict";

const { spawnSync } = require("node:child_process");

const PLATFORMS = {
  "linux-x64": "@pvfm/cdd-linux-x64",
  "linux-arm64": "@pvfm/cdd-linux-arm64",
};

const key = `${process.platform}-${process.arch}`;
const pkg = PLATFORMS[key];

if (!pkg) {
  console.error(`custom-docker-db: plataforma não suportada: ${key} (só Linux x64 e arm64)`);
  process.exit(1);
}

let binary;
try {
  binary = require.resolve(`${pkg}/bin/custom-docker-db`);
} catch {
  console.error(
    `custom-docker-db: o pacote ${pkg} não está instalado.\n` +
      "Reinstale sem --no-optional / --omit=optional."
  );
  process.exit(1);
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
  console.error(`custom-docker-db: ${result.error.message}`);
  process.exit(1);
}
if (result.signal) {
  process.kill(process.pid, result.signal);
}
process.exit(result.status === null ? 1 : result.status);
