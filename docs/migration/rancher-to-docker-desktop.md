# Rancher Desktop to Docker Desktop Migration Roadmap

**Status:** Planning only. No migration, backup, runtime, DNS, or network changes have been executed.
**Target:** Docker Desktop with one local Kubernetes cluster, subject to host-feasibility and runtime-choice gates.
**Required developer DNS target:** Technitium.
**Keep:** Kubernetes CoreDNS for internal cluster DNS, Traefik for ingress, and cert-manager for TLS; verify each live role before cutover.
**DNS access scope:** local development on the current Mac only. The Mac running Docker Desktop opens `*.local.dev` through its scoped resolver.
**Current source state:** Rancher `rancher-desktop` node was read-only verified Ready on Kubernetes `v1.36.4+k3s1` (ARM64, containerd `2.3.2`). Docker Desktop app bundle is `4.93.0`; daemon/Kubernetes health remains unverified.

This roadmap is the operating record for future Codex/Nova sessions. It does not authorize starting either desktop, resetting or installing a cluster, backing up/restoring data, changing DNS/firewall settings, moving volumes, publishing a release, or deleting the old environment. Preserve Rancher and the old DNS chain as rollback resources until all migration gates pass. Never put secret values in this document or evidence.

## Checkpoint history

- **2026-10-04:** Read-only check verified `lima-rancher-desktop` Ready on `rancher-desktop` (`v1.36.4+k3s1`, ARM64, containerd `2.3.2`); this supersedes earlier Broken status. Docker Desktop app bundle reports `4.93.0`, but daemon/Kubernetes health is unverified. A prior session reported pending KubeDeck-folder changes and missing identity key names in the Infisical UI; this is a historical observation only, its exact observation timestamp is unavailable here, and current saved state has not been rechecked. Verify required key names through a metadata-only check before treating this as a current blocker; do not inspect values. No runtime, backup, DNS/network, or deletion changes were made.

## Migration record

Use these states in the tables below: `Not started`, `In progress`, `Blocked`, `Passed`, `Accepted`, and `Rolled back`. Update state only with evidence from the actual target. `Passed` means the stated checks succeeded; `Accepted` means the user accepted the gate. No step may be marked accepted by a source-file render alone.

| Gate | Current status | Owner | Required evidence to pass |
| --- | --- | --- | --- |
| Live Rancher inventory | **In progress** — Rancher node read-only verified Ready; complete inventory pending | Codex records bounded read-only inventory | Ready node and API/runtime plus complete inventory captured without reset |
| Infisical backup and isolated restore | Not started; mandatory before any old-volume loss | User selects protected backup destination; Codex performs approved backup/restore with secret values kept private | Encrypted backup outside Rancher VM, integrity checks, isolated restore and decryption validation |
| Docker Desktop Kubernetes profile | Not started | User chooses limits/flavor; Codex validates after authorization | Fresh target cluster, distinct context, Ready node, host capacity and limits/storage/network documented |
| Technitium Mac-local DNS | **Required target, placement unresolved** | User approves placement/listener scope; Codex configures after authorization | Mac scoped resolver and UDP/TCP DNS reach Technitium; required hostnames resolve to the current local ingress address and trusted HTTPS reaches each configured service |
| Per-service, Finance, KubeDeck acceptance | Not started | Codex verifies; user/app owner accepts Finance behavior | Dependency-ordered service checklist, measurable health/resource criteria and rollback evidence pass; required local service workflows and Infisical restore are verified |
| Legacy Rancher and old DNS retirement | Not started; gated on all prior acceptance | User authorizes destructive cleanup; Codex inventories and removes only confirmed obsolete items | Target stable through agreed rollback window; exact old resources removed; new CoreDNS, Traefik, CA trust, Technitium, and backups retained |

## Last-known baseline and feasibility (not a complete current inventory)

Treat older inventory below as historical until reconciled with read-only live state. A failed/stale endpoint is not evidence that a workload or record is absent. Before selecting a target profile, measure Mac CPU/RAM/disk headroom and current workload resource requests/limits and observed use; compare against Docker Desktop limits and the complete intended image/storage footprint.

