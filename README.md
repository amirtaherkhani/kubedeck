# KubeDeck agents

This repository contains two Go agents. The Kubernetes agent discovers cluster resources and exposes an internal API; the macOS host agent reconciles this Mac's `*.local.dev` DNS records through an existing Technitium installation. No dashboard or shared service deployment is included.

| Component | Purpose |
| --- | --- |
| [`kubedeck-agent/`](kubedeck-agent/README.md) | Kubernetes discovery, metrics, events, and optional authenticated management and CoreDNS aliases |
| [`kubedeck-agent/chart/`](kubedeck-agent/chart/README.md) | Standalone Helm chart for the Kubernetes agent |
| [`host-agent/`](host-agent/README.md) | macOS local DNS reconciler |

The cluster agent's service is internal and has no dashboard or public hostname. Its catalog parser fixtures live with its Go tests. The host agent depends on a separately managed Technitium service; this repository does not install Technitium or other platform services.

## Verify

```bash
(cd kubedeck-agent && go test ./... && go vet ./...)
(cd host-agent && go test ./...)
helm lint kubedeck-agent/chart --set image.repository=example.invalid/kubedeck-agent --set image.tag=0.2.0
helm template kubedeck-agent kubedeck-agent/chart --set image.repository=example.invalid/kubedeck-agent --set image.tag=0.2.0
```
