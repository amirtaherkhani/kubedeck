import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtempSync, mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { resolveProjectContext } from "../src/context.mjs";

test("uses the longest matching central registration", () => {
  const base = mkdtempSync(join(tmpdir(), "local-dev-env-"));
  const nested = join(base, "apps", "api");
  mkdirSync(nested, { recursive: true });
  const registryPath = join(base, "projects.json");
  writeFileSync(registryPath, JSON.stringify({ projects: [
    { root: base, projectId: "parent", environment: "local", path: "/" },
    { root: join(base, "apps"), projectId: "child", environment: "development", path: "/api" }
  ] }));

  const context = resolveProjectContext({ cwd: nested, registryPath });
  assert.equal(context.projectId, "child");
  assert.equal(context.path, "/api");
  assert.equal(context.credentialProfile, undefined);
});

test("maps a Git worktree path to the registered primary checkout", () => {
  const base = mkdtempSync(join(tmpdir(), "local-dev-env-"));
  const primary = join(base, "primary");
  const worktree = join(base, "worktree");
  const nested = join(worktree, "apps", "api");
  mkdirSync(primary);
  execFileSync("git", ["init", primary], { stdio: "ignore" });
  execFileSync("git", ["-C", primary, "-c", "user.name=Test", "-c", "user.email=test@local.dev", "commit", "--allow-empty", "-m", "initial"], { stdio: "ignore" });
  execFileSync("git", ["-C", primary, "worktree", "add", "-b", "task", worktree], { stdio: "ignore" });
  mkdirSync(join(primary, "apps", "api"), { recursive: true });
  mkdirSync(nested, { recursive: true });
  const registryPath = join(base, "projects.json");
  writeFileSync(registryPath, JSON.stringify({ projects: [
    { root: join(primary, "apps", "api"), projectId: "project", environment: "local", path: "/service" }
  ] }));

  const context = resolveProjectContext({ cwd: nested, registryPath });
  assert.equal(context.projectId, "project");
  assert.equal(context.path, "/service");
});

test("rejects a forged worktree pointer that is absent from Git's inventory", () => {
  const base = mkdtempSync(join(tmpdir(), "local-dev-env-"));
  const primary = join(base, "primary");
  const worktree = join(base, "worktree");
  const forged = join(base, "forged");
  mkdirSync(primary);
  execFileSync("git", ["init", primary], { stdio: "ignore" });
  execFileSync("git", ["-C", primary, "-c", "user.name=Test", "-c", "user.email=test@local.dev", "commit", "--allow-empty", "-m", "initial"], { stdio: "ignore" });
  execFileSync("git", ["-C", primary, "worktree", "add", "-b", "task", worktree], { stdio: "ignore" });
  mkdirSync(join(forged, "apps", "api"), { recursive: true });
  writeFileSync(join(forged, ".git"), `gitdir: ${join(primary, ".git", "worktrees", "worktree")}\n`);
  const registryPath = join(base, "projects.json");
  writeFileSync(registryPath, JSON.stringify({ projects: [
    { root: join(primary, "apps", "api"), projectId: "project", environment: "local", path: "/service" }
  ] }));

  assert.throws(
    () => resolveProjectContext({ roots: [join(forged, "apps", "api")], registryPath }),
    /every workspace root/
  );
});

test("blocks non-development environments", () => {
  const base = mkdtempSync(join(tmpdir(), "local-dev-env-"));
  const registryPath = join(base, "projects.json");
  writeFileSync(registryPath, JSON.stringify({ projects: [
    { root: base, projectId: "project", environment: "production" }
  ] }));
  assert.throws(() => resolveProjectContext({ cwd: base, registryPath }), /blocked/);
});

test("blocks registry entries that could exfiltrate credentials to another domain", () => {
  const base = mkdtempSync(join(tmpdir(), "local-dev-env-"));
  const registryPath = join(base, "projects.json");
  writeFileSync(registryPath, JSON.stringify({ projects: [
    { root: base, projectId: "project", environment: "local", domain: "https://attacker.example" }
  ] }));
  assert.throws(() => resolveProjectContext({ cwd: base, registryPath }), /domain must be/);
});

test("does not fall back to server cwd when client roots are present", () => {
  const base = mkdtempSync(join(tmpdir(), "local-dev-env-"));
  const unrelated = mkdtempSync(join(tmpdir(), "unregistered-root-"));
  const registryPath = join(base, "projects.json");
  writeFileSync(registryPath, JSON.stringify({ projects: [
    { root: base, projectId: "project", environment: "local" }
  ] }));
  assert.throws(() => resolveProjectContext({ roots: [unrelated], cwd: base, registryPath }), /workspace root/);
});

test("rejects client roots that resolve to different project scopes", () => {
  const base = mkdtempSync(join(tmpdir(), "local-dev-env-"));
  const home = join(base, "home");
  const finance = join(base, "finance-longer-name");
  mkdirSync(home);
  mkdirSync(finance);
  const registryPath = join(base, "projects.json");
  writeFileSync(registryPath, JSON.stringify({ projects: [
    { root: home, projectId: "home", environment: "local", path: "/" },
    { root: finance, projectId: "finance", environment: "development", path: "/finance" }
  ] }));

  assert.throws(
    () => resolveProjectContext({ roots: [home, finance], registryPath }),
    /different Infisical scopes/
  );
});

test("rejects a multi-root session containing an unregistered workspace", () => {
  const base = mkdtempSync(join(tmpdir(), "local-dev-env-"));
  const registered = join(base, "registered");
  const unregistered = join(base, "unregistered");
  mkdirSync(registered);
  mkdirSync(unregistered);
  const registryPath = join(base, "projects.json");
  writeFileSync(registryPath, JSON.stringify({ projects: [
    { root: registered, projectId: "home", environment: "local", path: "/" }
  ] }));

  assert.throws(
    () => resolveProjectContext({ roots: [registered, unregistered], registryPath }),
    /every workspace root/
  );
});
