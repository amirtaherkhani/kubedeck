import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import test from "node:test";
import { Miniflare } from "miniflare";

async function createRuntime() {
  const runtime = new Miniflare({
    compatibilityDate: "2026-05-22",
    compatibilityFlags: ["nodejs_compat"],
    d1Databases: { DB: "kubedeck-test-db" },
    modules: true,
    modulesRules: [
      { type: "ESModule", include: ["**/*.js"], fallthrough: true },
    ],
    scriptPath: fileURLToPath(
      new URL("../dist/server/index.js", import.meta.url),
    ),
    serviceBindings: {
      ASSETS: async () => new Response("Not found", { status: 404 }),
    },
  });
  let db;
  try {
    db = await runtime.getD1Database("DB");
    const migration = await readFile(
      new URL("../drizzle/0000_fixed_forge.sql", import.meta.url),
      "utf8",
    );
    for (const statement of migration.split("--> statement-breakpoint")) {
      const sql = statement.trim();
      if (sql) await db.prepare(sql).run();
    }
  } catch (error) {
    await runtime.dispose();
    throw error;
  }

  return {
    request(pathname, options = {}) {
      return runtime.dispatchFetch(`http://localhost${pathname}`, {
        redirect: "manual",
        ...options,
      });
    },
    getAdminRecord: () =>
      db
        .prepare(
          `SELECT
            email,
            password_hash AS passwordHash,
            password_salt AS passwordSalt,
            password_iterations AS passwordIterations
          FROM admin_users
          WHERE id = 1`,
        )
        .first(),
    dispose: () => runtime.dispose(),
  };
}

async function setupAdmin(runtime) {
  const response = await runtime.request("/api/auth/setup", {
    method: "POST",
    headers: {
      "content-type": "application/json",
      "oai-authenticated-user-email": "owner@example.com",
    },
    body: JSON.stringify({
      firstName: "Amir",
      lastName: "Taherkhani",
      email: "admin@example.com",
      password: "Longer-Test-Password-2026",
    }),
  });
  assert.equal(response.status, 201);

  const setCookie = response.headers.get("set-cookie") ?? "";
  assert.match(setCookie, /__Host-kubedeck_admin=/);
  assert.match(setCookie, /HttpOnly/i);
  assert.match(setCookie, /Secure/i);
  assert.match(setCookie, /SameSite=Strict/i);

  return setCookie.split(";")[0];
}

test("renders one-time admin setup and creates a hashed admin account", async (t) => {
  const runtime = await createRuntime();
  t.after(() => runtime.dispose());
  const response = await runtime.request("/");
  assert.equal(response.status, 200);

  const html = await response.text();
  assert.match(html, /Create the admin account/);
  assert.match(html, /First name/);
  assert.match(html, /Last name/);
  assert.match(html, /Admin email/);
  assert.match(html, /Confirm password/);
  assert.match(html, /stores only a salted hash/);
  assert.match(html, /KubeDeck Kubernetes ecosystem banner/);

  const unverifiedSetup = await runtime.request("/api/auth/setup", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      firstName: "Unverified",
      lastName: "User",
      email: "unverified@example.com",
      password: "Another-Long-Password-2026",
    }),
  });
  assert.equal(unverifiedSetup.status, 403);

  await setupAdmin(runtime);
  const storedAdmin = await runtime.getAdminRecord();
  assert.equal(storedAdmin.email, "admin@example.com");
  assert.notEqual(storedAdmin.passwordHash, "Longer-Test-Password-2026");
  assert.ok(storedAdmin.passwordSalt.length >= 20);
  assert.equal(storedAdmin.passwordIterations, 210_000);

  const duplicate = await runtime.request("/api/auth/setup", {
    method: "POST",
    headers: {
      "content-type": "application/json",
      "oai-authenticated-user-email": "owner@example.com",
    },
    body: JSON.stringify({
      firstName: "Another",
      lastName: "Admin",
      email: "another@example.com",
      password: "Another-Long-Password-2026",
    }),
  });
  assert.equal(duplicate.status, 409);
});

