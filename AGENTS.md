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
10. Use the repository-local `$docker-image-deploy` skill for custom images; keep one verified current version per local registry repository. Rancher Desktop `nerdctl`/containerd is the primary build path. Docker CLI is optional and must not block deployment when the image is verified in the local registry and Kubernetes runtime checks pass.
11. Personal/company applications remain in their own repositories. This repository provides orchestration references and deployment configuration only.
12. When a service or web UI requires an administrator account, use the standard local identity: username `admin`, first name `admin`, family name `admin`, and email `admin@local.dev`. The administrator password is always generated or supplied through Infisical and must never be committed to Git or embedded in Helm values.
13. Every Helm chart and service values file must declare explicit chart metadata and immutable image metadata: chart `version`, `appVersion`, `description`, `home`/`sources` where applicable, recommended Kubernetes labels/annotations, and a non-`latest` image `tag`. Image repository and tag overrides must also be represented in the Infisical configuration for the service.
14. Deploy services individually in dependency order. Before each release, verify its Infisical-backed Secret/ConfigMap objects exist and are reconciled by the Infisical Secrets Operator; after each release, verify rollout, probes, Service/Ingress, HTTPS, DNS, persistence, and supported metrics/logs/traces.

## Infisical ownership and external-project access

- `my-home-lab` administers the Infisical organization and the shared `home-lab` project: project creation, environment/folder layout, machine identities, roles, audit, and revocation. Every application deployed by this repository uses the `home-lab` project with its service-specific path; the source repository's GitHub location does not change that deployment ownership.
- Projects deployed outside this repository keep their own Infisical project and Universal Auth machine identity. Never reuse administrator, Finance, home-lab, or another project's credentials across those external deployments.
- Humans use named Infisical accounts. AI agents and automation use project-scoped machine identities with short-lived tokens. Do not give agents organization-admin access or give humans shared machine credentials.
- Use `viewer` for read-only agents, `member` for agents that must add/edit/delete project secrets, and `admin` only for a designated project owner.
- Use stable project scopes and uppercase service-prefixed keys. For example, `vero-finance` uses `/finance` and keys such as `FINANCE_DATABASE_URL`; other Vero applications do not reuse its credentials.
- Kubernetes uses an Infisical Secrets Operator `credentialsRef` belonging to the same project identity. Verify `ReadyToSyncSecrets=True`; an existing managed Secret is not proof of current authorization.
- External agents use `scripts/infisical-agent-access.sh`, Keychain-backed credentials, and `infisical run`. They must not read arbitrary Kubernetes Secrets, use the admin token, export secrets to files, or put tokens in prompts, logs, CI artifacts, or Git.
- Record project, environment/path, principal, role, reason, rotation/expiry plan, and verification for every access change. Revoke access when an agent, human, project, or task ends.

## Application deployment contract

- A service deployed by this repository uses the shared `home-lab` Infisical project under a service-specific path, even when its source code lives in an external GitHub repository. External projects deployed outside this repository keep their own Infisical project.
- Before adding a dependency, inspect the live platform Services and Helm inventory. Reuse compatible platform PostgreSQL, Redis/Valkey, RabbitMQ, MinIO, NATS, storage class, TLS, DNS, and observability components; do not provision a duplicate database when the platform database can provide a dedicated schema or database.
- A new app must have a service README, Infisical requirements file, immutable image metadata, namespace, `*.local.dev` HTTPS route, health probes, persistence decision, dependency list, and verification commands. Runtime configuration includes non-secrets as well as secrets.
- If an app needs a first administrator, automate the supported bootstrap after the first Ready rollout with the standard `admin` / `admin` / `admin@local.dev` identity. Generate or retrieve the password only through Infisical, use a one-shot task with the app's persistent volume when necessary, remove it after success, and verify the account/health state without reporting the password.
- If a server-side agent adapter reports missing credentials, configure the adapter environment in the server workload through Infisical. A separate Codex/chat `/login` does not authenticate that server. Keep the app healthy without claiming the adapter is ready until its own credential is present and verified.
- For deployment failures, continue the bounded diagnose-fix-verify loop until rollout, probes, dependencies, Infisical reconciliation, HTTPS, DNS, persistence, and supported observability are healthy. Stop only for a missing credential, required decision, unavailable authority, or unsafe/destructive action.
- Keep Loki and Prometheus time-based retention at 10 days and use dashboard filters for currently Running pods. Do not manually delete historical pod-hash streams or metric series as routine cleanup.

## Change, merge, version, and deploy workflow

15. Start every repository change on a dedicated branch named `agent/<description>`; do not make change commits directly on `main`.
16. Before merge, run the relevant validation gates and review the complete diff. Preserve unrelated work and do not merge a branch with failing or missing required checks.
17. Merge completed branches through the repository's pull-request policy, push the resulting `main`, and remove the merged local and remote branches. Keep the repository's automatic branch deletion after merge enabled.
18. After a successful merge, create and push an annotated semantic-version tag and a matching GitHub release when the change affects project behavior, infrastructure, deployment configuration, or repository workflow. Use a patch version for compatible maintenance or rule changes.
19. Deploy again only when the merged change affects rendered Kubernetes resources, Helm values, images, runtime configuration, or service behavior. For documentation-only or repository-rule changes, do not restart workloads; still run repository validation and report that deployment was not required.
20. For every deployment, verify the `rancher-desktop` context and `Ready` node first, then run Helm lint/template and server-side dry-run, verify Infisical reconciliation, deploy in dependency order, and confirm rollout, probes, Service/Ingress, HTTPS, DNS, persistence, and supported observability.
21. Deliver every successful change in this order: build and test on the dedicated branch; push the branch and merge it through GitHub; push the merged `main`; create and push the required annotated version from that merged commit; then build the versioned artifact and deploy it to Kubernetes when the change affects runtime resources. Never deploy source that has not passed validation and been pushed to merged `main`.

## Validation gates

- Render Helm charts with `helm lint` and `helm template`.
- Validate Kubernetes manifests with server-side dry-run when the cluster is available.
- Check rendered namespaces, labels, selectors, ServiceMonitors, Ingress TLS, DNS, PVCs, RBAC, and endpoints.
- Treat local render/dry-run as validation, not proof that a live application is healthy.
- Never run load tests or destructive operations without explicit current-turn approval.
- Commit and push only when this directory is a valid Git checkout with the intended remote.
