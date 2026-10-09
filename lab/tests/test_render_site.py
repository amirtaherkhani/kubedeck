import copy
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from render_site import render, write_output  # noqa: E402


PROFILE = json.loads((Path(__file__).resolve().parents[1] / "site.json").read_text())


class SiteRenderTests(unittest.TestCase):
    def test_default_profile_matches_existing_site_and_fixes_fallback_certificate(self):
        resources, overlays, summary = render(PROFILE)
        self.assertEqual(summary["secretName"], "local-dev-tls")
        self.assertEqual(overlays["grafana-values.json"]["grafana.ini"]["server"]["root_url"],
                         "https://grafana.local.dev")
        self.assertEqual(resources["20-default-tlsstore.json"]["spec"]["defaultCertificate"]["secretName"],
                         resources["10-certificate-kube-system.json"]["spec"]["secretName"])

    def test_new_domain_and_existing_issuer_have_no_old_domain_or_ca_resources(self):
        profile = copy.deepcopy(PROFILE)
        profile["tls"]["issuer"] = {"type": "existing", "name": "acme-dns01"}
        resources, overlays, summary = render(profile, "example.internal")
        self.assertEqual(summary["secretName"], "example-internal-tls")
        self.assertFalse(any("ca-issuer" in name or "selfsigned" in name for name in resources))
        self.assertNotIn("duration", resources["10-certificate-kube-system.json"]["spec"])
        self.assertNotIn("local.dev", json.dumps([resources, overlays, summary]))

    def test_rejects_missing_fallback_secret_namespace_and_multilevel_host(self):
        profile = copy.deepcopy(PROFILE)
        profile["tls"]["certificateNamespaces"].remove("kube-system")
        with self.assertRaisesRegex(ValueError, "default TLSStore namespace"):
            render(profile)
        profile = copy.deepcopy(PROFILE)
        profile["hosts"]["grafana"] = "sub.grafana"
        with self.assertRaisesRegex(ValueError, "one label"):
            render(profile)

    def test_rerender_removes_stale_issuer_resources(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "site"
            write_output(output, *render(PROFILE))
            profile = copy.deepcopy(PROFILE)
            profile["tls"]["issuer"] = {"type": "existing", "name": "acme-dns01"}
            write_output(output, *render(profile, "example.internal"))
            self.assertFalse((output / "manifests/00-selfsigned-issuer.json").exists())


if __name__ == "__main__":
    unittest.main()
