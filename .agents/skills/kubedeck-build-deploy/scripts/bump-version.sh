#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

usage() {
  printf 'Usage: %s <major|minor|patch|x.y.z>\n' "$0"
}

increment="${1:-}"
[[ -n "$increment" ]] || { usage >&2; exit 2; }

current_version="$(node -p "require('./package.json').version")"
next_version="$({
  CURRENT_VERSION="$current_version" node - "$increment" <<'NODE'
const current = process.env.CURRENT_VERSION;
const requested = process.argv[2];
const match = /^(\d+)\.(\d+)\.(\d+)$/.exec(current);
if (!match) throw new Error(`Unsupported current version: ${current}`);

let next;
if (/^\d+\.\d+\.\d+$/.test(requested)) {
  next = requested;
} else if (["major", "minor", "patch"].includes(requested)) {
  const parts = match.slice(1).map(Number);
  const index = { major: 0, minor: 1, patch: 2 }[requested];
  parts[index] += 1;
  for (let i = index + 1; i < parts.length; i += 1) parts[i] = 0;
  next = parts.join(".");
} else {
  throw new Error(`Expected major, minor, patch, or x.y.z; received: ${requested}`);
}

console.log(next);
NODE
})"

VERSION="$next_version" node <<'NODE'
const fs = require("node:fs");
const version = process.env.VERSION;

for (const path of ["package.json", "package-lock.json"]) {
  const source = JSON.parse(fs.readFileSync(path, "utf8"));
  source.version = version;
  if (path === "package-lock.json" && source.packages?.[""]) {
    source.packages[""].version = version;
  }
  fs.writeFileSync(path, `${JSON.stringify(source, null, 2)}\n`);
}

for (const path of ["charts/kubedeck/Chart.yaml", "charts/kubedeck-agent/Chart.yaml"]) {
  let source = fs.readFileSync(path, "utf8");
  source = source.replace(/^version: .*$/m, `version: ${version}`);
  source = source.replace(/^appVersion: .*$/m, `appVersion: "${version}"`);
  fs.writeFileSync(path, source);
}

console.log(`Updated KubeDeck release version to ${version}`);
NODE
