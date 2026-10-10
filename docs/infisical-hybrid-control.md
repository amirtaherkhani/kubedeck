# Infisical hybrid control layer

## Scope and live baseline

The Host runs a local enrollment controller alongside its existing HTTPS bridge.
The human organization-Admin session is used only to enter explicitly approved
projects and establish machine memberships. Routine commands keep machine auth.
The Kubernetes bridge is read-only in both the cluster proxy and Host server;
local CLI/MCP administrative commands retain their separate explicit allowlists.
This does not change Kubernetes RBAC.

On 2026-10-10 the existing base project's display name was changed to **Kubedesk
Platform**, preserving UUID `85cbdbe6-80dd-4e31-aa45-2f0ac2263588` and slug
`home-lab-nb0-k`. Name and UUID were read back. No credentials, memberships,
folders, history or secrets were replaced. Grafana's live `grafana-admin`
InfisicalSecret selects the existing slug; it must not be changed independently.

`KUCHDESK_BASE_PROJECT_NAME` defaults to `Kubedesk Platform`. Optional
`KUCHDESK_BASE_PROJECT_ID` and `KUCHDESK_BASE_PROJECT_SLUG` describe the base tools
project only. They never select a managed project or grant permission. Managed
projects use immutable IDs in policy and explicit command scopes. Spaces and
non-ASCII names are supported; empty/unset selects the default, whitespace-only
or control characters are rejected. Process environment changes require a restart; `.env` metadata is re-read on
each CLI invocation and MCP capabilities request. The cluster chart maps `baseProject.name`, `id`, and `slug` to these
variables; `/v1/platform` and CLI/MCP capabilities expose the display metadata.
The Host CLI/MCP load these three literal keys from the same private project
`.env` selected by `KUCHDESK_PROJECT_ROOT` (or repository discovery). Non-empty
process environment overrides `.env`; defaults apply last. Empty process values
fall back to the file. Unknown keys are never exported or evaluated, and the
loader never writes the file. Keep the existing user-owned mode 0600 and 4 KiB
limit; do not shell-source it. Credential backend selection and credential
precedence remain unchanged. Kubernetes uses the same variable names from Helm
`baseProject` values; it never receives or mounts the Host `.env`.

## Compatibility

Enrollment REST contracts are pinned to installed **v0.151.0**. Unknown versions
fail closed. Human session refresh also models **v0.166.3**, but project enrollment
for that version reports unsupported until its full effective-permission
contracts are validated. Selecting a version does not upgrade the instance.
`POST /api/v1/organization-admin/projects/:id/grant-admin-access` grants the
acting human access, never a machine identity. Machine project memberships are
separate writes with read-back verification.

The official Go SDK v0.8.0 authentication package was evaluated. Dependency
installation was blocked by HTTP 403 for `google.golang.org/api@v0.267.0`;
a direct-source attempt did not complete. A follow-up official archive check
failed DNS in the restricted environment; escalation was rejected because the
user prohibited bypassing denied network access. Cached SDK source confirms
that even its auth package imports the shared error/util dependency chain.
No alternate mirror, dependency substitution, or network bypass was used. This change therefore retains the
existing bounded Universal Auth client, rather than claiming SDK adoption.
The typed enrollment adapter is isolated behind `Backend` so supported SDK
methods can replace REST methods after dependency and transport review.

## Local configuration and runtime state

Use `kuchdesk-enrollment -url ORIGIN -policy PRIVATE_JSON -state-dir PRIVATE_DIR`
for a local controller, or add `-enrollment-policy PRIVATE_JSON
-enrollment-state-dir PRIVATE_DIR` to the Host bridge. Do not run both against
the same state directory. `-once` runs one cycle; `apply:false` is dry-run and
never joins projects or writes identities/memberships. Startup immediately
reconciles, then polls with bounded exponential backoff. One cycle has a two
minute deadline; API requests are bounded and serialized, and discovery has a
configured cap. A failed project does not authorize access to that project.

Proposed state location (not installed by this code change):
`<KUCHDESK_PROJECT_ROOT>/.kuchdesk/infisical-control/`, user-owned mode 0700
and covered by the repository `.gitignore`.
It must be explicitly created for activation. `controller.lock`, `session.json`
and `ledger.json` are mode 0600. Saves use a temporary file, fsync, atomic rename
and directory fsync; the process lock prevents competing refresh writers.
Session and ledger JSON never appear in command output. Do not put this folder
in Git, cloud sync, logs or diagnostic bundles. Interrupted writes can leave
private `.pending-*` files; treat them as sensitive, not ordinary debug output.