| Area | Last recorded evidence | Migration implication |
| --- | --- | --- |
| Rancher Desktop | Version `1.24.0`; configured 28 GiB RAM, 10 CPUs, 100 GiB virtual disk; Rancher disk image about 82 GiB allocated on the Mac | Recover without reset. Capture current VM, cluster, process, and disk state before any change. |
| Kubernetes | Context `rancher-desktop`; last verified node was `lima-rancher-desktop`, ARM64, Kubernetes `v1.36.4+k3s1`; one node. The current API cannot be reached. | Re-inventory version, node readiness, namespaces, workload ownership, CRDs, RBAC, Services, ports, and Helm values. Do not assume Docker Desktop has compatible defaults. |
| Docker Desktop | Version `4.91.0` was the last recorded baseline. A read-only check during the user's in-progress update reported app bundle version `4.93.0`; daemon state and Kubernetes status were not queried, and this is not yet a post-update confirmation. The existing raw disk previously reported about 12 GiB allocated with a much larger logical capacity. Host has 48 GiB RAM. | After the user finishes the update, verify the installed version and settings before startup. The observed app bundle version matched the current [Docker Desktop release notes](https://docs.docker.com/desktop/release-notes/) at check time. Confirm entitlement, resources, disk, Kubernetes flavor/version, image store and host networking before creating a cluster. The old disk figures do not predict post-migration use. |
| KubeDeck | Last recorded app and agent version `0.1.2`; each was Ready 1/1 in the earlier snapshot. PR #15 proposed `0.1.3` but is a separate unresolved release and is not implicitly merged by this plan. | Use a verified merged commit and matching app/agent charts and immutable images. Do not deploy from an open PR or repair an API endpoint as part of this roadmap. |
| Platform and observability | Last read-only Helm inventory showed `alloy`, `caretta`, `cert-manager`, `grafana`, `infisical`, `infisical-operator`, `k6-operator`, `kubedeck`, `kubedeck-agent`, `loki`, `monitoring`, `platform-storage`, `tempo`, `traefik`, and `traefik-crd`. | Validate installed state and exact chart/image versions after recovery. Reinstall operators/CRDs before their custom resources. |
| KEDA | Source config declares chart `kedacore/keda` `2.20.2`, but its last install attempt rolled back. Source contains no app `ScaledObject`/`ScaledJob`; no current app depends on KEDA. | Optional capability, not a prerequisite for migration. Defer from the first target baseline unless a separate decision identifies an approved app-owned scaling resource. Do not call it deployed or delete source configuration here. |
| Namespaces and workloads | Earlier snapshot recorded 13 namespaces and KubeDeck dashboard/agent Ready. It also showed Vero Vault Finance API 3/3, file worker 1/1, RabbitMQ consumer and relay 4/4 each. Current health and exact namespace assignments are unknown. | Finance is an external client workload: preserve its own source/config and Infisical access. Record namespace, workload, image, dependency, and health evidence before cutover. |
| Images | Rancher runtime previously used containerd, with local images in the `k8s.io` namespace; previous Docker daemon was unavailable. The unauthenticated host registry on port 5001 was observed listening on all interfaces. | Capture image names, digests, platforms and source commits. Prefer verified pulls or Docker Desktop's Kubernetes image store; do not reuse the old all-interface registry endpoint without a separate security review. |

## Service categories and namespace proposal

This is a review proposal for catalog grouping and namespace boundaries. It does not authorize namespace creation, relabeling, Helm release changes, or moving workloads. Keep UI/service categories separate from Kubernetes namespaces: categories describe what a service does; namespaces describe ownership, lifecycle, access and policy boundaries. Do not infer one from the other.

| Existing source group or service | Proposed UI category | Existing namespace treatment for the first migration | Notes |
| --- | --- | --- | --- |
| PostgreSQL, MongoDB, Redis, MinIO, Kafka, NATS, RabbitMQ and related UIs | `data`, `cache`, `object-storage`, or `messaging` per component | Keep the current `platform-storage` release/namespace initially | One chart/release currently groups different products. Preserve Helm/PVC ownership; expose components separately in the catalog rather than splitting resources during migration. |
| Prometheus, Grafana, Loki, Tempo, Alloy, Caretta/VictoriaMetrics | `observability` | Keep `observability` and `observability-tests` as currently configured | Keep test/load tooling distinct from monitoring services. |
| KubeDeck, n8n, Plane, Temporal, Mailpit and developer UIs | `developer-tools` or `workflow` | Keep `development-tools` and existing source namespaces pending inventory | Group by purpose in the UI; do not rename or merge namespaces based on category. |
| Finance API, file worker, RabbitMQ consumers/relays | `external-application` / `backend` / `worker` | Preserve the verified Finance-owned namespace (`vero-vault-finance-load-test` was previously observed) | Finance remains an external client workload with separate source, ownership and Infisical access; do not import it into KubeDeck or the platform chart. |
| Infisical and its operator | `secrets-and-identity` | Keep `platform-secrets` | Treat as a protected stateful boundary; back up only Infisical PostgreSQL plus its original key/config, and complete isolated restore before any old data loss. |
| CoreDNS, Traefik, cert-manager; required Technitium | `networking`, `ingress`, or `certificate-management` | Keep distro-owned CoreDNS in `kube-system`; preserve existing Traefik/cert-manager namespaces pending inventory; Technitium placement remains undecided | Technitium is developer/local DNS; CoreDNS remains internal Kubernetes DNS. Do not place unrelated workloads in `kube-system`. |
| Cognee | `ai` | Keep `ai-tools` pending inventory | No namespace change implied. |
| KEDA | `platform-automation` | No install assumed; source config only | Defer unless an app owner identifies a required scaler and credential scope. |

