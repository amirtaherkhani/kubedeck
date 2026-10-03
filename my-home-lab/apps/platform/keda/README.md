# KEDA

Event-driven workload autoscaler for the local Rancher Desktop cluster.

| Field | Value |
| --- | --- |
| Namespace | `platform-system` |
| Chart | `kedacore/keda` `2.20.2` |
| UI | No |
| HTTPS/domain | Not applicable |
| Observability | Prometheus ServiceMonitors for operator, metrics server, and webhooks |
| Configuration | Structural Helm values only; application trigger credentials remain in the owning application's Infisical-backed Secret |

KEDA supplies the `ScaledObject`, `ScaledJob`, and authentication CRDs. It does
not create or own application scale policies. Finance must add its RabbitMQ
`ScaledObject` only after its queue names, quorum topology, and dedicated
least-privilege connection contract are approved.

This single-node Rancher Desktop installation can exercise autoscaling but
cannot prove node-loss availability. RabbitMQ quorum HA requires a multi-node
development/test cluster with independently scheduled broker replicas and
durable volumes before it can be accepted.

Validate and deploy the operator with:

```bash
make helm-validate RELEASE=keda
make helm-apply RELEASE=keda
```
