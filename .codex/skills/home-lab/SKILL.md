---
name: home-lab
description: Operate and refactor the local Rancher Desktop Kubernetes home lab with safe namespaces, local.dev HTTPS/DNS, Infisical configuration, and Grafana observability.
---

# Home Lab Skill

Use this workflow for every change in this repository.

1. Read `/AGENTS.md`, inspect `/core`, and check the current Rancher/Kubernetes context before any cluster-changing action.
2. Classify a service by purpose and place it under `/apps/dev`, `/apps/ai`, `/apps/entertainment`, `/apps/platform`, or `/apps/observability`.
3. Define a namespace, `*.local.dev` hostname, HTTPS/TLS route, ports, storage, dependencies, and Infisical paths before writing manifests. Treat Infisical as the source of truth for every configurable value, including image tags.
4. Prefer an upstream Helm chart and pinned image tags. For custom charts, keep only generated deployment artifacts in this repository; never vendor third-party source.
5. Add recommended Kubernetes metadata, probes, resource settings, ServiceMonitors, Grafana dashboards, and Alloy integrations when supported.
6. Validate with Helm lint/template and Kubernetes server-side dry-run. Use live Grafana as the source of truth for dashboard and datasource claims when its MCP is available.
7. Document the service in its local `README.md`; report unavailable integrations explicitly instead of implying they are enabled.
8. For custom local images, use `$docker-image-deploy` and keep only one verified complete image set, referenced by one tag, in the exact local registry repository.

## Application onboarding contract

- Inspect and reuse existing platform dependencies before adding PostgreSQL,
  Redis/Valkey, RabbitMQ, MinIO, NATS, storage, TLS, DNS, or observability.
  Do not create a second database when the platform database is compatible;
  use a dedicated schema or database on the existing service.
- Define the namespace, `*.local.dev` HTTPS route, ports, storage, dependency
  order, health endpoints, observability, and Infisical path before writing
  manifests. Put non-secret runtime configuration in Infisical too.
- A service README and `infisical.requirements.yaml` must document ownership,
  source/image, dependencies, runtime keys, admin bootstrap, deployment, and
  verification. Keep only structure and safe defaults in Git.

## First administrator and agent credentials

- Use the standard local identity: username `admin`, first name `admin`, family
  name `admin`, email `admin@local.dev`. Generate or supply its password only
  through Infisical.
- After the first rollout is Ready, use the vendor-supported idempotent admin
  bootstrap command or a one-shot Kubernetes host task with the app's runtime
  configuration and persistent volume. Remove the task after success and
  verify login/health without printing the password.
- A login in a separate Codex or chat session does not authenticate a server-
  side agent adapter. Adapter credentials must be visible in the server
  workload's own Infisical-backed environment; report the adapter unavailable
  until its credential is present and verified.

## macOS `local.dev` DNS self-healing

- Treat `home-lab-dns` and the `*.local.dev` ingress address as two different
  addresses. The `home-lab-dns` LoadBalancer is the DNS **server**; CoreDNS
  answers `*.local.dev` with the macOS ingress address (normally
  `192.168.1.100`). Never write that answer address to `/etc/resolver/local.dev`.
- Before diagnosing an application, Grafana, TLS, or ingress failure as a
  workload issue, compare the live `kube-system/home-lab-dns` LoadBalancer IP
  with `/etc/resolver/local.dev`, then query the DNS service directly. Desktop
  runtime restarts can change the LoadBalancer address while macOS retains the
  old resolver target.
- When the live DNS service is Ready and answers `grafana.local.dev` with the
  expected ingress address, run
  `./core/host/macos/configure-local-dev-resolver.sh`. It discovers and
  validates the current DNS LoadBalancer, updates only `/etc/resolver/local.dev`,
  flushes the macOS DNS cache, and reloads `mDNSResponder`. Use `--dry-run`
  or `make macos-dns-check` before a repair when diagnosing.
- The repair requires the user's macOS administrator authentication. Do not
  bypass the prompt, hard-code a stale ServiceLB address, alter `/etc/hosts`,
  or change CoreDNS/Traefik merely to compensate for a stale macOS resolver.
- Verify the operating-system resolver with
  `dscacheutil -q host -a name grafana.local.dev`; do not use a bare `dig` as
  proof of macOS scoped-resolver behavior. Verify the application separately
  over HTTPS.

## Delivery and recovery loop

- Validate with Helm lint/template and Kubernetes server-side dry-run. Check
  Infisical scope and `ReadyToSyncSecrets=True`, selectors, RBAC, Services,
  Ingress TLS, PVCs, and dependency readiness.
- Deliver from `agent/<description>`: build/test, push, merge by PR, push
  merged `main`, create the annotated semantic version/release when runtime
  or deployment configuration changed, then deploy the merged version. Remove
  merged local and remote branches.
- Deploy individually in dependency order through `helm-validate` and
  `helm-apply`/`scripts/platform-helm.sh`. If a release fails, inspect events,
  logs, probes, image pulls, dependencies, and Infisical reconciliation; apply
  the smallest safe fix and retry until rollout, health, and HTTPS are good.
  Stop only for a missing credential, required decision, unavailable external
  authority, or unsafe/destructive action.

## Data and image safety

- Keep Loki and Prometheus on the documented time-based retention policy
  (currently 5 days) and filter dashboards to currently Running pods. Do not
  manually delete old pod-hash streams or metric series as routine cleanup.
- For custom images use `$docker-image-deploy`, Rancher Desktop
  `nerdctl`/containerd, and immutable version tags. Delete and verify absence of
  the exact repository's complete existing image set before each build, then
  retain one verified image set referenced by one tag. Deploy its release with
  `HELM_AUTO_ROLLBACK=false` after the prior image is removed. Do not run load
  tests, production traffic, registry garbage collection, or cleanup outside
  that repository as deployment verification without current approval.
