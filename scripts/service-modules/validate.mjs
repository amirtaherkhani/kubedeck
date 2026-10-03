import { readFile, readdir } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import Ajv from 'ajv';
import yaml from 'js-yaml';

const repoRoot = fileURLToPath(new URL('../../', import.meta.url));
const ajv = new Ajv({ allErrors: true });
const moduleSchema = JSON.parse(await readFile(
  new URL('../../contracts/service-module/v1alpha1/schema.json', import.meta.url),
  'utf8',
));
const profileSchema = JSON.parse(await readFile(
  new URL('../../contracts/installation-profile/v1alpha1/schema.json', import.meta.url),
  'utf8',
));
const checkModuleSchema = ajv.compile(moduleSchema);
const checkProfileSchema = ajv.compile(profileSchema);

function schemaErrors(validator) {
  return (validator.errors ?? []).map((error) =>
    `${error.dataPath || '/'} ${error.message}`,
  );
}

function repeatedIDs(items, field) {
  const seen = new Set();
  const errors = [];
  for (const item of items ?? []) {
    const id = item?.[field];
    if (!id) continue;
    if (seen.has(id)) errors.push(id);
    seen.add(id);
  }
  return errors;
}

function isRepositoryRelative(filePath) {
  if (typeof filePath !== 'string' || !filePath) return false;
  if (path.posix.isAbsolute(filePath) || path.win32.isAbsolute(filePath)) return false;
  if (filePath.includes('\\')) return false;
  return !filePath.split('/').some((segment) => segment === '..' || segment === '.');
}

export function validateServiceModule(value) {
  const errors = [];
  if (!checkModuleSchema(value)) errors.push(...schemaErrors(checkModuleSchema));
  for (const id of repeatedIDs(value?.components, 'id')) {
    errors.push(`duplicate component id: ${id}`);
  }
  return errors;
}

export function validateInstallationProfile(profile, modules) {
  const errors = [];
  if (!checkProfileSchema(profile)) errors.push(...schemaErrors(checkProfileSchema));

  const byID = new Map();
  for (const descriptor of modules ?? []) {
    const id = descriptor?.module?.id;
    if (!id) continue;
    if (byID.has(id)) errors.push(`duplicate module id: ${id}`);
    byID.set(id, descriptor);
  }

  for (const id of repeatedIDs(profile?.installations, 'moduleId')) {
    errors.push(`duplicate installation module id: ${id}`);
  }
  for (const installation of profile?.installations ?? []) {
    const descriptor = byID.get(installation?.moduleId);
    if (!descriptor) {
      errors.push(`unknown module: ${installation?.moduleId ?? '(missing)'}`);
    } else if (descriptor.module.ownership === 'external') {
      errors.push(`cannot install external module: ${installation.moduleId}`);
    }

    for (const filePath of [installation?.chart?.localPath, ...(installation?.valuesFiles ?? [])]) {
      if (filePath !== undefined && !isRepositoryRelative(filePath)) {
        errors.push(`path must be repository-relative: ${filePath}`);
      }
    }
  }
  return errors;
}

async function loadYaml(filePath) {
  return yaml.load(await readFile(filePath, 'utf8'), { filename: filePath });
}

async function examplePaths(directory) {
  const root = path.join(repoRoot, directory);
  return (await readdir(root))
    .filter((name) => name.endsWith('.example.yaml'))
    .sort()
    .map((name) => path.join(root, name));
}

async function runCLI(args) {
  const modulePaths = [];
  const profilePaths = [];
  if (args.length === 0) {
    modulePaths.push(...await examplePaths('examples/service-modules'));
    profilePaths.push(...await examplePaths('examples/installation-profiles'));
  } else {
    for (let index = 0; index < args.length; index += 2) {
      const flag = args[index];
      const filePath = args[index + 1];
      if (!filePath || (flag !== '--module' && flag !== '--profile')) {
        throw new Error('usage: validate.mjs [--module path]... [--profile path]...');
      }
      (flag === '--module' ? modulePaths : profilePaths).push(path.resolve(filePath));
    }
  }

  const modules = await Promise.all(modulePaths.map(loadYaml));
  const profiles = await Promise.all(profilePaths.map(loadYaml));
  const failures = [];
  for (const [index, module] of modules.entries()) {
    failures.push(...validateServiceModule(module).map((error) => `${modulePaths[index]}: ${error}`));
  }
  for (const [index, profile] of profiles.entries()) {
    failures.push(...validateInstallationProfile(profile, modules).map(
      (error) => `${profilePaths[index]}: ${error}`,
    ));
  }
  if (failures.length) {
    throw new Error(failures.join('\n'));
  }
  process.stdout.write(`validated ${modules.length} modules and ${profiles.length} profile${profiles.length === 1 ? '' : 's'}\n`);
}

if (process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url) {
  runCLI(process.argv.slice(2)).catch((error) => {
    process.stderr.write(`${error.message}\n`);
    process.exitCode = 1;
  });
}
