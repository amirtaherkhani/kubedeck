# KubeDeck agents and home lab

This repository contains two standalone agents and the configuration for the local Docker Desktop Kubernetes home lab. The KubeDeck web dashboard, its API proxy, database, assets, and Helm release have been removed.

| Component | Purpose | Runtime |
| --- | --- | --- |
| [`kubedeck-agent/`](kubedeck-agent/README.md) | Kubernetes discovery, snapshots, events, metrics, and optional authenticated management/CoreDNS aliases | Go service in Kubernetes; internal `:8080` API |
| [`host-agent/`](host-agent/README.md) | Reconcile this Mac's local DNS integration | Go service on macOS |
| [`lab/`](lab/README.md) | Shared cluster services, observability, DNS, TLS, agent chart, and local Infisical connector | Docker Desktop Kubernetes and macOS |

The two agent Helm charts are under [`charts/kubedeck-agent/`](charts/kubedeck-agent/README.md) and [`lab/apps/dev/kubedeck-agent/`](lab/apps/dev/kubedeck-agent/README.md). Service-module contracts, templates, and examples remain for the Go agent's catalog parser. The local Infisical connector at [`lab/tools/kubedeck-env-mcp/`](lab/tools/kubedeck-env-mcp/README.md) serves development agents and is independent of the removed UI.

## Verify the agents

```bash
(cd kubedeck-agent && go test ./... && go vet ./...)
(cd host-agent && go test ./...)
helm lint charts/kubedeck-agent
helm lint lab/apps/dev/kubedeck-agent
```

The Kubernetes agent's Service is internal and has no public hostname or dashboard. Its `/healthz`, `/readyz`, `/v1/snapshot`, and `/v1/events` endpoints are documented in the [agent API](kubedeck-agent/README.md). The host agent's installation and launchd checks are documented in its [README](host-agent/README.md).

The dashboard's old PVC and administrator Secret are not required by either agent. The `kubedeck-agent-auth` Secret, local registry, and shared platform services are separate resources; keep them when maintaining the agents.
