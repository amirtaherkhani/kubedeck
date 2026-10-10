# Namespace organization proposal

Status: proposal only. Inventory observed on 2026-10-10; no namespace migration has been performed or approved as a whole.

The user selected **`platform-tests`** as the proposed destination for k6 and future general testing tools. The current live namespace remains **`observability-tests`**. This naming choice does not authorize moving its workloads or changing permissions.

| Current namespace and workloads | Proposed destination | Purpose / constraint |
|---|---|---|
| `kube-system`, `kube-public`, `kube-node-lease`, `local-path-storage`, `default` | Unchanged | Preserve system and default namespaces. |
| `technitium`: DNS; `platform-system`: Traefik and cert-manager | `platform-networking` | Group DNS, ingress and certificates. Proposed, not yet selected for migration. |
| `development-tools`: KuchDesk agent; `platform-system`: local registry | `platform-tools` | Group agents and developer infrastructure. Proposed, not yet selected for migration. |
| `platform-secrets`: Infisical, Operator, PostgreSQL and Redis | Unchanged | Keep the secret service and its dependencies together. |
| `observability`: Grafana/renderer, Prometheus, Alertmanager, Loki, Tempo, Alloy and exporters | Unchanged | Keep monitoring, logs and traces together. |
| `observability-tests`: k6 Operator | **`platform-tests`** | Selected proposed name for k6 and future general testing tools; live namespace unchanged. |
| `platform-storage`: no running workloads in the observed inventory, **10 bound PVCs** | Unchanged pending review | Preserve all data volumes; establish ownership and recovery requirements before considering changes. |
| Future application workloads | `app-<project>` when needed | Proposed convention; create only when a real application requires it. |

## Before any migration

Review and approve a concrete migration plan covering PVC ownership, backups and restore checks; Helm release ownership; RBAC and service accounts; service DNS; certificates and network policies; Secret/Operator references; and Infisical authentication bound to namespace and service account.

Keep namespace names configurable through Helm values and agent configuration. Apply supported reload behavior only where implemented; identity bindings and process settings may require restart. Do not hardcode the proposed names into running services before migration approval.

System namespaces and the 10 bound `platform-storage` PVCs remain untouched. This document changes neither cluster configuration nor permissions.
