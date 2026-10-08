# Home Lab Codex Rules

This repository manages the current Mac's Docker Desktop Kubernetes platform. Work from the user's current request and live service state; do not recreate a versioned roadmap or reinstall removed optional services without a new request.

## Operating rules

1. Before the first deployment or cluster-changing action in a task, verify Docker Desktop and the `docker-desktop` context. Stop if the node is not `Ready` or the runtime is unhealthy.
2. Keep base cluster configuration in `/core`; keep deployable services in `/apps/<category>/<service>`.
3. Use smart namespaces by ownership: `platform-system`, `platform-storage`, `platform-secrets`, `observability`, `development`, `ai`, and `entertainment`.
4. Use `*.local.dev` for every local hostname. HTTP routes must redirect to HTTPS and use the shared `local-dev-tls` certificate unless a service has a documented exception.
5. Register every service hostname in the local DNS/CoreDNS workflow. Kubernetes-only names use service DNS; macOS clients use the local DNS/hosts integration documented in `/core/dns`.
5a. For local DNS incidents, verify a Ready `docker-desktop` node, Technitium readiness, UDP/TCP port 53 listeners, the macOS scoped resolver, and the current Mac IPv4 address. Check for conflicts from the retired dnsmasq daemon before changing DNS records or restarting workloads.
6. Store all configurable application values in Infisical, including secrets and non-secrets such as image repositories/tags, domains, ports, resource settings, feature flags, and connection settings. Kubernetes workloads consume runtime values through the Infisical Secrets Operator; Helm deployment values are resolved from Infisical before rendering. Keep only structural chart/schema definitions in Git. Do not commit credentials, tokens, passwords, or environment-specific runtime values.
7. Verify readiness probes, logs, and service endpoints for active services. Add optional observability products only when the user requests and uses them.
8. Add Kubernetes recommended labels and annotations to generated resources. Pin chart versions and image tags; do not use `latest`.
9. Every service directory has a concise `README.md` covering purpose, ownership, namespace, source/image, domain, HTTPS, UI, ports, storage, dependencies, configuration, observability, deployment, and verification.
10. Use the repository-local `$docker-image-deploy` skill for custom images. After a successful push, each project repository in the local registry must contain exactly one deployable image set referenced by one tag; an OCI image index/manifest list and its platform manifests count as one image set. Before every custom-image build for a new version or a same-version redeploy, enumerate all existing tags, recursively resolve every referenced index/member manifest digest, delete the complete existing image sets in that exact project repository, and verify every discovered tag and manifest is absent; only then build and push. An explicit request to build, deploy, or redeploy a custom image authorizes only this exact-repository cleanup. Stop before the build if deletion or empty-state verification fails, and never extend that authorization to another repository or registry garbage collection. Because the prior image set no longer exists, deploy the custom-image release with `HELM_AUTO_ROLLBACK=false make helm-apply RELEASE=<release>`; fix forward after a failed rollout, or recover a rollback image only after an explicit user request and through the same replacement workflow. Docker Desktop is the active build and Kubernetes runtime. Verify the image in the intended registry and Kubernetes runtime before deployment. The KubeDeck cluster agent follows this custom-image workflow; the removed dashboard is not a deployment target.
11. Personal/company applications remain in their own repositories. This repository provides orchestration references and deployment configuration only.
12. When a service or web UI requires an administrator account, use the standard local identity: username `admin`, first name `admin`, family name `admin`, and email `admin@local.dev`. The administrator password is always generated or supplied through Infisical and must never be committed to Git or embedded in Helm values.
13. Every Helm chart and service values file must declare explicit chart metadata and immutable image metadata: chart `version`, `appVersion`, `description`, `home`/`sources` where applicable, recommended Kubernetes labels/annotations, and a non-`latest` image `tag`. Image repository and tag overrides must also be represented in the Infisical configuration for the service.
14. Deploy services individually in dependency order. Before each release, verify its Infisical-backed Secret/ConfigMap objects exist and are reconciled by the Infisical Secrets Operator; after each release, verify rollout, probes, Service/Ingress, HTTPS, DNS, persistence, and supported metrics/logs/traces.

## Infisical ownership and external-project access

