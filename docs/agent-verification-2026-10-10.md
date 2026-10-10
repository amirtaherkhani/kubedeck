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
| intervalSeconds | Re-read after a cycle; an existing sleep is not interrupted. This timing behavior is source-inspected, not directly regression-tested. |
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
