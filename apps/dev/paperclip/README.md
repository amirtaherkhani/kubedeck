# Paperclip

Paperclip is the private local AI-agent orchestration service. This chart runs the Paperclip server, its dedicated PostgreSQL instance, persistent Paperclip state, the bundled Codex local adapter, and HTTPS access at `https://paperclip.local.dev`.

## Ownership and namespace

- Owner: home-lab platform operations
- Namespace: `development-tools`
- Source: [paperclipai/paperclip](https://github.com/paperclipai/paperclip)
- Image: local immutable build `localhost:5001/homelab/dev/paperclip:v0.1.0`
- Dependencies: `platform-secrets` Infisical, `development-tools` namespace, `local-path` storage, Traefik `local-dev-tls`

## Runtime configuration

Paperclip is an external project and must use its own Infisical project and Universal Auth machine identity. Create the project slug `paperclip`, environment `local`, and path `/apps/development-tools/paperclip`, then create the Kubernetes credential Secret named `paperclip-infisical-universal-auth` in `platform-secrets` with only the Infisical Universal Auth `clientId` and `clientSecret` keys. The identity must be limited to this project/path and must not reuse the home-lab credential.

Create these keys in the Paperclip Infisical path. Values are intentionally not stored in Git:

```text
PAPERCLIP_DATABASE_URL
PAPERCLIP_DATABASE_USER
PAPERCLIP_DATABASE_NAME
PAPERCLIP_DATABASE_PASSWORD
BETTER_AUTH_SECRET
PAPERCLIP_TOOL_ACTION_SIGNING_SECRET
PAPERCLIP_SECRETS_MASTER_KEY
PAPERCLIP_AUTH_DISABLE_SIGN_UP
```

The chart maps the keys into a managed Secret named `paperclip-runtime`; it does not contain secret values. `PAPERCLIP_DATABASE_URL` must target `paperclip-postgresql.development-tools.svc.cluster.local:5432` with the database credentials stored in the same Infisical path.

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

Before deployment, verify the Paperclip InfisicalSecret reports `ReadyToSyncSecrets=True`. After deployment, verify the server and PostgreSQL pods are Ready, the HTTPS Ingress uses `local-dev-tls`, DNS resolves through the local.dev workflow, and the health endpoint returns success. No load test is required for this local control-plane service.
