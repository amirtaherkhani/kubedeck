# k6 Operator

Kubernetes-based performance-test runner with Prometheus and Grafana output.

| Field | Value |
|---|---|
| Namespace | `observability-tests` |
| UI | Ephemeral live UI: `https://k6-dashboard.local.dev` or `https://k6-live.local.dev` |
| Dashboard | Both hostnames open the active k6 web dashboard; Grafana is separate for durable history |
| Metrics | Prometheus remote-write and k6 ServiceMonitor |
| Approval | Load tests require explicit current-turn approval |

Durable dashboards are provisioned into Grafana. Runner resources are
temporary and must be bounded and cleaned up after an approved test.