Before any later namespace redesign, produce an explicit old-to-proposed namespace map from live ownership evidence and review cross-namespace dependencies: service FQDNs, NetworkPolicies, RBAC, Infisical sync scopes, ServiceMonitor/Prometheus selectors, Helm release ownership, PVCs, Ingress/DNS/TLS and the `kubedeck-env-mcp` URL. The first Docker migration should retain existing namespace names and Helm ownership wherever feasible; any exceptions need a separate review and rollback plan. No workloads have been moved as part of this roadmap.

### Candidate namespace map for the first Docker cluster

This is the initial mapping proposal, based on the checked-in namespace and release files. Verify every row against the recovered live cluster before accepting it. `Retain` means deploy the service into the same namespace and preserve its Helm release ownership; it does not mean move a live resource now.

| Current/source namespace or group | Initial Docker target | Examples | Decision |
| --- | --- | --- | --- |
| `platform-system` | `platform-system` | cert-manager; optional KEDA only if separately approved | Retain. CoreDNS remains Docker Kubernetes distribution-owned in `kube-system`. |
| `platform-storage` | `platform-storage` | PostgreSQL, MongoDB, Redis, MinIO, Kafka, NATS, RabbitMQ, Mailpit and related UIs | Retain the grouped release and PVC ownership for the first migration. Do not split the heterogeneous chart during cutover. |
| `platform-secrets` | `platform-secrets` | Infisical and secrets operator | Retain. Protected restore and decryption gate applies. |
| `observability` | `observability` | Prometheus stack, Grafana, Loki, Tempo, Alloy, Caretta/Radar | Retain. Validate monitoring selectors and scrape targets. |
| `observability-tests` | `observability-tests` | k6 operator and test resources | Retain separately from production-like observability services. |
| `development-tools` | `development-tools` | KubeDeck, KubeDeck agent, n8n, Plane, Temporal | Retain; verify actual release values and PVC ownership. |
| `ai-tools` | `ai-tools` | Cognee | Retain. |
| `kube-system` | Docker Kubernetes `kube-system` | Distribution CoreDNS | Do not replace or route internal cluster DNS through Technitium. |
| Finance namespace (previously observed as `vero-vault-finance-load-test`) | Same verified Finance-owned namespace | Finance API, file worker, RabbitMQ consumers/relays | Treat as external client workload; verify namespace and Finance owner before deploy. |
| Technitium (placement undecided) | Not selected | Developer/local DNS | Decide host/container versus a dedicated cluster namespace, bootstrap address and listener scope at Step 3. It must remain reachable when Kubernetes is stopped. |

This map intentionally leaves long-term consolidation open. A future category rename or namespace redesign needs a separate reviewed old-to-new map, workload owner, dependency assessment, and rollback plan; it must not be smuggled into the initial migration.

### Category and namespace decisions

- **In scope for review now:** approve or revise the UI taxonomy above and confirm the initial namespace-preservation principle before deployment.
- **Deferred:** exact long-term namespace consolidation, Technitium namespace/host placement, Traefik namespace, tenancy boundaries, and any service relocation. Record the chosen old-to-new map and owners before applying manifests.
- **Guardrail:** a category label or catalog descriptor must never itself grant Kubernetes permissions or cause a workload move.

## DNS, ingress, TLS and Mac-local access

Technitium is the **required final developer/local DNS service** for the current Mac running Docker Desktop. Configure the scoped `local.dev` resolver to reach it and resolve service names to the current local ingress address. The existing dnsmasq/CoreDNS `local.dev` forwarding chain is a rollback path only; it must not remain the final source of developer DNS records. CoreDNS itself remains in the target cluster for `*.svc.cluster.local` and Kubernetes DNS. Keep Traefik and cert-manager, but verify actual live routing/issuance roles before cutover.

### Existing local DNS contract

