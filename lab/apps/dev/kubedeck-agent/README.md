# KubeDeck Agent

Internal discovery, metrics, events, SSE, and optional CoreDNS alias agent.

| Field | Value |
|---|---|
| Namespace | `development-tools` |
| UI | No; internal API only |
| Service | ClusterIP `8080` |
| DNS | Managed CoreDNS aliases through `core/dns` |
| Configuration | Infisical secret `kubedeck-agent-auth` |
| Observability | Readiness, liveness, and agent metrics |

The service is not exposed through an Ingress. Authorized in-cluster clients
can consume its API directly; see the [agent API](../../../../kubedeck-agent/README.md).
