# Changelog

All notable KuchDesk changes are documented in this file.

## [Unreleased]

### Changed

- Select the Infisical host credential backend explicitly: project-local `.env` by default or an existing macOS Keychain item. Remove the global `host.env` setup command and automatic environment/file fallback; no credentials are provisioned or migrated.

### Added

- Add a read-only Infisical CLI project-access check that separates identity-scoped listing from project-detail authorization and never returns API error bodies.
- Add a scoped site-profile configuration CLI with snapshot, strict validation, field diff, dry-run, revision-confirmed apply, private rollback, and concurrent-write protection; live infrastructure still uses explicit deployment workflows.
- Add a project-local, Git-ignored `.env` credential reader shared by the Infisical CLI and MCP, with an explicit macOS Keychain backend and no automatic fallback.
- Gate local build and Helm apply on read-only Doctor checks for DNS, Docker/KIND, registry, disk, and node resource use; deployment profiles now require an explicit preflight domain and registry URL.
- Add a Doctor CLI and MCP diagnostic tool with structured, redacted checks; expose an MCP prompt, JSON schema, typed repair-plan validation, and fresh post-action verification through the existing AI harness.
- Add a local stdio Infisical MCP server with typed capability discovery and name-only secret listing. Its Universal Auth client renews short-lived access tokens in memory; no identity or live service is provisioned.
- Share Infisical name-listing logic between a noninteractive CLI and MCP; add bounded in-process start/status/cancel tools for concurrent MCP requests without a separate task service.
- Add bounded asynchronous scale/restart operations to the management-enabled Kubernetes agent, with per-workload serialization, status, deadlines, and cancellation.
- Add a host-side build/push/Helm/rollout CLI with an explicit profile, plan mode, revision check, image digest pin, and fail-fast behavior. It is not installed or run against the live cluster.
- Add an opt-in MCP deployment operation backed by the same local profile workflow and bounded in-process runner, with plan, start, status, and cancel tools.
- Document required MCP access to the proposed host-side Infisical service, unattended Universal Auth, Helm/Kubernetes Secret delivery, capability boundaries, and phased acceptance criteria. This is a plan; no new service or identity is deployed.
- Render site-specific HTTPS certificates, Traefik fallback TLS, and Grafana/Infisical domain overlays from one validated Go-powered profile. Support a different DNS zone and an existing external ClusterIssuer without editing application manifests.
- Scrape the existing macOS node_exporter and probe Technitium's site and recursive DNS plus Docker Desktop's host DNS path from Prometheus, with a Grafana Host and DNS dashboard.

### Fixed

- Stage a digest-pinned local image through KIND's containerd before registering it with CRI, avoiding a direct CRI manifest short-read while preserving the kubelet visibility gate.
- Resolve the informational Kubernetes capability catalog from one complete rules review, with exact access-review fallback when rules are incomplete or unavailable.
- Pre-pull digest-pinned KIND images through kubelet's CRI so `image.pullPolicy=Never` recognizes them during Helm rollouts.
- Bound parallel authorization reviews for the Kubernetes management capability catalog and make Helm installation notes describe the actual RBAC and CoreDNS values.
- Push locally built images through the macOS host's port-forwarded registry using `crane`, with optional digest pre-pull on dynamically discovered KIND nodes before Helm runs. A temporary Pod verified kubelet image pull and container creation with cached layers.
- Manage opt-in CoreDNS service aliases on Docker Desktop KIND through a bounded block in the existing Corefile, preserving unrelated directives and rejecting stale or unsafe updates.
- Let new Infisical PostgreSQL installations use the cluster's default StorageClass instead of assuming K3s `local-path`; preserve the existing StatefulSet and PVC during upgrades.
- Let the Kubernetes agent use a developer's kubeconfig when no in-cluster credentials exist, with an explicit context override and multi-file `KUBECONFIG` support.
- Handle Ingress resource backends without crashing the cluster snapshot or assigning their URLs to unnamed Services.
- Reject ambiguous private IPv4 addresses on the selected Mac interface instead of publishing an arbitrary DNS answer.
- Include a cert-manager Certificate in the default TLSStore namespace so Traefik's fallback certificate references a Secret that cert-manager can issue.
- Restore the macOS DNS reconciler with its explicit site zone, Kubernetes context, and physical LAN interface when a VPN owns the default route.

### Changed

- Require Go 1.25 for the host-agent module to use the official MCP Go SDK.
- Let operators configure the host DNS record TTL, run timeout, and macOS polling interval; pin the installed `kubectl` executable for launchd runs.
- Rename the repository, both agents, Helm chart, API identity, and macOS integration to KuchDesk. The agent chart is now 0.9.0 and its app version is 0.6.0.
- Remove embedded `local.dev` hostnames from deployment values and static TLS manifests. Document certificate-first domain migration and application configuration rollouts.

### Breaking Changes

