import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { dirname, join } from "node:path";
import { createInterface } from "node:readline";
import test from "node:test";
import { fileURLToPath } from "node:url";

const packageRoot = join(dirname(fileURLToPath(import.meta.url)), "..");

test("implements MCP initialization and protocol errors", async () => {
  const child = spawn(process.execPath, [join(packageRoot, "src", "cli.mjs"), "serve"], {
    cwd: packageRoot,
    stdio: ["pipe", "pipe", "pipe"]
  });
  const output = createInterface({ input: child.stdout, crlfDelay: Infinity });
  const messages = [];
  const waiters = [];
  output.on("line", (line) => {
    const message = JSON.parse(line);
    const waiter = waiters.shift();
    if (waiter) waiter(message);
    else messages.push(message);
  });

  function nextMessage() {
    if (messages.length) return Promise.resolve(messages.shift());
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error("MCP response timed out")), 2_000);
      waiters.push((message) => { clearTimeout(timeout); resolve(message); });
    });
  }

  function send(message) {
    child.stdin.write(`${typeof message === "string" ? message : JSON.stringify(message)}\n`);
    return nextMessage();
  }

  try {
    assert.equal((await send("not-json")).error.code, -32700);
    assert.equal((await send({ jsonrpc: "1.0", id: 99, method: "initialize" })).error.code, -32600);
    assert.equal((await send({ jsonrpc: "2.0", id: 1, method: "tools/list", params: {} })).error.code, -32002);
    assert.equal((await send({ jsonrpc: "2.0", id: 98, method: "initialize", params: {} })).error.code, -32600);
    assert.equal((await send({ jsonrpc: "2.0", id: 94, method: "initialize", unexpected: true, params: {
      protocolVersion: "2025-11-25", capabilities: {}, clientInfo: { name: "test", version: "1" }
    } })).error.code, -32600);
    assert.equal((await send({ jsonrpc: "2.0", id: 93, method: "initialize", params: {
      protocolVersion: "2025-11-25", capabilities: { roots: true }, clientInfo: { name: "test", version: "1" }
    } })).error.code, -32600);
    assert.equal((await send({ jsonrpc: "2.0", id: 90, method: "initialize", params: {
      protocolVersion: "2025-11-25", capabilities: { roots: { listChanged: "yes" } }, clientInfo: { name: "test", version: "1" }
    } })).error.code, -32600);
    assert.equal((await send({ jsonrpc: "2.0", id: 96, method: "initialize", params: {
      protocolVersion: "2025-11-25", capabilities: {}, clientInfo: { name: "test", version: "1" }, unexpected: true
    } })).error.code, -32600);
    const initialized = await send({
      jsonrpc: "2.0",
      id: 2,
      method: "initialize",
      params: { protocolVersion: "unsupported-version", capabilities: { roots: { listChanged: true } }, clientInfo: { name: "test", version: "1" } }
    });
    assert.equal(initialized.result.protocolVersion, "2025-11-25");
    child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", method: "notifications/initialized" })}\n`);
    const rootsRequest = await nextMessage();
    assert.equal(rootsRequest.method, "roots/list");
    child.stdin.write(`${JSON.stringify({
      jsonrpc: "2.0",
      id: rootsRequest.id,
      result: { roots: [{ uri: "file:///tmp" }] },
      error: { code: -32603, message: "malformed response" }
    })}\n`);
    assert.deepEqual((await send({ jsonrpc: "2.0", id: 3, method: "ping", params: {} })).result, {});
    child.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id: 77, result: {} })}\n`);
    assert.deepEqual((await send({ jsonrpc: "2.0", id: 88, method: "ping", params: {} })).result, {});
    assert.equal((await send({ jsonrpc: "2.0", id: 89, method: "notifications/initialized", params: {} })).error.code, -32600);
    assert.equal((await send({ jsonrpc: "2.0", id: 92, method: "ping", params: {}, unexpected: true })).error.code, -32600);
    assert.equal((await send({ jsonrpc: "2.0", id: 97, method: "tools/list", params: { unexpected: true } })).error.code, -32600);
    assert.equal((await send({ jsonrpc: "2.0", id: 91, method: "tools/list", params: { cursor: 42 } })).error.code, -32600);
    assert.equal((await send({ jsonrpc: "2.0", id: 95, method: "tools/call", params: {
      name: "env_context", arguments: {}, unexpected: true
    } })).error.code, -32600);
    assert.equal((await send({ jsonrpc: "2.0", id: 4, method: "tools/list", params: {} })).result.tools.length, 6);
    const noRoots = await send({ jsonrpc: "2.0", id: 5, method: "tools/call", params: { name: "env_context", arguments: {} } });
    assert.equal(noRoots.result.isError, true);
    assert.match(noRoots.result.content[0].text, /must provide a registered workspace root/);
  } finally {
    child.stdin.end();
  }
});
