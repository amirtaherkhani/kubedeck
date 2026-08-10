# Grafana

Primary home-lab observability UI.

| Field | Value |
|---|---|
| Namespace | `observability` |
| UI | Yes: `https://grafana.local.dev` |
| Port | HTTPS through Traefik; service `3000` |
| Storage | `local-path` PVC, 10 Gi |
| Datasources | Prometheus, Loki, Tempo, Alertmanager |
| Dashboards | Kubernetes, macOS, platform, k6, and service dashboards |

Dashboard ConfigMaps are discovered across namespaces using the Grafana sidecar.
Stable datasource UIDs are `prometheus`, `loki`, `tempo`, and `alertmanager`.

The local Grafana admin credential is stored in Infisical/Kubernetes Secret and
must never be committed to this repository.
