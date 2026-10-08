# Loki

Local log aggregation backend for Alloy.

| Field | Value |
|---|---|
| Namespace | `observability` |
| UI | No standalone UI; Grafana Explore |
| Port | HTTP `3100` |
| Storage | `standard` PVC, 10 Gi |
| Retention | 5 days for local development |

The PVC is retained if the StatefulSet scales down or is removed.
