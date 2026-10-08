# Grafana Alloy

Local telemetry pipeline for Kubernetes logs and application traces.

| Field | Value |
|---|---|
| Namespace | `observability` |
| UI | No public UI |
| Ports | OTLP gRPC `4317`, OTLP HTTP `4318` |
| Outputs | Loki logs and Tempo traces; own metrics scraped by Prometheus |
| Deployment | DaemonSet |

Alloy also scrapes Kubernetes pod logs and exposes a ServiceMonitor.
