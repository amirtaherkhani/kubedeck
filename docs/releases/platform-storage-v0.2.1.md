# platform-storage v0.2.1

## Fixed

The Helm workflow now resolves the active Infisical project slug and environment before rendering platform-storage. This prevents an upgrade from replacing the live scope with the chart's generic defaults and interrupting Secret reconciliation.

## Configuration

The existing Keychain-backed Infisical scope helper is used. Operators can still provide `INFISICAL_PROJECT_SLUG` and `INFISICAL_ENV_SLUG` explicitly.
