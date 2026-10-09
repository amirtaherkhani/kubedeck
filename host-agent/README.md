# KuchDesk Host DNS Agent

The Go CLI discovers the default-route interface and its private IPv4 address, opens a localhost-only Kubernetes port-forward to Technitium's admin API, and reconciles one configured DNS wildcard. It reads the admin password from a Kubernetes Secret; no credential or host IP is stored in this repository. It skips writes when the DNS record is already current. The supported host installation is macOS: the installer configures a per-user `launchd` job. Existing non-macOS source is retained but is outside this version's supported host workflow.

Prerequisites: a reachable Kubernetes cluster with a healthy Technitium Service, `kubectl`, Go for building, and access to the Technitium Secret. The macOS installer pins the selected Kubernetes context in the LaunchAgent, so later changes to the shell's current context do not redirect DNS writes. `RunAtLoad` and a 30-second interval cover IP changes and restart recovery on macOS. Install for the current macOS user:

```sh
KUCHDESK_HOST_AGENT_KUBE_CONTEXT=docker-desktop \
KUCHDESK_HOST_AGENT_ZONE=local.dev \
./host-agent/install-macos.sh
launchctl print "gui/$(id -u)/dev.kuchdesk.host-agent"
```

The installer requires `KUCHDESK_HOST_AGENT_ZONE` and otherwise uses the current `kubectl` context unless `KUCHDESK_HOST_AGENT_KUBE_CONTEXT` is set. To use another Technitium deployment, pass `KUCHDESK_HOST_AGENT_NAMESPACE`, `KUCHDESK_HOST_AGENT_SERVICE`, `KUCHDESK_HOST_AGENT_ADMIN_SECRET`, `KUCHDESK_HOST_AGENT_PASSWORD_KEY`, `KUCHDESK_HOST_AGENT_ADMIN_USER`, or `KUCHDESK_HOST_AGENT_API_PORT` to the installer. `KUCHDESK_HOST_AGENT_INTERFACE` selects a non-default interface; `KUCHDESK_HOST_AGENT_TARGET_IP` sets an explicit IPv4 address directly, including loopback for host-only DNS. `KUCHDESK_HOST_AGENT_TTL` controls the wildcard record TTL (default 30 seconds), `KUCHDESK_HOST_AGENT_TIMEOUT` limits a single run (default 30 seconds), and `KUCHDESK_HOST_AGENT_INTERVAL` controls the launchd interval (default 30 seconds; allowed 5–3600). The installer saves the discovered absolute `kubectl` path as an argument, so launchd does not depend on a shell PATH. A non-default `KUBECONFIG` is retained as absolute paths by the LaunchAgent.

For a single reconciliation without installing launchd, pass the zone and context explicitly:

```sh
cd host-agent && go run ./cmd/kuchdesk-host-agent -kube-context docker-desktop -zone local.dev
```

For a Mac whose default route does not lead to the intended ingress, pass `-interface en0` or an explicit `-target-ip` after checking the actual local network. The CLI also accepts `-kubectl`, `-ttl`, and `-timeout` for direct runs.

If a VPN changes the default route to a tunnel interface, the agent refuses to publish that address. On a multi-interface host, use `-interface` with the interface that reaches the local ingress after checking the routing profile, or use `-target-ip` with the intended IPv4 address. Automatic selection still chooses a private address; an explicit loopback address is useful only when the DNS clients run on that same host. The agent currently handles IPv4 only. It updates only the configured zone's wildcard DNS **answer**; it does not set system-wide DNS servers or routing.

The agent's Kubernetes credential can read the existing admin Secret. Treat that user account and kubeconfig as privileged; a dedicated scoped Technitium API credential and finer host permissions belong to later hardening work. The admin API is reachable only through a localhost port-forward.

The DNS reconciler opens its own temporary localhost port-forward. This repository does not install a fixed-port Technitium admin forwarder or a local image registry.

## Doctor diagnostics

`cmd/kuchdesk-doctor` runs bounded, read-only checks of the Mac default route,
one configured DNS name, Docker, Kubernetes node readiness, a loopback
registry, build-volume free space, node metrics, and optional Deployment
readiness. It reports ordered checks, sanitized evidence, findings, and
specific recommendations as JSON. A VPN-owned default route is a warning.
The command does not change network settings, cluster resources, or Secrets.

```sh
cd host-agent
go run ./cmd/kuchdesk-doctor \
  -domain infisical.local.dev \
  -kube-context docker-desktop \
  -registry-url http://127.0.0.1:5001 \
  -disk-path /absolute/path/to/project \
  -service development-tools/kuchdesk-agent \
  -service platform-system/kuchdesk-local-registry
```

