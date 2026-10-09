# Infisical integration plan

## Scope and current state

Build a reusable Infisical interface for KuchDesk on this Mac and its Docker Desktop KIND cluster. The first local MCP/authentication slice exists in source; it has not been deployed or tested with a real identity. No new identity, credentials, access grants, or data migration have been performed. Keep the existing Infisical Secrets Operator path for Grafana.

| Implemented in source | Still required |
| --- | --- |
| Stdio MCP discovery and bounded in-memory start/status/cancel for name-only listing; a CLI using the same typed service; two-variable Universal Auth bootstrap; in-memory access-token renewal, restart re-login, and bounded read retry. | Verify credential lifetime and live permissions; implement scoped secret writes, administration, host API and cluster-agent bridge, operator-backed chart delivery, Helm execution, and live MCP acceptance tests. |

| Area | Verified state | Limit or next check |
| --- | --- | --- |
| Backend | `infisical/infisical:v0.151.0`, 1/1 Ready; HTTPS UI, `/api/status`, and `/api/docs/json` return 200. Status reports `Ok` and Redis configured. | Image tag establishes the deployed version; edition and licensed features were not verified. |
| Data | PostgreSQL and Redis StatefulSets are 1/1 Ready; PostgreSQL accepts local connections; both 8 GiB PVCs are Bound to `standard`. | The live PostgreSQL StatefulSet template still requests `local-path`. Its existing PVC works, but a fresh claim from that template is unsafe. The template is immutable; do not reapply or replace it without a data migration plan. |
| Access | The Secrets Operator is 1/1 Ready. `observability/grafana-admin` reports successful Universal Auth and sync of two keys. | This proves the existing Grafana scope, not organization or project administration. No admin identity or role was tested. |
| Configuration | The backend reads an existing Kubernetes Secret through `envFrom`; it has no additional mounted configuration volume. | `SITE_URL` and similar process settings require a backend rollout to reload. The host agent currently runs a one-shot Technitium DNS job through launchd and has no persistent Infisical API. |
| KIND-to-host route | Prometheus currently scrapes `host.docker.internal:9101` with `up=1`. | This proves that one cluster workload reaches the host exporter; it does not validate a future host API listener, authentication, or TLS. |
| Invitations | Status reports `inviteOnlySignup=true` and `emailConfigured=false`. | Whether a specific invitation workflow is usable is unverified. Do not change signup or mail settings as part of this plan. |
| Secret rotation | Grafana secret sync is Ready; the operator reports zero deployments selected for automatic redeploy. | A changed Kubernetes Secret may need a separate Grafana rollout before the running process reads it. Verify with a safe rotation test before changing rollout policy. |
| Operator scope | The deployed `infisical/kubernetes-operator:v0.11.11` serves `InfisicalSecret` `v1alpha1` and watches only `platform-secrets` and `observability`. | Apps in another namespace need explicit namespace enrollment and RBAC before this operator can sync their Secrets. |

No warning events were found in `platform-secrets`, and sampled recent backend/operator logs contained no error or timeout matches. These checks do not prove every authenticated operation or backup/restore path. The existing PostgreSQL manifest now omits a storage class for **new** installs; the live StatefulSet and its data remain unchanged.

## Proposed boundary

```text
AI harness ─ MCP server (required) ─┐
Noninteractive CLI ─────────────────┼─ KuchDesk host commands ─ Infisical adapter ─ Infisical API
Kubernetes agent ─ authenticated API┘                                       │
                                                     PostgreSQL + Redis (Infisical-owned)

Chart deployer ─ Helm/Kubernetes API ─ chart with Secret references ─ application Pod
Infisical Secrets Operator ─ Kubernetes Secret ──────────────────────┘

Existing Grafana ─ Infisical Secrets Operator ─ Infisical API
```

