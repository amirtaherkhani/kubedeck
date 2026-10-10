# Proposed Operator TokenRequest narrowing

**Applied with explicit user approval on 2026-10-10, Helm revision 4.**
See the [rollout and observability audit](infisical-observability-audit-2026-10-10.md).
The proposal and pre-apply evidence below are retained for review. Human-session
handoff is still pending. This change is not an Infisical/Operator upgrade.

## Verified live ownership and consumers

Snapshot: 2026-10-10, Docker Desktop context. Helm release `infisical-operator`,
namespace `platform-secrets`, revision **3**, chart/image **0.11.11** owns all three
`infisical-opera-manager-role` Roles and their corresponding
`infisical-opera-manager-rolebinding` RoleBindings. Every binding targets
`system:serviceaccount:platform-secrets:infisical-opera-controller-manager`.

All installed Infisical CR kinds were inventoried across namespaces:

| Kind | Configured objects |
|---|---|
| InfisicalSecret | `observability/grafana-admin`, Universal Auth |
| InfisicalAuth, InfisicalConnection, InfisicalDynamicSecret | None |
| InfisicalPushSecret, InfisicalStaticSecret, ClusterGenerator | None |

There are no configured Kubernetes-auth Operator consumers requiring TokenRequest
in `development-tools`, `observability` or `platform-secrets`. This inventory
covers configured Operator consumers; it does not establish a history of every
out-of-band API request. Grafana's three synchronization conditions remain True.
Its existing credential reference is `platform-secrets/infisical-universal-auth`,
and its managed Secret is `observability/grafana-admin`; values were not read.

The approved preparation already created the dedicated reader service account,
its TokenReview role/binding, and the narrowly scoped TokenRequest role/binding.
They remain unchanged. No new Infisical identity, human-session import, consumer
cutover or Host restart has occurred. `apply:false` remains in the private policy.

## Exact additional security-setting approval

Only change the following rule inside each existing manager Role:

| Namespace | Before | Proposed after |
|---|---|---|
| development-tools | core `serviceaccounts/token`, verb `create`, unrestricted names | Same resource/verb, `resourceNames: [kubedesk-infisical-reader]` |
| observability | core `serviceaccounts/token`, verb `create`, unrestricted names | Remove only this TokenRequest rule |
| platform-secrets | core `serviceaccounts/token`, verb `create`, unrestricted names | Remove only this TokenRequest rule |

See the exact [rendered diff](infisical-tokenrequest-rules.diff). Preserve every
other rule: Secret/ConfigMap management, pod reads, service-account reads,
workload reload management, existing TokenReview rule, CR/finalizer/status access,
and all bindings/subjects. Preserve the already-created dedicated auth resources.
This approval concerns direct TokenRequest permissions; existing Secret-management
permissions remain unchanged.

## Persistence through Helm

The official chart exposes `scopedRBAC`/`scopedNamespaces`, but no per-account
TokenRequest setting. The offline [chart builder](../lab/scripts/operator-token-scope/prepare_chart.py)
requires the official `secrets-operator-0.11.11.tgz` archive with SHA-256:

`1dd2419b46f3b44d423a67bef2408a787e7f1b90598753d5730a46983bf0a82c`

Source: [official repository index](https://dl.cloudsmith.io/public/infisical/helm-charts/helm/charts/index.yaml).
The builder checks the exact upstream rule/include structure and changes only
`templates/manager-rbac.yaml`, plus explicit non-secret policy defaults. It never
calls the network, Helm or kubectl and never overwrites an existing output path.
Archive traversal, links, checksum mismatch, duplicate policy keys and unexpected
upstream layouts are rejected.

Helm renders the patched chart itself and stores its templates/values in the new
release revision. No post-install kubectl patch is used. Upgrades must continue
to use this workflow: directly using the unpatched upstream chart would restore
the broad grants. Keep chart/image version 0.11.11; no Deployment label/version
bump is introduced. Include the generated values overlay even with `--reuse-values`
so the new policy is explicit rather than relying on merged chart defaults.

The [policy file](../lab/scripts/operator-token-scope/policy.json) names the exact
operator subject and namespace/account allowlists. Rendering fails on wrong
operator namespace/account, cluster-scoped RBAC, missing namespace policy, or an
invalid account. Existing live Operator user values were compared in memory and
exactly match `lab/apps/platform/infisical-operator/values.yaml`.

## Future intended auth

The planned Grafana CR in `observability` references service account
`development-tools/kubedesk-infisical-reader`. TokenRequest is authorized in the
**referenced service account's namespace**, not the CR's namespace. Future
projects can use the same explicitly approved identity/account; project discovery
does not need new Kubernetes TokenRequest grants. A future distinct service
account requires an explicit policy entry and reviewed chart change before use.
Adding a watched namespace also requires matching policy and review of any
controller configuration/rollout effects.

## Correct permission checks and regression evidence

The earlier command `kubectl auth can-i create serviceaccounts/token` was wrong
for this question: its slash identifies resource/name, not the token subresource.
It checked service-account creation and produced a false conclusion about
TokenRequest. Correct CLI syntax includes both a name and `--subresource=token`:

```sh
kubectl auth can-i create serviceaccounts/kubedesk-infisical-reader \
  --subresource=token -n development-tools \
  --as=system:serviceaccount:platform-secrets:infisical-opera-controller-manager
```

The [checker](../lab/scripts/operator-token-scope/check_permissions.py) instead
submits typed, read-only SubjectAccessReviews with separate `resource`,
`subresource` and `name` fields and the service-account groups. It never mints a
token. Current results: the approved name is allowed, but all three negative
namespace checks are also allowed. Exit 1 is the expected **current gate failure**;
it is not proof of a token request or mutation. Evaluation failures are reported
as unknown/error, never interpreted as denial.

Ten tests passed, including the actual pinned-chart integration tests. They
verify every non-target rendered object remains byte-identical, all unrelated
Role rules are unchanged, exactly the three TokenRequest rules differ, wrong
scope/account values fail closed, and future accounts require explicit policy.
Tests reproduce the broad-permission negative failure and validate the separate
resource/subresource/name fields so the original mistake cannot silently recur.

## Application gate, impact and rollback

Before any application: recheck Helm revision/values, Role drift, all configured
CR consumers and their service-account references. Re-render against the current
release values and stop if anything beyond these three Role rules changes.
Capture the prior revision and the three old Role rules as non-secret rollback
evidence. Obtain explicit approval for this exact existing-RBAC change.

After approval only, upgrade the same release using the prepared local chart,
`--reuse-values` and its explicit policy overlay. Then run the typed checker:
approved account must be allowed; all three negative checks must be denied.
Verify Grafana/Operator readiness and unchanged workload templates before
resuming the previously approved activation bundle. Stop on any discrepancy;
do not request Admin-session details while this gate is unresolved.

Expected impact: no pod rollout, Secret update or current Universal Auth consumer
interruption from this RBAC-only change. The full render comparison includes the
Operator Deployment and every CRD/binding. Existing unexpected callers minting
tokens for other accounts would receive authorization denials. Grafana's later
auth cutover remains separate and may trigger its configured auto-reload.

Rollback proposal for the new approval: if the narrowing unexpectedly breaks a
current consumer, restore the captured pre-change Helm revision (currently 3)
with `helm rollback infisical-operator 3 -n platform-secrets --wait`. This restores
the old broad TokenRequest permissions and must be expressly included as a
contingency in the informed approval; otherwise stop and report before reopening
access. Preserve both existing Secrets and all unrelated resources. Do not run a
blanket Role deletion, uninstall, namespace deletion or credential rotation.
