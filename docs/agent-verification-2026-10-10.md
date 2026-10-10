# Agent command and reload verification — 2026-10-10

Scope: offline tests and source inspection at main `248e408`, plus the two
regression tests delivered with this report. No live login, identity grants,
service restarts, slug changes, upgrade, or credential inspection was performed.

## Tested results

Fresh `GOPROXY=off GOSUMDB=off go test -race -count=1` runs passed for:

- Cluster agent: `internal/httpapi`, `internal/management`,
  `internal/hostbridge`, `internal/dnsconfig`.
- Host: CLI Infisical, Doctor, deployment and MCP command packages;
  `internal/async` and `internal/hostbridge`.
- Targeted Host platform/enrollment tests for metadata, policy reload,
  dry-run/apply, revocation and pre-write authorization.

These cover bearer/auth boundaries, default-disabled management, bounded exec
and port-forward behavior, stale resource-version conflicts, same-resource
serialization, independent operations, queued/running cancellation, deadlines,
capacity release, panic containment, and redacted command responses.

New regressions:

- `TestMCPCancelsQueuedAndRunningJobsThroughTransport`: real in-memory MCP
  client/server calls cancel queued and running operations; canceled queued work
  never reaches the backend; a subsequent operation proves capacity is released.
- `TestBindingChangesImmediatelyDenyAccessAndRequireRestart`: each of version,
  organization, Host identity, K8 identity and K8 name changes immediately clears
  request eligibility, returns `restart_required`, and performs no new grants.

These are mock/fake-API tests, not proof of live cluster or external-provider
acceptance. No live acceptance is inferred from a passing unit suite.

## Restart matrix

| Configuration | Effective behavior |
|---|---|
| Base project name/ID/slug in `.env` | Re-read on CLI invocation and MCP capabilities request; nonempty process environment takes precedence. |
| Enrollment include/exclude/allProjects | Re-read per request; restrictions filter verified access immediately. New access waits for successful reconciliation. |
| apply, repairRevocations, createK8Identity, maxProjects | Re-read each cycle; write authorization rechecks policy immediately before mutations. |
| intervalSeconds | Re-read at bounded one-second intervals while waiting; deadline remains relative to the completed cycle. Shorter/longer intervals affect the active wait without restart; backoff and server Retry-After remain enforced. |
| Version, organization, Host identity, K8 identity/name | Restart required; changed bindings fail closed. Identity changes also need reviewed persisted-state migration. |
| Process environment, credentials, bridge listener/URL/TLS/static allowlist | Startup configuration; restart required. |
| Enrollment policy/state paths and enabling enrollment | Startup configuration; restart required. |
| Imported/replaced persisted human session | Stop/import/restart; controller loads it at startup. Automatic in-memory token refresh is separate. |

Source evidence: `host-agent/internal/platform/profile.go`,
`cmd/kuchdesk-infisical-mcp/main.go`, `internal/enrollment/policy.go`,
`internal/enrollment/controller.go`, `internal/enrollment/runtime.go`, and
`cmd/kuchdesk-host-bridge/main.go` under `host-agent`.

## Callback and slug boundaries

The official Copy-to-clipboard fallback is implemented and installed as a
separate recovery binary. Mock authority validation and terminal tests passed;
main `248e408` is pushed. No new user login was started. Exact Safari callback
transport failure is still unproven; browser evidence is needed to diagnose it.
Live human enrollment authority remains unverified. The importer alone is not an
end-to-end login flow: no standalone token-generation URL was found in the
pinned frontend; its fallback browser copy expires after 30 seconds. The new `./infisical-login` entry point now combines a fresh browser/callback
with hidden input. Mock integration tests cover both callback and pasted paths;
live user acceptance remains outstanding.

The existing slug migration inventory remains a read-only plan. Its external
CI/integration inventory gate is still open (`infisical-activation-plan.md`).
No fresh external inventory or live slug mutation was attempted here.

## Subsequent implementation and live activation checkpoint

Dynamic interval reload is now implemented in `reload_wait.go` and wired into
controller Run. Virtual-time race tests cover shortening, extension, an overdue
edit, Retry-After, failure backoff, malformed-policy fallback and cancellation.
Run also avoids starting a cycle when already canceled. No live restart was
performed to activate this new code.

Following the empty-JSON protocol fix (`fb76b87`), supported read-only status
returned human_authority_verified with all four authority/refresh flags true.
A fresh dry-run planned exactly one K8 identity. The approved bounded apply cycle
then returned ready for both approved projects, read back the required roles,
and recorded identity `d05fbb07-7b0c-402a-888d-980b9ac06cde` with no pending creation.
This does not claim successful Kubernetes-auth login or consumer cutover; those
and the Host bridge restart remain outstanding. No slug or service upgrade occurred.

## Live activation checkpoint — 2026-10-10

- Kubernetes-auth configuration verified for identity `d05fbb07-7b0c-402a-888d-980b9ac06cde`, dedicated `development-tools/kubedesk-infisical-reader`, TTL/max TTL 900 seconds, and API token review with no persisted reviewer JWT. Positive login and metadata reads of both approved projects succeeded. Default accounts in development-tools and observability were denied (401); this does not isolate a same-name cross-namespace test.
- Isolated secret test: Host create/update/delete succeeded; K8 PATCH returned 403. The disposable secret was deleted; no business-secret value was printed.
- Existing human-session refresh executed through the production Session refresh/persistence path. Server authority was revalidated after saving the refreshed session.
- Grafana Helm revision 3 deployed with a postrenderer preserving revision 2 resources except the authentication block. Helm merging retained the old Universal Auth block; an exact-test JSON Patch removed only that block. Live CR generation 3 reports only Kubernetes Auth, successful token loading and two secrets synced. Grafana remains 1/1 ready.
- Host bridge binary and plist backed up privately. Enrollment flags enabled against the existing absolute policy/state paths. One stop/start cycle performed; launchd bootstrap initially returned error 5, then a retry succeeded. PID 80221 stayed running across follow-up checks. Doctor passed 8/8 including enrollment ready. This verifies restart recovery and current service health; the public Doctor evidence does not expose a per-cycle timestamp, so repeated healthy reads alone are not proof of a second completed reconciliation cycle.
- Host `admin-login` shares the existing tested CLI implementation. Focused race tests passed for Host command, enrollment CLI and enrollment core. Built command installed at the existing Host executable path and help verified outside the checkout. The optional root script now uses this binary, with no separate login helper dependency.
- Rollback artifacts remain private under `.kuchdesk/activation-rollback` (original Grafana release values/manifest/CR and prior Host binaries/plist). No upgrade, project slug change, dependency deletion or unrelated identity/grant mutation was performed.

Remaining live acceptance gate: creating and later deleting a disposable Infisical project is outside the enumerated two-project activation bundle. Approve that isolated project lifecycle before testing discovery/enrollment of a newly created project against the live organization. Future-project discovery and policy reload have fixture coverage; do not call the live new-project scenario verified yet. See [helper inventory](host-helper-inventory.md).
