# Grafana

Primary home-lab observability UI.

| Field | Value |
|---|---|
| Namespace | `observability` |
| UI | Yes: generated `grafanaUrl` from `lab/site.json` |
| Port | HTTPS through Traefik; service `3000` |
| Storage | `standard` PVC, 10 Gi |
| Datasources | Prometheus, Loki, Tempo, Alertmanager |
| Dashboards | Kubernetes, macOS, platform, and service dashboards |
| Rendering | Remote image renderer `v5.12.2` for panel/dashboard exports |

Dashboard ConfigMaps are discovered across namespaces using the Grafana sidecar.
Stable datasource UIDs are `prometheus`, `loki`, `tempo`, and `alertmanager`.

The Platform / Observability dashboard limits its Kubernetes log pod filter to
pods currently reported by Prometheus as `Running`; Loki history is retained
for the configured backend retention period.

The local Grafana admin credential is stored in Infisical/Kubernetes Secret and
must never be committed to this repository. Supply the active Infisical project
and environment with `--set-string infisical.projectSlug=...` and
`--set-string infisical.envSlug=...` when deploying this chart.
