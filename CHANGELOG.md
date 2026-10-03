# Changelog

## [Unreleased]

### Fixed

- Aligned both provisioned k6 Prometheus dashboards with the `testrun_name`
  label emitted by the k6 operator, restoring Test Run filtering and panels.

## [1.0.40] - 2026-09-24

### Changed

- Added a guarded local-DNS recovery rule for canonical ingress address `192.168.1.100`, with explicit reporting when the live address differs.

### Fixed

- Enabled the k6 2.x runner control API so the k6 operator can start paused TestRuns through port `6565`.