Machine bootstrap remains the existing private project `.env`, with optional
explicit Keychain selection. No rotating secret rewrites `.env`. Human login is
a separately approved browser/SSO handoff into a private session file. There is
no password capture, browser extraction or AI-facing login endpoint. With the
controller stopped, `-import-session PRIVATE_FILE` validates the expected origin,
organization, version and unexpired human JWT claims, then writes session state.
Infisical still verifies signatures and permissions on every authenticated call.
Re-login uses the same explicit handoff; runtime import must never be automated
through chat. Activation must approve this storage before credentials are entered.

v0.151.0 refresh reuses its refresh cookie. v0.166.3 rotates it. A durable pending
marker is written before refresh; a restart or ambiguous failure during a
rotating refresh requires re-login rather than risking reuse. A revoked or
expired session is reported explicitly. No indefinite unattended session is
promised. Ordinary machine commands can continue while new enrollment needs a
human session.

## Policy and permissions

See `host-agent/config/enrollment-policy.example.json`. Start with explicit IDs,
`allProjects:false`, `apply:false`, `createK8Identity:false`, and
`repairRevocations:false`. Policy edits reload at each cycle and bridge request;
invalid JSON or changed organization/version/identity bindings deny access.
Identity binding changes require restart and reviewed state migration.
Includes/all-project discovery authorize nothing outside the approved policy;
excludes always win. Eligibility is withdrawn as each project is revalidated, so an observed
revocation cannot stay eligible while later projects are checked. Upstream
changes otherwise become visible on the next bounded polling cycle.
Policy does not remove upstream memberships when excluding
a project; it removes bridge eligibility.

The Host identity must already exist from the manual Host bootstrap. Its
organization permissions must permit discovery and identity administration;
missing permission is a denied/enrollment-required result, never a fallback
user token for routine commands. The controller reuses a configured K8 identity
ID, or an unambiguous configured name on first bootstrap. Optional creation uses
organization `no-access` with delete protection. It does not create auth methods
or return credentials. Kubernetes-auth/credential binding is a separate live
setup decision.

The Host gets project `admin`, K8 gets project `viewer`. Eligibility requires
exact permanent roles, no K8 organization role beyond `no-access`, and no
additional K8 project privileges. Failed or unsupported checks deny eligibility.
Existing differing roles and observed missing memberships are preserved as
revocations unless the reviewed policy explicitly enables repair. Never enable
repair merely to make a warning disappear. An uncertain write/create is journaled;
subsequent runs observe the result and otherwise require operator review instead
of replaying writes. The controller never deletes identities or projects.

Only project/environment/folder/secret metadata reads cross the K8 bridge.
Membership, role, project writes, secret writes/deletes and unknown operations
are forbidden before invoking Host commands. Doctor includes a value-free
`infisical-enrollment` check when the controller is enabled in the bridge.

## Staged slug migration (not executed)

Target slug: `kubedesk-platform`; UUID stays unchanged.

1. Inventory all external consumers as well as repository references. The live
   cluster currently has one InfisicalSecret consumer: observability/grafana-admin.
   Check any external CI/integrations before declaring the inventory complete.
2. Save non-secret desired Helm values and the exact CR selector/version. Prepare
   the new slug in the base Grafana profile and any explicitly scoped Helm probe.
3. In a coordinated window, patch only the Infisical project slug by UUID, then
   the Operator selector. Retain the existing Kubernetes Secret during the gap.
4. Persist the same value in the owning Helm release without changing the workload
   template. Review rendered diff before any upgrade; do not trigger a cosmetic
   restart. Verify Operator reconciliation and read back UUID/name/slug.
5. On failure restore both selectors to the previous slug. Do not delete/recreate
   the project, Secret, PVC or namespace. Historical release notes retain their
   original names and dates.

## Activation gates and acceptance

No controller, new credential store, identity, grant, or new bridge build was
activated by this implementation. Before activation, approve the exact origin,
organization ID, existing Host identity ID, K8 identity ID/name and auth method,
included/excluded project IDs (or future-project policy), repair policy, private
state path and human session handoff. Review dry-run output first. Revalidate
v0.151.0 discovery, identity permissions, self-enrollment and read-only effective
access against that scope before enabling apply. No Infisical upgrade is implied.

Tests cover session refresh/rotation, concurrent requests, crash markers, disk
failure, restart, pagination, cross-organization rejection, repeated startup,
external enrollment, identity reuse, uncertain writes, partial failures,
revocation, reload, renamed projects, K8 admin denial and redacted errors.

The concrete [activation approval bundle](infisical-activation-plan.md) records
verified identity IDs, all-project scope, project-local state and K8 auth/RBAC
effects. It supersedes the earlier open-ended activation questions.