- DNS management now targets `kube-system/coredns` key `Corefile`. Replace `KUCHDESK_COREDNS_CUSTOM_CONFIGMAP` and `KUCHDESK_COREDNS_OVERRIDE_KEY` with `KUCHDESK_COREDNS_CONFIGMAP` and `KUCHDESK_COREDNS_COREFILE_KEY`. Old Helm `dnsManagement.overrideKey` and `createConfigMap` values are rejected.
- The Kubernetes agent Go module, chart/release name, resource names, `KUCHDESK_*` environment variables, `kuchdesk.io/v1alpha1` snapshot identity, and `X-KuchDesk-Confirm` request header replace their former KubeDeck equivalents. Client integrations and existing agent installations must update together.
- The macOS LaunchAgent label and binary name change to `dev.kuchdesk.host-agent` and `kuchdesk-host-agent`. The installer does not remove an existing KubeDeck job.

### Migration

- Back up the existing Corefile, enable DNS writes only after a dry run, and remove old alias overrides through a separate reviewed change if present. Never reapply the changed Infisical PostgreSQL StatefulSet template over an existing installation; its volume claim template is immutable.
- Update imports, Helm paths, image references, bearer Secret names, and agent environment variables. Recreate or explicitly migrate the old chart release and Secret; Helm treats the renamed chart as a separate release. Preserve the existing CoreDNS override until the new agent and aliases are verified.
- Stop the old macOS LaunchAgent before installing the new one with the same DNS zone, context, and interface. Verify DNS resolution, then remove the old job and binary. Preserve retained PVCs when renaming local Kubernetes resources.

## [0.9.0] - 2026-10-09

### Changed

- Require a configured cluster ID and name for direct Kubernetes agent runs, matching the Helm chart's explicit identity requirement.
- Derive the agent's HTTP listener port from the chart Service target port; the chart version is 0.6.0 and the agent version is 0.3.0.
- Make the host DNS CLI usable on Linux with an OS-specific default-route adapter while keeping the macOS installer separate. The installer now requires an explicit DNS zone and stores absolute kubeconfig paths without Homebrew-specific PATH assumptions.
- Use cluster-domain-independent Service names for observability connections and configure Grafana's Infisical endpoint and credential location through chart values.

### Breaking Changes

- Direct Kubernetes agent invocations must set `KUBEDECK_CLUSTER_ID` and `KUBEDECK_CLUSTER_NAME`.
- Chart users must replace `agent.listenAddress` with `agent.listenHost` and `service.targetPort`.
- macOS host-agent installations must set `KUBEDECK_HOST_AGENT_ZONE` explicitly.

### Migration

- Preserve the previous cluster identity when upgrading an existing agent. Reapply the previous HTTP port through `service.targetPort` if it differed from 8080.
- Set the existing DNS zone explicitly before rerunning the macOS installer. The installer and this release do not change any running agent or DNS configuration automatically.

## [0.8.0] - 2026-10-09

### Added

- Configure the macOS host agent's DNS zone, Kubernetes context, Technitium resource names, and optional private target IP without editing source. The installer pins the selected context and preserves a custom kubeconfig.

### Changed

- Remove the `homelab` cluster identity from Kubernetes agent chart defaults. Each installation now declares its own cluster ID and name; the chart version is 0.5.0.
- Identify `lab/` as a local deployment profile rather than a requirement of either agent.

### Breaking Changes

- The host-agent CLI now requires `-zone` and `-kube-context`. The Kubernetes agent chart requires explicit `cluster.id` and `cluster.name` values.

### Migration

- Reinstall the macOS LaunchAgent with `host-agent/install-macos.sh` when ready to adopt the new binary and pinned context. Keep the existing DNS zone and context values. For an existing chart installation, pass its previous cluster ID and name during the next upgrade to preserve identity.

## [0.7.0] - 2026-10-08

### Added

- Restore the pinned k6 Operator, Grafana k6 dashboards, and a bounded one-iteration smoke TestRun in the existing `observability-tests` namespace.

### Changed

- Document the operator's Prometheus remote-write and Grafana dashboard dependencies alongside the deployment order.

## [0.6.0] - 2026-10-08

### Added

- Restore the Grafana and image-renderer chart, Prometheus Stack, Loki, Tempo, Alloy, and their Infisical, TLS, and ingress deployment configuration after these services were explicitly retained.

### Changed

- Keep Loki's existing PVC when its StatefulSet is scaled down or removed. Document the pinned observability deployment order and the required Infisical scope override.

## [0.5.0] - 2026-10-08

### Removed

- Remove the home-lab service catalog, deployment scripts, shared-service charts, local environment connector, and standalone contract/template documentation from the repository. The source tree now contains only the Kubernetes and macOS agents, their documentation, and the Kubernetes agent chart.
- Remove the chart's InfisicalSecret resource and implicit local-registry image reference.

### Changed

- Consolidate duplicate Kubernetes agent charts into `kubedeck-agent/chart` and keep catalog parser fixtures beside the agent tests.
- Require an explicit image repository and immutable tag when rendering the agent chart. The bearer Secret is provisioned separately in the target namespace.

### Migration

- Point Helm commands at `kubedeck-agent/chart`, provide `image.repository` and `image.tag`, and create `kubedeck-agent-auth` before deploying. Existing shared services are no longer managed from this repository.

