# Namespace organization proposal

Status: architecture/proposal only. Inventory observed on 2026-10-10; no namespace migration, shared database deployment or new credential provisioning is authorized by this document.

The retained functional groups are `observability`, `platform-secrets`, `platform-tools`, `platform-networking` and `platform-tests`. Use **`platform-databases`** for proposed shared PostgreSQL and future database services, and **`platform-messaging`** for NATS, RabbitMQ, Kafka and similar brokers. Redis belongs with the function it actually serves; its proposed shared-service placement is not yet a deployed fact. Application workloads use the proposed `app-<project>` namespace convention when needed. With messaging added, these are eight functional groups, not a requirement to rename system namespaces or create every namespace now.

## Consolidated eight-group proposal

| Group / proposed namespace | Current location and observed workloads | Proposed responsibility |
|---|---|---|
| `observability` | `observability`: Grafana/renderer, Prometheus, Alertmanager, Loki, Tempo, Alloy and exporters | Monitoring, logs and traces; retain grouping. |
| `platform-secrets` | `platform-secrets`: Infisical, Operator, Infisical PostgreSQL and Redis | Secret management and its internal dependencies; retain grouping and data. |
| `platform-tools` | `development-tools`: KuchDesk agent; `platform-system`: local registry | Agents and developer infrastructure. |
| `platform-networking` | `technitium`: DNS; `platform-system`: Traefik and cert-manager | DNS, ingress and certificates. |
| **`platform-tests`** | Current live namespace remains **`observability-tests`**, containing k6 Operator | k6 and future general testing tools. Selected proposed name; no migration performed. |
| **`platform-databases`** | No running shared database workload verified in `platform-storage`; retained PVC inventory below | Shared PostgreSQL and future database services for managed projects, with engine-specific project isolation. Redis used as a data/cache service may belong here. This is a new service design, not permission to adopt old volumes. |
| **`platform-messaging`** | No running NATS, RabbitMQ or Kafka workload or matching Service/current Helm release found in the fresh inventory | Shared broker infrastructure and engine-specific project accounts/ACLs for NATS, RabbitMQ, Kafka and similar systems; proposed only. |
| Application workloads: `app-<project>` | Future project application workloads | Separate application lifecycles; consume scoped endpoints from shared database services instead of deploying an instance for every project by default. |

`kube-system`, `kube-public`, `kube-node-lease`, `local-path-storage` and `default` remain unchanged. The existing `platform-storage` namespace and all **10 bound PVCs** remain untouched pending ownership and recovery review.

## Shared database boundaries

Shared infrastructure does not mean shared application credentials or unrestricted data access. Configure an endpoint, logical database/key namespace, project identity and credential reference for each project and environment. Keep credentials in the established secret workflow, with separate rotation and revocation; provision none as part of this proposal.

| Engine | Proposed project boundary | Required design checks before deployment |
|---|---|---|
| PostgreSQL | Separate logical database per project/environment; distinct application roles and credentials; separate restricted migration/owner roles where needed | Restrict connection, schema and object privileges, including inherited and `PUBLIC` grants. Applications receive neither superuser nor role/database administration privileges. Verify denied cross-project access and backup/restore boundaries. |
| Redis | Separate ACL user and credential per project/environment; restricted key prefixes, Pub/Sub channel patterns and command allowlists | A numeric Redis database selected with `SELECT` is not a security boundary. Avoid unrestricted administrative/global commands. Verify every required client command and cross-project denial. Shared process, memory, persistence and availability remain common; choose a dedicated service when a workload needs stronger isolation. |
| Future engines | Native database, schema, tenant or equivalent boundary plus separate identities | Define engine-specific access, quotas, backup/restore, rotation and cross-project tests before adoption; do not assume PostgreSQL or Redis semantics apply. |

