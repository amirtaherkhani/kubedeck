# Plane

Plane Community Edition is the home lab project-management and issue-tracking
service for personal projects.

| Field | Value |
|---|---|
| Category | Development / project management |
| Namespace | `development-tools` |
| UI | Yes: `https://plane.local.dev` |
| Primary port | HTTPS `443`; service HTTP is internal to the chart |
| Dependencies | PostgreSQL, Valkey, RabbitMQ, MinIO, Traefik, Infisical |
| Storage | `local-path`: PostgreSQL 1 Gi, MinIO 1 Gi, Valkey 100 Mi, RabbitMQ 100 Mi |
| Chart | `makeplane/plane-ce` `1.6.2`, app `v1.4.1` |
| Credentials | Generated credentials are stored in Infisical under `/apps/development-tools/plane/*` |
| Initial administrator | `admin@local.dev`; first name `admin`, last name `admin` |
| Initial workspace | `home-lab` with slug `home-lab` |
| Admin configuration | Infisical `/apps/development-tools/plane/admin` |
| Monitoring | Alloy collects Kubernetes logs; no upstream Prometheus endpoint is enabled by this chart |

Plane is exposed through the shared `local.dev` TLS certificate and native
Traefik `IngressRoute` resources. Create a GitHub App after deployment using
Plane's documented callback URLs, then store its credentials in Infisical.
Plane signs users in with the administrator email; its first-run flow generates
an internal username automatically.

## GitHub connection

The GitHub App must be created in the user's GitHub account. Use these exact
URLs when creating it:

| GitHub App field | Value |
|---|---|
| Homepage URL | `https://plane.local.dev` |
| Setup URL | `https://plane.local.dev/silo/api/github/auth/callback` |
| Callback URL | `https://plane.local.dev/silo/api/github/auth/callback` |
| User callback URL | `https://plane.local.dev/silo/api/github/auth/user/callback` |
| Webhook URL | `https://plane.local.dev/silo/api/github/github-webhook` |

After creation, add the App ID, client ID, client secret, private key, and
webhook secret to Infisical under
`/apps/development-tools/plane/github` using `PLANE_GITHUB_`-prefixed keys,
then finish the integration in Plane's God Mode. These account credentials are
not generated or stored in this repository.
