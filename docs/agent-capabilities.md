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
| Pod exec | Required, opt-in | Noninteractive, bounded endpoint with exact confirmation, Pod UID/version check, bearer auth, output and timeout limits | Source and chart tests passed; the live release still has exec disabled, and no live Pod command was run |
| Pod port-forward | Required, opt-in | WebSocket binary tunnel to one running Pod TCP port with exact confirmation, Pod UID/version check, bearer auth, lifetime/idle/traffic/concurrency bounds | Source tests passed; disabled in the live release, with no live tunnel or agent rollout |
| Attach, Job lifecycle, Secret value operations | Required for full management | Not implemented as dedicated capabilities; generic resource operations do not cover these safely | Need separate transport, authorization and audit designs |
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
| Infisical via CLI and MCP | Required | Shared typed name-listing service, read-only CLI project-access check, stdio MCP, Universal Auth token lifecycle | Host CLI authenticated live; its known project is not listed and detail is forbidden for that identity. Admin and value-delivery functions are absent; no live MCP process is registered |
| DNS/router/firewall/process/service mutation | Required only where an explicit action is needed; always opt-in | No general mutation API | Requires scoped commands, confirmation, recovery, audit and tests before enablement |
| Build orchestration and cluster deployment from the Mac | Required | Local CLI and opt-in stdio MCP plan/start/status/cancel use the same typed profile workflow | Local registry transport and manual Helm rollout work; no live end-to-end apply through this workflow yet |

The Infisical MCP server and management-enabled Kubernetes agent accept independent operations concurrently with bounded in-process execution, context-driven cancellation and timeouts, and conflict control for writes to the same target. This is an agent execution contract, not a separate task-manager service. The current host DNS job remains one-shot. The Infisical MCP server has in-memory start/status/cancel tools for name-only reads: at most four operations run concurrently, 64 records are retained, and each operation times out after 30 seconds. Same-key operations serialize inside its runner for future writes. The Infisical and deployment CLIs run synchronously; the host MCP process can also run a configured deployment profile in its bounded runner with per-release serialization. The Kubernetes agent has the same four-operation and 64-record bounds for scale/restart jobs; it serializes actions on the same workload. Each process loses job state on restart, and cancellation cannot undo a Kubernetes mutation already accepted.

## Capability test evidence

| Capability | Source tests | Live result / remaining gap |
| --- | --- | --- |
| Kubernetes snapshot, metrics and bearer auth | Collector and HTTP tests, including unauthorized requests | Snapshot 200 with metrics; unauthenticated snapshot 401; extended SSE and long-running metrics behavior untested |
| Management resource and workload operations | Fake API tests for confirmation, resource versions, dry runs, error paths and async job bounds | Self-ServiceAccount `cluster-admin` verified; scale dry run and unchanged replica/version verified; real writes intentionally untested |
| Capabilities catalog | Discovery, Secret exclusion, complete rules-review and fallback access-review tests | After deployment of the rules-review build, the authenticated catalog returned HTTP 200 with 218 resources in 0.030 seconds for `development-tools`; the previous build took 39.78 seconds. Actual writes remain separately authorized. |
| CoreDNS aliases | Parser, idempotency, conflict, dry-run and preservation tests | GET and empty-alias dry run 200; Corefile version unchanged; no live alias write |
| Pod exec | Fake Pod and runner tests for disabled default, bearer auth, exact Pod/container identity, bounded output, concurrency, cancellation and timeout; chart lint and opt-in render passed | No live command, RBAC change, or agent rollout; live exec remains disabled |
| Pod port-forward | Fake Pod and net.Pipe tests for disabled default, bearer auth, exact Pod confirmation and identity, binary relay, traffic/lifetime bounds, and capacity release; chart lint and opt-in render passed | No live tunnel, RBAC change, or agent rollout; live port-forward remains disabled |
| Host DNS and Technitium | Interface and reconciliation tests | LaunchAgent last exit 0, existing zone and interface retained; no extended failure-injection run |
| Infisical MCP and Universal Auth | Mock transport tests for renewal, 401 retry, redaction, project-list/detail status, malformed responses, typed tools, bounded jobs, private project `.env` parsing, and explicit Keychain selection | Host CLI Universal Auth succeeded. `home-lab-nb0-k` returned `listed=false` and `detailStatus=forbidden` for the host identity; this does not mean the project is absent. No live MCP registration or cluster-agent Infisical Secret exists |
| Build, registry and Helm delivery | Fake command runner tests for push failure, digest pin, staged containerd/CRI failure, and MCP profile boundaries | Full CLI apply completed with digest-pinned image, Helm deployed revision 4, and the new agent Pod reached 1/1 Ready. Helm failure rollback and MCP apply remain untested live. |
| Deployment preflight | Injected Doctor tests for health, failure, warning, error redaction, and block-before-build behavior | Integrated live preflight passed all seven checks before build and Helm apply. |
| Doctor diagnostics and MCP-guided repair | Fake command, resolver, HTTP, disk, report-schema, prompt, plan-limit, unsupported-tool, redaction and failed-recheck tests | Live deterministic command passed route, DNS, Docker, KIND, registry, disk, metrics and two Deployments. MCP prompt and typed repair flow are source-tested; no AI-driven live action has run |

