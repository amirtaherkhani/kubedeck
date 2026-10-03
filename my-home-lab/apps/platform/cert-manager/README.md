# cert-manager

Cluster certificate controller and CRDs.

| Field | Value |
|---|---|
| Namespace | `platform-system` |
| UI | No |
| HTTPS/domain | Not applicable; issues `local-dev-tls` |
| Observability | Kubernetes controller metrics through the monitoring stack |

Rancher Desktop owns the ingress controller; cert-manager owns certificate
resources used by home-lab HTTPS routes.
