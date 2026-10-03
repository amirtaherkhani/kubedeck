# Infisical access for local AI agents

Use `local-dev-env-mcp` as the single local connector. It resolves the
workspace project and service path automatically, reads one development-only
Universal Auth identity from macOS Keychain, and refreshes short-lived access
tokens without an interactive login.

## One-time setup

```bash
cd /Users/mac/Documents/GitHub/my-home-lab
INFISICAL_LOCAL_DEV_PROJECT_IDS=<home-lab-id>,<other-dev-project-id> \
  make local-dev-env-bootstrap
```

The bootstrap creates or reuses `local-dev-agents`, grants `member` access to
the explicitly listed development projects, and stores its client credential
in the Keychain service:

```text
my-home-lab.infisical.agent.local-dev
```

The client credential expires after **365 days**. Connector access tokens are
short-lived (one hour) and renew automatically. Verify without printing a
token or secret value:

```bash
make local-dev-env-doctor
```

Agents use the globally installed MCP server and the installed
`.codex/skills/local-dev-env/SKILL.md`. They should never ask for an Infisical
password, project ID, profile, token, or path.

## Access boundary

```text
AI client
  ↓ global MCP server
local-dev-env-mcp
  ↓ macOS Keychain
local-dev-agents Universal Auth
  ↓ renewable token
approved development projects
```

Do not include production projects in `INFISICAL_LOCAL_DEV_PROJECT_IDS`. Do
not use the administrator credential, a personal token, or `INFISICAL_TOKEN`.

## Kubernetes

Kubernetes does not use the MCP server. Keep the existing
`InfisicalSecret → managed Secret → envFrom/secretKeyRef` flow. Each workload
uses its Kubernetes `credentialsRef` and service-specific Infisical path;
local AI access and cluster runtime access remain separate.
