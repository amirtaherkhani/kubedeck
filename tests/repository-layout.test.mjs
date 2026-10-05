import assert from 'node:assert/strict';
import { existsSync } from 'node:fs';
import test from 'node:test';
import { resolve } from 'node:path';

const repositoryRoot = process.cwd();

test('homelab deployment sources live under lab/', () => {
  const requiredPaths = [
    'lab/AGENTS.md',
    'lab/apps/dev/kubedeck/values.yaml',
    'lab/apps/dev/kubedeck-agent/values.yaml',
    'lab/core/helm/releases.conf',
    'lab/core/helm/repositories.conf',
    'lab/scripts/platform-helm.sh',
  ];

  for (const path of requiredPaths) {
    assert.equal(existsSync(resolve(repositoryRoot, path)), true, `missing ${path}`);
  }
});

test('the previous repository directory name is not present', () => {
  assert.equal(existsSync(resolve(repositoryRoot, 'my-home-lab')), false);
});
