import { execFileSync } from "node:child_process";
import { existsSync, readFileSync, realpathSync, statSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, isAbsolute, join, normalize, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const DEFAULT_DOMAIN = "https://infisical.local.dev";
export const DEFAULT_REGISTRY_PATH = join(homedir(), ".config", "local-dev-env", "projects.json");
export const ALLOWED_ENVIRONMENTS = new Set(["local", "development", "dev"]);

function readJson(path) {
  return JSON.parse(readFileSync(path, "utf8"));
}

function normalizeRoot(root) {
  const path = root.startsWith("file:") ? fileURLToPath(root) : root;
  const absolute = normalize(resolve(path));
  return existsSync(absolute) ? realpathSync(absolute) : absolute;
}

function contains(parent, child) {
  return child === parent || child.startsWith(`${parent}/`);
}

function worktreeAlias(start) {
  let current = normalizeRoot(start);
  for (;;) {
    const marker = join(current, ".git");
    if (existsSync(marker) && statSync(marker).isFile()) {
      try {
        const inventory = execFileSync("git", ["-C", current, "worktree", "list", "--porcelain"], {
          encoding: "utf8",
          stdio: ["ignore", "pipe", "ignore"]
        });
        const worktrees = [...inventory.matchAll(/^worktree (.+)$/gm)].map((match) => normalizeRoot(match[1]));
        if (!worktrees.includes(current)) return null;
        return join(worktrees[0], relative(current, normalizeRoot(start)));
      } catch {
        return null;
      }
    }
    if (existsSync(marker) && statSync(marker).isDirectory()) return null;
    const parent = dirname(current);
    if (parent === current) return null;
    current = parent;
  }
}

function validateContext(raw, root, source) {
  const projectId = raw.projectId ?? raw.workspaceId;
  const environment = raw.environment ?? raw.defaultEnvironment;
  const secretPath = raw.path ?? raw.secretPath ?? "/";
  const domain = raw.domain ?? DEFAULT_DOMAIN;

  if (!projectId || !environment) {
    throw new Error(`${source} must define projectId/workspaceId and environment/defaultEnvironment`);
  }
  if (!ALLOWED_ENVIRONMENTS.has(environment)) {
    throw new Error(`environment ${environment} is blocked; kubedeck-env-mcp permits only local, development, or dev`);
  }
  if (domain !== DEFAULT_DOMAIN) throw new Error(`${source} domain must be ${DEFAULT_DOMAIN}`);
  if (!secretPath.startsWith("/")) throw new Error(`${source} path must start with /`);
  return { projectId, environment, path: secretPath, domain, root, source };
}

export function resolveProjectContext({ roots = [], cwd = process.cwd(), registryPath = DEFAULT_REGISTRY_PATH } = {}) {
  const directCandidates = (roots.length > 0 ? roots : [cwd]).filter(Boolean).map(normalizeRoot);

  if (existsSync(registryPath)) {
    const registry = readJson(registryPath);
    const projects = (registry.projects ?? []).map((entry) => ({ ...entry, root: normalizeRoot(entry.root) }));
    const contexts = directCandidates.flatMap((candidate) => {
      const candidates = [candidate, worktreeAlias(candidate)].filter(Boolean);
      const match = projects
        .filter((entry) => candidates.some((resolved) => contains(entry.root, resolved)))
        .sort((a, b) => b.root.length - a.root.length)[0];
      return match ? [validateContext(match, match.root, registryPath)] : [];
    });
    if (roots.length > 0 && contexts.length !== directCandidates.length) {
      throw new Error("every workspace root must have an Infisical project mapping");
    }
    if (contexts.length > 0) {
      const signatures = new Set(contexts.map(({ projectId, environment, path, domain }) =>
        JSON.stringify([projectId, environment, path, domain])));
      if (signatures.size > 1) {
        throw new Error("workspace roots resolve to different Infisical scopes; open one project scope per agent session");
      }
      return contexts[0];
    }
  }

  throw new Error("no Infisical project mapping found; run kubedeck-env-mcp register from this project");
}

export function contextSummary(context) {
  return {
    root: context.root,
    environment: context.environment,
    path: context.path,
    domain: context.domain,
    source: context.source
  };
}

export function projectRegistration({ root, projectId, environment, path = "/", domain = DEFAULT_DOMAIN }) {
  if (!isAbsolute(root)) throw new Error("project root must be an absolute path");
  return validateContext({ projectId, environment, path, domain }, normalizeRoot(root), "registration");
}
