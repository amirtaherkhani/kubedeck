# Home Lab Codex Rules

This repository manages the local Rancher Desktop Kubernetes home lab.

## Operating rules

1. Before the first deployment or cluster-changing action in a task, verify Rancher Desktop and the `rancher-desktop` context. Stop if the node is not `Ready` or the runtime is unhealthy.
2. Keep base cluster configuration in `/core`; keep deployable services in `/apps/<category>/<service>`.
3. Use smart namespaces by ownership: `platform-system`, `platform-storage`, `platform-secrets`, `observability`, `development`, `ai`, and `entertainment`.
4. Use `*.local.dev` for every local hostname. HTTP routes must redirect to HTTPS and use the shared `local-dev-tls` certificate unless a service has a documented exception.
5. Register every service hostname in the local DNS/CoreDNS workflow. Kubernetes-only names use service DNS; macOS clients use the local DNS/hosts integration documented in `/core/dns`.
6. Store all configurable application values in Infisical, including secrets and non-secrets such as image repositories/tags, domains, ports, resource settings, feature flags, and connection settings. Kubernetes workloads consume runtime values through the Infisical Secrets Operator; Helm deployment values are resolved from Infisical before rendering. Keep only structural chart/schema definitions in Git. Do not commit credentials, tokens, passwords, or environment-specific runtime values.
7. Enable observability for every supported service: Prometheus metrics, ServiceMonitor, Alloy/Loki logs, Alloy/Tempo traces, health probes, Grafana datasource configuration, dashboards, and alerts where supported.
8. Add Kubernetes recommended labels and annotations to generated resources. Pin chart versions and image tags; do not use `latest`.
9. Every service directory has a concise `README.md` covering purpose, ownership, namespace, source/image, domain, HTTPS, UI, ports, storage, dependencies, configuration, observability, deployment, and verification.
10. Use the repository-local `$docker-image-deploy` skill for custom images; keep one verified current version per local registry repository.
11. Personal/company applications remain in their own repositories. This repository provides orchestration references and deployment configuration only.
12. When a service or web UI requires an administrator account, use the standard local identity: username `admin`, first name `admin`, family name `admin`, and email `admin@local.dev`. The administrator password is always generated or supplied through Infisical and must never be committed to Git or embedded in Helm values.
13. Every Helm chart and service values file must declare explicit chart metadata and immutable image metadata: chart `version`, `appVersion`, `description`, `home`/`sources` where applicable, recommended Kubernetes labels/annotations, and a non-`latest` image `tag`. Image repository and tag overrides must also be represented in the Infisical configuration for the service.
14. Deploy services individually in dependency order. Before each release, verify its Infisical-backed Secret/ConfigMap objects exist and are reconciled by the Infisical Secrets Operator; after each release, verify rollout, probes, Service/Ingress, HTTPS, DNS, persistence, and supported metrics/logs/traces.

## Validation gates

- Render Helm charts with `helm lint` and `helm template`.
- Validate Kubernetes manifests with server-side dry-run when the cluster is available.
- Check rendered namespaces, labels, selectors, ServiceMonitors, Ingress TLS, DNS, PVCs, RBAC, and endpoints.
- Treat local render/dry-run as validation, not proof that a live application is healthy.
- Never run load tests or destructive operations without explicit current-turn approval.
- Commit and push only when this directory is a valid Git checkout with the intended remote.
