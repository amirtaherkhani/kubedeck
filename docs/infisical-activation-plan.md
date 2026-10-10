> Update 2026-10-10: human authority is now server-verified, including organization
> scope, AccessAllProjects and refresh availability. The approved apply cycle
> completed with `ready` for both project UUIDs below. The new K8 read-only
> identity is `d05fbb07-7b0c-402a-888d-980b9ac06cde`; identity creation is not pending.
> Policy apply is enabled; this was one bounded cycle, not a running daemon.
> Kubernetes-auth configuration/login tests, Grafana consumer cutover and the
> Host bridge restart remain outstanding. Operator RBAC narrowing is already
> applied at revision 4. Earlier preflight findings below are historical.

# Infisical activation approval bundle

Prepared from read-only checks on 2026-10-10. **Identity/project stage completed; consumer activation remains pending.**
The approved private state/template and five dedicated SA/RBAC resources have
since been created. Human session and identity/project reconciliation are now
verified; consumer cutover and Host restart have not occurred. The separately approved
[targeted TokenRequest correction](infisical-tokenrequest-narrowing.md) is now applied.
This plan uses the
installed Infisical v0.151.0 and Operator v0.11.11; no upgrade is included.

## Verified targets

- Origin: `https://infisical.local.dev`.
- Organization: `b46cad64-747c-4cec-a59a-e401cede95bd`.
- Existing `Kubedesk Host Agent`: identity `28ff63a8-e1dd-4929-bfb5-16dca7e238c3`,
  organization `admin`, Universal Auth, permanent base-project `admin`.
- Existing `local-dev-agents`: identity `7f02cfbe-ee13-4efa-bf5f-6bd020a58919`,
  organization `member`, Universal Auth, base-project `member`. Leave it intact:
  this is not the proposed read-only identity and its other consumers are unknown.
- Create/reuse exactly one identity named `kubedesk-k8-readonly`; no identity with
  that name exists in the complete two-identity listing. Organization role
  `no-access`, permanent project `viewer`, no additional project privileges,
  delete protection enabled. Bind its returned immutable ID in the local journal.
- Scope: **all current and future projects in this organization**, with explicit
  excludes winning and observed revocations preserved (`repairRevocations:false`).
  Current projects are Kubedesk Platform (`85cbdbe6-80dd-4e31-aa45-2f0ac2263588`,
  slug `home-lab-nb0-k`) and vero-finance (`fd27d482-48bc-4b6c-b194-5901bdbc497b`,
  slug `vero-finance-da-ih`). Host membership inspection of vero-finance is
  currently denied; human organization-Admin enrollment is needed there.

## Proposed local configuration

Keep the existing private project `.env` and its Host bootstrap pair. No Keychain
access, new Host credential, or file replacement is proposed. Add metadata only
through a reviewed edit preserving every existing line, if desired:

```dotenv
KUCHDESK_BASE_PROJECT_NAME=Kubedesk Platform
KUCHDESK_BASE_PROJECT_ID=85cbdbe6-80dd-4e31-aa45-2f0ac2263588
KUCHDESK_BASE_PROJECT_SLUG=home-lab-nb0-k
```

Non-empty process environment overrides these values; absent/empty file values
use defaults (`Kubedesk Platform`, empty ID, empty slug). File metadata reloads on
CLI invocation/MCP capabilities calls; process environment changes require restart.
These settings never select or authorize managed projects.

Create, only after approval, the user-owned mode-0700 directory
`/Users/mac/Documents/GitHub/kuchdesk/.kuchdesk/infisical-control/` and private
mode-0600 `policy.json`, `ledger.json`, `session.json`, and `controller.lock` as
needed. `.kuchdesk/` is Git-ignored. Do not sync or include it in backups without
credential-grade protection. Session JSON contains a human access token and
refresh token in plaintext protected by OS ownership/mode; it is not encrypted
by Keychain. The rotating state is never placed in `.env` or sent to the cluster.
Private interrupted-write `.pending-*` files are sensitive too.

Initial `policy.json`:

```json
{
  "version": "v0.151.0",
  "organizationId": "b46cad64-747c-4cec-a59a-e401cede95bd",
  "hostIdentityId": "28ff63a8-e1dd-4929-bfb5-16dca7e238c3",
  "k8IdentityName": "kubedesk-k8-readonly",
  "createK8Identity": true,
  "allProjects": true,
  "include": [],
  "exclude": [],
  "apply": false,
  "repairRevocations": false,
  "intervalSeconds": 60,
  "maxProjects": 1000
}
```