The PostgreSQL proposal follows its [privilege model](https://www.postgresql.org/docs/current/ddl-priv.html). Redis isolation uses [ACL users, command and key restrictions](https://redis.io/docs/latest/operate/oss_and_stack/management/security/acl/); [Redis's SELECT documentation](https://redis.io/docs/latest/commands/select/) cautions against using logical database numbers for unrelated applications. Logical access isolation does not provide separate resource capacity or failure domains.

Infisical's existing internal PostgreSQL and Redis **remain in `platform-secrets`**. Do not reuse their credentials, move their data, or expose them as the shared application service. Any later consolidation would require its own explicit design, backup/restore verification and migration approval.

## Messaging: current versus proposed

A fresh read-only cluster-wide inventory on 2026-10-10 scanned **91 Pods, StatefulSets, Deployments and Services**, plus the current Helm release listing. Names and container images were checked for NATS, RabbitMQ, Kafka and common related broker names. This is a bounded cluster inventory, not a check of externally hosted services.

| Service | Current namespace | Verified status | Proposed placement |
|---|---|---|---|
| NATS | None running found | No matching Pod, StatefulSet, Deployment, Service or current Helm release | `platform-messaging` if later deployed |
| RabbitMQ | None running found | No matching Pod, StatefulSet, Deployment, Service or current Helm release | `platform-messaging` if later deployed |
| Kafka | None running found | No matching Pod, StatefulSet, Deployment, Service or current Helm release | `platform-messaging` if later deployed |
| Infisical Redis | `platform-secrets` | `redis-master-0` Running and container ready; StatefulSet `redis-master` **1/1 ready**; `redis-master` and `redis-headless` Services exist | Remains an Infisical-internal dependency in `platform-secrets` |
| Future shared Redis | Not deployed/verified | No separate shared Redis workload found | Classify by actual role: data/cache under `platform-databases`; a dedicated messaging role may fit `platform-messaging` after design review |

Retained NATS/RabbitMQ/Kafka PVCs are not proof of running brokers. Old monitoring links are not runtime evidence. Infisical Redis is not treated as shared broker capacity, and its actual internal usage is not inferred solely from the Redis image.

The messaging design should specify per-project broker-native boundaries (accounts, vhosts, topics/subjects or equivalent), separate credentials and ACLs, and engine-specific cross-project denial tests. No broker, account, credential or permission is provisioned by this proposal.

## Existing ownership evidence

Read-only Kubernetes metadata and Helm inventory showed:

- `platform-storage`: no Deployments, StatefulSets, DaemonSets, Pods or Services returned; the current Helm listing returned no releases. This does not establish historical ownership or prove that the retained data is unused.
- All ten PVCs are `Bound`, have no owner references and no `meta.helm.sh/release-name` annotation. Labels provide historical hints only; actual data owners and consumers are **unverified**.
- `platform-secrets/infisical-postgresql` exists with no Helm release annotation. Repository manifests under `lab/apps/platform/infisical/manifests/postgresql.yaml` define the Infisical-specific database. Its Service selector identifies the PostgreSQL/database component; metadata does not establish shared-project ownership.
- `platform-secrets/redis-master` is labeled and annotated as belonging to Helm release `infisical`; its Service selector also targets that release.

| Retained PVC(s) in `platform-storage` | Application label | Instance label |
|---|---|---|
| `data-postgresql-0` | `postgresql` | `platform-storage` |
| `data-redis-0` | `redis` | `platform-storage` |
| `data-mongodb-0` | `mongodb` | `platform-storage` |
| `data-kafka-0` | `kafka` | `platform-storage` |
| `data-minio-0` | `minio` | `platform-storage` |
| `data-nats-0`, `data-nats-1`, `data-nats-2` | `nats` | `platform-storage` |
| `data-rabbitmq-0` | `rabbitmq` | `platform-storage` |
| `data-object-storage-0` | `object-storage` | `object-storage` |

The retained inventory also includes messaging and object-storage data. Those labels do not make every volume a database or assign it to `platform-databases`. No data contents or credentials were inspected.

## Before any migration

Review and approve a concrete migration plan covering PVC ownership, backups and restore checks; Helm release ownership; RBAC and service accounts; service DNS; certificates and network policies; Secret/Operator references; and Infisical authentication bound to namespace and service account. Specify shared-service resource limits, availability and per-project restore expectations.

Keep namespace names, service endpoints, database names and credential references configurable through Helm values and agent configuration. Apply supported reload behavior only where implemented; identity bindings and process settings may require restart. Do not hardcode proposed names into running services before migration approval.

System namespaces and the 10 bound `platform-storage` PVCs remain untouched. This document changes neither cluster configuration, permissions nor credentials.
