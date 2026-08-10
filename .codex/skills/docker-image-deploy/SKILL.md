---
name: docker-image-deploy
description: Build, tag, push, deploy, verify, and prune custom Docker images in the local Rancher Desktop registry. Use when a home-lab service needs a locally built image, a registry image update, or removal of older local image versions.
---

# Docker Image Deploy

Use this workflow for custom images deployed to the home lab.

## Image naming

Use one repository per service:

```text
localhost:5001/homelab/<category>/<service>:v<major>.<minor>.<patch>
```

Normalize category and service to lowercase kebab-case. Examples:

```text
localhost:5001/homelab/dev/kubedeck:v0.1.0
localhost:5001/homelab/observability/custom-exporter:v1.4.0
```

Use the `v<major>.<minor>.<patch>` tag format. Never use `latest`, `edge`, or
an unversioned tag for a Kubernetes workload. Store the selected tag in
Infisical as a non-secret value and resolve it into Helm during deployment.
Do not commit the environment-specific tag as a hard-coded runtime value.

## Workflow

1. Read the repository `AGENTS.md`, inspect `/core`, select the app category and
   namespace, and verify Rancher Desktop plus the `rancher-desktop` context.
2. Confirm the local registry is reachable at `localhost:5001`. Stop if it is
   unavailable; do not silently push to a public registry.
3. If the project has no public image, confirm that building its source is
   allowed before cloning or building it. Keep third-party source outside this
   repository; commit only the generated chart/manifests.
4. Inspect the Dockerfile, required build arguments, architecture, and runtime
   port. Build for Rancher Desktop's node architecture, normally `linux/arm64`:

   ```bash
   docker buildx build --platform linux/arm64 \
     -t localhost:5001/homelab/<category>/<service>:v<major>.<minor>.<patch> \
     --load <source-directory>
   docker push localhost:5001/homelab/<category>/<service>:v<major>.<minor>.<patch>
   ```

   Use `nerdctl` when Docker is unavailable and preserve the same image name.
5. Verify that the new `v<major>.<minor>.<patch>` manifest and digest exist in
   the local registry. Do not continue if the new version is missing.
6. Resolve the exact previous tag and digest, then delete the old registry tag.
   Never delete the newly verified tag. This cleanup happens before the Helm
   update, as required by the home-lab policy.
7. Store the new tag in the service's Infisical configuration, resolve it into
   Helm values, and add `app.kubernetes.io/version`. Render and lint the chart.
8. Deploy through the repository release workflow, using `--wait` and
   rollback-on-failure behavior where available.
9. Verify the rollout, image ID/digest, probes, HTTPS route, logs, metrics,
   dashboards, and application endpoint.

## Retention rule

Keep exactly one version tag per local image repository: the new verified image
version. If the registry does not support tag deletion, stop before changing
Helm values and report the limitation. Registry garbage collection is separate
from tag deletion and should run only when explicitly configured.

## Failure rules

- Do not deploy an image that cannot be pulled by the Kubernetes node.
- Do not overwrite a live tag with a different image.
- Verify the new image before deleting the old registry tag.
- Do not update Helm values until old-tag cleanup succeeds.
- Do not commit credentials, registry tokens, Dockerfiles containing secrets,
  or third-party source files.
- If build, push, registry cleanup, render, rollout, or health verification
  fails, stop before the next step and report the exact failing gate.