- User-facing hostname suffix is `local.dev`; the app configurations contain 23 distinct names: `cognee`, `grafana`, `grpcui`, `infisical`, `jaeger-ui`, `k6-dashboard`, `k6-live`, `kafka-ui`, `kubedeck`, `mailpit-ui`, `mongo-ui`, `n8n`, `nats-monitor`, `nats-ui`, `pgadmin`, `plane`, `rabbitmq-ui`, `radar`, `redis-ui`, `s3-ui`, `s3`, `schema-registry-ui`, `schema-registry`, and `temporal` (all with `.local.dev`). Preserve these hostnames initially so callbacks, CORS, dashboards, tools and service links continue to work.
- Current repo custom CoreDNS ConfigMap returns a legacy private address for the `local.dev` wildcard. `home-lab-dns` exposes CoreDNS over UDP/TCP 53 using a LoadBalancer Service. The repo describes Rancher Desktop forwarding web traffic to the Mac host and macOS Homebrew dnsmasq forwarding `local.dev` to that CoreDNS Service. Treat endpoint values as private preflight evidence, not public roadmap content.
- Prior read-only host-file inspection found a scoped `/etc/resolver/local.dev` route and Homebrew dnsmasq forwarding to a legacy private endpoint; `/etc/hosts` had no `local.dev` entries. This historical host snapshot does not establish current resolver/service health. Recheck the Mac's active interface, resolver path, port-53 listener and DNS answers in the private preflight before choosing target values; do not publish private IPs or carry legacy addresses forward blindly.
- `core/host/macos/configure-local-dev-resolver.sh` discovers the Rancher `home-lab-dns` ServiceLB address and writes `/etc/resolver/local.dev`; `configure-local-dev-dns-forwarding.sh` configures Homebrew dnsmasq and edits the project PF anchor references. `make macos-dns` runs both. The checked-in README's resolver flow and the actual host resolver file differ; reconcile this before cutover, but do not run the repair while Rancher is Broken.
- Traefik is Rancher/K3s-owned today. `core/ingress/traefik-helmchartconfig.yaml` redirects HTTP to HTTPS and selects the `local-dev-tls` default certificate. Ingress hosts live in each app's Helm values/templates; 23 names were found across the app YAML. Source does not establish Docker Desktop's reachable Traefik IP or host port mapping.
- cert-manager issues `local-dev-tls` for `local.dev` and `*.local.dev`. The CA private key is held in Kubernetes Secret `platform-system/local-dev-ca`; `core/host/macos/trust-local-dev-tls.sh` trusts the CA on macOS. Preserve the original CA key in the protected backup if keeping the same trust chain, or reissue a new CA and explicitly replace/trust it on macOS. Never put CA keys or certificate private keys in Git, this roadmap, or logs.
- KubeDeck's agent DNS endpoint is a separate internal feature. `GET/PUT /v1/dns/config` manages exact aliases in `kube-system/coredns-custom`, key `kubedeck.override`; it does not manage macOS, Technitium, Ingress or public/local host records. The root agent chart defaults writes off, but the homelab agent override currently sets `dnsManagement.enabled: true`. Live effective state is unknown. Keep CoreDNS for cluster DNS; disable this writer in the target unless a separately approved internal alias use case remains.

### Technitium acceptance criteria

1. Run Technitium with persistent configuration and a secure admin path from the Mac. Avoid a DNS bootstrap loop: keep its UI/API reachable by a fixed local bootstrap address while its own zone is being configured.
2. Configure Technitium as the Mac's authoritative local resolver for `local.dev`. Add the existing service records or a documented wildcard plus explicit exceptions, pointing to the current Docker Desktop/Traefik ingress endpoint. Do not carry forward old fixed IPs blindly. Forward unrelated DNS zones to the selected upstream.
3. Configure the current Mac's scoped resolver to use Technitium. Remove/disable the old dnsmasq forwarding path only in the final retirement phase, after a rollback window and after resolver tests pass. Store DNS administration credentials in an approved secret store, never in the repo or command output.
4. Verify UDP and TCP DNS queries, every required hostname, expected A records, non-local forwarding, behavior after Docker Desktop restart, and no dependency on the old `home-lab-dns` ServiceLB IP. Test macOS resolution with `dscacheutil` as well as direct `dig @<Technitium-address>`.
5. Keep `cluster.local` queries served by Kubernetes CoreDNS. Verify from a pod that core Kubernetes Service names resolve. Do not route cluster Service DNS through Technitium or remove CoreDNS.
6. Verify every existing Ingress still matches its unchanged hostname; confirm Traefik entrypoints, redirects, certificate chain, wildcard hostname coverage, macOS CA trust, and HTTPS responses. Reconfigure app URLs only if an existing hostname cannot be preserved, and update each callback/CORS/tool reference in the owning source/config.

## Data and configuration that must survive

### Infisical: mandatory restore gate

**Backup scope is deliberately limited to Infisical PostgreSQL and the material required to restore it:** the original `ENCRYPTION_KEY` plus exact version-specific auth/database configuration. Do not back up other database/service volumes as part of this migration. Other services get a documented fresh/reseed decision; do not delete any old volume during planning.

Infisical is the shared environment and secret source for Finance and KubeDeck. Its PostgreSQL database contains the encrypted values and organization/project/environment configuration. Its last-known storage included Infisical PostgreSQL and Redis PVCs, each requesting 8 GiB. The original Infisical encryption key is required to decrypt a restored database. Authentication/session and database configuration needed by the deployed version must also be preserved securely.

Before any Rancher VM, Infisical PVC, Secret, project or database data can be discarded:

