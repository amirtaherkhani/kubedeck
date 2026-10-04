# KubeDeck Agent

Internal discovery, metrics, events, SSE, and optional CoreDNS alias agent for
KubeDeck.

| Field | Value |
|---|---|
| Namespace | `development-tools` |
| UI | No; internal API only |
| Service | ClusterIP `8080` |
| DNS | Managed CoreDNS aliases through `core/dns` |
| Configuration | Infisical secret `kubedeck-agent-auth` |
| Observability | Readiness, liveness, and agent metrics |

The service is consumed by KubeDeck and is not exposed through an Ingress.
