# Infisical RBAC rollout and observability access audit — 2026-10-10

## Result

The explicitly approved three-Role narrowing is applied in Docker Desktop.
Helm release `infisical-operator`, namespace `platform-secrets`, is **deployed at
revision 4**, with the same chart/image version **0.11.11**. Rollback was not needed.
The rollback to revision 3 was explicitly authorized as a failure contingency;
it would restore the previous broad TokenRequest access.

The stored revision-3 manifest matched the reviewed official chart render. Live
Role rules matched that manifest. Rendering the pinned patched chart with actual
release values and the explicit overlay changed exactly three Role rules; every
other rendered object was identical. Post-apply live Role rules matched the exact
approved result. This is a permission change, not an application-version upgrade.

| Namespace | Effective Operator TokenRequest check | Result |
|---|---|---|
| development-tools | `kubedesk-infisical-reader` | Allowed |
| development-tools | Unapproved account | Denied |
| observability | Unapproved account | Denied |
| platform-secrets | Unapproved account | Denied |

Checks used typed SubjectAccessReviews with separate resource, subresource and
name fields, including service-account groups. No tokens were minted. All other
Role rules, bindings and dedicated auth resources were preserved.

## Actual deployed observability services

All 11 observability Pods remained ready, with the same UIDs and zero container
restarts before and after the change. All 15 service EndpointSlices had a ready
endpoint. Ten HTTP checks below ran from the actual Grafana container through
cluster service DNS, before and after the rollout.

| Service/component | Evidence | Result |
|---|---|---|
| Grafana 13.1.0 | `/api/health` 200; database `ok`; HTTPS login/health 200 with certificate verification | Pass |
| Grafana image renderer | `/healthz` 200 from Grafana; Prometheus scrape up; `/render` without credentials 401 | Health/access pass; authenticated image generation untested |
| Prometheus | `/-/ready` 200; provisioned FQDN query endpoint 200; all 21 scrape targets up | Pass |
| Alertmanager and config reloader | `/-/ready` 200; both scrape targets up; both containers ready | Pass; external notification delivery untested |
| Loki and rules sidecar | `/ready` 200; provisioned FQDN labels endpoint 200; both containers ready | Access pass; historical discard warning below |
| Tempo | `/ready` 200; provisioned FQDN search-tags endpoint 200 | Pass; fresh trace production untested |
| Alloy | `/-/ready` 200; scrape up; log/span counters described below | Pass |
| DNS blackbox | `/metrics` 200; three DNS scrape targets up | Pass |
| kube-state-metrics | `/healthz` 200; scrape up | Pass |
| node exporter | `/metrics` 200; scrape up | Pass |
| Prometheus Operator | Deployment ready; scrape target up | Pass |
| k6 Operator (observability-tests) | Deployment ready; scrape target up | Pass; no test workload submitted |
| Grafana dashboard sidecar | Container ready with zero restarts | Readiness pass; dashboard contents untested |

Other existing Prometheus targets—API server, CoreDNS, kubelet and macOS host—were
also up. Management/reloader service ports are represented by their existing
scrape targets; no new credentials or test grants were introduced.

## Auth, routes and data flow

Grafana is the only Ingress in `observability`: `grafana.local.dev`, through
Traefik. Its certificate resource is Ready, and host-side HTTPS verification
returned `ssl_verify_result=0`. `/api/user` returned **401 without credentials**;
that is expected auth enforcement, not a service-health failure. No signed-in
Grafana session was available, so user/organization permissions and Grafana's
authenticated datasource-health API were not tested.

The mounted provisioning configuration names Prometheus, Loki and Tempo, with
proxy access to their actual `observability.svc.cluster.local` URLs. Read-only
queries to all three URLs returned 200 from Grafana. Secret values and secure
provisioning fields were not read. Renderer access is restricted by the existing
NetworkPolicy to Grafana and Prometheus; it was preserved. Grafana and renderer
reference the same `grafana-image-renderer` Secret, but its token was not read or
used to perform an authenticated render.

The only installed Infisical consumer is `observability/grafana-admin`, using
Universal Auth; its three conditions remain True. All six other installed
Infisical CR kinds contain zero objects. The CR exposes conditions only, without
a last-successful-sync timestamp, so this audit confirms current reported sync
health, not an independently timestamped fresh secret rotation. Operator readiness
is 1/1. Existing Grafana admin, renderer, Alertmanager, Prometheus and admission
Secret references were inventoried by name only and preserved.

Alloy reported 291,600 sent log entries, zero batch retries and zero dropped
entries at sampling. Loki reported the matching received-line count. Alloy
reported two accepted/exported spans; Tempo reported two received spans and zero
discarded spans. These cumulative counters show existing pipeline activity; they
do not prove fresh trace ingestion. No synthetic logs, traces, alerts or render
jobs were written during this read-only audit.

## Findings and remaining limits

- Loki's cumulative discarded-sample counter is **8,569**, all reason
  `too_far_behind`; it stayed unchanged on repeated checks. This is a warning
  about older samples, not evidence of current authorization denial. No retention,
  ingestion-limit or security settings were changed.
- `CPUThrottlingHigh` is firing at informational severity for
  `kube-system/kindnet-2vkj9`, container `kindnet-cni`, active since
  **2026-10-10 01:21:45 UTC**, before this rollout. Watchdog and InfoInhibitor are
  also present. No resource-limit or CNI configuration change was made.
- No new Infisical identity, human-session import, auth cutover, Host restart,
  Infisical upgrade or slug change occurred. The private policy remains dry-run.
- Authenticated rendering, user-level Grafana access, external alert delivery and
  new trace ingestion remain untested; they must not be inferred from health
  endpoints or cumulative counters.

## Remaining user login

The RBAC gate is cleared. At the audit snapshot no human session existed.
The subsequent [browser-login implementation](infisical-human-login.md) now
provides a supported user-run callback; manual token handoff is no longer the
normal workflow. Follow that document for the exact command and read-only
server-authority check. No live login was performed by the assistant.
