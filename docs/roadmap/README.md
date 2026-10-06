# KubeDeck delivery roadmap

**Baseline:** [v0.2.0](https://github.com/amirtaherkhani/kubedeck/releases/tag/v0.2.0), released 2026-10-05. The current `main` also contains unreleased management-read and image-cleanup changes after that tag. The version numbers below are targets, not released versions.

**Current priority:** recover a usable, single-Mac development platform on Docker Desktop Kubernetes. The [live baseline](baseline-2026-10-06.md) and [migration record](../migration/rancher-to-docker-desktop.md) are inputs, not proof that workloads or client connections work.

| Target | Outcome | Must follow |
| --- | --- | --- |
| [v1.0.0](v1.0.0.md) | Docker Desktop services, secrets, storage, DNS, ingress, and dependent-project connections usable | v0.2.0 baseline and live inventory |
| [v1.1.0](v1.1.0.md) | Go host agent and complete local Kubernetes/Docker control capabilities | v1.0.0 stable access |
| [v1.2.0](v1.2.0.md) | Declarative service modules and independent tool lifecycle | v1.1.0 operations contract |
| [v1.3.0](v1.3.0.md) | Logs, volumes, builds, histories, and container metadata | v1.1.0 agents; v1.2.0 module identity |
| [v1.4.0](v1.4.0.md) | Repeatable installation on another Mac and enforceable GUI policies | v1.0.0 bootstrap; v1.1.0 host agent |
| [v2.0.0](v2.0.0.md) | Mature build/test/observe workflows and optional CI integrations | v1.2.0–v1.4.0 contracts |

The version sequence is an order of delivery, not a calendar promise. v1.0.0 is the release gate for actual dependent-project development; later versions must not delay it. A release can ship only after its acceptance evidence is recorded in [STATUS.md](STATUS.md). Cross-version work may proceed in parallel if it does not change the v1.0.0 critical path.

For v1.0.0, Phase 0 is inventory (V1-01); Phase 1 is data and Docker prerequisites (V1-02–03); Phase 2 is DNS/ingress (V1-04); Phase 3 is tool recovery (V1-05); Phase 4 is KubeDeck and client validation (V1-06–07); Phase 5 is release acceptance (V1-08). These phase numbers stay stable in status reports. Calendar dates for later releases are set after V1-01 measures actual service scope and V1-08 verifies recovery.

## Boundaries

```text
macOS host: browser + scoped resolver -> Technitium DNS -> Traefik ingress
            Host Agent (Go) -> Docker Desktop / Docker Engine / host network
                                        |
Docker Desktop Kubernetes: Traefik -> managed tools and client workloads
                           CoreDNS -> *.svc.cluster.local
                           Kubernetes Agent (Go) -> Kubernetes API
Web/desktop UI -> authenticated KubeDeck API -> agents
```

- Docker Desktop and its **built-in** Kubernetes are prerequisites. Docker Desktop may report its internal mode as `kind`; KubeDeck does not create or own a separate kind cluster.
- CoreDNS resolves Kubernetes services. Technitium owns Mac-local developer names. Traefik owns HTTP(S) ingress. These roles must remain distinct.
- The Kubernetes Agent manages cluster resources; the Host Agent manages macOS network state, Docker Desktop/Engine, and DNS reconciliation. Technitium is the DNS server, not a host-control agent.
- V1 is one local Mac and one Docker Desktop Kubernetes cluster. Other Macs must be installable from the same portable project; remote host enrollment and multiple clusters are later design decisions.
- Existing `lab/core/helm/releases.conf` remains the deployment source of truth until a reviewed module lifecycle replaces it. `ServiceModule` and `InstallationProfile` schemas are already present but do not deploy services by themselves.
- Finance and other applications are **external client workloads**. Their ownership, charts, secrets, and acceptance remain with those projects; KubeDeck exposes and validates their dependencies without silently absorbing them.
- Do not treat `Running` pods, `deployed` Helm releases, or a DNS answer alone as service acceptance. Verify the Mac-local path and application behavior.

## How to maintain this roadmap

1. Each task has a stable ID in its version file. Record dependency, deliverable, and observable acceptance there; do not renumber IDs.
2. Update its row in [STATUS.md](STATUS.md) when work starts or evidence changes. Use `Not started`, `In progress`, `Waiting`, `Blocked`, `Implemented`, `Verified`, or `Released`. `Implemented` means code/config exists; `Verified` requires the stated checks against the target; `Released` requires a published tag/release.
3. Append a dated entry to [HISTORY.md](HISTORY.md) for each meaningful transition, with commit/PR, target context, evidence location, and residual risk. Do not rewrite old entries to make a failed attempt disappear. Never record secret values.
4. Keep actual rollout commands, logs, and private environment data outside these public files. Link a safe evidence summary or commit. Refresh the live inventory before changing a workload or calling a gate complete.
5. For each release, compare the final product state with the previous tag. Update changelog and release notes using SemVer and include only externally relevant changes, migration steps, and configuration. A planned target is never a release claim.

## Related records

- [Requirements and product scope](requirements.md)
- [Module authoring contract](service-module-authoring.md)
- [Rancher to Docker Desktop migration and rollback gates](../migration/rancher-to-docker-desktop.md)

The older migration record includes historical observations and Rancher-specific commands. Reconcile them with the current Docker Desktop context before execution; preserve its rollback history.
