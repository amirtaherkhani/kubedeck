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

Native Kubernetes auth is opt-in via `infisical.authMethod=kubernetesAuth`. Set
`infisical.kubernetesAuth.identityId` to the approved identity UUID and provide
`serviceAccountRef.name` and `.namespace` under the same key. The Operator requests
short-lived tokens; no Universal Auth credential reference is rendered in this
mode. The default remains `universalAuth`. Unknown modes or missing native fields
fail rendering. Prepare the service account, TokenRequest/TokenReview permissions
and Infisical auth configuration from the [activation plan](../../../../docs/infisical-activation-plan.md)
before applying. Template tests verify auth changes do not change either Grafana
Deployment; keep the existing managed Secret during cutover. The checked-in
homelab environment now matches the verified live `dev` scope.
