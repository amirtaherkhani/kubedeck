# KubeDeck Build and Deploy Policy

This policy governs releases of the dashboard, `kubedeck-agent`, and
KubeDeck-owned subservices. Verify the active Kubernetes context and image
store; do not assume the legacy Rancher Desktop defaults match Docker Desktop.

## Version and image identity

- Increase the semantic version before building an artifact. Use
  `bash .agents/skills/kubedeck-build-deploy/scripts/bump-version.sh patch`
  for the next compatible patch release, or pass an explicit semver.
- Keep `package.json`, `package-lock.json`, and both Helm `Chart.yaml` files at
  the same version and `appVersion`.
- Use the same immutable tag for both components. The default format is:
  `localhost:5001/kubedeck:dev-0.1.2` and
  `localhost:5001/kubedeck-agent:dev-0.1.2`.
- The release channel defaults to `dev` and may be changed with
  `KUBEDECK_RELEASE_CHANNEL`. Never use `latest`, timestamps alone, or a tag
  that already exists in the registry. The version is increased before the
  build so the artifact, OCI label, chart, and deployment all identify the
  same release.

## Build and registry cleanup

1. Validate the clean, remote commit and all application, agent, and Helm
   gates.
2. Build both images from that exact Git commit with Rancher Desktop
   `nerdctl`.
3. For a registry release, push both `dev-<semver>` images to
   `localhost:5001`. For an approved local-only install, load both images in
   containerd's `k8s.io` namespace and verify their exact version tags there;
   do not run the registry-pushing deployment script.
4. After the new deployment, readiness, endpoints, logs, and any dependent
   application path are healthy, inspect older versions of only the
   deployed KubeDeck-owned components. Use
   `KUBEDECK_CLEANUP_OLD_IMAGES=false` with the legacy deployment script;
   perform image cleanup as a separate reviewed step because its automatic
   tag sweep does not establish the safeguards below.

For each exact tag and digest candidate, inventory Docker Desktop's local
image store and the KIND node's containerd `k8s.io` cache separately, plus
any registry used for that deployment. Check all namespaces and running
containers/workloads for references by tag, digest, and image ID. Keep the
current image, images in use, and any required rollback image. Remove an older
unused image only after proving it is recoverable from its exact source commit
or a verified registry digest, checking delete permissions, and confirming the
target store and image ID/digest. Delete only explicit candidates; never use
`prune`, wildcard deletes, or force. Preserve unique or unrecoverable images,
volumes, Secrets, other projects' images, and unrelated registry repositories.
Record cleaned and retained versions, stores, identifiers, and reasons. If a
check is unavailable, retain the image and report why.

## Kubernetes rollout cleanup

- Deploy the agent before the dashboard.
- Wait for both releases, verify every Ready pod uses the new image, and only
  then clean up old zero-replica ReplicaSets and non-running old pods.
- The cleanup must not delete a running pod, namespace, PVC, Secret,
  Infisical resource, Service, Ingress, or platform-owned resource.
- Keep the dashboard PVC and all Infisical-managed configuration intact.
- If the new rollout or authenticated dashboard-to-agent verification fails,
  stop cleanup and report the failure. Do not remove the previous running
  workload before the replacement is healthy.

## Required evidence

Report the exact commit, semantic version, both image references, registry push
result, Helm revisions, Ready pod images, cleanup counts, endpoints, logs, and
authenticated dashboard-to-agent verification. Never report credentials,
tokens, or passwords.
