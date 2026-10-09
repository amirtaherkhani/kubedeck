# cert-manager

Cluster certificate controller and CRDs.

| Field | Value |
|---|---|
| Namespace | `platform-system` |
| UI | No |
| HTTPS/domain | Not applicable; issues certificates rendered from `lab/site.json` |
| Observability | Kubernetes controller metrics through the monitoring stack |

Traefik serves the HTTPS routes; cert-manager issues and renews the
domain-derived TLS certificates in each configured namespace, including the
namespace containing Traefik's default TLSStore.
