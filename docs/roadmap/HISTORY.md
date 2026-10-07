# Roadmap decision and execution history

Append new entries. Record observation, implementation, target verification, merge, and release as separate events. Do not put credentials, secret values, private environment exports, or raw logs here.

| Date (Asia/Tehran) | ID | Transition | Evidence | Open item |
| --- | --- | --- | --- | --- |
| 2026-10-06 | Baseline | GitHub v0.2.0 selected as latest released version; `main` has four newer commits | [GitHub release](https://github.com/amirtaherkhani/kubedeck/releases/tag/v0.2.0), `git rev-list --count v0.2.0..origin/main` = 4 at inspection | Later commits must be included in the next release comparison |
| 2026-10-06 | V1-01 | Partial read-only Docker Desktop and Kubernetes inventory captured | [Baseline snapshot](baseline-2026-10-06.md) | Service functionality, DNS access, data restore, and consumer connectivity remain unverified |
| 2026-10-06 | V1-03/V1-04 | In progress: Docker host exposure, Technitium zone, and dynamic DNS reconciler implemented | [Live recovery record](v1-recovery-2026-10-06.md); Docker Desktop; Infisical HTTPS 200; Technitium UDP/TCP answer; launchd first run 0 | Port 53 occupied by legacy dnsmasq; TLS/LAN client/restart gates open |
| 2026-10-06 | V1-02/V1-05 | Dependency found: universal-auth Secret absent and Infisical connector fetch fails | [Live recovery record](v1-recovery-2026-10-06.md); live Kubernetes Secret/CR inventory | Restore and validate development secrets before deploying dependent charts |
| 2026-10-06 | V1-04 | Mac DNS/HTTPS path verified after legacy port-53 handoff | [Live recovery record](v1-recovery-2026-10-06.md); direct UDP/TCP 53, macOS resolver, HTTPS 200 without TLS bypass | Router/client and IP-change gates remain open; old dnsmasq files retained |
| 2026-10-06 | V1-02/V1-05 | Export values matched live Infisical targets and new connector/Operator auth worked | [Live recovery record](v1-recovery-2026-10-06.md); 51/51 and 347/347 private comparisons; 19 CR sync statuses | Full DB/key backup gate and workload checks still open |
| 2026-10-06 | V1-05 | Core tool profile narrowed after image pull failures; unauthenticated Redis LAN listener removed | [Live recovery record](v1-recovery-2026-10-06.md); Redis in-Pod `PONG`, Service `ClusterIP`, host listener absent | PostgreSQL/RabbitMQ/NATS readiness, S3-compatible storage and external client checks remain open |
| 2026-10-06 | V1-05 | PostgreSQL, RabbitMQ and NATS became Ready; authenticated cluster and Mac client protocol checks passed | [Live recovery record](v1-recovery-2026-10-06.md); PostgreSQL query, RabbitMQ HTTPS API, NATS `PONG` | Restore compatible S3 and prove application workflows/persistence |
| 2026-10-06 | V1-05 | Rejected incompatible PostgreSQL image change and restored the original image without deleting its PVC | [Live recovery record](v1-recovery-2026-10-06.md); existing PostgreSQL 18 data and Ready Pod | Back up and plan migration before any image family or major version change |
| 2026-10-06 | V1-04/V1-06 | Traefik namespace watch expanded; KubeDeck Agent/dashboard and Metrics Server became Ready | [Live recovery record](v1-recovery-2026-10-06.md); trusted dashboard login, snapshot, SSE, `kubectl top` | LAN client DNS, application workflows, and restart gates remain open |
| 2026-10-06 | V1-05 | Separate SeaweedFS S3 module deployed without changing the retained MinIO PVC; HTTPS API contract and Pod restart readback passed | [Live recovery record](v1-recovery-2026-10-06.md); `object-storage` Helm release, versioned checksum/SSE/tag/multipart checks | Validate backup/restore, external project workflow, and a second LAN client |
| 2026-10-06 | V1 scope | User confirmed V1 DNS and HTTPS use is Mac-local; phone and other LAN-client browsing is removed from scope | [V1.0.0](v1.0.0.md), [migration record](../migration/rancher-to-docker-desktop.md), and root README updated to specify Mac-local hostnames and wildcard routing | Verify required Mac-local hostnames after IP change and Docker Desktop restart; no router or phone setup |
| 2026-10-07 | V1-03 | Docker Desktop restart recovery verified; node Ready, all 15 PVCs Bound, and all 30 Pods Running after transient recovery errors | [2026-10-07 recovery record](v1-recovery-2026-10-07.md); Docker Desktop restart; Kubernetes node/PVC/Pod state | Measure actual volume usage and workload budget; complete data policy |
| 2026-10-07 | V1-04 | Mac-local DNS and normally trusted HTTPS verified after restart; NATS Monitor route remains failed | [2026-10-07 recovery record](v1-recovery-2026-10-07.md); Technitium UDP/TCP answers; HTTPS 200; NATS HTTPS 404 persists after Traefik rollout | Repair Traefik CRD middleware discovery/routing; verify Host Agent after an IP change |

## Entry template

```markdown
| YYYY-MM-DD | V1-XX | In progress / Implemented / Verified / Blocked / Released | PR or commit; target context; redacted evidence summary | Remaining action or none |
```

On a failed attempt, append a second row showing the failure and a later row for the repair. Keep the original history.
