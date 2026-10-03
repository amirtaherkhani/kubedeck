# Changelog

## [Unreleased]

### Added

- Added explicit multi-project bootstrap through `INFISICAL_LOCAL_DEV_PROJECT_IDS` for the central local development agent identity.
- Added a pinned Grafana image renderer deployment with Prometheus scraping support.

### Changed

- Extended the Keychain-held Universal Auth client credential to 365 days while keeping one-hour renewable access tokens.
- Consolidated local AI environment access on one project-aware `local-dev-env-mcp` connector and removed per-project profile selection from the agent surface.
- Configured the global MCP clients to use the macOS system CA store so `*.local.dev` HTTPS is trusted by Node.
- Applied the same system-CA setting to bootstrap, doctor, and live smoke verification commands.
- Updated pgAdmin's configured PostgreSQL application user to `app`.

### Removed

- Removed the legacy `infisical-agent-access.sh` and Finance-specific agent helper workflows.
- Removed obsolete PostgreSQL legacy credential entries from the generated Secret.

## [1.0.41] - 2026-10-03

### Fixed

- Aligned both provisioned k6 Prometheus dashboards with the `testrun_name`
  label emitted by the k6 operator, restoring Test Run filtering and panels.

## [1.0.40] - 2026-09-24

### Changed

- Added a guarded local-DNS recovery rule for canonical ingress address `192.168.1.100`, with explicit reporting when the live address differs.

### Fixed

- Enabled the k6 2.x runner control API so the k6 operator can start paused TestRuns through port `6565`.
