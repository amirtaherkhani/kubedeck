# Infisical integration plan

## Scope and current state

Build a reusable Infisical interface for KuchDesk on this Mac and its Docker Desktop KIND cluster. This document is a plan; it does not authorize a new identity, credentials, access grants, service deployment, or data migration. Keep the existing Infisical Secrets Operator path for Grafana while evaluating the new interface.

| Area | Verified state | Limit or next check |
| --- | --- | --- |
| Backend | `infisical/infisical:v0.151.0`, 1/1 Ready; HTTPS UI, `/api/status`, and `/api/docs/json` return 200. Status reports `Ok` and Redis configured. | Image tag establishes the deployed version; edition and licensed features were not verified. |
| Data | PostgreSQL and Redis StatefulSets are 1/1 Ready; PostgreSQL accepts local connections; both 8 GiB PVCs are Bound to `standard`. | The live PostgreSQL StatefulSet template still requests `local-path`. Its existing PVC works, but a fresh claim from that template is unsafe. The template is immutable; do not reapply or replace it without a data migration plan. |
| Access | The Secrets Operator is 1/1 Ready. `observability/grafana-admin` reports successful Universal Auth and sync of two keys. | This proves the existing Grafana scope, not organization or project administration. No admin identity or role was tested. |
| Configuration | The backend reads an existing Kubernetes Secret through `envFrom`; it has no additional mounted configuration volume. | `SITE_URL` and similar process settings require a backend rollout to reload. The host agent currently runs a one-shot Technitium DNS job through launchd and has no persistent Infisical API. |
| KIND-to-host route | Prometheus currently scrapes `host.docker.internal:9101` with `up=1`. | This proves that one cluster workload reaches the host exporter; it does not validate a future host API listener, authentication, or TLS. |
| Invitations | Status reports `inviteOnlySignup=true` and `emailConfigured=false`. | Whether a specific invitation workflow is usable is unverified. Do not change signup or mail settings as part of this plan. |
| Secret rotation | Grafana secret sync is Ready; the operator reports zero deployments selected for automatic redeploy. | A changed Kubernetes Secret may need a separate Grafana rollout before the running process reads it. Verify with a safe rotation test before changing rollout policy. |

No warning events were found in `platform-secrets`, and sampled recent backend/operator logs contained no error or timeout matches. These checks do not prove every authenticated operation or backup/restore path. The existing PostgreSQL manifest now omits a storage class for **new** installs; the live StatefulSet and its data remain unchanged.

## Proposed boundary

```text
Noninteractive CLI / AI client ─┐
                               ├─ KuchDesk host API ─ Infisical adapter ─ Infisical API
Kubernetes agent ───────────────┘                                  │
                                                PostgreSQL + Redis (Infisical-owned)

Existing Grafana ─ Infisical Secrets Operator ─ Infisical API
```

The host API owns authentication and policy. Clients discover typed capabilities and invoke named commands; they do not receive the Infisical client secret or access token. The Kubernetes agent uses a configurable, authenticated host address rather than a hard-coded Mac IP or domain. Confirm KIND-to-host routing and the selected TLS/auth transport before implementation. The current host agent exits after each DNS reconciliation; the preferred design is a separate long-lived host API process sharing Go packages with that job. This avoids turning DNS polling into a server lifecycle. The host is a single point of availability for *new* KuchDesk commands; the existing operator-to-Infisical sync remains independent. A thin MCP adapter may expose the same command schema later, but MCP is optional and must not become a required runtime dependency.

