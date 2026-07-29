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
  assert.match(html, /Your Kubernetes ecosystem,/);
  assert.match(html, /Web Apps/);
  assert.match(html, /Services/);
  assert.match(html, /grafana\.dev\.local/);
  assert.match(html, /postgresql\.storage\.svc\.cluster\.local/);
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
