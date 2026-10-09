# KuchDesk Agent Helm chart

This standalone chart deploys the Kubernetes agent as one internal cluster service. Discovery is read-only by default. It does not install an image registry, secret manager, DNS server, dashboard, or other shared services.

Supply an image already available to the Kubernetes runtime and use an immutable tag. Set a unique cluster ID and display name for each cluster; the chart has no home-lab identity baked in. Create the bearer token Secret in the target namespace before installation; the chart reads it but does not manage its lifecycle. For example, with a token obtained through your normal secret workflow:

```bash
kubectl create namespace development-tools --dry-run=client -o yaml | kubectl apply -f -
kubectl -n development-tools create secret generic kuchdesk-agent-auth \
  --from-literal=KUCHDESK_AGENT_TOKEN="${KUCHDESK_AGENT_TOKEN}"
helm upgrade --install kuchdesk-agent ./kuchdesk-agent/chart \
  --namespace development-tools \
  --set cluster.id=development \
  --set cluster.name='Development cluster' \
  --set image.repository=REGISTRY/kuchdesk-agent \
  --set image.tag=IMMUTABLE_TAG
```

Do not put the token in Helm values or Git. If the image is loaded directly into Docker Desktop Kubernetes, use its matching repository and tag. The chart does not require a local registry. Its `appVersion` tracks the Go binary; the chart `version` tracks chart changes.

Clients send `Authorization: Bearer <token>` to the internal Service. Limit which clients can reach it. `networkPolicy.enabled` is off by default; when enabling it, set `networkPolicy.ingressPodSelector` to select authorized client Pods.

The default ClusterRole allows discovery and metrics reads. CoreDNS alias writes and management are disabled by default. Management requires an explicit bearer Secret and `rbac.clusterAdmin=true`, granting the agent ServiceAccount full cluster access. Review the [agent API](../README.md) before enabling it. On Docker Desktop KIND, leave CoreDNS management disabled because the K3s `coredns-custom` import is unavailable.

The agent can start without `metrics.k8s.io`; CPU and memory usage remain unavailable until a compatible metrics-server is installed.

`service.targetPort` sets both the container port and the agent's HTTP listener. Use `agent.listenHost` only to change the bind host; the default listens on all interfaces. Remove any old `agent.listenAddress` override when upgrading from chart 0.5.x. Direct binary invocations also require `KUCHDESK_CLUSTER_ID` and `KUCHDESK_CLUSTER_NAME`; they no longer silently identify an unknown cluster as `default`.

When upgrading from chart 0.4.x, set `cluster.id` and `cluster.name` explicitly. Reuse the previous values for an existing installation so its identity does not change.

```bash
helm lint ./kuchdesk-agent/chart \
  --set cluster.id=example \
  --set cluster.name='Example cluster' \
  --set image.repository=example.invalid/kuchdesk-agent \
  --set image.tag=0.4.0
helm template kuchdesk-agent ./kuchdesk-agent/chart \
  --namespace development-tools \
  --set cluster.id=example \
  --set cluster.name='Example cluster' \
  --set image.repository=example.invalid/kuchdesk-agent \
  --set image.tag=0.4.0
```
