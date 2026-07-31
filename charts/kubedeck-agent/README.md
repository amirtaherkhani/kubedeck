# KubeDeck Agent Helm chart

This chart deploys one discovery agent for a Kubernetes cluster. Discovery is
read-only. CoreDNS Service-alias writes are optional, disabled by default, and
limited to one custom ConfigMap. The agent Service stays internal; KubeDeck
consumes its snapshot, SSE, and DNS configuration endpoints through the cluster
network.

```bash
helm upgrade --install kubedeck-agent ./charts/kubedeck-agent \
  --namespace kubedeck \
  --create-namespace \
  --set image.repository=localhost:5001/kubedeck-agent \
  --set image.tag=IMMUTABLE_TAG \
  --set cluster.id=homelab \
  --set cluster.name=Homelab \
  --set auth.existingSecret=kubedeck-agent-auth
```

For bearer authentication, create a Secret separately and set:

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
  tokenKey: token
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
