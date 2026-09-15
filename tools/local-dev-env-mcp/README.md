# local-dev-env-mcp

`local-dev-env-mcp` is the home-lab's central, project-aware Infisical connector for local coding agents. Agents receive six short tools and do not need to know an Infisical password, token, project ID, environment, path, or CLI profile.

The connector is intentionally limited to `local`, `development`, and `dev` environments. It automatically selects a project-scoped Universal Auth machine identity from macOS Keychain. Home-lab credentials live under `my-home-lab.infisical.agent.home-lab`; external projects keep their own identity and Keychain profile. Secret values are never stored in the project registry.

## Tools

- `env_context`: show the automatically selected non-secret context.
- `env_list`: list names without values.
- `env_get`: read one requested value.
- `env_set` and `env_set_many`: create or update values.
- `env_delete`: delete one explicitly requested value.

Each data tool also accepts an optional `path`. It must stay beneath the automatically selected scope and is useful only for charts, such as platform-storage, that own several service folders.

Project selection uses `~/.config/local-dev-env/projects.json`, choosing the longest matching workspace path. Git worktrees are mapped back to the corresponding registered path in the primary checkout. A multi-root session is rejected if its roots resolve to different Infisical scopes. Unregistered repositories cannot select a Keychain profile, and the connector accepts only `https://infisical.local.dev`. The repository bootstrap registers each home-lab chart directory with its literal `secretsPath` or safe common parent, plus `verovault-finance` with `/finance` when that checkout exists.

## Install and verify

From the home-lab repository:

```sh
make local-dev-env-bootstrap
make local-dev-env-test
make local-dev-env-doctor
```

The bootstrap creates or reuses the home-lab machine identity, grants it `member` access only to the `home-lab` project, stores a seven-day Universal Auth client secret and its ID in Keychain, installs the package globally, registers project paths, and installs the skill for Codex and OpenCode. Access tokens last 15 minutes. Re-running the bootstrap is safe and replaces expired or nonconforming credentials before revoking superseded connector credentials. The installed Infisical edition does not support custom Universal Auth IP ranges, so the fixed local domain, development-only environment guard, short credential lifetimes, project role, and Keychain storage provide the connector boundary. External registrations, such as Finance, select that project's existing profile instead of sharing the home-lab identity.

Register another local checkout without handling credentials:

```sh
local-dev-env-mcp register --project-id PROJECT_ID --environment development --path /service --profile project-profile --root /absolute/project/path
```

## Kubernetes contract

The MCP server is not used inside Kubernetes. Each chart connects directly to Infisical through an `InfisicalSecret` with `includeAllSecrets: true`, scoped to that service path. The operator writes every folder value into a native Secret. Workloads consume that service-scoped Secret through `envFrom.secretRef` where the chart supports whole-Secret environment injection, or through the chart's native `existingSecret`/`secretKeyRef` interface when it requires named keys. The controller annotation `secrets.infisical.com/auto-reload: "true"` rolls pods when the native Secret changes.

This separation keeps agent access simple while Kubernetes continues to use the official Infisical Secrets Operator and its own `credentialsRef`.
