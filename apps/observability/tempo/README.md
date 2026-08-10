# Tempo

Local distributed tracing backend for Alloy and application OpenTelemetry.

| Field | Value |
|---|---|
| Namespace | `observability` |
| UI | No standalone UI; Grafana Explore |
| Ports | HTTP `3200`, OTLP gRPC `4317` |
| Storage | `local-path` PVC, 10 Gi |
| Metrics | Span metrics and service graphs remote-written to Prometheus |
