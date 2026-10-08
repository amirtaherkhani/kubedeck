# cert-manager

Cluster certificate controller and CRDs.

| Field | Value |
|---|---|
| Namespace | `platform-system` |
| UI | No |
| HTTPS/domain | Not applicable; issues `local-dev-tls` |
| Observability | Kubernetes controller metrics through the monitoring stack |

Traefik serves the HTTPS routes; cert-manager issues and renews their
`local-dev-tls` certificates.
