# Changelog

All notable KubeDeck changes are documented in this file.

## [Unreleased]

### Added

- Render site-specific HTTPS certificates, Traefik fallback TLS, and Grafana/Infisical domain overlays from one validated profile. Support a different DNS zone and an existing external ClusterIssuer without editing application manifests.

### Fixed

- Include a cert-manager Certificate in the default TLSStore namespace so Traefik's fallback certificate references a Secret that cert-manager can issue.

### Changed

- Remove embedded `local.dev` hostnames from deployment values and static TLS manifests. Document certificate-first domain migration and application configuration rollouts.

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

[0.1.1]: https://github.com/amirtaherkhani/kubedeck/releases/tag/v0.1.1
[0.1.2]: https://github.com/amirtaherkhani/kubedeck/compare/v0.1.1...v0.1.2
[0.1.3]: https://github.com/amirtaherkhani/kubedeck/compare/v0.1.2...v0.1.3
[0.1.4]: https://github.com/amirtaherkhani/kubedeck/compare/v0.1.3...v0.1.4
[0.2.0]: https://github.com/amirtaherkhani/kubedeck/compare/v0.1.4...v0.2.0
[0.4.0]: https://github.com/amirtaherkhani/kubedeck/compare/v0.3.4...v0.4.0
[0.5.0]: https://github.com/amirtaherkhani/kubedeck/compare/v0.4.0...v0.5.0
[0.6.0]: https://github.com/amirtaherkhani/kubedeck/compare/v0.5.0...v0.6.0
[0.7.0]: https://github.com/amirtaherkhani/kubedeck/compare/v0.6.0...v0.7.0
[0.8.0]: https://github.com/amirtaherkhani/kubedeck/compare/v0.7.0...v0.8.0
[0.9.0]: https://github.com/amirtaherkhani/kubedeck/compare/v0.8.0...v0.9.0
