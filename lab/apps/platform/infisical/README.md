# Infisical

Secret backend used by the Grafana observability stack.

| Field | Value |
|---|---|
| Namespace | `platform-secrets` |
| UI | Yes: generated `infisicalUrl` from `lab/site.json` |
| Storage | Dedicated PostgreSQL and Redis in `platform-secrets` (Infisical exception to shared data services) |
| HTTPS | Traefik and the domain-derived cert-manager TLS Secret |
| Integration | Infisical Secrets Operator |

The active project and environment are configured at deployment time. Grafana
uses `/apps/observability/grafana` for its admin Secret. Credentials are
intentionally absent from the repository.

The runtime requires the existing `infisical-secrets`,
`infisical-postgresql`, and `infisical-universal-auth` Kubernetes Secrets.
Provision these through the established secret workflow before a fresh
installation; this repository does not store their values. The separate
PostgreSQL StatefulSet manifest and the Redis subchart reuse their PVCs.
When changing the site domain, update `SITE_URL` in `infisical-secrets` through
the existing secret workflow, then roll the backend so it reads the new URL.
