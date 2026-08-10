# Radar

Kubernetes observability and activity dashboard.

| Field | Value |
|---|---|
| Namespace | `observability` |
| UI | Yes: `https://radar.local.dev` |
| Port | HTTP `9280` |
| Dependencies | Caretta and Prometheus |
| Storage | `local-path` PVC, 1 Gi |
| Access | Read-only Kubernetes API capabilities by default |
