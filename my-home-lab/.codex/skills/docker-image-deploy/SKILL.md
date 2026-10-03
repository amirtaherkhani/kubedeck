---
name: docker-image-deploy
description: Use when a home-lab service needs a locally built Docker image, a local-registry image update, a same-version redeploy, or removal of an existing project image.
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
4. Inspect the Dockerfile, required build arguments, architecture, runtime port,
   and source readiness before deleting the existing image.
5. Before running any image-build command, set the exact target repository and
   list all of its registry tags. Resolve each tag with the OCI/Docker manifest
   media types, recursively collect every platform/member digest referenced by
   an image index or manifest list, and deduplicate the complete digest set.
   Delete member manifests before their indexes plus every ordinary manifest in
   that repository. Then verify the tag list is empty and every discovered
   digest is unavailable. This gate applies to a new version and to rebuilding
   or redeploying the current version. If the repository is already empty,
   record that state and continue. If any deletion fails or any discovered
   tag/manifest remains, stop before the build. Do not delete another repository
   or run registry garbage collection.
6. Build for Rancher Desktop's node architecture, normally `linux/arm64`:

   ```bash
   nerdctl build --platform linux/arm64 \
     -t localhost:5001/homelab/<category>/<service>:v<major>.<minor>.<patch> \
     <source-directory>
   nerdctl push localhost:5001/homelab/<category>/<service>:v<major>.<minor>.<patch>
   ```

   Rancher Desktop's `nerdctl`/containerd path is authoritative. Use Docker
   `buildx` and `docker push` only when `nerdctl` cannot access the source or
   local registry; a missing or stopped Docker daemon is not itself a failure.
7. Verify that the requested `v<major>.<minor>.<patch>` tag is the repository's
   only tag and resolves to one complete image set: either one image manifest or
   one index whose referenced platform manifests are all available. Verify that
   no manifest from the previous image sets remains and that the Kubernetes node
   can pull the requested image. Do not continue if the requested image is
   incomplete or if another deployable image set remains.
8. Store the new tag in the service's Infisical configuration, resolve it into
   Helm values, and add `app.kubernetes.io/version`. Render and lint the chart.
9. Deploy the individual custom-image release through the repository wrapper,
   explicitly disabling automatic rollback while retaining its wait and cleanup
   behavior:

   ```bash
   HELM_AUTO_ROLLBACK=false make helm-apply RELEASE=<release>
   ```

   Do not use the wrapper's default automatic rollback mode when the previous
   release points to the image set deleted by step 5. If rollout fails, retain
   the requested image as the repository's one image set, diagnose, and fix
   forward. A rollback requires an explicit recovery request and restoration of
   the selected image through this complete replacement workflow before changing
   the release.
10. Verify the rollout, image ID/digest, probes, HTTPS route, logs, metrics,
   dashboards, and application endpoint.

## Retention rule

| Gate | Required state for the exact project repository |
| --- | --- |
| Before every image build | Zero tags and zero available manifests from all previous image sets |
| Before deployment | One requested version tag resolving to one complete, verified, pullable image set |

Do not interpret this as one image per version: keep one deployable image total
for the project repository. For retention purposes, an OCI index/manifest list
and all platform manifests it references form one image set. A same-version
rebuild may reuse its version tag only after the complete previous image set has
been deleted and absence verified. If the registry does not support manifest
deletion, stop before building. Registry garbage collection is separate from
manifest deletion and runs only when explicitly configured.

If the build or push fails after the required cleanup, leave the target
repository empty, do not deploy, and report that state. Do not restore the stale
image unless the user explicitly requests recovery.

If deployment fails after the requested image is verified, keep that image as
the repository's one image set and fix forward. Never roll back a release to an
image that is absent from the registry. An explicitly requested rollback first
restores its selected image through this workflow, replacing the failed image
set before the Helm rollback or redeployment.

## Failure rules

- Do not deploy an image that cannot be pulled by the Kubernetes node.
- Do not start the image build until exact-repository cleanup and empty-state
  verification succeed.
- Do not overwrite an existing tag. For a same-version rebuild, reuse the tag
  only after its previous manifest is absent.
- Delete only manifests in the exact target repository; preserve every other
  project repository.
- Do not update Helm values until the new repository contains exactly one
  verified image set.
- For custom-image replacement, set `HELM_AUTO_ROLLBACK=false` on the repository
  Helm wrapper. Fix forward unless the user explicitly authorizes image recovery
  and rollback.
- Do not commit credentials, registry tokens, Dockerfiles containing secrets,
  or third-party source files.
- If registry cleanup, build, push, render, rollout, or health verification
  fails, stop before the next step and report the exact failing gate.
