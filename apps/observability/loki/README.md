# Loki

Local log aggregation backend for Alloy.

| Field | Value |
|---|---|
| Namespace | `observability` |
| UI | No standalone UI; Grafana Explore |
| Port | HTTP `3100` |
| Storage | `local-path` PVC, 10 Gi |
| Retention | 7 days for local development |