The `kuchdesk_doctor` MCP tool uses the same collector and returns report schema
`kuchdesk.doctor/v2`: ordered check/component/source/context/timing/status,
typed findings, dependencies, limitations, and short redacted command excerpts.
For AI-assisted diagnosis, use the existing MCP client's
`kuchdesk_doctor_repair` prompt. It creates a fresh report and supplies the
catalog of implemented typed tools. The report schema and English prompt
template are available as MCP resources at `kuchdesk://doctor/report/v2` and
`kuchdesk://doctor/prompt/v2`, and as
[`internal/doctor/doctor-report.schema.json`](internal/doctor/doctor-report.schema.json)
and [`internal/doctor/prompt.md`](internal/doctor/prompt.md) in source.
No separate model, provider, API key, or CLI `-ai` mode is used.

The prompt requires a bounded typed plan through `kuchdesk_doctor_validate_plan`,
uses `kuchdesk_deploy_plan` and `kuchdesk_deploy_preflight` before an authorized
`kuchdesk_deploy_start`, waits for job status, and reruns affected checks with
`kuchdesk_doctor_verify`. Only the deployment start is a supported repair
action in the current MCP catalog. It cannot repair DNS, firewall, RBAC, or
credentials by inventing commands. Plan validation is not authorization;
the existing deployment opt-in and exact confirmation still apply. Failed
post-checks remain unresolved.

## Infisical MCP: first implementation slice

`cmd/kuchdesk-infisical-mcp` is a separate, long-running stdio MCP server. It does not change the periodic DNS LaunchAgent or run as a network listener. It uses the official MCP Go SDK v1.6.0 and Infisical's documented REST API from the deployed v0.151.0 OpenAPI; the host-agent Go module now requires Go 1.25. The local client implements Universal Auth login, in-memory token renewal from `expiresIn`, restart re-login, one bounded retry of a read after HTTP 401, and a pause after a repeated 401. It does not assume a year-long access token.

Only `INFISICAL_CLIENT_ID` and `INFISICAL_CLIENT_SECRET` carry the bootstrap credential. For one-time local setup, run the no-echo command below yourself in a terminal. It creates `~/.config/kuchdesk/infisical/host.env` with directory mode `0700` and file mode `0600`, owned by your user. It never prints the values and refuses to overwrite an existing file. This is a persistent plaintext file readable by your user and system administrators; keep it outside Git and backups you do not trust.

```sh
cd /Users/mac/Documents/GitHub/kuchdesk/host-agent
go run ./cmd/kuchdesk-infisical-setup
```

The CLI and MCP stdio process each load this file at startup through a strict two-key parser. The file must contain only `INFISICAL_CLIENT_ID` and `INFISICAL_CLIENT_SECRET`; it is never sourced as shell code. A complete process environment pair takes precedence, and `KUCHDESK_INFISICAL_ENV_FILE` can select another absolute private file. Do not put values in command arguments, Git, MCP configuration text, prompts, or chat. Pass the HTTPS origin separately:

```sh
cd host-agent
go run ./cmd/kuchdesk-infisical-mcp -url https://YOUR-INFISICAL-HOST
```

Restart the MCP stdio process after setup or rotation so it loads the new file. Test locally with `kuchdesk-infisical ... capabilities` (`configured:true` proves only that the pair loaded), then use `list-secret-names` with the intended project ID, environment, and narrow path to verify Universal Auth and read permission. The existing Grafana Operator Secret is a separate identity and must not be copied into this host file.

The Infisical MCP tools are `infisical_capabilities`, `infisical_list_secret_names`, `infisical_start_list_secret_names`, `infisical_job_status`, and `infisical_cancel_job`. Name listing requires an explicit project ID, environment slug, and absolute secret path. It requests `viewSecretValue=false` and returns only names and paths, discarding any value fields from the API response. Discovery works without credentials; a secret-list call then returns `credentials_not_configured`.

The persistent MCP process executes up to four independent operations in parallel, retains at most 64 in-memory records, and gives each operation a 30-second deadline. Operations targeting the same nonempty resource key serialize in the runner; the current name-only reads use no key. Status distinguishes queued, running, succeeded, failed, canceled, and timed out. A cancellation request is cooperative through Go context; process exit discards job history and cancels in-flight work. This is lightweight agent execution, not a durable task service.

The command-line client shares the same typed Infisical service and runs synchronously:

```sh
cd host-agent
go run ./cmd/kuchdesk-infisical -url https://YOUR-INFISICAL-HOST capabilities
go run ./cmd/kuchdesk-infisical -url https://YOUR-INFISICAL-HOST list-secret-names -project PROJECT_ID -environment dev -path /app
```

No real identity, credential, role, or live MCP service is provisioned by this source change. Permissions and bootstrap credential lifetime remain unverified against this installation.

Secret writes, project/identity administration, value delivery, a host HTTP API, and Kubernetes-agent bridging are not implemented. Existing Grafana Operator sync remains independent. The same host MCP process now exposes a separate opt-in KuchDesk deployment workflow described below; this does not use Infisical credentials.

## Local build and deployment workflow

