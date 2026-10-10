# Host control plane

The macOS host owns the Infisical Universal Auth bootstrap pair in its private
project `.env` (or the explicitly selected Keychain backend). The Kubernetes
agent never receives that pair. The periodic Technitium DNS LaunchAgent stays
one-shot; a separate, opt-in host process serves the command API.

```text
AI client ── bearer ── Kubernetes agent ── TLS + bridge bearer ── host API
AI MCP client ───────────────────────────── stdio ────────────────┤
local CLI ────────────────────────────────────────────────────────┤
                                                               Infisical client
                                                                    │
                                                               Infisical API

Infisical Operator ── Kubernetes Secret ── application Pod
```

The host command service validates named operations and explicit project,
environment, and folder scope before using the installed Infisical v0.151 API.
It returns typed metadata and write receipts, never bootstrap tokens, secret
values, or upstream response bodies. A secret value may enter a create/update
request over authenticated TLS but is not echoed, logged, stored in a Helm
release, or returned to an AI client. Non-idempotent writes are not replayed
after an authentication failure. Infisical enforces the machine identity's
project role on every call; the host service also restricts project IDs when a
project allowlist is configured. Project creation is a separate opt-in action.

The network bridge is disabled by default. To enable it later, an operator
must supply a host listener address, TLS certificate and private key, a
dedicated bridge bearer token, and an explicit project allowlist. The agent
needs only the bridge URL, a trusted CA certificate, and a separate bridge
bearer Secret referenced by name in Helm values. Its existing API bearer
protects AI-facing routes. The chart does not accept a literal token or
Infisical bootstrap pair. Both transports bound request and response sizes
and timeouts. Host availability affects new commands, not existing Operator
Secret synchronization.

Doctor is a separate host command using its existing typed checks. The agent
can request a Doctor report through the same bridge, so a Helm-installed agent
needs no Docker socket or host kubeconfig. Chart render tests must prove the
bridge starts disabled and, when enabled, renders only Secret/ConfigMap
references. A live rollout, listener, project role, MCP registration, and DNS
probe remain separate operator decisions.
