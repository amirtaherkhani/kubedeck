---
name: home-lab
description: Operate and refactor the local Rancher Desktop Kubernetes home lab with safe namespaces, local.dev HTTPS/DNS, Infisical configuration, and Grafana observability.
---

# Home Lab Skill

Use this workflow for every change in this repository.

1. Read `/AGENTS.md`, inspect `/core`, and check the current Rancher/Kubernetes context before any cluster-changing action.
2. Classify a service by purpose and place it under `/apps/dev`, `/apps/ai`, `/apps/entertainment`, `/apps/platform`, or `/apps/observability`.
3. Define a namespace, `*.local.dev` hostname, HTTPS/TLS route, ports, storage, dependencies, and Infisical paths before writing manifests. Treat Infisical as the source of truth for every configurable value, including image tags.
4. Prefer an upstream Helm chart and pinned image tags. For custom charts, keep only generated deployment artifacts in this repository; never vendor third-party source.
5. Add recommended Kubernetes metadata, probes, resource settings, ServiceMonitors, Grafana dashboards, and Alloy integrations when supported.
6. Validate with Helm lint/template and Kubernetes server-side dry-run. Use live Grafana as the source of truth for dashboard and datasource claims when its MCP is available.
7. Document the service in its local `README.md`; report unavailable integrations explicitly instead of implying they are enabled.
8. For custom local images, use `$docker-image-deploy` and keep only the verified current image version in the local registry.
