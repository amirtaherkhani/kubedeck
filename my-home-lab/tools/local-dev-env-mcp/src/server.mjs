import { createInterface } from "node:readline";
import { resolveProjectContext } from "./context.mjs";
import { InfisicalStore } from "./infisical.mjs";
import { loadCredentials } from "./keychain.mjs";
import { callTool, toolDefinitions } from "./tools.mjs";

const LATEST_PROTOCOL_VERSION = "2025-11-25";
const SUPPORTED_PROTOCOL_VERSIONS = new Set([LATEST_PROTOCOL_VERSION, "2025-06-18", "2025-03-26", "2024-11-05"]);

function send(message) {
  process.stdout.write(`${JSON.stringify(message)}\n`);
}

function invalidRequest(id, message = "invalid request") {
  send({ jsonrpc: "2.0", id: id ?? null, error: { code: -32600, message } });
}

function isObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function hasOnlyKeys(value, allowed) {
  return isObject(value) && Object.keys(value).every((key) => allowed.has(key));
}

export async function serve() {
  let runtimePromise;
  let rootsPromise = Promise.resolve(null);
  let supportsRoots = false;
  let negotiated = false;
  let initialized = false;
  let rootRequestId = 0;
  const pendingRootRequests = new Map();

  function requestRoots() {
    if (!supportsRoots) return Promise.resolve(null);
    const id = `local-dev-env-roots-${++rootRequestId}`;
    send({ jsonrpc: "2.0", id, method: "roots/list", params: {} });
    return new Promise((resolve) => {
      const timeout = setTimeout(() => {
        pendingRootRequests.delete(id);
        resolve(null);
      }, 2_000);
      pendingRootRequests.set(id, (message) => {
        clearTimeout(timeout);
        const roots = message.result?.roots;
        resolve(Array.isArray(roots) && roots.every((root) => isObject(root) && typeof root.uri === "string")
          ? roots.map(({ uri }) => uri)
          : null);
      });
    });
  }

  function refreshRoots() {
    runtimePromise = null;
    rootsPromise = requestRoots();
  }

  async function runtime() {
    if (!runtimePromise) {
      runtimePromise = rootsPromise.then((roots) => {
        if (!roots || roots.length === 0) {
          throw new Error("the MCP client must provide a registered workspace root");
        }
        const context = resolveProjectContext({ roots });
        const store = new InfisicalStore({ credentials: loadCredentials(), context });
        return { context, store };
      });
    }
    return runtimePromise;
  }

  const input = createInterface({ input: process.stdin, crlfDelay: Infinity });
  input.on("line", async (line) => {
    let message;
    try {
      message = JSON.parse(line);
      const validId = message?.id === undefined || typeof message.id === "string" || typeof message.id === "number";
      if (!isObject(message) || message.jsonrpc !== "2.0" || !validId ||
        (message.method !== undefined && typeof message.method !== "string")) {
        invalidRequest(message?.id);
        return;
      }
      if (message.method === undefined && pendingRootRequests.has(message.id)) {
        const hasResult = Object.hasOwn(message, "result");
        const hasError = Object.hasOwn(message, "error");
        const validError = !hasError || (isObject(message.error) && typeof message.error.code === "number" &&
          Number.isInteger(message.error.code) && typeof message.error.message === "string" &&
          hasOnlyKeys(message.error, new Set(["code", "message", "data"])));
        if (!hasOnlyKeys(message, new Set(["jsonrpc", "id", "result", "error"])) || hasResult === hasError || !validError) {
          pendingRootRequests.get(message.id)({});
          pendingRootRequests.delete(message.id);
          return;
        }
        pendingRootRequests.get(message.id)(message);
        pendingRootRequests.delete(message.id);
        return;
      }
      if (message.method === undefined) return;
      if (message.method !== undefined && !hasOnlyKeys(message, new Set(["jsonrpc", "id", "method", "params"]))) {
        if (message.id !== undefined) invalidRequest(message.id);
        return;
      }
      if (message.method === "notifications/initialized") {
        if (message.id !== undefined) {
          invalidRequest(message.id);
          return;
        }
        if (!negotiated || (message.params !== undefined && !hasOnlyKeys(message.params, new Set()))) return;
        initialized = true;
        refreshRoots();
        return;
      }
      if (message.method === "notifications/roots/list_changed") {
        if (message.id !== undefined) {
          invalidRequest(message.id);
          return;
        }
        if (message.params !== undefined && !hasOnlyKeys(message.params, new Set())) return;
        refreshRoots();
        return;
      }
      if (message.id === undefined) return;
      if (message.method === "initialize") {
        if (negotiated) {
          send({ jsonrpc: "2.0", id: message.id, error: { code: -32600, message: "server is already initialized" } });
          return;
        }
        const params = message.params;
        if (!isObject(params) || typeof params.protocolVersion !== "string" || !isObject(params.capabilities) ||
          !isObject(params.clientInfo) || typeof params.clientInfo.name !== "string" || typeof params.clientInfo.version !== "string" ||
          !hasOnlyKeys(params, new Set(["protocolVersion", "capabilities", "clientInfo", "_meta"]))) {
          invalidRequest(message.id, "invalid initialize parameters");
          return;
        }
        if (params.capabilities.roots !== undefined && (!hasOnlyKeys(params.capabilities.roots, new Set(["listChanged"])) ||
          (params.capabilities.roots.listChanged !== undefined && typeof params.capabilities.roots.listChanged !== "boolean"))) {
          invalidRequest(message.id, "invalid roots capability");
          return;
        }
        negotiated = true;
        supportsRoots = params.capabilities.roots !== undefined;
        const requestedVersion = params.protocolVersion;
        const protocolVersion = SUPPORTED_PROTOCOL_VERSIONS.has(requestedVersion) ? requestedVersion : LATEST_PROTOCOL_VERSION;
        send({
          jsonrpc: "2.0",
          id: message.id,
          result: {
            protocolVersion,
            capabilities: { tools: { listChanged: false } },
            serverInfo: { name: "local-dev-env-mcp", version: "1.0.0" }
          }
        });
        return;
      }
      if (!initialized) {
        send({ jsonrpc: "2.0", id: message.id, error: { code: -32002, message: "server is not initialized" } });
        return;
      }
      if (message.method === "ping") {
        if (message.params !== undefined && !hasOnlyKeys(message.params, new Set())) {
          invalidRequest(message.id, "invalid ping parameters");
          return;
        }
        send({ jsonrpc: "2.0", id: message.id, result: {} });
        return;
      }
      if (message.method === "tools/list") {
        if (message.params !== undefined && (!hasOnlyKeys(message.params, new Set(["cursor"])) ||
          (message.params.cursor !== undefined && typeof message.params.cursor !== "string"))) {
          invalidRequest(message.id, "invalid tools/list parameters");
          return;
        }
        send({ jsonrpc: "2.0", id: message.id, result: { tools: toolDefinitions } });
        return;
      }
      if (message.method === "tools/call") {
        if (!isObject(message.params) || typeof message.params.name !== "string" ||
          (message.params.arguments !== undefined && !isObject(message.params.arguments)) ||
          !hasOnlyKeys(message.params, new Set(["name", "arguments", "_meta"]))) {
          invalidRequest(message.id, "invalid tools/call parameters");
          return;
        }
        try {
          const result = await callTool(message.params, await runtime());
          send({ jsonrpc: "2.0", id: message.id, result });
        } catch (error) {
          send({
            jsonrpc: "2.0",
            id: message.id,
            result: {
              isError: true,
              content: [{ type: "text", text: `local-dev-env-mcp: ${String(error?.message ?? error)}` }]
            }
          });
        }
        return;
      }
      send({ jsonrpc: "2.0", id: message.id, error: { code: -32601, message: `method not found: ${message.method}` } });
    } catch (error) {
      if (message?.id !== undefined) {
        send({ jsonrpc: "2.0", id: message.id, error: { code: -32603, message: String(error?.message ?? error) } });
      } else if (message === undefined) {
        send({ jsonrpc: "2.0", id: null, error: { code: -32700, message: "parse error" } });
      }
    }
  });

  await new Promise((resolve) => input.once("close", resolve));
}
