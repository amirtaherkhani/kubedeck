import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import test from "node:test";
import { Miniflare } from "miniflare";

async function createRuntime(bindings = {}) {
  const runtime = new Miniflare({
    bindings,
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
            first_name AS firstName,
            last_name AS lastName,
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
  assert.match(html, /Your Kubernetes ecosystem/);
  assert.match(html, /Multi-cluster access across all Kubernetes nodes/);
  assert.match(html, /aria-label="KubeDeck scope"/);
  assert.match(html, /kubedeck-banner\.png/);
  assert.doesNotMatch(html, /Rancher Desktop/i);

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

  const deniedSettings = await runtime.request("/settings");
  assert.equal(deniedSettings.status, 307);
  assert.match(deniedSettings.headers.get("location") ?? "", /\/$/);

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
  assert.match(html, /Your Kubernetes ecosystem/);
  assert.match(html, /kubedeck-banner\.png/);
  assert.match(html, /kubedeck-live-graph/);
  assert.match(html, /Cluster DNS/);
  assert.match(html, /Observability &amp; Metrics/);
  assert.match(html, /Databases &amp; Storage/);
  assert.match(html, /Web Applications/);
  assert.match(html, /Automation &amp; Workflows/);
  assert.match(html, /Deployments &amp; Testing/);
  assert.match(html, /AI &amp; MCP Services/);
  assert.match(html, /Developer Tools/);
  assert.match(html, /grafana\.dev\.local/);
  assert.match(html, /grafana\.monitoring\.svc\.cluster\.local/);
  assert.match(html, /postgresql\.storage\.svc\.cluster\.local/);
  assert.match(html, /kube-dns\.kube-system\.svc\.cluster\.local/);
  assert.match(html, /10\.43\.0\.10/);
  assert.match(html, /Fleet resources/);
  assert.match(html, /Cluster node review/);
  assert.match(html, /Resource history/);
  assert.match(html, /4 properties/);
  assert.match(html, /control-plane-01/);
  assert.match(html, /worker-data-01/);
  assert.match(html, /Illustrative multi-node snapshot/);
  assert.match(html, /Select node telemetry scope/);
  assert.match(html, /CPU/);
  assert.match(html, /Memory/);
  assert.match(html, /Storage/);
  assert.match(html, /Pod allocation/);
  assert.match(html, /Global discovery connected/);
  assert.match(html, /aria-label="Primary navigation"/);
  assert.match(html, /data-slot="animated-radio-group"/);
  assert.match(html, /role="radiogroup"/);
  assert.match(html, /role="radio"/);
  assert.match(html, /aria-label="Filter by resource kind"/);
  assert.match(html, /aria-label="Filter catalog by status"/);
  assert.match(html, /Last deploy/);
  assert.match(html, /unread status notifications/);
  assert.match(html, /brand\/kubedeck-mark\.svg/);
  assert.match(html, /href="\/settings"/);
  assert.match(html, /Sign out/);
  assert.doesNotMatch(html, /rancher[- ]desktop/i);
  assert.doesNotMatch(html, /codex-preview|Your site is taking shape/);

  const settings = await runtime.request("/settings", {
    headers: { accept: "text/html", cookie: loginCookie },
  });
  assert.equal(settings.status, 200);
  const settingsHtml = await settings.text();
  assert.match(settingsHtml, /Settings and access/);
  assert.match(settingsHtml, /Current user/);
  assert.match(settingsHtml, /Kubernetes/);
  assert.match(settingsHtml, />App</);
  assert.match(settingsHtml, />Users</);
  assert.match(settingsHtml, /Admin/);
  assert.match(settingsHtml, /All registered clusters/);

  const logout = await runtime.request("/api/auth/logout", {
    method: "POST",
    headers: { cookie: loginCookie },
  });
  assert.equal(logout.status, 303);
  assert.match(logout.headers.get("set-cookie") ?? "", /Max-Age=0/i);
});

test("bootstraps the first admin from complete backend environment values", async (t) => {
  const runtime = await createRuntime({
    KUBEDECK_ADMIN_FIRST_NAME: "Environment",
    KUBEDECK_ADMIN_LAST_NAME: "Admin",
    KUBEDECK_ADMIN_EMAIL: "env-admin@example.com",
    KUBEDECK_ADMIN_PASSWORD: "Environment-Password-2026",
  });
  t.after(() => runtime.dispose());

  const response = await runtime.request("/");
  assert.equal(response.status, 200);
  const html = await response.text();
  assert.match(html, /Sign in to KubeDeck/);
  assert.doesNotMatch(html, /Create the admin account/);

  const storedAdmin = await runtime.getAdminRecord();
  assert.equal(storedAdmin.firstName, "Environment");
  assert.equal(storedAdmin.lastName, "Admin");
  assert.equal(storedAdmin.email, "env-admin@example.com");
  assert.notEqual(storedAdmin.passwordHash, "Environment-Password-2026");

  const login = await runtime.request("/api/auth/login", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      email: "env-admin@example.com",
      password: "Environment-Password-2026",
    }),
  });
  assert.equal(login.status, 200);
  assert.match(login.headers.get("set-cookie") ?? "", /HttpOnly/i);
});

