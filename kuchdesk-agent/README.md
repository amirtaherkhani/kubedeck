# KuchDesk Agent

KuchDesk Agent is a small Go service that runs once per Kubernetes cluster. It
watches the Kubernetes API, reads the resource Metrics API, builds a
cluster snapshot, and streams updates to authorized clients over
Server-Sent Events (SSE). Discovery is read-only. An optional, disabled-by-
default CoreDNS module can manage exact internal aliases for Kubernetes
Services.

The agent talks to the Kubernetes API server, not directly to every node. This
is the safest and most portable approach: `client-go` maintains efficient
list/watch caches while metrics-server already aggregates CPU and memory from
each kubelet.

## Package choices

- [`k8s.io/client-go`](https://github.com/kubernetes/client-go) provides the
  official typed Kubernetes client, discovery client, shared informers, listers,
  watch reconnection, and local caches.
- [`k8s.io/metrics`](https://github.com/kubernetes/metrics) provides the typed
  `metrics.k8s.io/v1beta1` client used for node and pod CPU/memory snapshots.
- [`github.com/coredns/caddy/caddyfile`](https://github.com/coredns/caddy)
  provides CoreDNS's maintained Corefile lexer and parser. It validates the
  generated override before the Kubernetes API is updated.
- Go's [`net/http`](https://pkg.go.dev/net/http) provides the SSE server,
  streaming flush support, connection cancellation, and production HTTP
  timeouts without another runtime dependency.

The module pins the Kubernetes packages to `v0.36.3`, matching Kubernetes
`v1.36.x`. Keep the `client-go`, API machinery, API, and metrics module minor
versions aligned when upgrading.

## Collected data

- Nodes, readiness conditions, roles, addresses, versions, capacity, pod
  allocation, CPU, memory, and requested ephemeral storage
- Pods, containers, readiness, restart counts, owners, IPs, images, and
  CPU/memory
- Deployments, StatefulSets, and DaemonSets with desired/ready counts, selectors,
  and the latest matching Pod creation time as `lastDeployedAt`
- Services, cluster DNS names, ClusterIPs, ports, matching Pods, EndpointSlice
  readiness, related workload references, ingress URLs, uptime, and intelligent
  category assignment
- Ingress routes and TLS hosts
- PersistentVolumes and PersistentVolumeClaims
- CoreDNS service name, DNS service IP, ports, endpoint readiness, cluster
  domain, and search path
- Recent Kubernetes events for the notification surface

Storage percentage is explicitly
`ephemeralStorageRequestPercent`: Kubernetes Metrics API does not publish node
filesystem usage. The agent intentionally does not request broad `nodes/proxy`
permission to scrape kubelet Summary APIs.

## HTTP and SSE contract

| Endpoint | Purpose |
| --- | --- |
| `GET /healthz` | Process liveness |
| `GET /readyz` | Informer cache readiness |
| `GET /v1/snapshot` | Current `kuchdesk.io/v1alpha1` JSON snapshot |
| `GET /v1/events` | Reconnecting SSE stream |
| `GET /v1/dns/config` | CoreDNS management state and service aliases |
| `PUT /v1/dns/config` | Validate, preview, or replace managed service aliases |
| `GET /v1/manage/capabilities?namespace=apps` | Served resources and subresources, their verbs, point-in-time authorization checks, and Metrics API availability |
| `GET/POST/PUT/PATCH/DELETE /v1/manage/resources/{group}/{version}/{resource}` | Bounded Kubernetes resource operations; use `core` for the core API group |
| `GET /v1/manage/pods/{namespace}/{name}/logs` | Up to 1 MiB and 1,000 tail lines of pod logs |
| `GET /v1/manage/events/{namespace}` | Up to 100 namespace events |
| `GET /v1/manage/workloads/{kind}/{namespace}/{name}/{action}` | Deployment/StatefulSet status or scale; DaemonSet status |
| `POST /v1/manage/workloads/{kind}/{namespace}/{name}/{action}` | Deployment/StatefulSet scale, restart, status; DaemonSet restart/status |

Management is disabled by default with `KUCHDESK_MANAGEMENT_ENABLED=false`.
It requires a non-empty bearer token and an intentionally granted Kubernetes
ServiceAccount role. The Helm chart requires `rbac.clusterAdmin=true` when
management is enabled. Clients must authenticate directly to the internal
agent API; avoid caching management responses.

Resource operations use discovery for served resource scope. `GET` supports
`name` or a paginated list (`continue`), and a bounded newline-delimited watch
with `watch=true&resourceVersion=...`. Namespace-scoped writes require
`namespace`; `PUT` requires `metadata.resourceVersion`; merge `PATCH` and
server-side apply require `If-Match` with the resource version. Apply also
requires `fieldManager`. Write operations require exact
`X-KuchDesk-Confirm: namespace/name`; `DELETE` additionally requires
`If-Match-UID` and `If-Match` with the current resource version. Pass
`dryRun=true` to ask the API server to validate without
persisting a write. Scale accepts JSON `replicas` (0–100) and
`resourceVersion`; restart requires `If-Match` and patches only the workload's
pod-template restart annotation. The API server still enforces its own RBAC.
Capabilities include `allowedVerbs` from SelfSubjectAccessReviews for the
agent ServiceAccount. Pass a concrete `namespace` to evaluate namespaced
resources; without one, their `allowedVerbs` are empty. Authorization results
are informational snapshots, not grants, and Kubernetes checks every actual
operation again. `GET` workload status returns the typed workload object;
`GET` scale returns the typed Scale object with its resource version.

Raw Secret resources are excluded from generic discovery and operations. Logs,
events, ConfigMaps, and other resources can contain sensitive content; limit
agent API client access and avoid saving these responses. Audit logs
record action metadata and request IDs, not request or response bodies.
Exec, attach, port-forward, Helm operations, and generic subresource writes
are not implemented; each needs separate streaming and authorization design.

The stream sends an initial snapshot, debounced resource-change snapshots,
metrics snapshots, and heartbeat comments. It supports the standard
`Last-Event-ID` header and replays a bounded in-memory history. When the client
falls behind that history, the agent sends only the newest full snapshot so
stale replay cannot roll a client backward:

```text
retry: 3000

id: 42
event: snapshot
data: {"id":42,"name":"snapshot","clusterId":"homelab","sentAt":"...","data":{...}}
```

If `KUCHDESK_AGENT_TOKEN` is set, snapshot and stream requests require
`Authorization: Bearer <token>`. Keep the token in a trusted client process;
native browser `EventSource` cannot set an Authorization header.

## CoreDNS service aliases

CoreDNS does not expose a remote configuration CRUD API. The agent therefore
uses the official Kubernetes client to update one dedicated key in the
CoreDNS custom ConfigMap, and the CoreDNS Caddyfile package to validate the
generated override. It never edits the K3s-owned main `Corefile`.

For K3s, the main Corefile imports `/etc/coredns/custom/*.override`. The Helm
chart can create `kube-system/coredns-custom`, and the agent owns only the
`kuchdesk.override` key. Each alias is an exact CoreDNS rewrite to an existing
Service:

```text
rewrite stop name exact grafana.home.arpa grafana.monitoring.svc.cluster.local
```

Example request:

```json
{
  "resourceVersion": "18",
  "aliases": [
    {
      "hostname": "grafana.home.arpa",
      "service": "grafana",
      "namespace": "monitoring"
    }
  ],
  "dryRun": false
}
```

First `GET /v1/dns/config`, then pass its opaque `resourceVersion` to `PUT`.
This provides optimistic concurrency, so two administrators cannot silently
overwrite each other's changes. Set `dryRun: true` to validate and preview the
rendered override without updating Kubernetes.

The agent validates DNS names, confirms every target Service exists, rejects
duplicate aliases and aliases that shadow native `*.svc.<cluster-domain>`
records, refuses to replace unrecognized content in its key, and publishes a
`dns.config.changed` SSE event after a successful write.

This feature configures internal cluster DNS only. Public DNS and Ingress host
records should be managed with an authoritative DNS provider, typically
through `external-dns`.

## Configuration

| Environment variable | Default |
| --- | --- |
| `KUCHDESK_AGENT_LISTEN_ADDRESS` | `:8080` |
| `KUCHDESK_CLUSTER_ID` | required; unique, stable cluster identity |
| `KUCHDESK_CLUSTER_NAME` | required; human-readable display name |
| `KUCHDESK_CLUSTER_DOMAIN` | `cluster.local` |
| `KUCHDESK_AGENT_TOKEN` | empty, internal endpoint unauthenticated |
| `KUCHDESK_DNS_MANAGEMENT_ENABLED` | `false` |
| `KUCHDESK_COREDNS_NAMESPACE` | `kube-system` |
| `KUCHDESK_COREDNS_CUSTOM_CONFIGMAP` | `coredns-custom` |
| `KUCHDESK_COREDNS_OVERRIDE_KEY` | `kuchdesk.override` |
| `KUCHDESK_METRICS_INTERVAL` | `10s` |
| `KUCHDESK_REFRESH_DEBOUNCE` | `250ms` |
| `KUCHDESK_SSE_HEARTBEAT` | `15s` |
| `KUCHDESK_SSE_HISTORY` | `256` |
| `KUCHDESK_EVENT_LIMIT` | `100` |
| `KUBECONFIG` | in-cluster ServiceAccount configuration |

DNS management requires a non-empty `KUCHDESK_AGENT_TOKEN`; the agent refuses
to start with unauthenticated DNS writes. The Helm chart adds a namespaced Role
that can `get` and `update` only the configured custom ConfigMap. It does not
grant ConfigMap creation, Secret access, or node proxy access. If
`dnsManagement.createConfigMap=false`, create the ConfigMap separately before
using the write endpoint.

For local development:

```bash
KUBECONFIG="$HOME/.kube/config" \
KUCHDESK_CLUSTER_ID=local \
KUCHDESK_CLUSTER_NAME='Local cluster' \
go run ./cmd/kuchdesk-agent
```

Run the complete agent verification suite with race detection and static
analysis:

```bash
go test -race ./...
go vet ./...
```
