import assert from "node:assert/strict";
import test from "node:test";
import { callTool } from "../src/tools.mjs";

const runtime = {
  context: { root: "/project", environment: "local", path: "/apps/storage", domain: "https://infisical.local.dev", source: "test" },
  store: { list: async (path) => path === "/apps/storage/postgres" ? ["POSTGRES_PASSWORD"] : [] }
};

test("rejects undeclared tool arguments", async () => {
  await assert.rejects(
    callTool({ name: "env_context", arguments: { unexpected: true } }, runtime),
    /no fields/
  );
  await assert.rejects(
    callTool({ name: "env_get", arguments: { name: "VALID", unexpected: true } }, runtime),
    /only: name, path/
  );
});

test("allows an explicit descendant scope and blocks scope escape", async () => {
  const listed = await callTool({ name: "env_list", arguments: { path: "/apps/storage/postgres" } }, runtime);
  assert.deepEqual(listed.structuredContent, { path: "/apps/storage/postgres", names: ["POSTGRES_PASSWORD"] });
  await assert.rejects(
    callTool({ name: "env_list", arguments: { path: "/apps/other" } }, runtime),
    /must stay within \/apps\/storage/
  );
});
