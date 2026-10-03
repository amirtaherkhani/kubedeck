#!/usr/bin/env node

import { resolve } from "node:path";
import { contextSummary, resolveProjectContext } from "./context.mjs";
import { InfisicalStore } from "./infisical.mjs";
import { loadCredentials } from "./keychain.mjs";
import { registerProject, unregisterProjectsUnder } from "./registry.mjs";
import { serve } from "./server.mjs";

function usage() {
  console.log(`Usage:
  kubedeck-env-mcp serve
  kubedeck-env-mcp doctor [--root DIR]
  kubedeck-env-mcp register --project-id ID --environment SLUG [--path PATH] [--domain URL] [--root DIR]
  kubedeck-env-mcp unregister --root DIR

register stores non-secret project metadata in ~/.config/local-dev-env/projects.json.`);
}

function options(args) {
  const parsed = {};
  for (let index = 0; index < args.length; index += 2) {
    if (!args[index]?.startsWith("--") || args[index + 1] === undefined) throw new Error(`invalid option: ${args[index] ?? ""}`);
    parsed[args[index].slice(2)] = args[index + 1];
  }
  return parsed;
}

async function doctor(args) {
  const flags = options(args);
  if (Object.keys(flags).some((key) => key !== "root")) throw new Error("doctor accepts only --root");
  const context = resolveProjectContext({ cwd: resolve(flags.root ?? process.cwd()) });
  const store = new InfisicalStore({ credentials: loadCredentials(), context });
  const names = await store.list();
  console.log(JSON.stringify({ ok: true, context: contextSummary(context), secretCount: names.length }, null, 2));
}

async function main() {
  const [command = "serve", ...args] = process.argv.slice(2);
  if (command === "serve") return serve();
  if (command === "doctor") return doctor(args);
  if (command === "register") {
    const flags = options(args);
    const context = registerProject({
      root: resolve(flags.root ?? process.cwd()),
      projectId: flags["project-id"],
      environment: flags.environment,
      path: flags.path ?? "/",
      domain: flags.domain,
    });
    console.log(JSON.stringify({ registered: contextSummary(context) }, null, 2));
    return;
  }
  if (command === "unregister") {
    const flags = options(args);
    const root = resolve(flags.root ?? process.cwd());
    console.log(JSON.stringify({ root, removed: unregisterProjectsUnder(root) }, null, 2));
    return;
  }
  if (command === "help" || command === "--help" || command === "-h") return usage();
  throw new Error(`unknown command: ${command}`);
}

main().catch((error) => {
  console.error(`kubedeck-env-mcp: ${String(error?.message ?? error)}`);
  process.exitCode = 1;
});
