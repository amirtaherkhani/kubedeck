---
name: kubedeck-build-deploy
description: Build and deploy both KubeDeck components from the latest remote main branch or a named remote branch to Rancher Desktop Kubernetes. Use when Codex is asked to build, deploy, redeploy, release, or update KubeDeck or kubedeck-agent in Kubernetes.
---

# KubeDeck Build and Deploy

Act as the release engineer for this repository. Treat KubeDeck as one release
made of two separately built and deployed components:

- `Dockerfile` and `charts/kubedeck` for the vinext/Node dashboard.
- `kubedeck-agent/Dockerfile` and `charts/kubedeck-agent` for the Go cluster
  agent.

Use the bundled deployment script instead of recreating the workflow with
ad-hoc commands:

```bash
bash .agents/skills/kubedeck-build-deploy/scripts/deploy.sh --branch <branch>
```

## Branch Selection

- Default to `main` when the user does not mention a branch.
- If the user asks to deploy from a branch but does not provide its exact name,
  ask for the branch name before changing the checkout.
- Use only an existing branch from `origin`.
- The script fetches, switches to the requested branch, pulls with
  `--ff-only`, and verifies that local `HEAD` exactly equals
  `origin/<branch>`.
- Never build uncommitted, untracked, unpushed, ahead-of-remote, or diverged
  source.
- Never stash, discard, reset, commit, or push source as part of deployment.
  Stop and report a dirty or diverged checkout.

## Required Sequence

Always keep this order:

1. Resolve the branch (`main` by default).
2. Confirm the checkout is clean.
3. Fetch and fast-forward to the latest remote commit.
4. Validate the dashboard with `npm ci`, lint, build/tests.
5. Validate the agent with Go race tests and `go vet`.
6. Lint and render both Helm charts.
7. Build both immutable images from Git archives of the verified commit.
8. Push both images to the local registry.
9. Deploy `kubedeck-agent` first and wait for readiness.
10. Deploy `kubedeck` and wait for readiness.
11. Verify workloads, endpoints, pod images, and recent logs.

Do not reverse the deployment order. The dashboard proxies cluster requests to
the agent and should roll out only after the agent is ready.

## Live Defaults

- Kubernetes context: `rancher-desktop`
- Namespace: `kubedeck`
- Registry: `localhost:5001`
- Dashboard release/image: `kubedeck`
- Agent release/image: `kubedeck-agent`
- Dashboard admin Secret: `kubedeck-admin`
- Shared agent token Secret: `kubedeck-agent-auth`

The script refuses a context mismatch. To target another context, the user must
name it explicitly, then pass it as `KUBE_CONTEXT`:

```bash
KUBE_CONTEXT=<context> \
  bash .agents/skills/kubedeck-build-deploy/scripts/deploy.sh \
  --branch <branch>
```

Optional overrides:

```bash
KUBEDECK_NAMESPACE=<namespace>
KUBEDECK_REGISTRY=<registry>
KUBEDECK_VALUES_FILE=<dashboard-values.yaml>
KUBEDECK_AGENT_VALUES_FILE=<agent-values.yaml>
KUBEDECK_HELM_TIMEOUT=10m
KUBEDECK_TARGET_PLATFORM=linux/arm64
```

Use repository-relative values-file paths from the repository root. Existing
installed Helm values, including computed values required by immutable
resources, are preserved and merged with new chart defaults. Explicit optional
values files are applied afterward. Immutable image values and Secret wiring
are applied last.

## Secrets and DNS Safety

- Require `kubedeck-admin` to exist with all four expected administrator keys.
  Never print their values.
- If `kubedeck-agent-auth` is absent, create it with a random 256-bit token.
  Never print or persist the token outside Kubernetes.
- Wire the same token Secret into both releases.
- Do not create, replace, or reveal administrator credentials.
- Do not enable CoreDNS management during a normal deployment.
- Preserve an existing agent release's DNS settings. The chart default remains
  disabled for a first install.
- Never edit the main CoreDNS Corefile. The application owns only
  `kubedeck.override` when DNS management was separately authorized and
  enabled.

## Build and Rollout Rules

- Use `nerdctl`, not Docker Desktop, for Rancher Desktop builds.
- Stream `git archive` output into isolated temporary build directories inside
  the Rancher Desktop VM so each image contains exactly the verified remote
  commit even though the macOS checkout is not mounted in the VM.
- Tag both images with the same branch, short commit, and UTC timestamp.
- Never deploy `latest`.
- Preserve the dashboard PVC and existing Helm configuration.
- Use Helm rollback-on-failure and wait for both releases.
- Do not delete namespaces, PVCs, Secrets, releases, or old images.
- Do not modify Traefik, monitoring, or other platform-owned releases.

## Success Evidence

Do not report success from image builds alone. Report:

- requested branch and exact commit SHA;
- Kubernetes context and namespace;
- both immutable image references;
- both Helm release revisions and statuses;
- every owned Deployment rollout and Ready pod;
- Service EndpointSlices;
- recent logs for both components;
- whether the agent auth Secret was reused or created;
- any unavailable check or failure.

If one component fails, report the task as failed even if the other component
is healthy.
