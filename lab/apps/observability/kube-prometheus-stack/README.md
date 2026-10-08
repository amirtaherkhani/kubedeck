# Prometheus Monitoring

Prometheus Operator, kube-state-metrics, node exporter, Alertmanager, and
Kubernetes recording/alert rules.

| Field | Value |
|---|---|
| Namespace | `observability` |
| UI | Internal services; Grafana is the primary UI |
| Ports | Prometheus `9090`, Alertmanager `9093` |
| Sources | Kubernetes and observability ServiceMonitors |
| Retention | 5 days for local development |
| Dashboards | Provisioned into Grafana |

Docker Desktop control-plane endpoints are scraped only when available.
