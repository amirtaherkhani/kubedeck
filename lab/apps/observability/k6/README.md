# k6 Operator

Kubernetes-based performance-test runner with Prometheus and Grafana output.

| Field | Value |
|---|---|
| Namespace | `observability-tests` |
| UI | Ephemeral live UI: `https://k6-live.local.dev` |
| Dashboard | `https://k6-dashboard.local.dev/` redirects to the canonical Grafana URL |
| Metrics | Prometheus remote-write and k6 ServiceMonitor |
| Approval | Load tests require explicit current-turn approval |

Durable dashboards are provisioned into Grafana. Runner resources are
temporary and must be bounded and cleaned up after an approved test.

Both provisioned k6 dashboards filter Prometheus remote-write results by the
operator-provided `testrun_name` label. Select **Test Run** in Grafana after a
completed run; no InfluxDB exporter is required.

## k6 2.x runner contract

The k6 operator starts runners paused and resumes them through the k6 REST API
on port `6565`. k6 2.x disables that API by default, so every TestRun using a
k6 2.x runner must set `K6_ADDRESS=0.0.0.0:6565` under `spec.runner.env`.
