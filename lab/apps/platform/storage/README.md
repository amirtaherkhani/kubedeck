# Platform Storage

Shared development dependencies for personal projects.

| Field | Value |
|---|---|
| Namespace | `platform-storage` |
| UI | Yes, multiple admin UIs |
| HTTPS | `*.local.dev` through Traefik and `local-dev-tls` |
| Storage | `local-path` PVCs |
| Configuration | Infisical; no credentials in Git |
| Observability | Prometheus metrics, Grafana dashboards where supported; Alloy/Loki logs |

Services include PostgreSQL, pgAdmin, Redis, RabbitMQ,
Kafka, Kafka UI, MongoDB, Mongoku, NATS, NATS UI, MinIO, Jaeger, Mailpit,
Apicurio Registry, and grpcui.

UI domains are defined in `values.yaml`; common routes include:
`pgadmin.local.dev`, `rabbitmq-ui.local.dev`,
`kafka-ui.local.dev`, `mongo-ui.local.dev`, `nats-ui.local.dev`,
`nats-monitor.local.dev`, `s3-ui.local.dev`, `jaeger-ui.local.dev`,
`mailpit-ui.local.dev`, `schema-registry-ui.local.dev`, and
`grpcui.local.dev`.

Mailpit SMTP authentication is enabled for the local development listener.
The platform Infisical path `/apps/platform-storage/mailpit` supplies
`MAIL_SMTP_USER` and `MAIL_SMTP_PASSWORD`; the operator combines them into the
namespace-local `mailpit-smtp-auth` Secret key `auth`, and the Deployment passes
that value to Mailpit as `MP_SMTP_AUTH`. `MP_SMTP_AUTH_ALLOW_INSECURE=true` is
intentional while this local listener has no SMTP TLS certificate; do not expose
port 1025 outside the trusted home-lab network. Finance uses its own
`MAIL_USER` and `MAIL_PASSWORD` values from its external `vero-finance` project.

Internal clients must use Kubernetes service DNS in the
`platform-storage.svc.cluster.local` namespace. macOS clients use the
documented `*.local.dev` DNS names.

Deploy with `make helm-validate RELEASE=platform-storage` and
`make helm-apply RELEASE=platform-storage` after Infisical-backed secrets are
available.

Templated Infisical secrets such as Basic Auth `users` files include only the
rendered data key; ordinary application secrets continue to include all keys
from their Infisical path. Verify this contract with
`make platform-storage-test`.

MinIO's bucket/user bootstrap Job runs on the initial chart installation. It
does not run on routine upgrades; set `minio.initOnUpgrade=true` only when its
bucket or policy setup must be refreshed. Validate that lifecycle with
`python3 apps/platform/storage/tests/test_minio_hook.py`.

RabbitMQ is scraped from its built-in Prometheus endpoint on port `15692` by
the `rabbitmq` ServiceMonitor. The provisioned **RabbitMQ Overview** Grafana
dashboard appears in the `RabbitMQ` folder and covers scrape health,
connections, queue depth, message throughput, consumer backlog, memory, and
disk headroom. A separate bounded scrape of `/metrics/detailed` collects
per-queue data only for the Finance vhost `/vero-vault-finance`; it supports
queue-specific backlog, consumer capacity, storage, disk-I/O, virtual-host, and
exchange-topology panels without turning on high-cardinality metrics globally.
Broker-wide message throughput retains delivery, acknowledgement, and
redelivery visibility.
Finance application metrics (outbox, relay, inbox, retry, and DLQ behaviour)
remain owned by the Finance service dashboards.

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
`NATS_FINANCE_TEST_PASSWORD`; its publish permissions are limited to the
dedicated `vv.finance.test.internal.>` and
`vv.finance.test.internal-dlq.>` trees, and its JetStream API permissions are limited to
`VV_FINANCE_INTERNAL_EVENTS_TEST` and
`VV_FINANCE_INTERNAL_EVENTS_TEST_DLQ`. The external Finance runner receives
the corresponding seven-key contract, including its two test subject-prefix
values, from its own Infisical project at
`/finance/test-internal-events`.

Redis ACL-file loading and Redis authentication are intentionally disabled for
the local home lab. Redis has no Infisical-managed Secret or Kubernetes secret
mount, and clients connect through the in-cluster Service endpoint without
credentials. The legacy `/apps/platform-storage/redis` Infisical path is not
consumed by Kubernetes.