test("keeps browser setup available for incomplete backend environment values", async (t) => {
  const runtime = await createRuntime({
    KUBEDECK_ADMIN_EMAIL: "incomplete@example.com",
  });
  t.after(() => runtime.dispose());

  const response = await runtime.request("/");
  assert.equal(response.status, 200);
  assert.match(await response.text(), /Create the admin account/);
  assert.equal(await runtime.getAdminRecord(), null);
});

test("ships the admin schema, migration, and finished product assets", async () => {
  const [
    loginPage,
    bannerComponent,
    logoComponent,
    notificationsComponent,
    dashboardPage,
    dashboardClient,
    settingsPage,
    settingsClient,
    monitoringSource,
    chartComponent,
    tableComponent,
    manifestSource,
    authSource,
    schema,
    migration,
    layout,
    hosting,
    packageJson,
    socialImage,
    bannerImage,
    logoSvg,
    logo16,
    logo32,
    logo48,
    logo64,
    logo128,
    logo180,
    logo192,
    logo256,
    logo512,
    logo1024,
    liquidGlassPage,
    environmentExample,
    globalStyles,
  ] = await Promise.all([
    readFile(new URL("../app/page.tsx", import.meta.url), "utf8"),
    readFile(
      new URL("../components/kubedeck-banner.tsx", import.meta.url),
      "utf8",
    ),
    readFile(
      new URL("../components/kubedeck-logo.tsx", import.meta.url),
      "utf8",
    ),
    readFile(
      new URL("../components/notifications-menu.tsx", import.meta.url),
      "utf8",
    ),
    readFile(new URL("../app/dashboard/page.tsx", import.meta.url), "utf8"),
    readFile(
      new URL("../app/dashboard/dashboard-client.tsx", import.meta.url),
      "utf8",
    ),
    readFile(new URL("../app/settings/page.tsx", import.meta.url), "utf8"),
    readFile(
      new URL("../app/settings/settings-client.tsx", import.meta.url),
      "utf8",
    ),
    readFile(
      new URL("../lib/kubedeck-monitoring.ts", import.meta.url),
      "utf8",
    ),
    readFile(new URL("../components/ui/chart.tsx", import.meta.url), "utf8"),
    readFile(new URL("../components/ui/table.tsx", import.meta.url), "utf8"),
    readFile(new URL("../app/manifest.ts", import.meta.url), "utf8"),
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
    readFile(new URL("../public/kubedeck-banner.png", import.meta.url)),
    readFile(
      new URL("../public/brand/kubedeck-mark.svg", import.meta.url),
      "utf8",
    ),
    readFile(new URL("../public/brand/kubedeck-mark-16.png", import.meta.url)),
    readFile(new URL("../public/brand/kubedeck-mark-32.png", import.meta.url)),
    readFile(new URL("../public/brand/kubedeck-mark-48.png", import.meta.url)),
    readFile(new URL("../public/brand/kubedeck-mark-64.png", import.meta.url)),
    readFile(new URL("../public/brand/kubedeck-mark-128.png", import.meta.url)),
    readFile(new URL("../public/brand/kubedeck-mark-180.png", import.meta.url)),
    readFile(new URL("../public/brand/kubedeck-mark-192.png", import.meta.url)),
    readFile(new URL("../public/brand/kubedeck-mark-256.png", import.meta.url)),
    readFile(new URL("../public/brand/kubedeck-mark-512.png", import.meta.url)),
    readFile(new URL("../public/brand/kubedeck-mark-1024.png", import.meta.url)),
    readFile(
      new URL("../public/kubedeck-liquid-glass.html", import.meta.url),
      "utf8",
    ),
    readFile(new URL("../.env.example", import.meta.url), "utf8"),
    readFile(new URL("../app/globals.css", import.meta.url), "utf8"),
  ]);

  assert.match(loginPage, /<AdminAuthForm mode=\{isSetup/);
  assert.match(loginPage, /KubeDeckBanner/);
  assert.match(bannerComponent, /src="\/kubedeck-banner\.png"/);
  assert.match(bannerComponent, /Your Kubernetes ecosystem/);
  assert.match(bannerComponent, /liveGraph/);
  assert.match(bannerComponent, /kubedeck-live-link/);
  assert.match(bannerComponent, /kubedeck-live-beacon/);
  assert.match(logoComponent, /src="\/brand\/kubedeck-mark\.svg"/);
  assert.match(notificationsComponent, /Services and Kubernetes nodes/);
  assert.match(notificationsComponent, /kubedeck-notify-services/);
  assert.match(notificationsComponent, /kubedeck-notify-nodes/);
  assert.match(dashboardPage, /getCurrentAdmin/);
  assert.match(dashboardPage, /<DashboardClient admin=\{admin\}/);
  assert.match(dashboardClient, /KubeDeckBanner/);
  assert.match(dashboardClient, /const webApps:/);
  assert.match(dashboardClient, /const operationalMeta:/);
  assert.match(dashboardClient, /AI & MCP Services/);
  assert.match(dashboardClient, /MultiNodeReview/);
  assert.match(dashboardClient, /ChartContainer/);
  assert.match(dashboardClient, /LineChart/);
  assert.match(settingsPage, /getCurrentAdmin/);
  assert.match(settingsClient, /Single-admin authentication is active/);
  assert.match(settingsClient, /kubedeck-compact-catalog/);
  assert.match(settingsClient, /Multi-node cluster review/);
  assert.match(settingsClient, /Service status notifications/);
  assert.match(settingsClient, /Node status notifications/);
  assert.match(monitoringSource, /Illustrative multi-node snapshot/);
  assert.match(monitoringSource, /control-plane-01/);
  assert.match(monitoringSource, /worker-data-01/);
  assert.match(chartComponent, /ResponsiveContainer/);
  assert.match(tableComponent, /data-slot="table"/);
  assert.match(manifestSource, /kubedeck-mark-192\.png/);
  assert.match(manifestSource, /kubedeck-mark-512\.png/);
  assert.doesNotMatch(dashboardClient, /rancher[- ]desktop/i);
  assert.match(authSource, /PBKDF2/);
  assert.match(authSource, /KUBEDECK_ADMIN_FIRST_NAME/);
  assert.match(authSource, /KUBEDECK_ADMIN_PASSWORD/);
  assert.match(authSource, /httpOnly: true/);
  assert.match(authSource, /sameSite: "strict"/);
  assert.match(schema, /adminUsers/);
  assert.match(migration, /CREATE TABLE `admin_users`/);
  assert.match(hosting, /"d1": "DB"/);
  assert.match(layout, /summary_large_image/);
  assert.match(layout, /kubedeck-mark\.svg/);
  assert.match(layout, /kubedeck-mark-180\.png/);
  assert.match(layout, /Manrope/);
  assert.doesNotMatch(layout, /\bGeist\b/);
  assert.match(globalStyles, /font-family: var\(--font-manrope\)/);
  assert.match(globalStyles, /\.dashboard-sidebar/);
  assert.match(globalStyles, /\.liquid-orbit/);
  assert.match(packageJson, /"recharts": "\^3\.8\.0"/);
  assert.doesNotMatch(packageJson, /react-loading-skeleton/);
  assert.match(environmentExample, /KUBEDECK_ADMIN_EMAIL=/);
  assert.match(liquidGlassPage, /<html lang="en">/);
  assert.match(
    liquidGlassPage,
    /<title>KubeDeck — Kubernetes Ecosystem<\/title>/,
  );
  assert.match(liquidGlassPage, /--u: min\(100vw \/ 1357, 100dvh \/ 871\)/);
  assert.match(liquidGlassPage, /id="i-grid"/);
  assert.match(liquidGlassPage, /id="i-network"/);
  assert.match(liquidGlassPage, /@media \(prefers-reduced-motion: reduce\)/);
  assert.doesNotMatch(liquidGlassPage, /<script\b/i);
  assert.deepEqual(
    [...socialImage.subarray(0, 8)],
    [137, 80, 78, 71, 13, 10, 26, 10],
  );
  assert.equal(socialImage.readUInt32BE(16), 1731);
  assert.equal(socialImage.readUInt32BE(20), 909);
  assert.deepEqual(
    [...bannerImage.subarray(0, 8)],
    [137, 80, 78, 71, 13, 10, 26, 10],
  );
  assert.equal(bannerImage.readUInt32BE(16), 1731);
  assert.equal(bannerImage.readUInt32BE(20), 909);
  assert.match(logoSvg, /<title id="title">KubeDeck KB logo<\/title>/);
  assert.match(logoSvg, /fill-rule="evenodd"/);
  assert.match(logoSvg, /linearGradient id="monogram"/);
  for (const [asset, size] of [
    [logo16, 16],
    [logo32, 32],
    [logo48, 48],
    [logo64, 64],
    [logo128, 128],
    [logo180, 180],
    [logo192, 192],
    [logo256, 256],
    [logo512, 512],
    [logo1024, 1024],
  ]) {
    assert.deepEqual(
      [...asset.subarray(0, 8)],
      [137, 80, 78, 71, 13, 10, 26, 10],
    );
    assert.equal(asset.readUInt32BE(16), size);
    assert.equal(asset.readUInt32BE(20), size);
  }

  await assert.rejects(
    access(new URL("../app/_sites-preview", import.meta.url)),
  );
  await assert.rejects(
    access(new URL("../components/kubedeck-topology-hero.tsx", import.meta.url)),
  );
});
