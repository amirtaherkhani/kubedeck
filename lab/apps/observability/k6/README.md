# k6 Operator

Kubernetes-based performance-test runner with Prometheus and Grafana output.

| Field | Value |
|---|---|
| Namespace | `observability-tests` |
| UI | Provisioned Grafana dashboards |
| Metrics | Prometheus remote-write and k6 ServiceMonitor |
| Deployment | Helm chart `grafana/k6-operator` version `4.5.0` |

Durable dashboards are provisioned into Grafana. Runner resources are
temporary and must be bounded and cleaned up after a test. The included
smoke TestRun makes one request to Grafana's internal health endpoint.

Both provisioned k6 dashboards filter Prometheus remote-write results by the
operator-provided `testrun_name` label. Select **Test Run** in Grafana after a
completed run; no InfluxDB exporter is required.

The Helm release pins the k6 Operator v1.5.0 controller image by digest. The
smoke TestRun pins the runner and initializer to the k6 2.0.0
multi-architecture image digest and the starter to the matching Operator v1.5.0
starter digest. When upgrading either component, update the digests together
and verify that each image includes the cluster's architecture.

Deploy the operator in `observability-tests`; Prometheus must have its remote
write receiver enabled. Apply `dashboard/` in `observability` for Grafana's
dashboard sidecar. Run `kubectl apply -f tests/smoke-test.yaml` only when a new
smoke test is wanted. With `cleanup: post`, the operator removes the completed
TestRun and its jobs; capture runner logs during the run or query Prometheus
metrics afterward to verify it.

## k6 2.x runner contract

The k6 operator starts runners paused and resumes them through the k6 REST API
on port `6565`. k6 2.x disables that API by default, so every TestRun using a
k6 2.x runner must set `K6_ADDRESS=0.0.0.0:6565` under `spec.runner.env`.