Preserve any existing exclusions/journal instead of replacing them with this
initial example. `apply:false` prevents upstream writes even with creation
requested. The first dry-run will report the missing identity/create plan; it
cannot verify read-only effective access before that identity exists. After
creation, repeat dry-run and inspect per-project status before enabling apply.

## Recommended K8 authentication

Use **native Kubernetes Auth with short-lived service-account TokenRequests**,
not a new Universal Auth client secret. The installed v1alpha1 CRD exposes
`kubernetesAuth`, `autoCreateServiceAccountToken`, `serviceAccountRef` and token
audiences. Infisical v0.151.0 supports using the incoming token as reviewer when
no separate reviewer JWT is supplied ([versioned route](https://github.com/Infisical/infisical/blob/v0.151.0/backend/src/server/routes/v1/identity-kubernetes-auth-router.ts),
[versioned service](https://github.com/Infisical/infisical/blob/v0.151.0/backend/src/services/identity-kubernetes-auth/identity-kubernetes-auth-service.ts)).

The exact [RBAC manifest](../host-agent/config/k8-auth-rbac.homelab.plan.yaml) is
prepared for review only. It contains no token or Secret and has not been applied.

Exact proposed changes after approval:

1. Create service account `development-tools/kubedesk-infisical-reader`.
2. Create ClusterRole and ClusterRoleBinding `kubedesk-infisical-token-reviewer`,
   granting only `create` on `authentication.k8s.io/tokenreviews` to that account.
   This is an authentication permission, not permission to read/write cluster
   workloads or Secrets. Existing KuchDesk agent RBAC stays unchanged.
3. Create namespaced Role/RoleBinding `development-tools/kubedesk-infisical-token-request`,
   allowing only `create` on `serviceaccounts/token`, restricted to resource name
   `kubedesk-infisical-reader`, for existing service account
   `platform-secrets/infisical-opera-controller-manager`. The initial permission check was incorrect: it did not query the token
   subresource. Correct named-subresource checks found broader pre-existing
   TokenRequest grants in all three watched namespaces. The new dedicated Role
   is scoped correctly but cannot override those grants; see the correction plan.
4. Configure Kubernetes Auth on `kubedesk-k8-readonly`: API review mode,
   host `https://kubernetes.default.svc`, actual public cluster CA with TLS
   verification, allowed namespace `development-tools`, allowed service-account
   name `kubedesk-infisical-reader`, no persisted reviewer JWT. Use API-server
   default audience initially (omit custom TokenRequest audiences; empty
   `allowedAudience`), access-token TTL/max TTL 900 seconds. Validate an actual
   login and negative namespace/account checks before consumer cutover. No
   long-lived Kubernetes service-account token Secret is needed.
5. Configure the existing `observability/grafana-admin` InfisicalSecret through
   its owning Grafana chart to use the returned identity ID,
   `autoCreateServiceAccountToken:true` and the exact service-account reference.
   Preserve its existing scope, managed Secret and Grafana Deployment template.
   The chart now has an opt-in `infisical.authMethod=kubernetesAuth` path,
   requiring the identity UUID and exact service-account name/namespace. Offline
   tests verify both Deployment renders stay byte-identical across auth modes.
   Keep the default Universal Auth until the new identity and RBAC are ready.

The enrollment controller currently provisions identities and memberships only;
Kubernetes-auth/RBAC wiring above is a distinct activation step, not functionality
silently performed by enabling `apply`. KuchDesk's cluster agent continues its
read-only HTTPS Host bridge; it does not currently log in directly as the K8
Infisical identity. The Operator is the first consumer of that identity.

## Human login: user-only handoff

Use the supported [user-run browser login](infisical-human-login.md). The local
`kuchdesk-enrollment -login` command receives the official v0.151.0 callback after
the user completes login/MFA/SSO and organization selection in Infisical. It
validates server authority and saves the private session without token copying.
Run `-status` separately for sanitized, read-only server-authority validation.
The older `-import-session` remains an explicit compatibility path; local import
alone is not server-authority proof and is no longer the normal user workflow.

## One approval bundle and execution order

Approve the exact targets and effects in this document together:

- **Persistence:** private project-local session/journal files containing human
  tokens; existing Host bootstrap remains in its private `.env`; K8 uses transient
  TokenRequest/Infisical tokens, with no new long-lived K8 client secret.
- **Access expansion:** Host project Admin and K8 project Viewer across the two
  current and future organization projects, preserving exclusions/revocations.
  The human session can self-enroll into approved projects as Admin. Existing
  `local-dev-agents` access is not revoked or changed.
- **Cluster changes:** only the dedicated service account/authentication RBAC and
  reviewed Grafana InfisicalSecret auth selector described above. This is not
  cluster-wide workload administration or an Infisical upgrade.
- **Network/process effects:** Host polls Infisical HTTPS roughly every 60 seconds
  with backoff; Operator requests short-lived tokens from the Kubernetes API and
  calls Infisical; Infisical calls the API TokenReview endpoint. Keep the existing
  HTTPS Host listener/address/TLS/bearer. Restart the Host bridge once to load
  the new binary and policy flags; project-policy updates then need no restart.
  Render consumer changes first and reject any unintended workload rollout.

Order: review the private user handoff and prepared auth/RBAC manifests; run no-session dry-run;
user login/import; create the single identity under approved scope; repeat dry-run;
enable reviewed apply policy; verify exact roles/effective read-only access and
negative writes; validate K8 login; cut over the one consumer; verify Doctor and
Operator readiness. On failures disable apply and stop enrollment; preserve the
journal and known-good Secret. Reverting apply does not revoke memberships already
created; any removal needs a separate reviewed rollback action.

## Authorized slug migration: exact read-only preflight

Target `kubedesk-platform`, same base-project UUID/name. Do this separately from
auth cutover so failures have one cause. Current `observability/grafana-admin`
conditions LoadedInfisicalToken, ReadyToSyncSecrets and AutoRedeployReady are True.
Its Helm release is `grafana` in `observability`, revision 2; the Helm manifest
contains this InfisicalSecret despite missing release annotations on the live CR.

Affected selectors/references:

- Infisical project UUID above: slug `home-lab-nb0-k` -> `kubedesk-platform`.
- Live CR `.spec.authentication.universalAuth.secretsScope.projectSlug` today;
  after auth cutover use `.spec.authentication.kubernetesAuth.secretsScope.projectSlug`.
  Environment `dev`, path `/apps/observability/grafana` stay unchanged.
- `lab/apps/observability/grafana/values.homelab.yaml`, key
  `infisical.projectSlug`; template `templates/infisicalsecret.yaml` consumes it.
  The checked-in `envSlug` was stale (`local`) and is corrected to verified `dev`;
  existing explicit deployment overrides still take precedence.
- Owning Helm release values/rendered CR; `lab/README.md` install override
  `INFISICAL_PROJECT_SLUG` must agree. Inspect any generated/explicit overrides
  before applying rather than replacing unrelated values.
- Optional local `KUCHDESK_BASE_PROJECT_SLUG` metadata, if configured.
- `lab/tests/infisical-helm-probe/README.md` example and any explicit probe slug.
  Probe defaults have no slug; no live probe release/CR was found.
- Active Infisical documentation naming the current slug; historical release
  notes retain their original evidence.

Preserve `observability/grafana-admin` Secret and its Grafana references
(`envFromSecret`, `admin.existingSecret`). Preserve the existing bootstrap Secret
`platform-secrets/infisical-universal-auth` for rollback; never print its data.
The `home-lab/managed-by` label is unrelated to project selection and stays.

Remaining slug gate: external CI/integration inventory is not established by
cluster/repository inspection. Once resolved, render the exact release change,
coordinate project slug and CR/Helm selector updates, read back UUID/name/slug and
Operator readiness, and restore both selectors to `home-lab-nb0-k` if needed.
No rename, Helm apply, secret mutation or restart was performed in this follow-up.

## SDK boundary

Keep the current bounded REST adapter and tests. No retry or bypass of the denied
SDK network access is proposed. Supported next options are to retain REST, or
review an official SDK/dependency set only when it is available through an
already authorized network path or verified existing cache. No alternate mirror,
untrusted dependency, or silent version substitution is part of this bundle.
