# Infisical Secrets Operator

Synchronizes Infisical secret and configuration values into Kubernetes Secrets.

| Field | Value |
|---|---|
| Namespace | `platform-secrets` |
| UI | No |
| Dependency | Infisical backend |
| Observability | Controller health and metrics when supported |

The operator watches only `platform-secrets` and `observability` for this
stack. Grafana's `InfisicalSecret` creates the `grafana-admin` Secret; the
operator owns its contents.
