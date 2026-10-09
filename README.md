# KubeDeck agents, observability, and k6

This repository contains two Go agents, the Grafana observability stack, and k6 Operator. The Kubernetes agent discovers cluster resources and exposes an internal API; the macOS host agent reconciles a configured DNS wildcard through an existing Technitium installation. The KubeDeck dashboard is not included.

| Component | Purpose |
| --- | --- |
| [`kubedeck-agent/`](kubedeck-agent/README.md) | Kubernetes discovery, metrics, events, and optional authenticated management and CoreDNS aliases |
| [`kubedeck-agent/chart/`](kubedeck-agent/chart/README.md) | Standalone Helm chart for the Kubernetes agent |
| [`host-agent/`](host-agent/README.md) | Host DNS reconciler CLI and macOS installer |
| [`lab/`](lab/README.md) | Grafana, image renderer, Prometheus Stack, Loki, Tempo, Alloy, k6 Operator, and their HTTPS/secret dependencies |

The cluster agent's service is internal and has no dashboard or public hostname. Its catalog parser fixtures live with its Go tests. The host agent depends on a separately managed Technitium service. The files under [`lab/`](lab/README.md) are a local deployment profile; the agent binaries and chart do not depend on those hostnames or that cluster context. Unrelated home-lab services are not managed here.

## Portability

The Kubernetes agent runs in any compatible Kubernetes cluster when given a unique cluster identity and an available image. The host DNS reconciler CLI runs on macOS and Linux; only the installer uses macOS `launchd`. Its DNS zone, Kubernetes context, Technitium resources, and optional target IP are configuration. The `lab/` manifests are a separate profile for this local Docker Desktop cluster, not universal agent defaults.

## Verify

```bash
(cd kubedeck-agent && go test ./... && go vet ./...)
(cd host-agent && go test ./...)
helm lint kubedeck-agent/chart --set cluster.id=example --set cluster.name='Example cluster' --set image.repository=example.invalid/kubedeck-agent --set image.tag=0.3.0
helm template kubedeck-agent kubedeck-agent/chart --set cluster.id=example --set cluster.name='Example cluster' --set image.repository=example.invalid/kubedeck-agent --set image.tag=0.3.0
```