Use small Go interfaces for authentication, secret operations, project structure, and administration. Prefer the official [Go SDK](https://github.com/Infisical/go-sdk) where its released version covers a required operation; use a typed, version-pinned HTTP adapter only for gaps. Keep transport, authorization, and Infisical request mapping separate so the host API, CLI, and optional AI adapter share one behavior. Make endpoint, project, environment, path, timeouts, retry policy, and host address configurable. Reload non-secret configuration when safe; do not promise hot reload for settings that require process restart.

## Capability evidence and boundaries

The deployed [OpenAPI document](https://infisical.local.dev/api/docs/json) lists the paths below. A listed path is **not** proof that the current identity has permission, that the installed edition enables it, or that the Go SDK implements it. The official [Universal Auth description](https://infisical.com/blog/introducing-machine-identities) explains the client-ID/client-secret exchange and short-lived access token; organization and project roles are separate. Current [Go SDK source](https://github.com/Infisical/go-sdk) exposes authentication, secrets, and folders; broader administrative coverage must be checked against a pinned SDK release. Its current [secret types](https://github.com/Infisical/go-sdk/blob/main/secrets.go) name V3 requests while the deployed API also exposes V4, so the chosen client/API version needs contract tests.

| Capability | Deployed API evidence | Implementation status |
| --- | --- | --- |
| Universal Auth login | `POST /api/v1/auth/universal-auth/login` | Path confirmed; new identity, permissions, token lifetime, and retry behavior untested. |
| Secret read/write | `GET/POST/PATCH/DELETE /api/v4/secrets/{secretName}` and list/batch paths | Path confirmed; project/environment/path scope and write rights untested. |
| Folders | `GET/POST /api/v1/folders`, update and delete paths | Path confirmed; access scope untested. |
| Projects and environments | Project list/create and project environment paths | Path confirmed; management rights and edition untested. |
| Identities, roles, memberships | Identity, project role, project membership, and organization membership paths | Some paths confirmed; do not combine these into one assumed “admin” permission. |
| Human-user invitation | No matching invitation operation found in the deployed OpenAPI scan | Unsupported or undocumented for this version until separately verified. Do not invent an endpoint. |

## Phases and acceptance criteria

| Phase | Prerequisite and deliverable | Acceptance test | Approval boundary |
| --- | --- | --- | --- |
| 0. Capability matrix | Pin the deployed OpenAPI and candidate Go SDK version. Map each command to API method, edition, role, scope, and response shape. | Read-only schema checks and least-privilege permission probes distinguish 401, 403, unsupported, and working operations. | No new access for schema review; permission probes need an agreed test identity. |
| 1. API and configuration contract | Agree on Go interfaces, versioned command names, JSON input/output/error schemas, noninteractive CLI, host endpoint, TLS, timeouts, configuration source, and host-side credential storage. Include a machine-readable command catalog with capability and permission metadata. | Schema validation, invalid-config failures, command discovery, and a KIND-to-host connectivity test. | Review the contract before building the new module; no credential setup yet. |
| 2. Authentication lifecycle | After choosing a scoped identity, provision its bootstrap credential once and store it on the host (macOS Keychain is an option); keep access tokens in memory. Track expiry, re-login before expiry, single-flight login for concurrent callers, restart recovery, and revocation. Do not assume a refresh endpoint exists. | Simulate expiry, restart, concurrent calls, one bounded 401 re-login with retry only for safe/idempotent requests, 403 without retry, and revoked credentials. No secret appears in logs, prompts, or errors. | Creating an identity, assigning roles, and storing a client secret require separate explicit approval. |
| 3. Secret operations | Implement scoped metadata/list/read/create/update/delete with project, environment, and path as explicit inputs. Use bounded pagination and safe retry/idempotency rules. AI-facing responses are redacted by default; any value-use path stays within an authorized local process. | Contract tests against a disposable project, 404/409/429/5xx behavior, duplicate-write prevention, redaction, and audit visibility. | Test writes and any access to real secret values require exact scope approval. |
| 4. Project and administrative operations | Add project/environment/folder commands separately from identity, role, membership, and user administration. Return `unsupported` for missing/version-limited operations. | Per-command allow/deny tests with scoped identities; no privilege expansion from secret access to administration. | Each administrative role or identity grant and live write needs separate approval. |
| 5. Cluster integration | Let the Kubernetes agent call the authenticated host API through a configured KIND-reachable address; keep Infisical credentials host-side. Preserve the existing operator sync. | DNS/TLS/auth, host outage, timeout, retry, and recovery tests from KIND; no accidental public listener or token exposure. | Review host listener, network access, Kubernetes RBAC, and any live deployment change before applying it. |
| 6. Client usability and documentation | Publish command catalog, examples, structured JSON errors, and operator guidance. Optionally add a thin MCP adapter over the same contract. | An AI client discovers a command, invokes it noninteractively, receives clear permission and unsupported errors, and handles bounded retries without credential access. | No new mandatory service or MCP dependency without a separate decision. |

## Safety rules for implementation

- Do not put Infisical admin credentials in Git, Helm values, Pods, AI prompts, or logs. Redact values, tokens, URLs with credentials, and server error bodies before returning structured errors.
- Retry only safe reads and explicitly idempotent writes with a bounded backoff; never turn 403 into re-authentication or replay a non-idempotent admin write automatically.
- Keep secret CRUD and administrative commands in separate capability groups and roles. A machine identity with access to Grafana's secret path does not imply organization administration.
- Preserve the current PVCs and healthy releases. Treat the `local-path` StatefulSet template mismatch as a separate storage migration decision, not a reason to delete or recreate data during this integration.
