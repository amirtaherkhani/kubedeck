# Platform Storage

The `platform-storage` release provides the four shared services currently used by local projects: PostgreSQL, Redis, RabbitMQ, and NATS JetStream. It runs in the `platform-storage` namespace on Docker Desktop Kubernetes. Infisical synchronizes broker and database credentials; credentials are not committed to Git. Infisical itself keeps separate PostgreSQL and Redis backing stores in `platform-secrets`.

| Service | Purpose | Client endpoint | Data |
| --- | --- | --- | --- |
| PostgreSQL | Shared relational database server | `postgresql.platform-storage.svc.cluster.local:5432` | `data-postgresql-0` PVC |
| Redis | Shared cache and task coordination | `redis.platform-storage.svc.cluster.local:6379` | `data-redis-0` PVC |
| RabbitMQ | Message broker and management UI | `rabbitmq.platform-storage.svc.cluster.local:5672`; `https://rabbitmq-ui.local.dev` | `data-rabbitmq-0` PVC |
| NATS JetStream | Internal events and protected monitoring UI | `nats.platform-storage.svc.cluster.local:4222`; `https://nats-monitor.local.dev` | `data-nats-0`, `data-nats-1`, `data-nats-2` PVCs |

NATS monitoring uses Traefik Basic Auth. The `nats` and `nats-monitoring` Secrets are synchronized from separate Infisical paths. Finance uses a dedicated NATS identity and subject scope; its client credentials remain in Finance's own Infisical project. Redis is unauthenticated and restricted to a ClusterIP Service; do not expose it outside the cluster.

The chart no longer includes the unused admin UIs, Kafka, MongoDB, MinIO, tracing, mail testing, or schema registry. Historical PVCs from removed services are retained separately and must not be deleted by a normal release upgrade.

From `lab/`, validate with `make helm-validate RELEASE=platform-storage` and apply with `make helm-apply RELEASE=platform-storage`. Verify Pod readiness, service endpoints, persistence, the two HTTPS routes, and the application protocols after a change. `make platform-storage-test` checks the rendered Infisical Secret templates without revealing values.
