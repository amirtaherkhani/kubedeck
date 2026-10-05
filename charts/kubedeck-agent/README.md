# KubeDeck Agent Helm chart

This chart deploys one discovery agent for a Kubernetes cluster. Discovery is
read-only. CoreDNS Service-alias writes are optional, disabled by default, and
limited to one custom ConfigMap. The agent Service stays internal; KubeDeck
consumes its snapshot, SSE, and DNS configuration endpoints through the cluster
network.

```bash
helm upgrade --install kubedeck-agent ./charts/kubedeck-agent \
  --namespace development-tools \
  --create-namespace \
  --set image.repository=localhost:5001/kubedeck-agent \
  --set image.tag=IMMUTABLE_TAG \
  --set cluster.id=homelab \
  --set cluster.name=Homelab \
  --set auth.existingSecret=kubedeck-agent-auth \
  --set auth.tokenKey=KUBEDECK_AGENT_TOKEN
```

By default, the bearer Secret is synchronized by Infisical from project
`home-lab`, environment `local`, path
`/apps/development-tools/kubedeck-agent`, using
`platform-secrets/infisical-universal-auth`. For a standalone installation,
create a Secret separately and set `tokenKey: token`:

```yaml
auth:
  existingSecret: kubedeck-agent-auth
  tokenKey: token
```

Configure the KubeDeck application with:

```yaml
agent:
  url: http://kubedeck-agent:8080
  existingSecret: kubedeck-agent-auth
  tokenKey: KUBEDECK_AGENT_TOKEN
```

For a standalone installation that uses the `token` key above, set the
application's `agent.tokenKey` to `token` as well.

## Cluster administrator access

The default ClusterRole allows discovery and metrics reads only. To grant this
agent's ServiceAccount full Kubernetes API access on a dedicated cluster, set:

```yaml
rbac:
  create: true
  clusterAdmin: true
serviceAccount:
  create: true
```

This binds the agent ServiceAccount to Kubernetes' built-in `cluster-admin`
ClusterRole. It can then read Secrets and change or delete resources in every
namespace. The chart requires a dedicated ServiceAccount in this mode. Keep
the bearer token Secret configured, limit access to the
internal agent Service, and review Kubernetes audit logs for this identity.
The current HTTP API still exposes snapshot/SSE reads and the separately
enabled, validated CoreDNS alias operation; this setting does not add generic
Kubernetes write endpoints. Any future destructive operation needs its own
authenticated API, authorization checks, dry-run/confirmation flow, and tests.

For Docker Desktop KIND, leave `dnsManagement.enabled=false`: the K3s
`coredns-custom` import is not part of KIND's default CoreDNS configuration.
Load or publish a `linux/arm64` agent image into the Docker Desktop Kubernetes
image store, and set `image.repository` and immutable `image.tag` to that image.
The agent starts without `metrics.k8s.io`; node and pod CPU/memory usage stay
unavailable until a compatible metrics-server is installed.
Verify the rendered binding before installing:

```bash
helm template kubedeck-agent ./charts/kubedeck-agent \
  --namespace development-tools --set rbac.clusterAdmin=true \
  --set image.tag=IMMUTABLE_TAG
```

## CoreDNS aliases

K3s mounts the optional `kube-system/coredns-custom` ConfigMap and imports
`*.override` files inside its main server block. Enable KubeDeck's dedicated
`kubedeck.override` key with:

```yaml
dnsManagement:
  enabled: true
  namespace: kube-system
  configMapName: coredns-custom
  overrideKey: kubedeck.override
  createConfigMap: true
```

When enabled, `auth.existingSecret` is mandatory. The chart creates a
namespaced Role with only `get` and `update` on the named ConfigMap; the agent
cannot create arbitrary ConfigMaps or write workloads and Secrets. Set
`createConfigMap: false` if another release already owns the custom ConfigMap.
