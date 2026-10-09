# Agent capability inventory

This is a capability and deployment inventory, not a grant of cluster-admin, host-root, or secret access. **Required** means the requested product capability; **opt-in** means a powerful operation must be explicitly enabled and scoped when implemented. Status distinguishes source code, current runtime, and a tested end-to-end operation.

## Kubernetes agent

The current chart is not installed on the checked Docker Desktop KIND cluster. Its source has unit tests, but no live agent endpoint was tested there. Defaults are read-only: `agent.managementEnabled=false`, `dnsManagement.enabled=false`, and `rbac.clusterAdmin=false`.

| Capability | Requirement | Source status | Runtime / gap |
| --- | --- | --- | --- |
| Nodes, Pods, Services, workloads, Ingress, storage, DNS and events inventory | Required | Snapshot/list-watch/SSE implemented | Agent not installed; no live endpoint proof |
| CPU and memory metrics | Required | Metrics API client implemented | metrics-server runs; agent collection not live-tested |
| Generic Kubernetes GET/list/watch/create/update/patch/delete/apply | Required, writes opt-in | Management API implemented for discovered resources; Secret resources excluded | Disabled; enabling currently requires bearer auth and cluster-admin RBAC |
| Logs, workload status, scale, restart | Required, actions opt-in | Bounded log and status/scale/restart endpoints plus in-process async scale/restart jobs implemented | Disabled and not live-tested; API acceptance is distinct from rollout health |
| CoreDNS exact service aliases | Required DNS capability, write opt-in | Managed Corefile block, dry run, resource-version check and rollback implemented | Disabled; live Corefile has no KuchDesk block; aliases are entered by explicit API request, not auto-discovered |
| Exec/attach/port-forward, Job lifecycle, Secret value operations | Required for full management | Not implemented as dedicated capabilities; generic resource operations do not cover these safely | Need transport, authorization and audit designs |
| Helm chart plan/apply/rollback and Infisical Secret readiness | Required | Not implemented | Must avoid secret values in rendered manifests and Helm history |
| Build image, publish to the local registry, then deploy/test a pinned image through Helm | Required | Not implemented as one workflow | Needs build context validation, immutable image digest, registry reachability from KIND, redacted Helm plan, rollout/health checks, and rollback; the current registry pull path is failing |
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
| Host network, route, DNS, connectivity and port inspection | Required | Only interface/default-route selection and DNS reconciliation in this agent | Host DNS probes and node_exporter are separate observability services; general inspection API absent |
| CPU, memory, disk, process and service monitoring/actions | Required | No host-agent management API | macOS node_exporter metrics are monitored separately; generic host actions absent |
| Infisical via CLI and MCP | Required | Shared typed name-listing service, CLI, stdio MCP, Universal Auth token lifecycle implemented in source | Not installed or authenticated live; admin, value delivery and Helm functions absent |
| DNS/router/firewall/process/service mutation | Required only where an explicit action is needed; always opt-in | No general mutation API | Requires scoped commands, confirmation, recovery, audit and tests before enablement |
| Build orchestration and cluster deployment from the Mac | Required | No host build/deploy command API | Docker build, registry push, Helm execution and tests need one scoped workflow with separate credentials and clear rollback boundaries |

The Infisical MCP server and management-enabled Kubernetes agent accept independent operations concurrently with bounded in-process execution, context-driven cancellation and timeouts, and conflict control for writes to the same target. This is an agent execution contract, not a separate task-manager service. The current host DNS job remains one-shot. The Infisical MCP server has in-memory start/status/cancel tools for name-only reads: at most four operations run concurrently, 64 records are retained, and each operation times out after 30 seconds. Same-key operations serialize inside its runner for future writes. The CLI runs synchronously. The Kubernetes agent has the same four-operation and 64-record bounds for scale/restart jobs; it serializes actions on the same workload. Each process loses job state on restart, and cancellation cannot undo a Kubernetes mutation already accepted. Build/deploy/test jobs are not yet implemented.

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

The Kubernetes agent uses official `client-go`/`apimachinery`/Metrics API packages and the CoreDNS Caddyfile parser already in the project. The Infisical slice uses the official MCP Go SDK v1.6.0 for protocol/schema handling and the deployed Infisical v0.151.0 OpenAPI REST contract for Universal Auth and V4 name-only listing; SDK secret methods name V3 requests, so broader SDK coverage still needs a pinned contract check. The host DNS agent uses Go's standard library plus `kubectl`. No blanket version upgrade or new live permission was made for this inventory.
