# KubeDeck + homelab integration requirements

Status: requirements captured; architecture, implementation plan, and roadmap sequencing remain subject to review.

The approved first foundation design is in `docs/superpowers/specs/2026-10-03-service-module-foundation-design.md`; authoring steps are in `docs/roadmap/service-module-authoring.md`.

## Product goal

Combine KubeDeck and the homelab repository into a usable local platform for inventorying and operating services, developing and validating applications locally, and observing service health. The daily target is the Vero Finance project: simulate company services locally before staging or production. Finance is an external client workload, not a native KubeDeck platform component.

## User requirements

- Present one dashboard for service inventory and management.
- Support local deployment and testing of backend and frontend applications.
- Manage environments, configuration, and secrets; Infisical is the shared environment/secrets manager.
- Surface logs, monitoring data, metric dashboards, uptime, and service stability.
- Support incident notification delivery through Slack, Telegram, and WhatsApp, subject to user-approved credentials and audiences.
- Keep both the UI and backend/agent modular and plugin-based, so plugins can be added and deployed for later platform deployments.
- Keep the code clean, maintainable, reusable, and performant for the current Docker Desktop system. Keep machine-specific values in local configuration rather than shared assets.
- Treat Node.js, Go, and Python as examples of possible application stacks, not a requirement to rewrite components or support multiple languages in the platform itself.
- Verify architecture and engineering recommendations against current authoritative documentation when design begins.

## First architecture milestone: standardized service/tool modules

Before wider feature work, inspect current repository layouts and propose a reusable service/tool module template plus a validated shared metadata contract that both the agent and UI can consume consistently. Cover databases, brokers, microservices, UI applications, and open-source tools without assuming their vendor-specific configuration is identical. Preserve vendor charts and tool-specific values.

The conceptual metadata should consider: stable identity, type, purpose, chart source and version, namespace, dependencies, endpoints, health checks, observability links/signals, and Infisical secret-reference concepts. It must never contain raw secret values. Distinguish infrastructure owned/deployed by this homelab from external applications registered for observation, including the Vero Finance client workloads.

The exact directory layout and contract format are design decisions for review; this requirement does not choose a schema or authorize migrating all existing services.

## Design constraints and open decisions

- Preserve and extend the existing KubeDeck agent; it already provides read-only Kubernetes discovery and streaming. Design any future management actions as explicit, scoped capabilities with least-privilege access and auditable behavior.
- Current user requirements do not require hot-loading or arbitrary code execution. Decide the plugin packaging, lifecycle, compatibility, and deployment model during reviewed design.
- Identify UI extension points and backend capability contracts during design; do not preselect an invasive plugin architecture here.
- Define plugin failure isolation, permissions, compatibility checks, and automated contract/integration tests before implementation.
- Separate portable configuration from secrets; use reproducible setup and do not check credentials into source control.
- Do not add, remove, or redefine service catalog entries without the user's explicit confirmation.
- Do not classify Vero Finance client workloads as KubeDeck's own platform services.
- The roadmap targets the current single-node Docker Desktop Kubernetes environment. Discovery, control, and service workflows stay scoped to this cluster.
- Local runtime validation must verify the intended local cluster/context and avoid disrupting existing workloads. Staging/production actions require separate authorization.
- Notification credentials, destination audiences, and authorization are not presumed to exist.

## Candidate work sequence

This is an ordering of work areas, not an approved implementation plan:

1. Preserve repository history and existing local changes; document the integrated local-development workflow and its portability boundary.
2. Inspect current layouts and present a reusable service/tool module template and shared metadata contract for review. Separate owned infrastructure from external registered apps; retain vendor-specific chart values.
3. Review the current UI, agent, homelab charts, observability, and secrets integrations; establish architecture choices and plugin contracts.
4. Specify and test plugin lifecycle, compatibility, UI extension points, backend capability boundaries, permissions, and failure isolation.
5. Add local application deployment/testing workflows suitable for the Finance simulation use case, with reproducible environment/configuration handling.
6. Improve service health, logs, metrics, uptime, and incident views using existing homelab telemetry where possible.
7. Design and implement delivery integrations for approved notification providers and audiences.
8. Validate a portable setup from a clean developer environment and document local-only operation separately from staging/production.

V1 targets the current local single-node Docker Desktop environment. Discovery, control, and service workflows are scoped to this cluster.

Each work area needs its own reviewed design and execution plan before invasive implementation. No feature implementation is authorized by this requirements capture alone.
