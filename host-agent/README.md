# KubeDeck Host DNS Agent

The Go CLI discovers the default-route interface and its private IPv4 address, opens a localhost-only Kubernetes port-forward to Technitium's admin API, and reconciles one configured DNS wildcard with a 30-second TTL. It reads the admin password from a Kubernetes Secret; no credential or host IP is stored in this repository. It skips writes when the DNS record is already current. macOS uses `route` for automatic interface discovery, Linux uses `iproute2`, and other operating systems can pass `-interface` or `-target-ip` explicitly. The included installer configures a macOS `launchd` job; the CLI can be scheduled separately on Linux.

Prerequisites: a reachable Kubernetes cluster with a healthy Technitium Service, `kubectl`, Go for building, and access to the Technitium Secret. The macOS installer pins the selected Kubernetes context in the LaunchAgent, so later changes to the shell's current context do not redirect DNS writes. `RunAtLoad` and a 30-second interval cover IP changes and restart recovery on macOS. Install for the current macOS user:

```sh
KUBEDECK_HOST_AGENT_KUBE_CONTEXT=docker-desktop \
KUBEDECK_HOST_AGENT_ZONE=local.dev \
./host-agent/install-macos.sh
launchctl print "gui/$(id -u)/dev.kubedeck.host-agent"
```

The installer requires `KUBEDECK_HOST_AGENT_ZONE` and otherwise uses the current `kubectl` context unless `KUBEDECK_HOST_AGENT_KUBE_CONTEXT` is set. To use another Technitium deployment, pass `KUBEDECK_HOST_AGENT_NAMESPACE`, `KUBEDECK_HOST_AGENT_SERVICE`, `KUBEDECK_HOST_AGENT_ADMIN_SECRET`, `KUBEDECK_HOST_AGENT_PASSWORD_KEY`, `KUBEDECK_HOST_AGENT_ADMIN_USER`, or `KUBEDECK_HOST_AGENT_API_PORT` to the installer. `KUBEDECK_HOST_AGENT_INTERFACE` selects a non-default interface; `KUBEDECK_HOST_AGENT_TARGET_IP` sets an explicit IPv4 address directly, including loopback for host-only DNS. These settings are saved as LaunchAgent arguments, not as repository defaults. A non-default `KUBECONFIG` is retained as absolute paths by the LaunchAgent.

For a single reconciliation without installing launchd, pass the zone and context explicitly:

```sh
cd host-agent && go run ./cmd/kubedeck-host-agent -kube-context docker-desktop -zone local.dev
```

The same CLI can run on Linux with a reachable Kubernetes context. For example, run `go run ./cmd/kubedeck-host-agent -kube-context kind-team -zone dev.example -target-ip 10.2.3.4` from `host-agent/`; provide the target IP explicitly if the default route does not lead to the intended ingress. An external scheduler can repeat the command. The repository does not install a Linux service or modify Linux DNS settings.

If a VPN changes the default route to a tunnel interface, the agent refuses to publish that address. On a multi-interface host, use `-interface` with the interface that reaches the local ingress after checking the routing profile, or use `-target-ip` with the intended IPv4 address. Automatic selection still chooses a private address; an explicit loopback address is useful only when the DNS clients run on that same host. The agent currently handles IPv4 only. It updates only the configured zone's wildcard DNS **answer**; it does not set system-wide DNS servers or routing.

The agent's Kubernetes credential can read the existing admin Secret. Treat that user account and kubeconfig as privileged; a dedicated scoped Technitium API credential and finer host permissions belong to later hardening work. The admin API is reachable only through a localhost port-forward.

The DNS reconciler opens its own temporary localhost port-forward. This repository does not install a fixed-port Technitium admin forwarder or a local image registry.