## [0.4.0] - 2026-10-08

### Removed

- Remove the KubeDeck web UI, its API proxy, database, Node build chain, static assets, dashboard Helm charts, and combined UI/agent deployment scripts. The repository now keeps the Kubernetes and macOS agents without a dashboard.

### Changed

- Keep the cluster agent's API, auth Secret, service-module contract, and Helm charts independent of a dashboard. Operators who enable the agent NetworkPolicy must now select authorized client Pods explicitly.

### Migration

- Existing dashboard Helm releases should be uninstalled separately. The dashboard administrator Secret and UI PVC are not needed by either agent; retain or delete UI data according to the operator's data-retention decision.

## [0.3.4] - 2026-10-08

### Removed

- Remove unused D1 notes examples, archived release-note files, the empty Grafana dashboard placeholder, the completed NATS HA migration helper, and the optional k6 live-dashboard TestRun example.

## [0.3.3] - 2026-10-08

### Fixed

- Pin the k6 Operator controller image by digest so a fresh Helm deployment uses the same controller binary as the TestRun examples.

## [0.3.2] - 2026-10-08

### Fixed

- Pin the k6 smoke and live dashboard TestRun job images by digest so reruns use the same k6 and Operator starter versions.

## [0.3.1] - 2026-10-08

### Fixed

- Keep the Technitium admin page and local image registry available on fixed Mac localhost ports 5380 and 5001 across pod restarts and user logins.
- Reuse the persistent registry listener when preparing Docker Desktop images.

## [0.2.0] - 2026-10-05

### Added

- Add an opt-in, bearer-authenticated Kubernetes management API for discovery,
  bounded resource reads and writes, watches, pod logs, events, and workload
  scale, restart, and status actions.
- Add an authenticated dashboard management proxy and chart configuration.
  Resource writes require identity checks, version preconditions, and explicit
  confirmation; generic Secret access remains excluded.


## [0.1.4] - 2026-10-05

### Added

- Add an opt-in cluster-admin binding for the KubeDeck agent ServiceAccount,
  with chart guards requiring a bearer token Secret, RBAC creation, and a
  dedicated ServiceAccount.
- Document Docker Desktop KIND deployment constraints and the optional
  Metrics API dependency.


## [0.1.3] - 2026-10-05

### Changed

- Refresh the dashboard visual style and KubeDeck brand, including a static
  purple gradient with grain texture and a simplified logo mark.
- Align the dashboard package and both Helm charts at version `0.1.3`.

## [0.1.2] - 2026-10-03

### Added

- Add versioned ServiceModule and InstallationProfile schemas, examples, a
  validator, and shared Go and TypeScript catalog types.
- Enable first administrator setup in local development while retaining
  verified workspace identity checks for hosted setup.
- Add reduced-motion-aware transitions to shared UI components and an
  interactive dashboard banner with API and data layers.

### Changed

- Carry forward Infisical-backed dashboard and agent deployment configuration.
- Align the dashboard package and both Helm charts at version `0.1.2`.

## [0.1.1] - 2026-08-01

### Fixed

- Forward the in-cluster agent URL and bearer token into the Wrangler Worker
  runtime so authenticated dashboard snapshot, SSE, and DNS proxy routes can
  reach `kubedeck-agent`.
- Preserve empty service endpoint arrays in the agent contract and tolerate
  older snapshots that omitted them, so live catalog routes render reliably.

### Changed

- Keep the npm package, dashboard Helm chart, agent Helm chart, and Kubernetes
  client user-agent on version `0.1.1`.
- Add an authenticated post-deployment dashboard-to-agent snapshot gate to the
  Rancher Desktop release workflow.
- Document the verified immutable build and agent-first Helm deployment flow.

[0.1.1]: https://github.com/amirtaherkhani/kuchdesk/releases/tag/v0.1.1
[0.1.2]: https://github.com/amirtaherkhani/kuchdesk/compare/v0.1.1...v0.1.2
[0.1.3]: https://github.com/amirtaherkhani/kuchdesk/compare/v0.1.2...v0.1.3
[0.1.4]: https://github.com/amirtaherkhani/kuchdesk/compare/v0.1.3...v0.1.4
[0.2.0]: https://github.com/amirtaherkhani/kuchdesk/compare/v0.1.4...v0.2.0
[0.4.0]: https://github.com/amirtaherkhani/kuchdesk/compare/v0.3.4...v0.4.0
[0.5.0]: https://github.com/amirtaherkhani/kuchdesk/compare/v0.4.0...v0.5.0
[0.6.0]: https://github.com/amirtaherkhani/kuchdesk/compare/v0.5.0...v0.6.0
[0.7.0]: https://github.com/amirtaherkhani/kuchdesk/compare/v0.6.0...v0.7.0
[0.8.0]: https://github.com/amirtaherkhani/kuchdesk/compare/v0.7.0...v0.8.0
[0.9.0]: https://github.com/amirtaherkhani/kuchdesk/compare/v0.8.0...v0.9.0
