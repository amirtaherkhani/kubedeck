# Infisical Secrets Operator

Synchronizes Infisical secret and configuration values into Kubernetes Secrets.

| Field | Value |
|---|---|
| Namespace | `platform-secrets` |
| UI | No |
| Dependency | Infisical backend |
| Observability | Controller health and metrics when supported |

The operator watches `platform-secrets`, `observability`, and
`development-tools` for this stack. Grafana's `InfisicalSecret` creates the
`grafana-admin` Secret; the operator owns its contents. The
`development-tools` scope supports a separately enabled, value-free Helm
integration probe. The operator reads its existing Universal Auth Secret in
`platform-secrets`; the credentials are not copied into an application chart.
