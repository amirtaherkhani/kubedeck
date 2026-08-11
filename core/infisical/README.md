# Infisical access for local AI agents

Use one Infisical Universal Auth machine identity per external project. Give
that identity only the project role and folders it needs. Do not use the
Infisical administrator account, a shared service token, or credentials from a
different project.

The helper stores each project's `clientId` and `clientSecret` in macOS
Keychain, mints a short-lived token for each command, and writes only
non-secret `.infisical.json` metadata into the external project. Credentials
and access tokens are not written to Git or printed by the helper.

## Install

```bash
cd /Users/mac/Documents/GitHub/my-home-lab
./scripts/infisical-agent-access.sh install
```

## Configure a project profile

The Kubernetes Secret must contain the client credentials for a machine
identity that belongs to the same Infisical project:

```bash
./scripts/infisical-agent-access.sh configure \
  --profile vero-finance \
  --project-id 82075663-04cb-4b35-a685-1d7a46b6193d \
  --environment development \
  --path /finance \
  --kube-secret vero-vault-finance-infisical-auth \
  --project-dir /Users/mac/Documents/GitHub/verovault-finance
```

The helper verifies the `rancher-desktop` context and a Ready node before
reading Kubernetes credentials. It creates or updates the Keychain service
`my-home-lab.infisical.agent.vero-finance` and writes this non-secret file to
the external project:

```json
{
  "workspaceId": "<project-id>",
  "defaultEnvironment": "development",
  "domain": "https://infisical.local.dev"
}
```

## Use from an AI agent or project task

Inject secrets into a command without putting credentials in the command line:

```bash
./scripts/infisical-agent-access.sh run \
  --profile vero-finance -- npm test
```

Export secrets to stdout for a controlled pipe:

```bash
./scripts/infisical-agent-access.sh export \
  --profile vero-finance --format dotenv
```

Check the short-lived token without printing it:

```bash
./scripts/infisical-agent-access.sh status --profile vero-finance
```

Prefer `run` over redirecting `export` to a file. Keep `.env` files, exported
secret files, client secrets, and access tokens out of Git and agent logs.

## Access model

Each project should have its own machine identity, Universal Auth client
secret, Keychain profile, and Infisical project membership. Use the `viewer`
role for read-only agents and `member` only when an agent must edit project
secrets. Organization admin access is not required.

For Kubernetes workloads, continue using the Infisical Secrets Operator and a
project-scoped identity Secret in `platform-secrets`. This helper is for
external project directories and local AI-agent commands.