## Configuration and propagation

| Input or change | Current mechanism | Required apply/restart |
| --- | --- | --- |
| `lab/site.json` domain, hostnames or TLS issuer | Go renderer writes generated Certificates, TLSStore, Grafana/Infisical Helm values and DNS probe config | Render again; review/apply generated resources and upgrade affected Helm releases. Update Technitium zone and host-agent `-zone` separately. Domain changes need the new certificate Ready before switching Ingress/TLS. |
| Host DNS zone, context, interface, target IP, TTL or interval | Installer writes LaunchAgent arguments/environment; each one-shot run chooses the current IP unless `-target-ip` is set | Reinstall/rebootstrap the LaunchAgent for changed saved settings. IP changes are detected on the next interval; a new subdomain under the wildcard needs an Ingress route but no new wildcard record. |
| Kubernetes agent identity, listener, cluster domain, auth reference, DNS/management flags | Helm values become Deployment environment and RBAC resources | Helm upgrade and Pod rollout. The flags are not runtime-watched. `cluster.domain` defaults to `cluster.local`; the DNS target ConfigMap defaults to `kube-system/coredns` key `Corefile`. |
| CoreDNS alias list | Explicit authenticated `PUT /v1/dns/config` with full list, current resource version and optional dry run | When management is enabled, the agent updates only its marked ConfigMap block; CoreDNS `reload` applies it without a Pod restart. No automatic Service-to-alias watcher exists. |
| Infisical MCP bootstrap pair | Default project-local `.env`, or explicitly selected macOS Keychain generic-password item; `KUCHDESK_PROJECT_ROOT` can identify the checkout | Restart MCP process after backend selection or credential rotation. No backend fallback or migration; bootstrap credential lifetime must be verified. |
| Infisical Operator-synced Kubernetes Secret | Operator resync updates the native Secret | Existing Pod environment variables require a rollout; Secret volumes can update but the application must reload. Existing Grafana sync has no automatic redeploy target. |
| Certificate renewal | cert-manager updates Secret, Traefik reads it | No manual Traefik restart for routine renewal. A domain or application base-URL change still needs the affected Helm upgrade/application rollout and DNS transition. |

The Kubernetes agent uses official `client-go`/`apimachinery`/Metrics API packages and the CoreDNS Caddyfile parser already in the project. The Infisical slice uses the official MCP Go SDK v1.6.0 for protocol/schema handling and the deployed Infisical v0.151.0 OpenAPI REST contract for Universal Auth and V4 name-only listing; SDK secret methods name V3 requests, so broader SDK coverage still needs a pinned contract check. The host DNS agent uses Go's standard library plus `kubectl`. The live `cluster-admin` binding is specific to this approved local installation; other installations retain the chart's read-only defaults.
