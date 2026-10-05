import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const root = new URL("..", import.meta.url);
const pkg = JSON.parse(readFileSync(new URL("package.json", root), "utf8"));
const lock = JSON.parse(readFileSync(new URL("package-lock.json", root), "utf8"));

test("new KubeDeck command retains the existing CLI alias and registry", () => {
  assert.equal(pkg.name, "@kubedeck/kubedeck-env-mcp");
  assert.equal(pkg.bin["local-dev-env-mcp"], "src/cli.mjs");
  assert.equal(pkg.bin["kubedeck-env-mcp"], "src/cli.mjs");
  assert.deepEqual(lock.packages[""].bin, pkg.bin);
  const context = readFileSync(new URL("src/context.mjs", root), "utf8");
  assert.match(context, /\.config", "local-dev-env", "projects\.json"/);
});

test("bootstrap registers one preferred MCP client name", () => {
  const surfaces = readFileSync(new URL("../../scripts/lib/local-dev-env-bootstrap-surfaces.sh", root), "utf8");
  assert.match(surfaces, /\[mcp_servers\.kubedeck-env\]/);
  assert.match(surfaces, /\.mcp\["kubedeck-env"\]/);
  assert.match(surfaces, /del\(\.mcp\["local-dev-env"\]\)/);
});