test("authenticates the configured admin and protects the dashboard", async (t) => {
  const runtime = await createRuntime();
  t.after(() => runtime.dispose());
  const setupCookie = await setupAdmin(runtime);

  const loginPage = await runtime.request("/");
  assert.equal(loginPage.status, 200);
  assert.match(await loginPage.text(), /Sign in to KubeDeck/);

  const denied = await runtime.request("/dashboard");
  assert.equal(denied.status, 307);
  assert.match(denied.headers.get("location") ?? "", /\/$/);

  const invalidLogin = await runtime.request("/api/auth/login", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      email: "admin@example.com",
      password: "incorrect-password",
    }),
  });
  assert.equal(invalidLogin.status, 401);

  const validLogin = await runtime.request("/api/auth/login", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      email: "admin@example.com",
      password: "Longer-Test-Password-2026",
    }),
  });
  assert.equal(validLogin.status, 200);
  const loginCookie =
    (validLogin.headers.get("set-cookie") ?? "").split(";")[0] || setupCookie;

  const dashboard = await runtime.request("/dashboard", {
    headers: { accept: "text/html", cookie: loginCookie },
  });
  assert.equal(dashboard.status, 200);
  assert.match(dashboard.headers.get("content-type") ?? "", /^text\/html\b/i);

  const html = await dashboard.text();
  assert.match(html, /<title>KubeDeck — Kubernetes Launchpad<\/title>/i);
  assert.match(html, /Every cluster route,/);
  assert.match(html, /Cluster DNS/);
  assert.match(html, /Observability/);
  assert.match(html, /Databases &amp; Storage/);
  assert.match(html, /MCP &amp; Developer Tools/);
  assert.match(html, /grafana\.dev\.local/);
  assert.match(html, /grafana\.monitoring\.svc\.cluster\.local/);
  assert.match(html, /postgresql\.storage\.svc\.cluster\.local/);
  assert.match(html, /kube-dns\.kube-system\.svc\.cluster\.local/);
  assert.match(html, /10\.43\.0\.10/);
  assert.match(html, /Node resources/);
  assert.match(html, /Last deploy/);
  assert.match(html, /Sign out/);
  assert.doesNotMatch(html, /codex-preview|Your site is taking shape/);

  const logout = await runtime.request("/api/auth/logout", {
    method: "POST",
    headers: { cookie: loginCookie },
  });
  assert.equal(logout.status, 303);
  assert.match(logout.headers.get("set-cookie") ?? "", /Max-Age=0/i);
});

test("ships the admin schema, migration, and finished product assets", async () => {
  const [
    loginPage,
    dashboardPage,
    dashboardClient,
    authSource,
    schema,
    migration,
    layout,
    hosting,
    packageJson,
    socialImage,
  ] = await Promise.all([
    readFile(new URL("../app/page.tsx", import.meta.url), "utf8"),
    readFile(new URL("../app/dashboard/page.tsx", import.meta.url), "utf8"),
    readFile(
      new URL("../app/dashboard/dashboard-client.tsx", import.meta.url),
      "utf8",
    ),
    readFile(new URL("../lib/auth.ts", import.meta.url), "utf8"),
    readFile(new URL("../db/schema.ts", import.meta.url), "utf8"),
    readFile(
      new URL("../drizzle/0000_fixed_forge.sql", import.meta.url),
      "utf8",
    ),
    readFile(new URL("../app/layout.tsx", import.meta.url), "utf8"),
    readFile(new URL("../.openai/hosting.json", import.meta.url), "utf8"),
    readFile(new URL("../package.json", import.meta.url), "utf8"),
    readFile(new URL("../public/og.png", import.meta.url)),
  ]);

  assert.match(loginPage, /<AdminAuthForm mode=\{isSetup/);
  assert.match(loginPage, /unoptimized/);
  assert.match(dashboardPage, /getCurrentAdmin/);
  assert.match(dashboardClient, /const webApps:/);
  assert.match(dashboardClient, /const operationalMeta:/);
  assert.match(authSource, /PBKDF2/);
  assert.match(authSource, /httpOnly: true/);
  assert.match(authSource, /sameSite: "strict"/);
  assert.match(schema, /adminUsers/);
  assert.match(migration, /CREATE TABLE `admin_users`/);
  assert.match(hosting, /"d1": "DB"/);
  assert.match(layout, /summary_large_image/);
  assert.doesNotMatch(packageJson, /react-loading-skeleton/);
  assert.deepEqual(
    [...socialImage.subarray(0, 8)],
    [137, 80, 78, 71, 13, 10, 26, 10],
  );
  assert.equal(socialImage.readUInt32BE(16), 1731);
  assert.equal(socialImage.readUInt32BE(20), 909);

  await assert.rejects(
    access(new URL("../app/_sites-preview", import.meta.url)),
  );
});
