import assert from "node:assert/strict";
import test from "node:test";
import { InfisicalStore } from "../src/infisical.mjs";

function storeWith(secrets) {
  const calls = [];
  const requests = [];
  const response = (status, data) => ({ ok: status >= 200 && status < 300, status, json: async () => data });
  const fetchImpl = async (url, options) => {
    if (url.pathname.endsWith("/auth/universal-auth/login")) return response(200, { accessToken: "token", expiresIn: 3600 });
    const body = options.body ? JSON.parse(options.body) : undefined;
    requests.push({
      path: url.pathname,
      method: options.method,
      query: Object.fromEntries(url.searchParams),
      body
    });
    if (url.pathname === "/api/v3/secrets/raw") {
      return response(200, { secrets: Object.keys(secrets).map((secretKey) => ({ secretKey })) });
    }
    const name = decodeURIComponent(url.pathname.split("/").at(-1));
    if (options.method === "GET") {
      return name in secrets ? response(200, { secret: { secretKey: name, secretValue: secrets[name] } }) : response(404, {});
    }
    if (options.method === "PATCH") {
      if (!(name in secrets)) return response(404, {});
      calls.push(["update", name]); secrets[name] = body.secretValue; return response(200, {});
    }
    if (options.method === "POST") {
      calls.push(["create", name]); secrets[name] = body.secretValue; return response(200, {});
    }
    if (options.method === "DELETE") {
      calls.push(["delete", name]); delete secrets[name]; return response(200, {});
    }
    return response(405, {});
  };
  const store = new InfisicalStore({
    credentials: { clientId: "id", clientSecret: "secret" },
    context: { projectId: "p", environment: "local", path: "/", domain: "https://example.test" },
    fetchImpl
  });
  return { store, calls, requests };
}

test("lists names without values and creates or updates values", async () => {
  const { store, calls } = storeWith({ EXISTING: "old" });
  assert.deepEqual(await store.list(), ["EXISTING"]);
  assert.equal(await store.set("EXISTING", "new"), "updated");
  assert.equal(await store.set("NEW_VALUE", "value"), "created");
  assert.deepEqual(calls, [["update", "EXISTING"], ["create", "NEW_VALUE"]]);
  assert.equal((await store.get("NEW_VALUE")).secretValue, "value");
});

test("deletes an explicitly named shared value", async () => {
  const { store, calls } = storeWith({ TEMP_VALUE: "value" });
  await store.delete("TEMP_VALUE");
  assert.deepEqual(calls, [["delete", "TEMP_VALUE"]]);
});

test("refreshes authentication once after an API 401", async () => {
  let logins = 0;
  let lists = 0;
  const response = (status, data) => ({ ok: status >= 200 && status < 300, status, json: async () => data });
  const store = new InfisicalStore({
    credentials: { clientId: "id", clientSecret: "secret" },
    context: { projectId: "p", environment: "local", path: "/", domain: "https://example.test" },
    fetchImpl: async (url) => {
      if (url.pathname.endsWith("/auth/universal-auth/login")) {
        logins += 1;
        return response(200, { accessToken: `token-${logins}`, expiresIn: 3600 });
      }
      lists += 1;
      return lists === 1 ? response(401, {}) : response(200, { secrets: [] });
    }
  });

  assert.deepEqual(await store.list(), []);
  assert.equal(logins, 2);
  assert.equal(lists, 2);
});

test("concurrent stale-token 401 responses reuse one refreshed login", async () => {
  let logins = 0;
  let staleRequests = 0;
  const response = (status, data) => ({ ok: status >= 200 && status < 300, status, json: async () => data });
  const store = new InfisicalStore({
    credentials: { clientId: "id", clientSecret: "secret" },
    context: { projectId: "p", environment: "local", path: "/", domain: "https://example.test" },
    fetchImpl: async (url, options) => {
      if (url.pathname.endsWith("/auth/universal-auth/login")) {
        logins += 1;
        return response(200, { accessToken: `token-${logins}`, expiresIn: 3600 });
      }
      if (options.headers.authorization === "Bearer token-1") {
        staleRequests += 1;
        await new Promise((resolve) => setTimeout(resolve, staleRequests === 1 ? 10 : 30));
        return response(401, {});
      }
      return response(200, { secrets: [] });
    }
  });

  await store.token();
  assert.deepEqual(await Promise.all([store.list(), store.list()]), [[], []]);
  assert.equal(logins, 2);
});

test("sends the exact project, environment, and selected path on every operation", async () => {
  const { store, requests } = storeWith({ EXISTING: "old" });
  const path = "/apps/storage/postgres";
  await store.list(path);
  await store.get("EXISTING", path);
  await store.set("EXISTING", "new", path);
  await store.set("NEW_VALUE", "value", path);
  await store.delete("NEW_VALUE", path);

  for (const request of requests) {
    const scope = request.method === "GET" ? request.query : request.body;
    assert.equal(scope.workspaceId, "p");
    assert.equal(scope.environment, "local");
    assert.equal(scope.secretPath, path);
  }
  assert.deepEqual(requests[0].query, {
    workspaceId: "p",
    environment: "local",
    secretPath: path,
    viewSecretValue: "false"
  });
  assert.equal(requests[1].query.type, "shared");
  assert.equal(requests[1].query.viewSecretValue, "true");
  assert.equal(requests.at(-1).method, "DELETE");
  assert.equal(requests.at(-1).body.type, "shared");
});
