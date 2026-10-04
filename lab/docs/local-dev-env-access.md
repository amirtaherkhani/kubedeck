# Local development agent access record

## Central development connector

- Principal: `local-dev-agents`
- Role: `member` in explicitly approved development projects only
- Reason: allow local Codex, OpenCode, and other MCP agents to read and maintain development configuration through `kubedeck-env-mcp`
- Authentication: Universal Auth credentials in the `my-home-lab.infisical.agent.local-dev` macOS Keychain service
- Rotation and expiry: client credential TTL is 365 days; access-token TTL is one hour and is renewed automatically by the connector
- Network boundary: the connector accepts only `https://infisical.local.dev` and only `local`, `development`, or `dev` environments
- Verification: `make kubedeck-env-doctor` and `npm run smoke --prefix tools/kubedeck-env-mcp -- <registered-project-root>`

Add project IDs through `INFISICAL_LOCAL_DEV_PROJECT_IDS` before running
`make kubedeck-env-bootstrap`. Never include production-only projects.

## Kubernetes

Kubernetes workloads keep their own Infisical Secrets Operator credentials and
service-specific paths. The local agent identity is not mounted into pods.
