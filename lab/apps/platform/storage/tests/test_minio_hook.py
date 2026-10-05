#!/usr/bin/env python3
"""Check the one-time MinIO bootstrap hook's Helm lifecycle."""

import re
import subprocess
import unittest
from pathlib import Path


CHART = Path(__file__).resolve().parents[1]


def rendered_jobs(*flags):
    manifest = subprocess.check_output(
        ['helm', 'template', 'platform-storage', str(CHART), '--namespace', 'platform-storage', *flags],
        text=True,
    )
    return len(re.findall(r'^kind: Job$', manifest, re.MULTILINE))


class MinioHookLifecycleTest(unittest.TestCase):
    def test_install_runs_bootstrap(self):
        self.assertEqual(rendered_jobs(), 1)

    def test_routine_upgrade_skips_bootstrap(self):
        self.assertEqual(rendered_jobs('--is-upgrade'), 0)

    def test_upgrade_can_opt_in(self):
        self.assertEqual(rendered_jobs('--is-upgrade', '--set', 'minio.initOnUpgrade=true'), 1)


if __name__ == '__main__':
    unittest.main()
