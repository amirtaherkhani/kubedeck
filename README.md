# Local Home Lab

Source of truth for services running on the local Rancher Desktop Kubernetes
cluster.

## Structure

```text
core/                         cluster, DNS, TLS, host, Helm ownership
apps/platform/                storage, cert-manager, Infisical
apps/observability/           Grafana, Prometheus, Loki, Tempo, Alloy, k6
apps/dev/                     KubeDeck, KubeDeck Agent, n8n, Temporal
scripts/                      safe validation and operation helpers
```

## Conventions

- Kubernetes context: `rancher-desktop`.
- Local domains: `*.local.dev`.
- Web access: HTTPS through Traefik and `local-dev-tls`.
- Namespaces: `platform-system`, `platform-storage`, `platform-secrets`,
  `observability`, `observability-tests`, `vero-vault-finance-load-test`, and
  `development-tools`.
- Configuration source: Infisical for secret and non-secret environment values.
- Telemetry: Prometheus metrics, Alloy/Loki logs, Alloy/Tempo traces, and
  Grafana dashboards where the service supports them.

## Operations

```bash
make helm-inventory
make helm-status
make helm-lint
make helm-validate RELEASE=all
make helm-apply RELEASE=all
```

`helm-validate` is traffic-free dry-run validation. `helm-apply` changes the
cluster and must only be run after reviewing the rendered configuration.

Rancher Desktop owns Traefik, Traefik CRDs, CoreDNS, Flannel,
local-path-provisioner, metrics-server, ServiceLB, and K3s system controllers.
The repository observes these components and does not upgrade or remove their
Helm releases.