`cmd/kuchdesk-deploy` is a local operator CLI. Its default mode prints a plan;
`-apply` runs it only with an exact `release/namespace` confirmation. A profile
is one JSON object with absolute `sourceDir`, `dockerfile`, and `chartDir`
paths, optional absolute `valuesFiles`, `imageRepository`, a 12-hex-commit
`imageTag` such as `git-123456789abc`, `kubeContext`, `namespace`, `release`,
`deployment`, and `test` (`go` or empty). Dockerfile and chart must be inside
the clean Git source tree. The Git tag must match that source's HEAD. The CLI
also requires `preflightDomain` (a host-resolvable DNS name) and
`preflightRegistryUrl` (a loopback registry origin). For the `host-crane`
profile, the registry URL must point to the local port 5001.
uses first-party Docker, Helm, kubectl, and Go commands with explicit arguments
and context; it never invokes a shell or passes Infisical bootstrap variables
to child commands. Values files are operator-supplied; review them before
apply and reference Kubernetes Secrets by name instead of putting secret
values in Helm release metadata.

For the KuchDesk agent chart, a values file must also provide `cluster.id`
and `cluster.name`; the workflow supplies image repository, tag, and digest.

```sh
cd host-agent
go run ./cmd/kuchdesk-deploy -profile /absolute/path/to/profile.json
go run ./cmd/kuchdesk-deploy -profile /absolute/path/to/profile.json -preflight
go run ./cmd/kuchdesk-deploy -profile /absolute/path/to/profile.json \
  -apply -confirm kuchdesk-agent/development-tools
```

Apply first runs the read-only Doctor checks for the default route, DNS,
Docker, Kubernetes node readiness, registry, source-volume free space, and
node CPU/memory use. Any failed, warning, or unavailable check blocks apply
before source validation and build. It then runs Helm lint, optional
`go test ./...`, Docker build/push, image digest
resolution, Helm template, atomic Helm upgrade, and `kubectl rollout status` in
that order. It stops before Helm if push or digest lookup fails. The Helm chart
uses `image.digest` to pin the pulled content. The CLI prints step names as
progress and reports failures without printing child output, rendered Secret
data, or Infisical credentials. A successful rollout is readiness evidence,
not an application smoke test. Helm's `--atomic` covers upgrade failures; if the subsequent
`kubectl rollout status` check fails, inspect the release before retrying
because the successful Helm upgrade remains installed. The workflow has
fake-runner and chart-render tests. A host-side `crane` push and node-side
`ctr` pull have succeeded against the live registry. A temporary unprivileged
Pod also reached `Pulled`, `Created`, and `Started` with
`imagePullPolicy=Always`; it exited because the deliberately omitted
`KUCHDESK_CLUSTER_ID` is required and was then removed. The image layers were
already cached. A manual Helm install and a subsequent full CLI build/push/
upgrade reached 1/1 Ready; the latter needed a corrective CRI pull after the
older `ctr` pre-pull step. A fresh-blob kubelet download remains unverified.

When a registry is available only through a macOS localhost port-forward,
Docker Desktop's image daemon cannot use that host-local endpoint. Set
`pushMode` to `host-crane`, `hostPushRepository` to
`127.0.0.1:5001/kuchdesk-agent`, and `imageRepository` to
`localhost:5001/kuchdesk-agent`. Install `crane` v0.20.6 from
`github.com/google/go-containerregistry/cmd/crane` on the host PATH. The
workflow saves the built Docker image to a temporary archive, pushes it from
the host with `crane`, and resolves its registry digest. `kindPrepull: true`
discovers the nodes from the explicit Kubernetes context, pre-pulls the exact
tag and digest on each node through `crictl` (the kubelet's CRI), and sets Helm's
`image.pullPolicy=Never` only after every pre-pull succeeds. This is a
Docker Desktop KIND profile, not a generic Kubernetes registry solution.
The first live workflow run exposed that a direct `ctr` pull alone could leave
kubelet at `ErrImageNeverPull`; a matching `crictl pull` made the image visible
and the digest-pinned Helm rollout reached 1/1 Ready. Future workflow runs use
CRI directly.
The registry Service and persistent data are retained. Existing cloud-hosted
images in other charts are unaffected.

The host MCP process offers `kuchdesk_deploy_plan`, `kuchdesk_deploy_preflight`, `kuchdesk_deploy_start`,
`kuchdesk_job_status`, and `kuchdesk_cancel_job`. Set an absolute
`KUCHDESK_DEPLOY_PROFILE_DIR` to permit plan reads of named `.json` files in
that directory. Start stays disabled unless `KUCHDESK_DEPLOY_ENABLED=true`
is set in the trusted host process environment; it also requires the exact
`release/namespace` value from the profile. A start runs in the existing
bounded in-process runner with a 20-minute deadline and serializes operations
for the same context, namespace, and release. Profile names cannot escape the
configured directory through path traversal or symlinks. Status and cancel
are per-process only. The MCP process requires local Docker, `crane`, Helm,
kubectl, and access to the selected Kubernetes context. This deployment path
has source tests but has not performed a live agent rollout; enabling
cluster-admin RBAC and CoreDNS edits remain separate operator decisions.
