import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import test from "node:test";

async function render() {
  const workerUrl = new URL("../dist/server/index.js", import.meta.url);
  workerUrl.searchParams.set("test", `${process.pid}-${Date.now()}`);
  const { default: worker } = await import(workerUrl.href);

  return worker.fetch(
    new Request("http://localhost/", {
      headers: { accept: "text/html" },
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
  const [page, layout, packageJson, socialImage] = await Promise.all([
    readFile(new URL("../app/page.tsx", import.meta.url), "utf8"),
    readFile(new URL("../app/layout.tsx", import.meta.url), "utf8"),
    readFile(new URL("../package.json", import.meta.url), "utf8"),
    readFile(new URL("../public/og.png", import.meta.url)),
  ]);

  assert.match(page, /const webApps:/);
  assert.match(page, /const services:/);
  assert.match(page, /const operationalMeta:/);
  assert.match(page, /const categoryConfig:/);
  assert.match(page, /lastDeployedAt: "2026-07-24T19:07:45Z"/);
  assert.match(page, /metrics-server \+ kubelet summary/);
  assert.match(page, /Ingress \+ Service/);
  assert.match(layout, /summary_large_image/);
  assert.doesNotMatch(packageJson, /react-loading-skeleton/);
  assert.deepEqual(
    [...socialImage.subarray(0, 8)],
    [137, 80, 78, 71, 13, 10, 26, 10],
  );

  await assert.rejects(
    access(new URL("../app/_sites-preview", import.meta.url)),
  );
});
