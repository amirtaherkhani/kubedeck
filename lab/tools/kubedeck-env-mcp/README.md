# kubedeck-env-mcp

`kubedeck-env-mcp` is the central, project-aware Infisical connector for
local AI agents. Agents receive short `env_*` tools and do not need an
Infisical password, token, project ID, environment, path, or profile.

The connector accepts only `local`, `development`, and `dev` environments. It
selects the project and path from the registered workspace root, including Git
worktrees, and uses one Keychain-held development machine identity for every
explicitly approved project. Secret values are never stored in the registry.

## Tools

- `env_context`: show the selected non-secret project context.
- `env_list`: list names without values.
- `env_get`: read one requested value.
- `env_set` and `env_set_many`: create or update values.
- `env_delete`: delete one explicitly requested value.

Each data tool accepts an optional descendant `path`; it cannot escape the
registered service scope. Multi-root sessions must resolve to one Infisical
scope.

## Install and verify

From the home-lab repository:

```sh
make kubedeck-env-bootstrap
make kubedeck-env-test
make kubedeck-env-doctor
```

The bootstrap creates or reuses the `local-dev-agents` machine identity,
grants it `member` access to the explicitly configured development projects,
stores a **365-day Universal Auth client credential** and its ID in macOS
Keychain, installs the package globally, registers project paths, and
installs the skill for Codex and OpenCode. Access tokens last one hour and are
renewed automatically by the connector, so agents never ask for login again.

By default the bootstrap includes the home-lab and Finance development
projects. Add more project IDs without changing the connector:

```sh
INFISICAL_LOCAL_DEV_PROJECT_IDS=home-project-id,finance-project-id,other-project-id \
  make kubedeck-env-bootstrap
```

Register an additional local checkout without handling credentials:

```sh
kubedeck-env-mcp register \
  --project-id PROJECT_ID \
  --environment development \
  --path /service \
  --root /absolute/project/path
```

The installed MCP configuration is global for the local AI clients. The
connector reads the client ID and secret from
`my-home-lab.infisical.agent.local-dev`; no interactive `infisical login` and
no `.infisical.json` is required.

The `local-dev-env-mcp` binary and `local-dev-env-*` Make targets remain
compatibility aliases. Existing `~/.config/local-dev-env/projects.json`,
Keychain credentials, and MCP client registrations are left in place. This
source rename does not install or register a new global MCP server. A client
using the old binary continues to work; changing global client registrations
requires a separate review. The approved bootstrap registration now replaces
the old client stanza with one `kubedeck-env` stanza instead of duplicating it.

The bootstrap configures a 365-day Universal Auth client credential and
one-hour access tokens. The connector requests a new access token 30 seconds
before expiry, retries once on an API 401, and shares a single refresh among
concurrent callers. The long-lived Keychain credential is not auto-rotated;
run the approved bootstrap/rotation flow before its expiry.

## Kubernetes contract

The MCP server is not used inside Kubernetes. Each chart connects directly to
Infisical through an `InfisicalSecret` with `includeAllSecrets: true`, scoped
to its service path. The operator writes every folder value into a native
Secret; workloads consume it through `envFrom.secretRef`, `existingSecret`, or
`secretKeyRef`, with `secrets.infisical.com/auto-reload: "true"` where
supported.
