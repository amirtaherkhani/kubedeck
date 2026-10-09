# Agent capability inventory

This is a capability and deployment inventory, not a grant of cluster-admin, host-root, or secret access. **Required** means the requested product capability; **opt-in** means a powerful operation must be explicitly enabled and scoped when implemented. Status distinguishes source code, current runtime, and a tested end-to-end operation.

## Kubernetes agent

The chart is installed as `development-tools/kuchdesk-agent` on the checked Docker Desktop KIND cluster. The local values explicitly enable management and CoreDNS APIs and bind its dedicated ServiceAccount to `cluster-admin`; chart defaults remain read-only for other installations. The release is 1/1 Ready behind a ClusterIP Service, with the existing bearer Secret retained. CoreDNS's `Corefile` remains unchanged.

| Capability | Requirement | Source status | Runtime / gap |
| --- | --- | --- | --- |
| Nodes, Pods, Services, workloads, Ingress, storage, DNS and events inventory | Required | Snapshot/list-watch/SSE implemented | Authenticated snapshot returned cluster `docker-desktop`, one node, 33 Pods and 34 Services; no full SSE soak test |
| CPU and memory metrics | Required | Metrics API client implemented | Live snapshot returned node CPU/memory usage with `metricsAvailable=true` |
| Generic Kubernetes GET/list/watch/create/update/patch/delete/apply | Required, writes opt-in | Management API implemented for discovered resources; Secret resources excluded | `cluster-admin` is active; authenticated capabilities and workload status succeeded. Generic writes beyond a scale dry run have not been exercised live |
| Logs, workload status, scale, restart | Required, actions opt-in | Bounded log and status/scale/restart endpoints plus in-process async scale/restart jobs implemented | Workload status and scale `dryRun=true` succeeded without changing replicas or resource version; real mutation and async jobs remain untested live |
| CoreDNS exact service aliases | Required DNS capability, write opt-in | Managed Corefile block, dry run, resource-version check and rollback implemented | GET and empty-alias dry run succeeded; live Corefile version 227 did not change. No alias write was authorized or performed |
| Exec/attach/port-forward, Job lifecycle, Secret value operations | Required for full management | Not implemented as dedicated capabilities; generic resource operations do not cover these safely | Need transport, authorization and audit designs |
| Helm chart plan/apply/rollback and Infisical Secret readiness | Required | Not implemented | Must avoid secret values in rendered manifests and Helm history |
| Build image, publish to the local registry, then deploy/test a pinned image through Helm | Required | Host CLI/MCP profile workflow supports host-side `crane` push and optional KIND CRI pre-pull | Host push, node pull and kubelet start succeeded. A manual Helm install and a full CLI apply both reached 1/1 Ready; the first CLI rollout required a corrective CRI pull after `ctr` alone left `ErrImageNeverPull`. Fresh-blob kubelet pull remains untested |
| Redeploy and replica changes | Required | Workload scale/restart endpoints implemented | Controller-managed Pods must be changed through workload desired state; deleting one Pod causes its controller to replace it |
| Pod details, logs, termination, and Job lifecycle | Required | Snapshot and bounded logs exist; generic Pod delete is behind management mode | Pod stop/delete semantics and Job lifecycle require explicit workflows and live verification |
| KIND node-count changes | Required where supported | Not implemented | Node topology is a cluster lifecycle operation, not a Pod scale operation; inspect Docker Desktop KIND support and preserve cluster data before designing it |
| RBAC, NetworkPolicy, storage and CRDs | Required for full coverage | Generic discovery/resource operations cover served top-level resources when enabled; no dedicated workflows or generic subresource support | Not fully implemented or verified; actual permissions still enforced by Kubernetes |

## macOS host agent

The current `dev.kuchdesk.host-agent` LaunchAgent is a **one-shot job every 30 seconds**. `not running` between runs is normal. After the latest binary update, its last exit was 0 and Technitium's wildcard remained current at `192.168.1.100`.

| Capability | Requirement | Source status | Runtime / gap |
| --- | --- | --- | --- |
| LAN interface and private IPv4 selection | Required | Reads selected/default-route interface, rejects ambiguous addresses, supports explicit target IP | Running for `en0`; current answer verified |
| Technitium zone and wildcard A reconciliation | Required | Creates configured zone if missing; updates only `*.zone` when IP/TTL differs | Periodic job healthy; current zone `local.dev` |
| Host network, route, DNS, connectivity and port inspection | Required | Doctor CLI/MCP collects sanitized route, DNS, Docker, KIND, registry, disk, metrics and Deployment facts read-only | Live deterministic run passed all nine configured checks; detailed port/process inventory is absent |
| CPU, memory, disk, process and service monitoring/actions | Required | No host-agent management API | macOS node_exporter metrics are monitored separately; generic host actions absent |
| Infisical via CLI and MCP | Required | Shared typed name-listing service, CLI, stdio MCP, Universal Auth token lifecycle implemented in source | Not installed or authenticated live; admin and value-delivery functions absent. The separate deployment MCP tools are source-tested and opt-in |
| DNS/router/firewall/process/service mutation | Required only where an explicit action is needed; always opt-in | No general mutation API | Requires scoped commands, confirmation, recovery, audit and tests before enablement |
| Build orchestration and cluster deployment from the Mac | Required | Local CLI and opt-in stdio MCP plan/start/status/cancel use the same typed profile workflow | Local registry transport and manual Helm rollout work; no live end-to-end apply through this workflow yet |

