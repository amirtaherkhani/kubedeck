# Local development agent access record

## home-lab

- Project: `home-lab` (`e064fd84-b51e-4318-8442-8f30fec2b316`)
- Environment and paths: `local`; repository root and service-scoped paths declared by `InfisicalSecret` resources
- Principal: `local-dev-agents`
- Role: `member`
- Reason: allow local Codex and OpenCode agents to read and maintain development configuration through `local-dev-env-mcp`
- Authentication: Universal Auth credentials in the `my-home-lab.infisical.agent.home-lab` macOS Keychain service
- Rotation and expiry: client secret TTL is 7 days and access-token TTL is 15 minutes. Rerun `make local-dev-env-bootstrap` to replace an expired or nonconforming credential; bootstrap stores the replacement before revoking superseded connector credentials.
- Network boundary: the installed Infisical edition does not support custom Universal Auth IP ranges, so trusted-IP fields remain unrestricted. The connector compensates by accepting only `https://infisical.local.dev`, limiting access to development environments, keeping credentials in Keychain, and using the short lifetimes above.
- Verification: `make local-dev-env-doctor` and `npm run smoke --prefix tools/local-dev-env-mcp -- /Users/mac/Documents/GitHub/my-home-lab`
- Last verified: 2026-09-15

## External projects

External projects retain separate machine identities and Keychain profiles. The central registry may select them automatically, but the home-lab identity must never receive membership in those projects. The current Finance mapping selects `my-home-lab.infisical.agent.vero-finance`; its access lifecycle remains owned by the Finance workflow.
