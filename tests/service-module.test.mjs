import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import { spawnSync } from 'node:child_process';
import yaml from 'js-yaml';

import {
  validateInstallationProfile,
  validateServiceModule,
} from '../scripts/service-modules/validate.mjs';

const managedText = await readFile(
  new URL('../examples/service-modules/managed.example.yaml', import.meta.url),
  'utf8',
);
const externalText = await readFile(
  new URL('../examples/service-modules/external.example.yaml', import.meta.url),
  'utf8',
);
const profileText = await readFile(
  new URL('../examples/installation-profiles/local-single-node.example.yaml', import.meta.url),
  'utf8',
);

const managed = yaml.load(managedText);
const external = yaml.load(externalText);
const profile = yaml.load(profileText);

test('validates the versioned managed and external module examples', () => {
  assert.deepEqual(validateServiceModule(managed), []);
  assert.deepEqual(validateServiceModule(external), []);
});

test('rejects duplicate component IDs', () => {
  const duplicate = structuredClone(managed);
  duplicate.components.push(structuredClone(duplicate.components[0]));

  assert.ok(validateServiceModule(duplicate).some((error) => error.includes('duplicate component id')));
});

test('installation profile accepts managed module refs and relative Helm values paths', () => {
  assert.deepEqual(validateInstallationProfile(profile, [managed, external]), []);
});

test('installation profile rejects external modules and machine-specific paths', () => {
  const invalidProfile = structuredClone(profile);
  invalidProfile.installations[0].moduleId = external.module.id;
  invalidProfile.installations[0].valuesFiles = ['/Users/someone/private/values.yaml'];

  const errors = validateInstallationProfile(invalidProfile, [managed, external]);
  assert.ok(errors.some((error) => error.includes('external module')));
  assert.ok(errors.some((error) => error.includes('must be repository-relative')));
});

test('rejects raw secret fields while accepting Infisical references', () => {
  const invalidModule = structuredClone(managed);
  invalidModule.components[0].secretRefs[0].password = 'must-never-be-stored';

  assert.ok(validateServiceModule(invalidModule).length > 0);
});

test('rejects an installation with an unknown module ID', () => {
  const invalidProfile = structuredClone(profile);
  invalidProfile.installations[0].moduleId = 'missing-module';

  assert.ok(validateInstallationProfile(invalidProfile, [managed, external]).some(
    (error) => error.includes('unknown module'),
  ));
});

test('catalog validation command checks the example set', () => {
  const result = spawnSync(process.execPath, ['scripts/service-modules/validate.mjs'], {
    cwd: new URL('..', import.meta.url),
    encoding: 'utf8',
  });

  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /validated 2 modules and 1 profile/);
});
