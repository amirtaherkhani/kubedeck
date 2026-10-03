import assert from "node:assert/strict";
import { mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { unregisterProjectsUnder } from "../src/registry.mjs";

test("registry cleanup removes stale nested mappings but preserves external projects", () => {
  const base = mkdtempSync(join(tmpdir(), "local-dev-env-registry-"));
  const registryPath = join(base, "projects.json");
  writeFileSync(registryPath, JSON.stringify({ projects: [
    { root: "/projects/home-lab", projectId: "home", environment: "local" },
    { root: "/projects/home-lab/apps/deleted", projectId: "home", environment: "local" },
    { root: "/projects/finance", projectId: "finance", environment: "development" }
  ] }));

  assert.equal(unregisterProjectsUnder("/projects/home-lab", registryPath), 2);
  const registry = JSON.parse(readFileSync(registryPath, "utf8"));
  assert.deepEqual(registry.projects.map(({ root }) => root), ["/projects/finance"]);
});
