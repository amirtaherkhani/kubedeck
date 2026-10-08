# KubeDeck macOS Host Agent: local DNS reconciler

This small Go process runs in the signed-in macOS user's `launchd` session. It discovers the default-route interface and the Mac's private IPv4 address, opens a localhost-only Kubernetes port-forward to Technitium's admin API, and reconciles `*.local.dev` with a 30-second TTL so the Mac can keep opening services by name. It reads the existing `technitium-admin` Secret through the user's `docker-desktop` kubeconfig; no credential or host IP is stored in this repository or process arguments. It skips writes when the DNS record is already current. `RunAtLoad` and a 30-second launchd interval cover IP changes and restart recovery.

Prerequisites: Docker Desktop Kubernetes running in the `docker-desktop` context, a healthy Technitium Service, `kubectl`, Go for installation, and access to the Technitium Secret. Install for the current macOS user:

```sh
./host-agent/install-macos.sh
launchctl print "gui/$(id -u)/dev.kubedeck.host-agent"
```

For a single reconciliation without installing launchd:

```sh
cd host-agent && go run ./cmd/kubedeck-host-agent
```

If a VPN changes the default route to `utun`, the agent refuses to publish that address. On a multi-interface Mac, run the binary with `-interface en0` or another interface that reaches the local ingress after checking the routing profile. The agent currently handles IPv4 only. It updates the `local.dev` DNS **answer** for use by the current Mac.

The agent's Kubernetes credential can read the existing admin Secret. Treat that user account and kubeconfig as privileged; a dedicated scoped Technitium API credential and finer host permissions belong to later hardening work. The admin API is reachable only through a localhost port-forward.

The DNS reconciler opens its own temporary localhost port-forward. This repository does not install a fixed-port Technitium admin forwarder or a local image registry.
