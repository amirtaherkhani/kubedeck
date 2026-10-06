---
name: kubedeck-build-deploy
description: Build, version, publish, deploy, and clean up both KubeDeck components from a verified remote branch to Rancher Desktop Kubernetes. Use when Codex is asked to build, deploy, redeploy, release, or update KubeDeck or kubedeck-agent in Kubernetes.
---

# KubeDeck Build and Deploy

Act as the release engineer for this repository. Treat KubeDeck as one release
made of two separately built and deployed components:

- `Dockerfile` and `charts/kubedeck` for the vinext/Node dashboard.
- `kubedeck-agent/Dockerfile` and `charts/kubedeck-agent` for the Go cluster
  agent.

Follow [the repository build and deploy policy](../../policies/kubedeck-build-deploy.md).
For registry-backed releases on the legacy Rancher Desktop target, use the
bundled deployment script with automatic image cleanup disabled:

```bash
KUBEDECK_CLEANUP_OLD_IMAGES=false \
  bash .agents/skills/kubedeck-build-deploy/scripts/deploy.sh --branch <branch>
```

The script pushes to `localhost:5001` and always removes old non-running pods
and zero-replica ReplicaSets after health verification. Run it only when that
Kubernetes cleanup is authorized. Its automatic image cleanup sweeps old tags
without proving recoverability or checking all consumers; use the reviewed
post-deployment procedure below instead. For an
approved local-only install, build the verified version-tagged images directly
into Rancher Desktop's `k8s.io` containerd namespace and use those exact tags
in Helm with `IfNotPresent`. Never use the script for an image-load-only
request or a Docker Desktop KIND target; verify that target's image store and
deployment method first.

For a release, increase the version before running the deployment workflow:

```bash
bash .agents/skills/kubedeck-build-deploy/scripts/bump-version.sh patch
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
8. Push both versioned images to the local registry for a registry release, or
   verify both tags in `k8s.io` containerd for an approved local-only install.
9. Deploy `kubedeck-agent` first and wait for readiness.
10. Deploy `kubedeck` and wait for readiness.
11. Verify workloads, endpoints, pod images, and recent logs.
12. Verify authenticated dashboard requests can proxy a live agent snapshot and render the deployments catalog.
13. After successful health verification, inspect old images for the deployed
    KubeDeck-owned components and remove only unused, recoverable, explicitly
    identified versions using the procedure below.

Do not reverse the deployment order. The dashboard proxies cluster requests to
the agent and should roll out only after the agent is ready.

## Live Defaults

- Kubernetes context: `rancher-desktop`
- Namespace: `development-tools`
- Registry: `localhost:5001`
- Dashboard release/image: `kubedeck`
- Agent release/image: `kubedeck-agent`
- Dashboard admin Secret: `development-tools/kubedeck-admin`
- Shared agent token Secret: `development-tools/kubedeck-agent-auth`
- Infisical project/environment: `home-lab`/`local`
- Default image tag: `dev-<package-version>`

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
KUBEDECK_RELEASE_CHANNEL=dev
KUBEDECK_CLEANUP_OLD_IMAGES=false
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

- Require the Infisical operator, `InfisicalSecret` CRD, and
  `platform-secrets/infisical-universal-auth` to exist when Infisical is
  enabled. Wait for the managed dashboard and agent Secrets after Helm
  installation. Never print their values.
- For an explicitly standalone deployment with Infisical disabled, require the
  administrator Secret and create a random agent token only in Kubernetes.
- Wire the same managed agent token Secret into both releases.
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
- Tag both images with the same release channel and semantic version, for
  example `localhost:5001/kubedeck:dev-0.1.2`.
- Refuse an existing registry tag; increase the version before building.
- Never deploy `latest` or a timestamp-only tag.
- Preserve the dashboard PVC and existing Helm configuration.
- Use Helm rollback-on-failure and wait for both releases.
- After health verification, use **Post-deployment image cleanup** below. Do
  not enable the script's automatic old-image sweep.
- Remove old zero-replica ReplicaSets and non-running old pods only when that
  Kubernetes cleanup is also in scope; do not use image cleanup permission as
  permission to delete Kubernetes objects.
- Never delete namespaces, PVCs, Secrets, Infisical resources, Services,
  Ingresses, or platform-owned releases.
- Do not modify Traefik, monitoring, or other platform-owned releases.

## Post-deployment image cleanup

Apply this to each newly deployed KubeDeck-owned component or subservice after
its rollout, endpoints, logs, and dependent application path are healthy. If
health fails, retain the previous images for recovery.

1. Inventory exact tags, digests, image IDs, platforms, and stores for the new
   and older component versions. On Docker Desktop, inspect the local image
   store and the KIND node's containerd `k8s.io` cache separately; inspect a
   registry too if it supplied an image. One store's inventory or deletion
   says nothing about another store.
2. Check all namespaces and running containers/workloads, including jobs and
   rollback targets, for references by tag, digest, and image ID. Retain the
   current image, every referenced image, and any required rollback version.
3. For each older unused candidate, verify recovery from its exact source
   commit and reproducible build instructions or from a reachable registry at
   the verified digest. Retain unique or unverified images. Check permission,
   store, and exact identifier before removal.
4. Remove only verified candidates by explicit ID or digest. Never use broad
   `prune`, wildcard removal, or force; never delete volumes, Secrets, other
   projects' images, or unrelated registry repositories. Record every removed
   and retained version, identifier, store, and reason. If any check is
   unavailable, retain the candidate and report the blocker.

## Success Evidence

Do not report success from image builds alone. Report:

- requested branch and exact commit SHA;
- Kubernetes context and namespace;
- both immutable image references;
- both Helm release revisions and statuses;
- every owned Deployment rollout and Ready pod;
- Service EndpointSlices;
- recent logs for both components;
- authenticated dashboard-to-agent snapshot status and live object counts;
- authenticated `/dashboard/catalog/deployments` rendering status;
- whether the agent auth Secret was reused or created;
- cleaned and retained image versions, stores, IDs/digests, and reasons;
- any unavailable check or failure.

If one component fails, report the task as failed even if the other component
is healthy.
