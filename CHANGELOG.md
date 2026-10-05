# Changelog

All notable KubeDeck changes are documented in this file.

## [Unreleased]

## [0.1.4] - 2026-10-05

### Added

- Add an opt-in cluster-admin binding for the KubeDeck agent ServiceAccount,
  with chart guards requiring a bearer token Secret and RBAC creation.
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
