# Retained platform-storage volumes — 2026-10-10

Read-only metadata audit. All ten claims below are **Bound**, use storage class `standard`, have hostPath-backed PVs with reclaim policy **Delete**, and have no PVC owner references. Total requested/bound capacity: **60 GiB**. No contents were read, mounted, changed, relocated or deleted.

No Pod, Deployment, StatefulSet, DaemonSet, Job or CronJob in `platform-storage` currently references these claims. No current Helm release or running service was found there. This does not prove the data is unused or establish historical ownership; external/manual consumers were not verified.

| Claim | Capacity | Backing PV | Current consumer/controller | Ownership confidence |
|---|---|---|---|---|
| `data-kafka-0` | 8 GiB | `pvc-62bbc10e-4a81-4d0f-b4ab-4fa676e809eb` | None found | Unverified; Kafka label only |
| `data-minio-0` | 8 GiB | `pvc-8b05ff69-1dbc-408c-8f7f-00f24ff662d3` | None found | Unverified; MinIO label only |
| `data-mongodb-0` | 8 GiB | `pvc-ea394dcc-63da-49e5-b77f-9672a39d4b69` | None found | Unverified; MongoDB label only |
| `data-nats-0` | 4 GiB | `pvc-459fb6c4-4963-4921-aa68-54e91984b64e` | None found | Unverified; NATS label only |
| `data-nats-1` | 4 GiB | `pvc-89a6f069-2565-4f9f-adf8-22ca98893e47` | None found | Unverified; NATS label only |
| `data-nats-2` | 4 GiB | `pvc-580a303c-bac6-4e80-abcb-de56d4314f69` | None found | Unverified; NATS label only |
| `data-object-storage-0` | 8 GiB | `pvc-58233832-15e8-455b-a5ab-85176b7631e6` | None found | Unverified; object-storage label only |
| `data-postgresql-0` | 8 GiB | `pvc-22d73d43-d3a8-422a-835a-0e1dc67b9a0d` | None found | Unverified; PostgreSQL label only |
| `data-rabbitmq-0` | 4 GiB | `pvc-3395270b-1731-4e21-95d6-2b1659ca7571` | None found | Unverified; RabbitMQ label only |
| `data-redis-0` | 4 GiB | `pvc-0a29477b-c9d7-4f39-a338-edc7c889b8ae` | None found | Unverified; Redis label only |

## Recovery prerequisites

Before proposing reuse or removal, establish the owning project/operator, original engine/version and configuration, data consistency state, topology and relevant private credential/key references. Broker clusters and replicated systems require their original membership/recovery context; a PVC name is insufficient. Identify actual node-local backing and survival limits before any node/cluster rebuild.

`Delete` is the current reclaim policy: deleting a claim can lead to backing-volume deletion. Do not delete claims, change their binding or mount them into a new service to discover their contents as part of this audit.

Any later recovery work must have a separate concrete scope, follow [native local backup defaults](backup-defaults.md), and verify restoration in an isolated destination before data migration or destructive cleanup. No usable backup or successful restoration was established by this metadata-only audit.

## Infisical checkpoint — paused

The installed backend remains `infisical/infisical:v0.151.0`, 1/1 ready. The user paused upgrade-specific preparation, backups and migration until a future rescheduling; no date or rollout is set.

Preserved findings: the enrollment adapter and live login implementation currently accept v0.151.0 only; modeled refresh behavior for v0.166.3 is not complete upgrade compatibility. Previously identified duplicate-secret migration risks, preservation requirements and restorable-database prerequisites remain unresolved findings, not completed checks. The current audit did not establish a verified database dump/restore artifact. No further upgrade-specific preparation, backup rehearsal or database mutation was performed after the pause.
