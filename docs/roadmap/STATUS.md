# Roadmap status

**As of:** 2026-10-07 (Asia/Tehran). **Current:** V1 recovery is still in progress; the [latest live recovery record](v1.0.0-recovery.md) records post-restart results and remaining blockers. v1.0.0 is not released.

| ID | Target | State | Last evidence | Next action |
| --- | --- | --- | --- | --- |
| V1-01 | v1.0.0 | In progress | [Live recovery record](v1.0.0-recovery.md): live Helm/PVC/Ingress inventory and all 24 historic Mac-local service names checked | Confirm required service owners and settle observability scope |
| V1-02 | v1.0.0 | In progress | Prior export-to-live target comparisons and connector auth remain recorded in [initial recovery evidence](v1.0.0-recovery-initial.md) | Still no protected Infisical DB backup plus isolated restore evidence; obtain data/reseed decisions for each PVC |
| V1-03 | v1.0.0 | In progress | 2026-10-07 Docker Desktop restart completed; node Ready, 30/30 Pods Running, 15/15 PVCs Bound afterward | Measure workload/storage use and finish data policy; restart recovery included temporary readiness failures |
| V1-04 | v1.0.0 | In progress | All 24 Mac-local names resolve; trusted HTTPS works; NATS Monitor now returns HTTP 401 with a Basic Auth challenge | Verify Host Agent reconciliation after a Mac IP change |
| V1-05 | v1.0.0 | In progress | Core database/broker Pods recovered; Redis PING, PostgreSQL readiness, RabbitMQ ping and direct NATS health checks pass | Revalidate storage data workflow/backup and determine required observability tools and project consumers |
| V1-06 | v1.0.0 | In progress | KubeDeck endpoint serves HTTPS 200; current live Helm app is 0.1.0 while repository target is 0.2.0 | Authenticated dashboard/agent workflows remain unverified after restart; resolve data gate before upgrade |
| V1-07 | v1.0.0 | Not started | No Finance client workload or process was found in the Docker Desktop cluster | Have the project owner select/approve a development smoke workflow and run it against the platform |
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
| V14-04 | v1.4.0 | Not started | — | Validate setup and recovery on the current Docker Desktop system |
| V2-01 | v2.0.0 | Not started | — | Build reproducible project workflows |
| V2-02 | v2.0.0 | Not started | — | Pilot and decide Tekton/Buildkite/native coordination |
| V2-03 | v2.0.0 | Not started | — | Connect workflow history and observability |
| V2-04 | v2.0.0 | Not started | — | Define resource limits and recovery boundaries for the current Docker Desktop cluster |

**Gate rule:** A parent version is `Verified` only when every required task in its version file is verified and its release acceptance checklist passes. `Released` requires the tag and GitHub release URL. A blocked task records the exact obstacle and owner in the history entry.

**Scope:** The roadmap targets the current Mac and its running Docker Desktop Kubernetes cluster. The required path is the Mac's scoped resolver to Technitium, local ingress, and service hostname. The Host Agent keeps DNS answers aligned with the current Mac address.
