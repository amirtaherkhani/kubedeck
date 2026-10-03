import { chmodSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";
import { DEFAULT_REGISTRY_PATH, projectRegistration } from "./context.mjs";

export function registerProject(input, registryPath = DEFAULT_REGISTRY_PATH) {
  const registration = projectRegistration(input);
  const registry = existsSync(registryPath) ? JSON.parse(readFileSync(registryPath, "utf8")) : { version: 1, projects: [] };
  const projects = (registry.projects ?? []).filter(({ root }) => root !== registration.root);
  projects.push({
    root: registration.root,
    projectId: registration.projectId,
    environment: registration.environment,
    path: registration.path,
    domain: registration.domain
  });
  projects.sort((a, b) => a.root.localeCompare(b.root));
  mkdirSync(dirname(registryPath), { recursive: true, mode: 0o700 });
  writeFileSync(registryPath, `${JSON.stringify({ version: 1, projects }, null, 2)}\n`, { mode: 0o600 });
  chmodSync(registryPath, 0o600);
  return registration;
}

export function unregisterProjectsUnder(root, registryPath = DEFAULT_REGISTRY_PATH) {
  if (!existsSync(registryPath)) return 0;
  const target = projectRegistration({ root, projectId: "unused", environment: "local" }).root;
  const registry = JSON.parse(readFileSync(registryPath, "utf8"));
  const projects = registry.projects ?? [];
  const retained = projects.filter((entry) => entry.root !== target && !entry.root.startsWith(`${target}/`));
  writeFileSync(registryPath, `${JSON.stringify({ version: 1, projects: retained }, null, 2)}\n`, { mode: 0o600 });
  chmodSync(registryPath, 0o600);
  return projects.length - retained.length;
}