- `my-home-lab` administers the Infisical organization and the shared `home-lab` project: project creation, environment/folder layout, machine identities, roles, audit, and revocation. Every application deployed by this repository uses the `home-lab` project with its service-specific path; the source repository's GitHub location does not change that deployment ownership.
- Local development agents use the central `local-dev-agents` Universal Auth identity through `kubedeck-env-mcp`; grant it `member` access only to explicitly approved development projects. Kubernetes workloads and external deployments keep their own project-scoped credentials and must never mount the local agent identity.
- Humans use named Infisical accounts. Local AI agents use the central development-only machine identity with short-lived tokens; workloads use project-scoped identities. Do not give agents organization-admin access or give humans shared machine credentials.
- Use `viewer` for read-only agents, `member` for agents that must add/edit/delete project secrets, and `admin` only for a designated project owner.
- Use stable project scopes and uppercase service-prefixed keys. For example, `vero-finance` uses `/finance` and keys such as `FINANCE_DATABASE_URL`; other Vero applications do not reuse its credentials.
- Kubernetes uses an Infisical Secrets Operator `credentialsRef` belonging to the same project identity. Verify `ReadyToSyncSecrets=True`; an existing managed Secret is not proof of current authorization.
- Local AI agents use the globally installed `kubedeck-env-mcp` connector, which reads its 365-day client credential from Keychain and renews short-lived access tokens automatically. They must not use `infisical login`, `INFISICAL_TOKEN`, the administrator token, arbitrary Kubernetes Secrets, exported secret files, or credentials in prompts, logs, CI artifacts, or Git.
- Record project, environment/path, principal, role, reason, rotation/expiry plan, and verification for every access change. Revoke access when an agent, human, project, or task ends.

## Application deployment contract

- A service deployed by this repository uses the shared `home-lab` Infisical project under a service-specific path, even when its source code lives in an external GitHub repository. External projects deployed outside this repository keep their own Infisical project.
- Before adding a dependency, inspect the live platform Services and Helm inventory. Reuse compatible platform PostgreSQL, Redis, RabbitMQ, NATS, storage class, TLS, and DNS components; do not provision a duplicate database when the platform database can provide a dedicated schema or database.
- A new app must have a service README, Infisical requirements file, immutable image metadata, namespace, `*.local.dev` HTTPS route, health probes, persistence decision, dependency list, and verification commands. Runtime configuration includes non-secrets as well as secrets.
- If an app needs a first administrator, automate the supported bootstrap after the first Ready rollout with the standard `admin` / `admin` / `admin@local.dev` identity. Generate or retrieve the password only through Infisical, use a one-shot task with the app's persistent volume when necessary, remove it after success, and verify the account/health state without reporting the password.
- If a server-side agent adapter reports missing credentials, configure the adapter environment in the server workload through Infisical. A separate Codex/chat `/login` does not authenticate that server. Keep the app healthy without claiming the adapter is ready until its own credential is present and verified.
- For deployment failures, continue the bounded diagnose-fix-verify loop until rollout, probes, dependencies, Infisical reconciliation, HTTPS, DNS, persistence, and service health are healthy. Stop only for a missing credential, required decision, unavailable authority, or unsafe/destructive action.

## Change, merge, version, and deploy workflow

15. Start every repository change on a dedicated branch named `agent/<description>`; do not make change commits directly on `main`.
16. Before merge, run the relevant validation gates and review the complete diff. Preserve unrelated work and do not merge a branch with failing or missing required checks.
17. Merge completed branches through the repository's pull-request policy, push the resulting `main`, and remove the merged local and remote branches. Keep the repository's automatic branch deletion after merge enabled.
18. After a successful merge, create and push an annotated semantic-version tag and a matching GitHub release when the change affects project behavior, infrastructure, deployment configuration, or repository workflow. Use a patch version for compatible maintenance or rule changes.
19. Deploy again only when the merged change affects rendered Kubernetes resources, Helm values, images, runtime configuration, or service behavior. For documentation-only or repository-rule changes, do not restart workloads; still run repository validation and report that deployment was not required.
20. For every deployment, verify the `docker-desktop` context and `Ready` node first, then run Helm lint/template and server-side dry-run, verify Infisical reconciliation, deploy in dependency order, and confirm rollout, probes, Service/Ingress, HTTPS, DNS, persistence, and supported observability.
21. Deliver every successful change in this order: build and test on the dedicated branch; push the branch and merge it through GitHub; push the merged `main`; create and push the required annotated version from that merged commit; then build the versioned artifact and deploy it to Kubernetes when the change affects runtime resources. Never deploy source that has not passed validation and been pushed to merged `main`.

## Validation gates

- Render Helm charts with `helm lint` and `helm template`.
- Validate Kubernetes manifests with server-side dry-run when the cluster is available.
- Check rendered namespaces, labels, selectors, ServiceMonitors, Ingress TLS, DNS, PVCs, RBAC, and endpoints.
- Treat local render/dry-run as validation, not proof that a live application is healthy.
- Never run load tests or destructive operations without explicit current-turn approval.
- Commit and push only when this directory is a valid Git checkout with the intended remote.
