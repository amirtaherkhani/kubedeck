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