1. Confirm the actual Infisical and secrets-operator versions/configuration and identify the original `ENCRYPTION_KEY` plus all version-specific restore-critical auth/database material. Do not generate replacement keys for a restore.
2. Back up the complete Infisical PostgreSQL database consistently, including every project, environment, folder, identity/policy and encrypted secret record. Preserve its original key/auth material and exact restore-critical configuration in an encrypted backup outside the Rancher VM and outside the repo. Select and record the protected destination before exporting anything. Do not back up other databases/PVCs under this plan.
3. Verify backup checksums and database archive readability. Restore a copy to an isolated, non-production target, supplying the original key/config privately. Compare project/environment identities and secret-name/count metadata, then perform an approved decryption/retrieval validation without printing or logging secret values.
4. Verify that Finance and KubeDeck environment paths, all required keys, Infisical Universal Auth identities, operator references, and Kubernetes sync resources work on Docker Desktop. Check status/metadata only in logs and record pass/fail; never include secret values, passwords, tokens, or key material here.
5. Retain the verified encrypted backup and original key material independently from both desktop VMs for the agreed retention period. A database dump without its original encryption key is not an accepted backup.

**Stop gate:** no Rancher, Infisical database/PVC, encryption key, authentication material, or backup copy may be removed until an isolated restore and decryption check passes and the user accepts the result.

### PVC and service data baseline

The last verified live inventory had 18 Bound `local-path` claims requesting 102 GiB total. These are requests, not measured use. Obtain exact claim names, capacity/actual bytes, owner, reclaim policy and backup decision from the recovered cluster; new PVCs on Docker Desktop do not carry Rancher data with them.

| Namespace | Last recorded PVC purpose and requested size |
| --- | --- |
| `development-tools` | KubeDeck D1 data, 1 GiB |
| `observability` | Grafana 10 GiB; Loki 10 GiB; Tempo 10 GiB |
| `platform-secrets` | Infisical PostgreSQL 8 GiB; Infisical Redis 8 GiB |
| `platform-storage` | Kafka 8 GiB; MinIO 8 GiB; MongoDB 8 GiB; NATS 3 × 4 GiB; PostgreSQL 8 GiB; RabbitMQ 4 GiB; Redis 4 GiB; Mailpit 1 GiB; NATS UI 1 GiB; pgAdmin 1 GiB |

The user has said non-Infisical volume contents may be recreated, but the per-volume data scope is not confirmed. Mark each claim `preserve`, `restore`, `reseed`, or `fresh` with its owner and evidence. Do not irreversibly drop any original claim or VM data until that decision is recorded; never treat a matching PVC name/size as data migration.

## Legacy cleanup inventory worksheet

Complete this worksheet from read-only inventory after Rancher recovers. Do not fill a location from memory or infer a path from an example. Record actual filesystem allocated bytes (for example, `du`/`stat` results and measurement method) separately from sparse-file apparent size. `TBD` is a stop state, not permission to remove an item.

| Candidate class | Exact path/resource, owner and references | Allocated bytes before | Decision and proof of Rancher-only ownership | Allocated bytes after / bytes returned |
| --- | --- | ---: | --- | ---: |
| Rancher Desktop app/support files and launch items | TBD after recovery | TBD | TBD | TBD |
| Rancher VM, virtual disk and Kubernetes/containerd state | TBD after recovery | TBD | TBD | TBD |
| Rancher-only images and image metadata | TBD after recovery; record digest/platform | TBD | TBD | TBD |
| Local registry on port 5001 and its stored image data | TBD after recovery; confirm listener, bind scope and clients | TBD | TBD | TBD |
| Rancher-only build caches, logs, diagnostics and temporary files | TBD after recovery; identify shared caches before excluding anything | TBD | TBD | TBD |
| Old Kubernetes contexts/generated config and Rancher-specific processes | Exact entries/process owners TBD | N/A | TBD | N/A |
| Legacy CoreDNS external `local.dev` alias and KubeDeck DNS writer | Exact ConfigMap key, Helm value and live consumers TBD | N/A | TBD | N/A |
| Homebrew dnsmasq local.dev config/service/PF references | Exact files, launch items and PF anchors TBD | N/A | TBD | N/A |
| macOS `/etc/resolver/local.dev` route | Confirm current contents and resolver usage at cleanup time | N/A | TBD | N/A |
| Rancher PVCs/volumes and Infisical state | Exact owner, claim, reclaim policy, backup and restore evidence TBD | TBD | Infisical isolated restore accepted; each other claim has user-approved disposition | TBD |

Only include measured bytes returned to the filesystem in the reclaimed-space total. Exclude target Docker Desktop data, shared caches, any files referenced by other processes/projects, and content whose owner or purpose is unclear. Retain the completed worksheet with the migration record before any final purge.

## Resource sizing, pod counts and sleep behavior

Default workloads to one pod. Use two replicas only for important services when the service is stateless or has validated HA/storage semantics and measured host capacity justifies them. Do not blindly set database replicas to two; two pods on the same Mac are not machine-level HA. Capture requests/limits and observed CPU/RAM before and after each rollout, and adjust only with measured evidence.

