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

## Infisical MCP: first implementation slice

`cmd/kuchdesk-infisical-mcp` is a separate, long-running stdio MCP server. It does not change the periodic DNS LaunchAgent or run as a network listener. It uses the official MCP Go SDK v1.6.0 and Infisical's documented REST API from the deployed v0.151.0 OpenAPI; the host-agent Go module now requires Go 1.25. The local client implements Universal Auth login, in-memory token renewal from `expiresIn`, restart re-login, one bounded retry of a read after HTTP 401, and a pause after a repeated 401. It does not assume a year-long access token.

Only `INFISICAL_CLIENT_ID` and `INFISICAL_CLIENT_SECRET` carry the bootstrap credential. Supply them in the trusted host process environment, never in command arguments, Git, MCP configuration text, or prompts. Pass the HTTPS origin separately:

```sh
cd host-agent
go run ./cmd/kuchdesk-infisical-mcp -url https://YOUR-INFISICAL-HOST
```

The MCP tools are `infisical_capabilities`, `infisical_list_secret_names`, `infisical_start_list_secret_names`, `infisical_job_status`, and `infisical_cancel_job`. Name listing requires an explicit project ID, environment slug, and absolute secret path. It requests `viewSecretValue=false` and returns only names and paths, discarding any value fields from the API response. Discovery works without credentials; a secret-list call then returns `credentials_not_configured`.

The persistent MCP process executes up to four independent operations in parallel, retains at most 64 in-memory records, and gives each operation a 30-second deadline. Operations targeting the same nonempty resource key serialize in the runner; the current name-only reads use no key. Status distinguishes queued, running, succeeded, failed, canceled, and timed out. A cancellation request is cooperative through Go context; process exit discards job history and cancels in-flight work. This is lightweight agent execution, not a durable task service.

The command-line client shares the same typed Infisical service and runs synchronously:

```sh
cd host-agent
go run ./cmd/kuchdesk-infisical -url https://YOUR-INFISICAL-HOST capabilities
go run ./cmd/kuchdesk-infisical -url https://YOUR-INFISICAL-HOST list-secret-names -project PROJECT_ID -environment dev -path /app
```

No real identity, credential, role, or live MCP service is provisioned by this source change. Permissions and bootstrap credential lifetime remain unverified against this installation.

Secret writes, project/identity administration, value delivery, a host HTTP API, Kubernetes-agent bridging, and Helm execution are not implemented. Existing Grafana Operator sync remains independent.