The Infisical MCP server and management-enabled Kubernetes agent accept independent operations concurrently with bounded in-process execution, context-driven cancellation and timeouts, and conflict control for writes to the same target. This is an agent execution contract, not a separate task-manager service. The current host DNS job remains one-shot. The Infisical MCP server has in-memory start/status/cancel tools for name-only reads: at most four operations run concurrently, 64 records are retained, and each operation times out after 30 seconds. Same-key operations serialize inside its runner for future writes. The Infisical and deployment CLIs run synchronously; the host MCP process can also run a configured deployment profile in its bounded runner with per-release serialization. The Kubernetes agent has the same four-operation and 64-record bounds for scale/restart jobs; it serializes actions on the same workload. Each process loses job state on restart, and cancellation cannot undo a Kubernetes mutation already accepted.

## Capability test evidence

| Capability | Source tests | Live result / remaining gap |
| --- | --- | --- |
| Kubernetes snapshot, metrics and bearer auth | Collector and HTTP tests, including unauthorized requests | Snapshot 200 with metrics; unauthenticated snapshot 401; extended SSE and long-running metrics behavior untested |
| Management resource and workload operations | Fake API tests for confirmation, resource versions, dry runs, error paths and async job bounds | Self-ServiceAccount `cluster-admin` verified; scale dry run and unchanged replica/version verified; real writes intentionally untested |
| Capabilities catalog | Discovery, Secret exclusion, authorization and bounded parallel review tests | Live catalog returned 218 resources but took about 40 seconds before parallel review change; remeasure after redeploy |
| CoreDNS aliases | Parser, idempotency, conflict, dry-run and preservation tests | GET and empty-alias dry run 200; Corefile version unchanged; no live alias write |
| Host DNS and Technitium | Interface and reconciliation tests | LaunchAgent last exit 0, existing zone and interface retained; no extended failure-injection run |
| Infisical MCP and Universal Auth | Mock transport tests for renewal, 401 retry, redaction, typed tools and bounded jobs | No approved live host identity; credential revocation and real admin/value permissions untested |
| Build, registry and Helm delivery | Fake command runner tests for push failure, digest pin, CRI pre-pull failure, and MCP profile boundaries | Registry/CRI/kubelet, manual Helm install and full CLI apply verified; failure rollback and MCP apply remain untested live |
| Doctor diagnostics and optional AI | Fake command, resolver, HTTP and disk tests for ordering, failures, redaction, bounds, malformed AI output, timeout and offline mode | Live deterministic command passed route, DNS, Docker, KIND, registry, disk, metrics and two Deployment checks. AI mode reported `provider_not_configured`; no data was sent externally |

## Configuration and propagation

| Input or change | Current mechanism | Required apply/restart |
| --- | --- | --- |
| `lab/site.json` domain, hostnames or TLS issuer | Go renderer writes generated Certificates, TLSStore, Grafana/Infisical Helm values and DNS probe config | Render again; review/apply generated resources and upgrade affected Helm releases. Update Technitium zone and host-agent `-zone` separately. Domain changes need the new certificate Ready before switching Ingress/TLS. |
| Host DNS zone, context, interface, target IP, TTL or interval | Installer writes LaunchAgent arguments/environment; each one-shot run chooses the current IP unless `-target-ip` is set | Reinstall/rebootstrap the LaunchAgent for changed saved settings. IP changes are detected on the next interval; a new subdomain under the wildcard needs an Ingress route but no new wildcard record. |
| Kubernetes agent identity, listener, cluster domain, auth reference, DNS/management flags | Helm values become Deployment environment and RBAC resources | Helm upgrade and Pod rollout. The flags are not runtime-watched. `cluster.domain` defaults to `cluster.local`; the DNS target ConfigMap defaults to `kube-system/coredns` key `Corefile`. |
| CoreDNS alias list | Explicit authenticated `PUT /v1/dns/config` with full list, current resource version and optional dry run | When management is enabled, the agent updates only its marked ConfigMap block; CoreDNS `reload` applies it without a Pod restart. No automatic Service-to-alias watcher exists. |
| Infisical MCP bootstrap pair | `INFISICAL_CLIENT_ID` and `INFISICAL_CLIENT_SECRET` in the MCP process environment | Restart MCP process to replace either value. Short-lived access tokens are reissued automatically before expiry and after restart; bootstrap credential lifetime must be verified. |
| Infisical Operator-synced Kubernetes Secret | Operator resync updates the native Secret | Existing Pod environment variables require a rollout; Secret volumes can update but the application must reload. Existing Grafana sync has no automatic redeploy target. |
| Certificate renewal | cert-manager updates Secret, Traefik reads it | No manual Traefik restart for routine renewal. A domain or application base-URL change still needs the affected Helm upgrade/application rollout and DNS transition. |

The Kubernetes agent uses official `client-go`/`apimachinery`/Metrics API packages and the CoreDNS Caddyfile parser already in the project. The Infisical slice uses the official MCP Go SDK v1.6.0 for protocol/schema handling and the deployed Infisical v0.151.0 OpenAPI REST contract for Universal Auth and V4 name-only listing; SDK secret methods name V3 requests, so broader SDK coverage still needs a pinned contract check. The host DNS agent uses Go's standard library plus `kubectl`. The live `cluster-admin` binding is specific to this approved local installation; other installations retain the chart's read-only defaults.
