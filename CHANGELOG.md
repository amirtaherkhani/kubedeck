# Changelog

All notable KubeDeck changes are documented in this file.

## [Unreleased]

### Changed

- Enable the first administrator browser setup flow in local development while
  retaining verified workspace identity checks for hosted setup.
- Add reduced-motion-aware entrance, hover, focus, and open/close transitions to
  the shared shadcn-style UI components.
- Replace the banner's continuous drift and scan with an interactive isometric
  interface, API, and data layer stack.
- Replace animated liquid grid and orbit backdrops with the supplied static
  purple gradient and grain texture.

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
