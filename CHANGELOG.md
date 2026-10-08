# Changelog

All notable KubeDeck changes are documented in this file.

## [Unreleased]

### Added

- Restore the Grafana, Prometheus Stack, Loki, Tempo, Alloy, and k6 Operator deployment catalog for the Docker Desktop cluster, including the Grafana dashboards and Tempo smoke check.

### Fixed

- Allow the local Prometheus pod to scrape the Grafana image renderer's metrics through its ingress NetworkPolicy.
- Use the pinned Grafana Enterprise image for the local Grafana release after the Docker Hub mirror rejected the OSS image tag.
- Disable anonymous Grafana administrator access and resolve Grafana's Infisical scope from the active Docker Desktop project.

### Removed

- Remove the redundant k6 dashboard shortcut hostname; k6 dashboards remain available in Grafana.
- Retire versioned delivery roadmaps and the legacy Rancher migration plan; service changes are now handled from the current request and live state.
- Remove unused storage add-ons, Caretta, Radar, and KEDA, while retaining the KubeDeck dashboard and agent source for future use.

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
