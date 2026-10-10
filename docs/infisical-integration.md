# Infisical integration

The [host control plane](host-control-plane.md) is the shared boundary for AI
clients and the Kubernetes agent. The host keeps the Universal Auth bootstrap
pair in its private project `.env` (or an explicitly selected Keychain item).
Neither Helm values nor the Kubernetes agent contain that pair.

## Implemented in source

- The host Infisical adapter targets the installed v0.151 API. It supports
  project list/get/create/update/delete; environment list/create/update/delete;
  folder list/create/update/delete; secret name list, metadata get,
  create/update/delete; project role list/create/update/delete; and machine
  identity membership list/add/update/delete. Secret metadata reads never
  return `secretValue`. Secret values can be supplied to create/update over
  authenticated TLS, but are never echoed. Writes require exact
  `operation:target` confirmation and are not replayed after HTTP 401.
  Role writes currently accept only simple subject/action rules; conditional
  policies need a separate typed contract and tests before use.
- `kuchdesk-host-bridge` is a separate HTTPS process, disabled by default.
  It requires an explicit project allowlist, TLS certificate/key, and a
  dedicated bridge bearer from the private host `.env`. Project creation is
  additionally opt-in. It exposes typed Infisical commands and Doctor checks.
- The Kubernetes agent can proxy the typed commands and Doctor through this
  HTTPS bridge. Its own existing API bearer protects both routes. The agent
  receives only a dedicated bridge bearer via `secretKeyRef` and an optional
  CA ConfigMap. The Helm chart renders no Infisical bootstrap credentials.
- MCP `-scope all -manage -project-ids ID` can expose the same typed commands
  to an AI harness. The live `doctor-repair` registration remains
  credential-free and does not expose Infisical management.

The command result intentionally contains metadata and write receipts, not
secret values. This supports secret value creation and rotation while keeping
AI-facing responses redacted. A future value-delivery operation must name a
trusted destination rather than return plaintext to an agent.

## Verified local runtime (2026-10-10)

The host bridge is running with HTTPS on this Mac's LAN address and an explicit
allowlist containing only the `home-lab-nb0-k` project. The cluster agent is
running a digest-pinned image and reaches the bridge with a dedicated bearer
Secret and a scoped CA ConfigMap. The `kuchdesk-doctor` MCP registration exposes
only the `doctor-repair` scope; deployment apply remains separately gated.
The Infisical Operator also watches `development-tools`. An opt-in probe
verified Infisical secret delivery through the Operator, a Kubernetes Secret,
and a Helm workload; the probe resources and Infisical test secret were then
removed. None of these runtime steps expands the bridge's project allowlist.

The host machine identity is Project Admin in `home-lab-nb0-k`. Its
organization-level Admin role is separate from project membership. Do not
assume that role grants access to every current or future project. See
[project onboarding](infisical-project-onboarding.md) for the verified
boundary and the manual bootstrap needed for other projects.
