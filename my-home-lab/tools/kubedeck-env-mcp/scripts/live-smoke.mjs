import assert from "node:assert/strict";
import { randomUUID } from "node:crypto";
import { spawn } from "node:child_process";
import { resolve } from "node:path";
import { createInterface } from "node:readline";
import { pathToFileURL } from "node:url";

const root = resolve(process.argv[2] ?? process.cwd());
const serverCwd = resolve(process.argv[3] ?? root);
const secretPath = process.argv[4];
const scope = secretPath ? { path: secretPath } : {};
const child = spawn("kubedeck-env-mcp", ["serve"], {
  cwd: serverCwd,
  env: { ...process.env, NODE_USE_SYSTEM_CA: "1" },
  stdio: ["pipe", "pipe", "pipe"]
});
const output = createInterface({ input: child.stdout, crlfDelay: Infinity });
const pending = new Map();
let nextId = 1;

output.on("line", (line) => {
  const message = JSON.parse(line);
  if (message.method === "roots/list") {
    child.stdin.write(`${JSON.stringify({
      jsonrpc: "2.0",
      id: message.id,
      result: { roots: [{ uri: pathToFileURL(root).href, name: "smoke-workspace" }] }
    })}\n`);
    return;
  }
  const handler = pending.get(message.id);
  if (handler) {
    pending.delete(message.id);
    handler(message);
  }
});

function request(method, params = {}) {
  const id = nextId++;
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id, method, params })}\n`);
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      pending.delete(id);
      reject(new Error(`${method} timed out`));
    }, 15_000);
    pending.set(id, (message) => {
      clearTimeout(timeout);
      if (message.error) reject(new Error(message.error.message));
      else resolve(message.result);
    });
  });
}

function data(result) {
  if (result.isError) throw new Error(result.content?.[0]?.text ?? "MCP tool failed");
  return result.structuredContent;
}

const key = `LOCAL_DEV_ENV_MCP_SMOKE_${Date.now()}`;
const value = randomUUID();
let created = false;

try {
  const initialized = await request("initialize", {
    protocolVersion: "2025-11-25",
    capabilities: { roots: { listChanged: true } },
    clientInfo: { name: "kubedeck-env-live-smoke", version: "1.0.0" }
  });
  assert.equal(initialized.serverInfo.name, "kubedeck-env-mcp");
  child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", method: "notifications/initialized" })}\n`);

  const tools = await request("tools/list");
  assert.deepEqual(tools.tools.map(({ name }) => name), [
    "env_context", "env_list", "env_get", "env_set", "env_set_many", "env_delete"
  ]);

  const context = data(await request("tools/call", { name: "env_context", arguments: {} }));
  assert.ok(["local", "development", "dev"].includes(context.environment));
  assert.equal(context.root, root);
  await request("tools/call", { name: "env_list", arguments: scope }).then(data);
  const setResult = data(await request("tools/call", { name: "env_set", arguments: { name: key, value, ...scope } }));
  assert.equal(setResult.action, "created");
  created = true;
  const getResult = data(await request("tools/call", { name: "env_get", arguments: { name: key, ...scope } }));
  assert.equal(getResult.value, value);
  data(await request("tools/call", { name: "env_delete", arguments: { name: key, ...scope } }));
  created = false;
  const names = data(await request("tools/call", { name: "env_list", arguments: scope })).names;
  assert.equal(names.includes(key), false);
  console.log(JSON.stringify({ ok: true, server: initialized.serverInfo.name, tools: tools.tools.length, createReadDelete: "passed" }));
} finally {
  if (created) {
    await request("tools/call", { name: "env_delete", arguments: { name: key, ...scope } }).catch(() => {});
  }
  child.stdin.end();
}
