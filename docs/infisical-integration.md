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
  to an AI harness. Existing `doctor-repair` remains credential-free and does
  not expose Infisical management. Neither scope is registered live.

The command result intentionally contains metadata and write receipts, not
secret values. This supports secret value creation and rotation while keeping
AI-facing responses redacted. A future value-delivery operation must name a
trusted destination rather than return plaintext to an agent.

## Current runtime boundary

The installed cluster agent still uses its earlier image. The host bridge is
not installed or listening, no bridge token/CA Secret has been created, and
no MCP registration or Helm rollout has occurred. The host machine identity's
read-only project check for `home-lab-nb0-k` returned `forbidden`; the
Infisical Operator's existing Grafana sync is independent. Its current
namespace scope excludes `development-tools`.

Enabling the bridge needs a reviewed project role, host listener address and
certificate, a private host bridge token, a matching dedicated Kubernetes
Secret and trusted CA, a tested cluster-to-host route, and an explicit Helm
rollout. The local Project Admin role grants secret-value access and project
administration; an organization role for project creation is separate. No
identity grant or live write is part of this source change. Chart secret
consumption by applications remains through the existing Infisical Operator;
changing its watched namespaces is a separate live operation.
