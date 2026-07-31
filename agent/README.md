# KubeDeck Agent

KubeDeck Agent is a small Go service that runs once per Kubernetes cluster. It
watches the Kubernetes API, reads the resource Metrics API, builds a
dashboard-ready cluster snapshot, and streams updates to KubeDeck over
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
- Deployments, StatefulSets, and DaemonSets with desired/ready counts and the
  latest matching Pod creation time as `lastDeployedAt`
- Services, cluster DNS names, ClusterIPs, ports, matching Pods, EndpointSlice
  readiness, ingress URLs, uptime, and intelligent category assignment
- Ingress routes and TLS hosts
- PersistentVolumes and PersistentVolumeClaims
- CoreDNS service name, DNS service IP, ports, cluster domain, and search path
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
| `GET /v1/snapshot` | Current `kubedeck.io/v1alpha1` JSON snapshot |
| `GET /v1/events` | Reconnecting SSE stream |
| `GET /v1/dns/config` | CoreDNS management state and service aliases |
| `PUT /v1/dns/config` | Validate, preview, or replace managed service aliases |

The stream sends an initial snapshot, debounced resource-change snapshots,
metrics snapshots, and heartbeat comments. It supports the standard
`Last-Event-ID` header and replays a bounded in-memory history:

```text
retry: 3000

id: 42
event: snapshot
data: {"id":42,"name":"snapshot","clusterId":"homelab","sentAt":"...","data":{...}}
```

If `KUBEDECK_AGENT_TOKEN` is set, snapshot and stream requests require
`Authorization: Bearer <token>`. KubeDeck should keep that token server-side and
proxy the SSE stream to an authenticated browser session; native browser
`EventSource` cannot set an Authorization header.

## CoreDNS service aliases

CoreDNS does not expose a remote configuration CRUD API. KubeDeck therefore
uses the official Kubernetes client to update one dedicated key in the
CoreDNS custom ConfigMap, and the CoreDNS Caddyfile package to validate the
generated override. It never edits the K3s-owned main `Corefile`.

For K3s, the main Corefile imports `/etc/coredns/custom/*.override`. The Helm
chart can create `kube-system/coredns-custom`, and the agent owns only the
`kubedeck.override` key. Each alias is an exact CoreDNS rewrite to an existing
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
| `KUBEDECK_AGENT_LISTEN_ADDRESS` | `:8080` |
| `KUBEDECK_CLUSTER_ID` | `default` |
| `KUBEDECK_CLUSTER_NAME` | `Kubernetes` |
| `KUBEDECK_CLUSTER_DOMAIN` | `cluster.local` |
| `KUBEDECK_AGENT_TOKEN` | empty, internal endpoint unauthenticated |
| `KUBEDECK_DNS_MANAGEMENT_ENABLED` | `false` |
| `KUBEDECK_COREDNS_NAMESPACE` | `kube-system` |
| `KUBEDECK_COREDNS_CUSTOM_CONFIGMAP` | `coredns-custom` |
| `KUBEDECK_COREDNS_OVERRIDE_KEY` | `kubedeck.override` |
| `KUBEDECK_METRICS_INTERVAL` | `10s` |
| `KUBEDECK_REFRESH_DEBOUNCE` | `250ms` |
| `KUBEDECK_SSE_HEARTBEAT` | `15s` |
| `KUBEDECK_SSE_HISTORY` | `256` |
| `KUBEDECK_EVENT_LIMIT` | `100` |
| `KUBECONFIG` | in-cluster ServiceAccount configuration |

DNS management requires a non-empty `KUBEDECK_AGENT_TOKEN`; the agent refuses
to start with unauthenticated DNS writes. The Helm chart adds a namespaced Role
that can `get` and `update` only the configured custom ConfigMap. It does not
grant ConfigMap creation, Secret access, or node proxy access. If
`dnsManagement.createConfigMap=false`, create the ConfigMap separately before
using the write endpoint.

For local development:

```bash
KUBECONFIG="$HOME/.kube/config" \
KUBEDECK_CLUSTER_ID=local \
go run ./cmd/kubedeck-agent
```
