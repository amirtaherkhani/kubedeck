# Paperclip

Paperclip is the private local AI-agent orchestration service. This chart runs the Paperclip server, persistent Paperclip state, the bundled Codex local adapter, and HTTPS access at `https://paperclip.local.dev`. It uses the shared home-lab PostgreSQL service in `platform-storage`.

## Ownership and namespace

- Owner: home-lab platform operations
- Namespace: `development-tools`
- Source: [paperclipai/paperclip](https://github.com/paperclipai/paperclip)
- Image: local immutable build `localhost:5001/homelab/dev/paperclip:v0.1.0`
- Dependencies: `platform-secrets` Infisical, `platform-storage/postgresql`, `development-tools` namespace, `local-path` storage, Traefik `local-dev-tls`

## Runtime configuration

Paperclip is an external project and must use its own Infisical project and Universal Auth machine identity. Create the project slug `paperclip`, environment `local`, and path `/apps/development-tools/paperclip`, then create the Kubernetes credential Secret named `paperclip-infisical-universal-auth` in `platform-secrets` with only the Infisical Universal Auth `clientId` and `clientSecret` keys. The identity must be limited to this project/path and must not reuse the home-lab credential.

The complete non-secret requirement is recorded in [`infisical.requirements.yaml`](infisical.requirements.yaml). Create the `paperclip` project, its `local` environment, the `/apps/development-tools/paperclip` path, and the viewer Universal Auth identity described there. Store the identity credentials only in the `paperclip-infisical-universal-auth` Secret in `platform-secrets`.

Create these keys in the Paperclip Infisical path. Values are intentionally not stored in Git:

```text
PAPERCLIP_DATABASE_URL
PAPERCLIP_BETTER_AUTH_SECRET
PAPERCLIP_TOOL_ACTION_SIGNING_SECRET
PAPERCLIP_SECRETS_MASTER_KEY
PAPERCLIP_PUBLIC_URL
PAPERCLIP_DEPLOYMENT_MODE
PAPERCLIP_DEPLOYMENT_EXPOSURE
PAPERCLIP_AUTH_DISABLE_SIGN_UP
PAPERCLIP_SECRETS_PROVIDER
PAPERCLIP_TELEMETRY_DISABLED
```

The chart maps these keys into a managed Secret named `paperclip-runtime`; it does not contain secret values. `PAPERCLIP_DATABASE_URL` must target the existing `postgresql.platform-storage.svc.cluster.local:5432` service. Use the existing platform PostgreSQL database and credentials; do not add another PostgreSQL release or StatefulSet for Paperclip.

## Codex configuration

The Paperclip image installs the upstream local Codex adapter and the Codex CLI. After the first login, create or select a Codex agent in the Paperclip UI and complete its Codex authentication through the UI's agent setup. Do not place an OpenAI token in this repository, a Helm value, or a Kubernetes manifest. Agent work is persisted under the Paperclip PVC at `/paperclip`.

## Deployment and verification

```sh
make -C /Users/mac/Documents/GitHub/my-home-lab core-validate
make -C /Users/mac/Documents/GitHub/my-home-lab helm-lint
make -C /Users/mac/Documents/GitHub/my-home-lab platform-helm RELEASE=paperclip apply
kubectl -n development-tools rollout status deployment/paperclip --timeout=15m
kubectl -n development-tools get infisicalsecret,pod,svc,ingress,pvc -l app.kubernetes.io/name=paperclip
curl --fail --silent --show-error https://paperclip.local.dev/health
```

Before deployment, verify the Paperclip InfisicalSecret reports `ReadyToSyncSecrets=True` and the shared `platform-storage/postgresql` StatefulSet is Ready. After deployment, verify the Paperclip server pod is Ready, the HTTPS Ingress uses `local-dev-tls`, DNS resolves through the local.dev workflow, the existing PostgreSQL service is reachable, and the health endpoint returns success. No load test is required for this local control-plane service.
