# Temporal

Workflow orchestration service for personal projects.

| Field | Value |
|---|---|
| Namespace | `development-tools` |
| UI | Yes: `https://temporal.local.dev` |
| Frontend | gRPC/TCP `7233` |
| Storage | PostgreSQL in `platform-storage` |
| HTTPS | Traefik, `local-dev-tls`, HTTP redirect |
| Observability | Prometheus ServiceMonitor, Alloy/Loki, Grafana |

Database credentials and non-secret runtime configuration are supplied through
Infisical-backed Kubernetes Secrets and ConfigMaps.
