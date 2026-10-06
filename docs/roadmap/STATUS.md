# Roadmap status

**As of:** 2026-10-06 (Asia/Tehran). **Current:** V1 recovery implementation is in progress. The [live recovery record](v1-recovery-2026-10-06.md) separates verified paths from open gates. v1.0.0 is not released.

| ID | Target | State | Last evidence | Next action |
| --- | --- | --- | --- | --- |
| V1-01 | v1.0.0 | In progress | [Live recovery record](v1-recovery-2026-10-06.md): runtime, listeners, Helm, PVCs and secret dependency checked | Confirm active client consumers, service ownership, exact data policy |
| V1-02 | v1.0.0 | In progress | 51 home-lab and 347 Finance export records exactly matched live `dev` targets; connector and Operator identity work | Prove isolated DB backup with original key and per-PVC policy |
| V1-03 | v1.0.0 | In progress | Docker Desktop runtime, node allocatable resources, PVCs and host 80/443 checked | Validate restart persistence, disk and selected workload budget |
| V1-04 | v1.0.0 | In progress | TCP/UDP 53, localhost resolver, Host Agent, and trusted Infisical HTTPS 200 pass on Mac | Verify Mac-local DNS after IP change and Docker Desktop restart; check all required local routes |
| V1-05 | v1.0.0 | In progress | PostgreSQL, Redis, RabbitMQ, NATS and SeaweedFS S3 are Ready; S3 contract and Pod restart persistence pass; unauthenticated Redis LAN listener removed | Prove external project workflows and backups; select remaining required tools |
| V1-06 | v1.0.0 | In progress | KubeDeck and Kubernetes Agent are Ready; trusted HTTPS login, cluster snapshot and SSE pass; Metrics Server reports node/Pod usage | Verify UI management workflows, restart persistence and scope of Agent permissions |
| V1-07 | v1.0.0 | Not started | — | Verify other-project connections and end-to-end workflows |
| V1-08 | v1.0.0 | Not started | — | Record rollback, release acceptance, and tag/release evidence |
| V11-01 | v1.1.0 | Not started | — | Define agent boundaries and capability contract |
| V11-02 | v1.1.0 | Not started | — | Extend Kubernetes and Helm operations |
| V11-03 | v1.1.0 | In progress | Go DNS reconciler and launchd installer implemented early for V1-04 | Extend host-control capabilities and supported OS adapters after V1 |
| V11-04 | v1.1.0 | Not started | — | Connect REST, SSE, and interactive streams |
| V12-01 | v1.2.0 | Not started | — | Adopt approved module descriptors |
| V12-02 | v1.2.0 | Not started | — | Build independent Helm lifecycle reconciler |
| V12-03 | v1.2.0 | Not started | — | Build Tool Manager API and UI |
| V12-04 | v1.2.0 | Not started | — | Publish module authoring and compatibility workflow |
| V13-01 | v1.3.0 | Not started | — | Provide bounded Kubernetes and Docker logs |
| V13-02 | v1.3.0 | Not started | — | Manage Docker volumes and Kubernetes PV/PVC distinctly |
| V13-03 | v1.3.0 | Not started | — | Track active/completed builds and container metadata |
| V13-04 | v1.3.0 | Not started | — | Correlate module, release, image, logs, and metrics |
| V14-01 | v1.4.0 | Not started | — | Remove machine-specific assumptions |
| V14-02 | v1.4.0 | Not started | — | Package and install Host Agent on macOS |
| V14-03 | v1.4.0 | Not started | — | Enforce UI policy in API and agents |
| V14-04 | v1.4.0 | Not started | — | Validate setup and recovery on a second Mac |
| V2-01 | v2.0.0 | Not started | — | Build reproducible project workflows |
| V2-02 | v2.0.0 | Not started | — | Pilot and decide Tekton/Buildkite/native coordination |
| V2-03 | v2.0.0 | Not started | — | Connect workflow history and observability |
| V2-04 | v2.0.0 | Not started | — | Decide any multi-host topology and enrollment |

**Gate rule:** A parent version is `Verified` only when every required task in its version file is verified and its release acceptance checklist passes. `Released` requires the tag and GitHub release URL. A blocked task records the exact obstacle and owner in the history entry.

**Scope decision (2026-10-06):** v1.0.0 serves the Mac that runs Docker Desktop. Phone and other LAN-client browsing, router DHCP/DNS integration, and remote resolver bootstrap are out of scope. The required path is the Mac's scoped resolver to Technitium, local ingress, and service hostname. The Host Agent must keep DNS answers aligned with the Mac's current address.