Docker Desktop [Resource Saver](https://docs.docker.com/desktop/use-desktop/resource-saver/) is compatible only when no containers are running; Kubernetes workloads mean the VM cannot sleep while still serving those services. Do not promise both VM sleep/Resource Saver and always-reachable services. Present separate essential always-on and nonessential on-demand profiles for user choice; test selected behavior with the installed version.

## Step-by-step execution plan

| Step | Status | Owner | Work and evidence required to accept | Rollback / failure handling |
| --- | --- | --- | --- | --- |
| 0. Freeze the boundary | **Passed for this document only** | User + Codex | Keep this work planning-only. No PR #15 merge, endpoint repair, DNS write, secret export, install, reset, or deletion is part of authoring this roadmap. | No runtime changes to roll back. |
| 1. Verify Rancher read-only without reset | **Partially passed** — node Ready verified; bounded full inventory pending | Codex performs authorized read-only checks | Capture full current inventory and host/runtime health without initializing, resetting, reinstalling or deleting the VM. Reconcile historical baseline with actual state. | If API/runtime is inaccessible or recovery threatens state, stop and retain VM/config unchanged. |
| 2. Complete source/live inventory | Not started | Codex; app owner confirms Finance scope | Capture a non-secret inventory of contexts, node architecture, namespaces, Helm releases/chart revisions, workloads, images/digests, ConfigMaps, Services/Endpoints/ports, Ingress/Traefik CRDs, actual TLS/cert-manager state, CoreDNS, CRDs/RBAC/NetworkPolicies/operators, PVC names/usage/reclaim policy, host ports/process/path references and the required DNS names. Add service purpose/owner/dependencies/config-name references, CPU/RAM/disk headroom, per-service resource requests/limits and observed use, the Mac's active interface/private IP, ingress bindings, Technitium resolver path, and host-to-service dependencies. Redact Secret data. Reconcile repository, Infisical references and live state. | Save the inventory outside ephemeral sessions but with no secret values. If source/live disagree, keep both facts and stop affected migration work until resolved. |
| 2a. Review service categories and namespace map | Not started | User reviews proposal; Codex records approved map | Review the proposal above against recovered namespace/workload ownership. Approve category labels and retain existing namespace/release ownership for initial migration, or record exact exceptions and dependency checks. This is a design review only; no resource moves. | If ownership or cross-namespace dependencies are unclear, retain the current names and defer redesign. |
| 3. Decide target and data scope | In progress; decisions open | User; Codex supplies evidence | Confirm Docker Desktop Kubernetes implementation/version, host feasibility, CPU/RAM/disk limits, entitlement, storage and Mac-local ingress; decide the local Technitium bootstrap/listener path; category/namespace map from Step 2a; CA reuse versus reissue; Infisical-only backup scope and per-service fresh/reseed choices (no other database backups); always-on versus on-demand profile; default one pod and reviewed replica exceptions; KEDA defer/include; maintenance window and rollback period. Technitium is required. | Do not provision until local target and storage choices are recorded. |
| 4. Build protected Infisical backup and prove restore | Not started — hard blocker before old data loss | User selects secure destination; Codex + Infisical owner | Back up Infisical PostgreSQL only with its original encryption key and restore-critical config. Do not back up any other database/service volume. Follow the Infisical gate for Finance/KubeDeck metadata/decryption checks. Record version, backup artifact identifier/checksum, encrypted destination, original key custody confirmation, isolated restore result and metadata/decryption checks only. | Restore failure: stop migration, preserve Rancher and original volumes, repair backup/restore procedure; never regenerate the key to bypass failure. |
| 5. Prepare Docker Desktop beside Rancher | Not started | User authorizes app/Kubernetes startup; Codex executes approved setup | Start Docker Desktop only after settings/entitlement are approved. Create a fresh, distinct Docker Kubernetes context; verify cluster version, single node Ready, runtime/image store, CNI, default storage class, measured resources and Mac-local host-port bindings. Default workloads to one pod; review each two-replica exception against HA semantics and host capacity. Keep Rancher installed and untouched. | Disable/stop only the new Docker Kubernetes cluster if checks fail; leave Rancher cluster and data unchanged. |
| 6. Install target platform foundations | Not started | Codex | Install pinned compatible CRDs/operators before dependent resources: CoreDNS is cluster-provided; install cert-manager, Infisical and its operator, Traefik/CRDs, storage provisioner and observability dependencies as the inspected manifests require. Apply only the reviewed Step 2a namespace map; initially preserve names and Helm ownership. Create scoped RBAC/network policies. Validate Helm renders and PVC bindings. KEDA is not required by current workloads; do not install it unless the decision gate includes an app-owned scaler. | Remove only newly created Docker resources if the fresh cluster fails; preserve Infisical backup and Rancher. Do not delete source definitions. |
| 7. Restore Infisical and establish Technitium | Not started | Codex + Infisical/DNS owner | Restore Infisical using the original key, verify secret sync for Finance/KubeDeck without revealing values, then establish Technitium with secure admin access, a local bootstrap path and the local zone. Configure required names to the discovered Docker/Traefik endpoint. Confirm UDP/TCP, forwarders, scoped Mac resolver, trusted HTTPS and restart persistence. | On failure, keep old dnsmasq/resolver files and Rancher DNS resources unchanged; do not point the Mac resolver at an unverified server. Restore the prior resolver only while the old path is healthy and verified. |
| 8. Restore ingress, TLS and service platform | Not started | Codex | Keep Traefik and cert-manager; recreate chart versions/CRDs and `IngressClass`, shared TLS secret issuance, redirects, DNS records and host ports. Install storage/brokers/database/monitoring releases in dependency order using pinned images/charts and verified values. Use the selected PVC policy per service; do not copy secret values to Helm overrides or Git. | Revert DNS/resolver clients to old service only if old path is healthy. Preserve target PVCs and backup while diagnosing; no old-volume deletion. |
| 9. Restore KubeDeck, tools and Finance client workloads | Not started | Codex; Finance owner validates | Deploy KubeDeck dashboard + agent from a verified merged commit with compatible immutable images. Confirm bearer-token wiring, read-only agent RBAC and intended DNS write setting. Reconnect `kubedeck-env-mcp` to the preserved `https://infisical.local.dev` URL and test with existing secure credential storage. Deploy Finance from its own approved source/config; verify API, file worker, RabbitMQ consumers/relays, database/broker connectivity and Infisical references. | Keep workloads in Rancher available. If target app validation fails, stop target traffic use; do not copy/rotate credentials as a workaround without a separate decision. |
| 10. Run cutover acceptance | Not started | Codex executes; user and Finance owner accept | Verify KubeDeck auth/UI, authenticated agent snapshot/SSE, all workload health, Endpoints/Ingress/ports, per-service dependency checklist, logs/dashboards, measured resource use versus requests/limits; run non-production Finance end-to-end/API and queue checks; verify Infisical project/environment/secret-name counts and successful private decryption checks; test all required Mac-local DNS names and HTTPS/TLS trust. Verify CoreDNS `cluster.local` service resolution from a pod. Capture exact Docker context/versions and results with no secret values. | Any critical check fails: stop cutover, preserve both clusters and all backups; record the failed test and retry only after correction. |
| 11. Hold rollback window | Not started | User chooses duration; Codex monitors | Keep Rancher VM, cluster, original Infisical PVC/key backup, old DNS files and CA trust as rollback assets through the agreed stable-use period. Watch Finance/KubeDeck health, DNS, certificates, storage and logs. | Roll back only to an available and verified Rancher path. Do not reset/remove Rancher while the rollback window is open. |
| 12. Retire old Rancher and DNS (conditional, destructive) | Not started — requires action-time approval | User approves exact removal set; Codex inventories/removes | After Infisical restore and all Mac-local service and DNS/TLS acceptance checks pass, the rollback window is accepted, and the user separately approves the exact cleanup list, remove only verified obsolete Rancher-specific items and legacy DNS. Inventory, by exact path/resource and owner: Rancher app/support files, VM, virtual disk, Kubernetes/containerd data, Rancher-only images and image metadata, local registry data on port 5001 if obsolete, Rancher-only build cache/logs, launch agents/processes, old `rancher-desktop` contexts and generated config; old `home-lab-dns` Service and legacy `local.dev` custom block; KubeDeck CoreDNS writer if unused; old Homebrew dnsmasq config/service/PF references and `/etc/resolver/local.dev` route. Check exact process/listener/path references and source/config consumers before each removal. Record both logical file size and allocated disk blocks for sparse VM images, image counts/sizes/digests, cache sizes, and PVC/volume data; compare pre/post using the same measurement method, report net allocated bytes actually returned by the filesystem, and state reclaimed capacity separately from logical VM capacity. Treat the old ~82 GiB figure as a prior observation, not reclaimable space. Preserve Docker Desktop's app/support data, VM/raw disk, images/volumes/cache; repositories/worktrees and shared caches; internal CoreDNS, Traefik/CRDs, cert-manager, Technitium, app hostnames, required Mac CA trust, unrelated DNS/PF/hosts settings, and encrypted Infisical backup/key material. Verify no process/listener/launch-agent, Helm value, project config, or local development consumer still depends on every removal candidate. | Prefer supported uninstall and a recoverable quarantine first. Do not `rm -rf` VM/application-support trees or broadly edit shared host files. Any irreversible purge, old-volume deletion, or backup/key retirement needs explicit approval at that time; retain rollback assets until approved retention expires. |

## Migration acceptance checklist

Do not declare migration complete until each required item has a dated result and owner acceptance:

- [ ] Host CPU/RAM/disk capacity and Docker Desktop limits are measured; per-service requests/limits and observed use are recorded before/after. Workloads default to one pod; any two-replica choice has explicit stateless/HA/storage and single-host capacity reasoning. Essential always-on versus on-demand services are chosen with Resource Saver behavior understood.
- [ ] Docker Kubernetes context is distinct, correct, Ready after restart, and has documented resource/disk/storage/network settings. CoreDNS resolves internal Service FQDNs; actual Traefik ingress and cert-manager issuer/renewal roles and health are verified.
- [ ] Technitium is the authoritative final developer DNS path; old dnsmasq/CoreDNS external forwarding is no longer required. All 23 existing `.local.dev` hostnames resolve to the current ingress endpoint over UDP and TCP, with non-local DNS forwarded correctly.
- [ ] All required HTTPS names return the expected certificate and redirect from the Mac after Docker Desktop restart; the Mac trusts the chosen CA and old Rancher IPs are not target endpoints.
- [ ] Infisical backup is outside both VM disks, integrity checked and restored with the original encryption/authentication material; all Finance/KubeDeck projects and environments are present and decryption validation succeeded without secret disclosure.
- [ ] Infisical PostgreSQL alone has an encrypted backup with original key/restore config and a successful isolated restore/decryption check. No other database/service PVC volume was backed up under this plan. Other services have documented fresh/reseed decisions; no volume is removed without separate approval.
- [ ] KubeDeck dashboard/agent and `kubedeck-env-mcp` are healthy and reach the expected Infisical/DNS services; the agent has only approved RBAC and authentication.
- [ ] The service category proposal and candidate namespace map were reviewed before deployment; the accepted old-to-target map is recorded, and live namespace/release ownership matches it. No unreviewed namespace move occurred.
- [ ] Finance API, file worker, broker consumers/relays, dependent data stores and required environment values pass owner-approved non-production checks.
- [ ] Observability, alerting and logs cover the migrated workloads; no secret values appear in logs or evidence.
- [ ] Rollback procedure was verified or its limitations accepted; the agreed rollback window ended before legacy cleanup.
- [ ] Old Rancher and old DNS cleanup is limited to the reviewed exact inventory; new core DNS/ingress/TLS/DNS services and protected backups remain intact. Pre/post allocated-byte measurements use the same method, distinguish sparse logical capacity from bytes actually freed, and make no unsupported reclaim estimate.

## Decisions still required

1. Docker Desktop Kubernetes implementation/version, resource limits, disk quota and entitlement; Mac capacity measurements and essential always-on versus on-demand profile.
2. Technitium placement, Mac-local bootstrap address, DNS zone/record strategy, forwarding upstreams, and local listener scope. Verify host port bindings and minimum local firewall requirements.
3. Infisical-only backup destination, original key custody/restore operators, retention period and Finance decryption verifier. Other database volumes are not part of this backup.
4. Reuse the original local CA versus issue a new CA and re-trust macOS; keep the same `*.local.dev` names either way.
5. Final data policy for each non-Infisical PVC; fresh data is allowed in principle but no old data may be destroyed until each item is confirmed.
6. Keep KEDA deferred for initial migration unless an owner defines a concrete `ScaledObject`/`ScaledJob` and its credential scope. No current app scaler is present in source.
7. Define service-specific measurable acceptance and rollback, app/Finance rollback window, and exact old Rancher/DNS cleanup approval after Mac-local DNS/TLS validation.
8. Review service categories and initial namespace map before deployment; record service dependency order, resource/replica exceptions, measurable acceptance and owners.

## Source references

- [Homelab DNS design](../../lab/core/dns/README.md), [custom CoreDNS ConfigMap](../../lab/core/dns/coredns-custom.yaml), [CoreDNS Service](../../lab/core/dns/service.yaml), [Mac DNS forwarding helper](../../lab/core/host/macos/configure-local-dev-dns-forwarding.sh), and [scoped-resolver helper](../../lab/core/host/macos/configure-local-dev-resolver.sh).
- [Traefik configuration](../../lab/core/ingress/traefik-helmchartconfig.yaml), [TLS issuers/certificates](../../lab/core/tls/clusterissuer.yaml) and [wildcard certificates](../../lab/core/tls/certificates.yaml), and [Mac CA-trust helper](../../lab/core/host/macos/trust-local-dev-tls.sh).
- [KubeDeck app values](../../lab/apps/dev/kubedeck/values.yaml), [KubeDeck agent values](../../lab/apps/dev/kubedeck-agent/values.yaml), [release inventory](../../lab/core/helm/releases.conf), and [platform storage chart](../../lab/apps/platform/storage/Chart.yaml).
- [Infisical deployment](../../lab/apps/platform/infisical/values.yaml), [Infisical operator](../../lab/apps/platform/infisical-operator/values.yaml), [KubeDeck agent DNS API contract](../../kubedeck-agent/README.md), and [`kubedeck-env-mcp`](../../lab/tools/kubedeck-env-mcp/README.md).


## Roadmap retirement condition

Only after the migration is complete, every roadmap step has passed, and the agreed tests pass may this roadmap file be removed from the working project, with its Git history retained and references updated. Until then, keep this roadmap as the operating record. This future condition does not authorize deletion now or deletion of any other document.
