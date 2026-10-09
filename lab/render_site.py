#!/usr/bin/env python3
"""Render site-specific Kubernetes resources and Helm overlays from one profile."""

import argparse
import json
import re
from pathlib import Path


DNS_LABEL = re.compile(r"^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$")


def dns_name(value, field, *, require_dot=False):
    if not isinstance(value, str) or len(value) > 253 or not value:
        raise ValueError(f"{field} must be a nonempty DNS name")
    labels = value.split(".")
    if (require_dot and len(labels) < 2) or any(not DNS_LABEL.fullmatch(label) for label in labels):
        raise ValueError(f"{field} must be a lowercase DNS name")
    return value


def render(profile, domain_override=None):
    domain = dns_name(domain_override or profile["domain"], "domain", require_dot=True)
    hosts = profile["hosts"]
    for service in ("grafana", "infisical"):
        dns_name(hosts[service], f"hosts.{service}")
        if "." in hosts[service]:
            raise ValueError(f"hosts.{service} must be one label for wildcard TLS")
    if hosts["grafana"] == hosts["infisical"]:
        raise ValueError("Grafana and Infisical host labels must differ")
    tls = profile["tls"]
    issuer = tls["issuer"]
    if issuer["type"] not in ("private-ca", "existing"):
        raise ValueError("tls.issuer.type must be private-ca or existing")
    issuer_name = dns_name(issuer["name"], "tls.issuer.name")
    namespaces = tls["certificateNamespaces"]
    if not isinstance(namespaces, list) or not namespaces or len(set(namespaces)) != len(namespaces):
        raise ValueError("certificateNamespaces must be a nonempty list without duplicates")
    for namespace in namespaces:
        dns_name(namespace, "certificate namespace")
    default_ns = dns_name(tls["defaultTlsStoreNamespace"], "defaultTlsStoreNamespace")
    if default_ns not in namespaces:
        raise ValueError("default TLSStore namespace needs its own Certificate and Secret")
    secret_name = dns_name(domain.replace(".", "-") + "-tls", "generated TLS Secret name")
    grafana_host = f'{hosts["grafana"]}.{domain}'
    infisical_host = f'{hosts["infisical"]}.{domain}'
    resources = {}
    if issuer["type"] == "private-ca":
        self_signed_name = dns_name(issuer["selfSignedName"], "selfSignedName")
        ca_secret = dns_name(issuer["caSecretName"], "caSecretName")
        ca_namespace = dns_name(issuer["caNamespace"], "caNamespace")
        if ca_namespace not in namespaces:
            raise ValueError("CA namespace must be present in certificateNamespaces")
        if not issuer.get("caCommonName"):
            raise ValueError("caCommonName is required")
        resources["00-selfsigned-issuer.json"] = {
            "apiVersion": "cert-manager.io/v1", "kind": "ClusterIssuer",
            "metadata": {"name": self_signed_name}, "spec": {"selfSigned": {}},
        }
        resources["01-ca-certificate.json"] = {
            "apiVersion": "cert-manager.io/v1", "kind": "Certificate",
            "metadata": {"name": ca_secret, "namespace": ca_namespace},
            "spec": {
                "secretName": ca_secret,
                "issuerRef": {"name": self_signed_name, "kind": "ClusterIssuer"},
                "commonName": issuer["caCommonName"], "isCA": True,
                "duration": "87600h", "renewBefore": "720h",
                "privateKey": {"algorithm": "ECDSA", "size": 256},
                "usages": ["cert sign", "crl sign", "digital signature"],
            },
        }
        resources["02-ca-issuer.json"] = {
            "apiVersion": "cert-manager.io/v1", "kind": "ClusterIssuer",
            "metadata": {"name": issuer_name}, "spec": {"ca": {"secretName": ca_secret}},
        }
    for namespace in namespaces:
        certificate_spec = {
            "secretName": secret_name,
            "issuerRef": {"name": issuer_name, "kind": "ClusterIssuer"},
            "commonName": domain, "dnsNames": [domain, f"*.{domain}"],
            "privateKey": {"algorithm": "ECDSA", "size": 256, "rotationPolicy": "Always"},
        }
        if issuer["type"] == "private-ca":
            certificate_spec.update({"duration": "8760h", "renewBefore": "720h"})
        resources[f"10-certificate-{namespace}.json"] = {
            "apiVersion": "cert-manager.io/v1", "kind": "Certificate",
            "metadata": {"name": secret_name, "namespace": namespace},
            "spec": certificate_spec,
        }
    resources["20-default-tlsstore.json"] = {
        "apiVersion": "traefik.io/v1alpha1", "kind": "TLSStore",
        "metadata": {"name": "default", "namespace": default_ns},
        "spec": {"defaultCertificate": {"secretName": secret_name}},
    }
    overlays = {
        "grafana-values.json": {
            "ingress": {"hosts": [grafana_host], "tls": [{"secretName": secret_name, "hosts": [grafana_host]}]},
            "grafana.ini": {"server": {"domain": grafana_host, "root_url": f"https://{grafana_host}"}},
        },
        "infisical-values.json": {
            "ingress": {"hostName": infisical_host, "tls": [{"secretName": secret_name, "hosts": [infisical_host]}]},
        },
    }
    return resources, overlays, {"domain": domain, "secretName": secret_name,
                                  "certificateNamespaces": namespaces,
                                  "caSecretName": issuer.get("caSecretName") if issuer["type"] == "private-ca" else None,
                                  "caNamespace": issuer.get("caNamespace") if issuer["type"] == "private-ca" else None,
                                  "grafanaUrl": f"https://{grafana_host}",
                                  "infisicalUrl": f"https://{infisical_host}"}


def write_output(output, resources, overlays, summary):
    marker = output / ".kubedeck-site-render"
    if output.exists() and any(output.iterdir()) and not marker.exists():
        raise ValueError(f"{output} is not a site-render output directory")
    output.mkdir(parents=True, exist_ok=True)
    manifests = output / "manifests"
    manifests.mkdir(exist_ok=True)
    for old in manifests.glob("*.json"):
        old.unlink()
    for name, obj in resources.items():
        (manifests / name).write_text(json.dumps(obj, indent=2) + "\n")
    for name, obj in overlays.items():
        (output / name).write_text(json.dumps(obj, indent=2) + "\n")
    (output / "site-summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    marker.write_text("Generated by lab/render_site.py; do not edit.\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--profile", type=Path, default=Path(__file__).with_name("site.json"))
    parser.add_argument("--domain", help="override the profile's DNS domain")
    parser.add_argument("--output", type=Path, default=Path(__file__).with_name(".generated"))
    args = parser.parse_args()
    try:
        resources, overlays, summary = render(json.loads(args.profile.read_text()), args.domain)
    except (ValueError, KeyError, TypeError) as exc:
        parser.error(str(exc))
    try:
        write_output(args.output, resources, overlays, summary)
    except ValueError as exc:
        parser.error(str(exc))
    print(f"Rendered {summary['domain']} to {args.output}")


if __name__ == "__main__":
    main()
