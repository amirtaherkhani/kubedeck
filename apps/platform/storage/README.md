# Platform Storage

Shared development dependencies for personal projects.

| Field | Value |
|---|---|
| Namespace | `platform-storage` |
| UI | Yes, multiple admin UIs |
| HTTPS | `*.local.dev` through Traefik and `local-dev-tls` |
| Storage | `local-path` PVCs |
| Configuration | Infisical; no credentials in Git |
| Observability | Prometheus metrics where supported; Alloy/Loki logs |

Services include PostgreSQL, pgAdmin, Redis, Redis Commander, RabbitMQ,
Kafka, Kafka UI, MongoDB, Mongoku, NATS, NATS UI, MinIO, Jaeger, Mailpit,
Apicurio Registry, and grpcui.

UI domains are defined in `values.yaml`; common routes include:
`pgadmin.local.dev`, `redis-ui.local.dev`, `rabbitmq-ui.local.dev`,
`kafka-ui.local.dev`, `mongo-ui.local.dev`, `nats-ui.local.dev`,
`nats-monitor.local.dev`, `s3-ui.local.dev`, `jaeger-ui.local.dev`,
`mailpit-ui.local.dev`, `schema-registry-ui.local.dev`, and
`grpcui.local.dev`.

Internal clients must use Kubernetes service DNS in the
`platform-storage.svc.cluster.local` namespace. macOS clients use the
documented `*.local.dev` DNS names.

Deploy with `make helm-validate RELEASE=platform-storage` and
`make helm-apply RELEASE=platform-storage` after Infisical-backed secrets are
available.

The `nats` Secret is synchronized from Infisical path
`/apps/platform-storage/nats`. In addition to the shared administrative and
cluster keys, it must contain `NATS_FINANCE_USERNAME` and
`NATS_FINANCE_PASSWORD`. That dedicated identity is limited to the
`vv.finance.internal.>` and `vv.finance.internal-dlq.>` data subjects plus the
specific JetStream API subjects required to inspect the two Finance streams,
manage Finance consumers, pull and acknowledge messages, and inspect the DLQ.
The Finance client uses NATS-generated request/reply inboxes, so its only
subscription permission is `_INBOX.>`; it has no subscription permission on
application data subjects. Narrowing that inbox tree further requires the
Finance client to configure a dedicated `inboxPrefix` first.

The development durable-source test identity is separate from the Finance
runtime identity. It is synchronized with `NATS_FINANCE_TEST_USERNAME` and
`NATS_FINANCE_TEST_PASSWORD`; its JetStream API permissions are limited to
`VV_FINANCE_INTERNAL_EVENTS_TEST` and
`VV_FINANCE_INTERNAL_EVENTS_TEST_DLQ`. The external Finance runner receives
the corresponding five-key contract from its own Infisical project at
`/finance/test-internal-events`.