The host API owns authentication and policy. The MCP server is a **required** client of the same versioned command service as the noninteractive CLI and Kubernetes agent. The AI harness must discover [MCP tools](https://modelcontextprotocol.io/specification/draft/server/index) with typed input/output schemas and invoke them without seeing Infisical credentials or secret values. A local stdio MCP transport is the first target; any network MCP transport needs its own access design. The Kubernetes agent uses a configurable, authenticated host address rather than a hard-coded Mac IP or domain. Confirm KIND-to-host routing and the selected TLS/auth transport before implementation. The current host agent exits after each DNS reconciliation; the preferred design is a separate long-lived host API process sharing Go packages with that job. This avoids turning DNS polling into a server lifecycle. The host is a single point of availability for *new* KuchDesk commands; existing operator-to-Infisical secret sync remains independent.

Use small Go interfaces for authentication, secret operations, project structure, administration, and chart deployment. Prefer the official [Go SDK](https://github.com/Infisical/go-sdk) where its released version covers a required operation; use a typed, version-pinned HTTP adapter only for gaps. Keep transport, authorization, Infisical request mapping, and Helm execution separate so the host API, CLI, and MCP server share one behavior. Make endpoint, project, environment, path, timeouts, retry policy, and host address configurable. Reload non-secret configuration when safe; do not promise hot reload for settings that require process restart.

## Continuous access decision

The operational requirement is **one host bootstrap credential configured once**, followed by unattended access for months or a year. A literal bearer token with that lifetime is not yet verified. A Universal Auth machine identity uses a client ID and client secret to issue access tokens; the host can obtain a new token automatically when the current token expires or after a restart. Supply the pair to the host agent as exactly two environment variables, `INFISICAL_CLIENT_ID` and `INFISICAL_CLIENT_SECRET`; do not add a Keychain or credential-store dependency. Configure them outside Git and keep the secret out of command output and logs. The existing Kubernetes operator already has its own credential and continues to use it; sharing a broad host-admin identity with every Pod would defeat scope isolation. [Infisical recommends identities over legacy Service Tokens](https://infisical.com/blog/introducing-machine-identities).

| Option | Evidence in this installation | Decision |
| --- | --- | --- |
| Legacy Service Token | The deployed OpenAPI exposes `GET /api/v2/service-token`, but no creation operation was found in the scanned public schema. Infisical describes Service Tokens as older, project/path-scoped credentials. | Do not base new host, admin, and Helm operations on this legacy token model. Keep existing tokens untouched. |
| Universal Auth machine identity | The deployed API exposes client-secret creation/revocation and `POST /api/v1/auth/universal-auth/login`. Its client-secret request has a configurable `ttl` field (schema default `0`), and the auth configuration has access-token TTL fields (schema defaults 2,592,000 seconds). | **Recommended**, subject to verification of actual client-secret expiry semantics, edition, identity roles, and accepted policy. Bootstrap once; re-login automatically using the host credential. |
| One literal bearer token for a year | No year-long bearer lifetime was verified. The login response supplies `expiresIn`; this and server policy, not a hard-coded duration, govern renewal. | Do not promise or require it. A stable client credential plus automatically reissued access tokens meets the operational goal. |

Before provisioning, confirm whether this deployed version treats client-secret `ttl=0` as non-expiring and whether a finite TTL of at least 365 days is allowed. The OpenAPI default alone is not proof of either behavior. Choose an explicit policy, verify its expiry metadata, and alert well before any configured bootstrap expiry. A finite one-year credential still needs planned rotation before year end; uninterrupted access cannot be guaranteed after revocation, role removal, lockout, or Infisical outage. The auth adapter must coalesce concurrent logins, renew from the returned `expiresIn`, retry one safe request after a 401, stop on 403, and fail closed on revocation. Avoid repeated invalid logins because this installation's auth schema enables lockout.

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

## Helm and Kubernetes Secret delivery

A Helm chart does not need a secret value at template time. It should declare the Infisical project/environment/path and the **name and keys** of a Kubernetes Secret, while the existing Secrets Operator reconciles the values into that Secret. The Pod then uses `secretKeyRef`, `envFrom`, or a Secret volume. This is the pattern already used by Grafana's `InfisicalSecret` chart template and `grafana-admin` Secret. It gives a Helm-deployed app its environment values at runtime without putting them into Git, `--set`, rendered manifests, or Helm release history. [Kubernetes documents Secret-backed environment variables](https://kubernetes.io/docs/tasks/inject-data-application/distribute-credentials-secure/); [Helm stores supplied values in release metadata](https://docs.helm.sh/docs/intro/using_helm/).

For each new app: enroll its namespace in the operator's watched/RBAC scope, create a path-scoped `InfisicalSecret`, wait for `ReadyToSyncSecrets=True` and the named Secret, then install or upgrade the chart with references to that Secret. Do not duplicate one broad Infisical credential into every app. The Kubernetes agent itself receives only explicitly named, required keys through a scoped Secret reference; broad Secret-list permissions are unnecessary. Environment variables do not update in an already running container after Secret rotation, so plan a controlled rollout; mounted Secret files can update, but the application must reload them. The existing Grafana sync has no auto-redeploy target, so test its rollout behavior before changing it.

Chart deployment is a **new planned agent capability**, not an existing feature. Its future Helm executor must accept pinned chart/image versions and non-secret values, show a redacted plan/diff, verify operator sync and namespace scope, and apply only within approved Kubernetes permissions. Avoid a Helm bootstrap cycle for Infisical itself: its existing database and backend bootstrap Secrets stay under the established workflow. Test that a chart can consume an environment-specific synced Secret without exposing its value in MCP output, CLI logs, `helm get values`, or rendered YAML.

## Phases and acceptance criteria

| Phase | Prerequisite and deliverable | Acceptance test | Approval boundary |
| --- | --- | --- | --- |
| 0. Capability and lifetime matrix | Pin the deployed OpenAPI and Go SDK version. Verify Universal Auth client-secret TTL/expiry semantics, edition, role scope, and each command's API method. Compare legacy Service Tokens without creating one. | Read-only schema checks distinguish supported, permission-denied, and unverified operations; the proposed one-year bootstrap policy is accepted or rejected by a controlled test. | Identity and permission probes need an agreed scoped test identity. |
| 1. API, MCP, and configuration contract | Agree on Go interfaces, versioned command names, JSON schemas/errors, required stdio MCP tools, noninteractive CLI, host endpoint/TLS, and the two host-agent variables `INFISICAL_CLIENT_ID` and `INFISICAL_CLIENT_SECRET`. Include machine-readable capability metadata and explicit unsupported results. | AI harness `tools/list` discovery, input validation, invalid-config failures, and a KIND-to-host connectivity test. | Review contract and host listener before building or exposing the new service. |
| 2. Unattended authentication | Provision one scoped Universal Auth client ID/secret pair after approval and supply it through the two host-agent environment variables; keep access tokens in memory. Re-login automatically from `expiresIn`, coalesce concurrent logins, survive restarts, and handle revocation without retry storms. Do not assume a refresh endpoint or indefinite token. | Fake-clock 30/365-day lifecycle, short-TTL soak, process restart, concurrent callers, safe 401 retry, 403 no retry, revocation/lockout, and redacted logs. | Creating an identity, assigning roles, selecting a long TTL, and supplying the client secret require separate explicit approval. |
| 3. Secret operations | Implement scoped metadata/list/read/create/update/delete with explicit project, environment, and path. AI-facing responses are redacted by default; authorized value use stays inside the local process or destination Pod. | Disposable-project contract tests, 404/409/429/5xx handling, duplicate-write prevention, redaction, and audit visibility. | Test writes and access to real values require exact scope approval. |
| 4. Project and administrative operations | Separate project/environment/folder commands from identity, role, membership, and user administration. Return `unsupported` for missing/version-limited operations. | Per-command allow/deny tests with scoped identities; secret access never implies admin rights. | Each administrative role, identity grant, and live write needs separate approval. |
| 5. Operator and chart secret delivery | Extend watched namespace scope deliberately. Charts create/reference path-scoped `InfisicalSecret` resources and consume only named Kubernetes Secrets. Preserve existing Grafana sync. | Sync Ready gate, missing Secret behavior, environment isolation, value-free Helm render/history, and rotation rollout tests for env and volume consumers. | Operator RBAC/namespace changes and any rollout policy change need review before live application. |
| 6. Kubernetes agent and Helm execution | Give the agent only its required named Secrets and a planned scoped Helm executor; route commands through the authenticated host service. Pin charts/images and expose a redacted plan before apply. | KIND DNS/TLS/auth, host outage independence of existing operator sync, namespace/RBAC denial, dry run, apply, rollback, and recovery tests. | Helm writes, Kubernetes RBAC, host listener, and live deployments require explicit scope approval. |
| 7. MCP harness and documentation | Ship the required MCP server over the shared command service plus operator examples and stable JSON error codes. | AI harness discovers tools, invokes read/plan commands, handles permission/unsupported/expiry errors, and cannot receive credentials or secret values. | MCP transport is required; any network exposure or privileged tool enablement needs separate review. |

## Safety rules for implementation

- Do not put Infisical admin credentials in Git, Helm values, Pods, AI prompts, or logs. Supply the host bootstrap pair only through `INFISICAL_CLIENT_ID` and `INFISICAL_CLIENT_SECRET` in the host agent's runtime environment; the existing operator credential remains scoped to its own job. Redact values, tokens, URLs with credentials, and server error bodies before returning structured errors.
- Retry only safe reads and explicitly idempotent writes with a bounded backoff; never turn 403 into re-authentication or replay a non-idempotent admin write automatically.
- Keep secret CRUD and administrative commands in separate capability groups and roles. A machine identity with access to Grafana's secret path does not imply organization administration.
- MCP must be present and tested, but it must expose only the permissions granted to its caller. Chart/Helm tools never return Secret contents to the AI harness.
- Preserve the current PVCs and healthy releases. Treat the `local-path` StatefulSet template mismatch as a separate storage migration decision, not a reason to delete or recreate data during this integration.
