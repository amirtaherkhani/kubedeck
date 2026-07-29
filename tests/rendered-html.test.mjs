import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import test from "node:test";

async function render(pathname = "/dashboard", requestHeaders = {}) {
  const workerUrl = new URL("../dist/server/index.js", import.meta.url);
  workerUrl.searchParams.set("test", `${process.pid}-${Date.now()}`);
  const { default: worker } = await import(workerUrl.href);

  return worker.fetch(
    new Request(`http://localhost${pathname}`, {
      headers: { accept: "text/html", ...requestHeaders },
    }),
    {
      ASSETS: {
        fetch: async () => new Response("Not found", { status: 404 }),
      },
    },
    {
      waitUntil() {},
      passThroughOnException() {},
    },
  );
}

test("server-renders the private KubeDeck login page", async () => {
  const response = await render("/", {
    "oai-authenticated-user-email": "amir@example.com",
    "oai-authenticated-user-full-name": "Amir%20Taherkhani",
    "oai-authenticated-user-full-name-encoding": "percent-encoded-utf-8",
  });
  assert.equal(response.status, 200);

  const html = await response.text();
  assert.match(html, /Sign in to KubeDeck/);
  assert.match(html, /KubeDeck Kubernetes ecosystem banner/);
  assert.match(html, /Amir Taherkhani/);
  assert.match(html, /amir@example\.com/);
  assert.match(html, /Continue to dashboard/);
  assert.match(html, /does not request or store Rancher tokens/);
  assert.match(html, /href="\/dashboard"/);
});

test("server-renders the KubeDeck cluster catalog", async () => {
  const response = await render();
  assert.equal(response.status, 200);
  assert.match(response.headers.get("content-type") ?? "", /^text\/html\b/i);

  const html = await response.text();
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
  assert.match(html, /Current run/);
  assert.match(html, /Node resources/);
  assert.match(html, /52\.3 \/ 97\.9 GiB/);
  assert.match(html, /Last deploy/);
  assert.match(html, /Jul 24, 2026 · 19:07 UTC/);
  assert.match(html, /rancher-desktop · v1\.36\.1\+k3s1/);
  assert.match(
    html,
    /<meta property="og:image" content="http:\/\/localhost:3000\/og\.png"/i,
  );
  assert.doesNotMatch(html, /codex-preview|Your site is taking shape/);
});

test("ships the finished product assets without starter preview code", async () => {
  const [loginPage, dashboardPage, layout, packageJson, socialImage] =
    await Promise.all([
      readFile(new URL("../app/page.tsx", import.meta.url), "utf8"),
      readFile(new URL("../app/dashboard/page.tsx", import.meta.url), "utf8"),
      readFile(new URL("../app/layout.tsx", import.meta.url), "utf8"),
      readFile(new URL("../package.json", import.meta.url), "utf8"),
      readFile(new URL("../public/og.png", import.meta.url)),
    ]);

  assert.match(loginPage, /export const dynamic = "force-dynamic"/);
  assert.match(loginPage, /oai-authenticated-user-email/);
  assert.match(loginPage, /src="\/og\.png"/);
  assert.match(loginPage, /width=\{1731\}/);
  assert.match(loginPage, /height=\{909\}/);
  assert.match(loginPage, /href="\/dashboard"/);
  assert.match(dashboardPage, /const webApps:/);
  assert.match(dashboardPage, /const services:/);
  assert.match(dashboardPage, /const operationalMeta:/);
  assert.match(dashboardPage, /const categoryConfig:/);
  assert.match(dashboardPage, /lastDeployedAt: "2026-07-24T19:07:45Z"/);
  assert.match(dashboardPage, /metrics-server \+ kubelet summary/);
  assert.match(dashboardPage, /Ingress \+ Service/);
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
