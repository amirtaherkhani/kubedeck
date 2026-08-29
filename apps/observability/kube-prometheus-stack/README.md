# Prometheus Monitoring

Prometheus Operator, kube-state-metrics, node exporter, Alertmanager, and
Kubernetes recording/alert rules.

| Field | Value |
|---|---|
| Namespace | `observability` |
| UI | Internal services; Grafana is the primary UI |
| Ports | Prometheus `9090`, Alertmanager `9093` |
| Sources | Kubernetes, macOS exporter, ServiceMonitors, k6 |
| Retention | 5 days for local development |
| Dashboards | Provisioned into Grafana |

Rancher Desktop embedded K3s control-plane components are excluded where their
standalone metrics endpoints do not exist.
