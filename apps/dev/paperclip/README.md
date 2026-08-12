# Paperclip

Paperclip is the private local AI-agent orchestration service. This chart runs the Paperclip server, persistent Paperclip state, the bundled Codex local adapter, and HTTPS access at `https://paperclip.local.dev`. It uses the shared home-lab PostgreSQL service in `platform-storage`.

## Ownership and namespace

- Owner: home-lab platform operations
- Namespace: `development-tools`
- Source: [paperclipai/paperclip](https://github.com/paperclipai/paperclip)
- Image: local immutable build `localhost:5001/homelab/dev/paperclip:v0.1.1`
- Dependencies: `platform-secrets` Infisical, `platform-storage/postgresql`, `development-tools` namespace, `local-path` storage, Traefik `local-dev-tls`

## Runtime configuration

Paperclip is deployed and managed by `my-home-lab`, so it uses the shared Infisical project `home-lab`, environment `local`, and path `/apps/development-tools/paperclip`. The chart uses the existing Kubernetes credential Secret `infisical-universal-auth` in `platform-secrets`; it contains only the home-lab Infisical Universal Auth `clientId` and `clientSecret` keys. External GitHub source ownership does not change this deployment boundary.

Paperclip is intentionally single-environment in this home lab. Only the
Infisical `local` environment is supported; staging and production releases,
extra values files, and multi-environment deployment modes are disabled.

The complete non-secret requirement is recorded in [`infisical.requirements.yaml`](infisical.requirements.yaml). The `/apps/development-tools/paperclip` path and required keys belong in the existing `home-lab` / `local` project. Do not create a second Paperclip project or credential Secret for this deployment.

Create these keys in the home-lab Infisical path. Values are intentionally not stored in Git:

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
PAPERCLIP_CODEX_AUTH_JSON
```

The chart maps these keys into a managed Secret named `paperclip-runtime`; it does not contain secret values. `PAPERCLIP_DATABASE_URL` must target the existing `postgresql.platform-storage.svc.cluster.local:5432` service. Use the existing platform PostgreSQL database and credentials; do not add another PostgreSQL release or StatefulSet for Paperclip.

## First administrator

After the first Paperclip rollout is Ready, create the first administrator with
the supported one-shot host bootstrap flow. Use `admin` for username, first
name, and family name, and `admin@local.dev` for email. Generate or supply the
password through the `PAPERCLIP_ADMIN_PASSWORD` key in Infisical and remove the
bootstrap task after the account is claimed. Never print the password, put it
in Git, or inject bootstrap-only credentials into the long-running pod.

## Codex configuration

The Paperclip image installs the upstream local Codex adapter and the Codex CLI. ACP runs in the Paperclip server process, so its credential must be visible to that server. The local setup stores the existing Codex login JSON as `PAPERCLIP_CODEX_AUTH_JSON` in Infisical; the chart mounts it only as `/var/run/paperclip-codex/auth.json` and sets `CODEX_HOME` to that directory. A `/login` in a separate Codex or chat session does not authenticate the Paperclip server unless its auth metadata is explicitly provisioned to the server. Do not place credentials in this repository, Helm values, or Kubernetes manifests. Agent work is persisted under the Paperclip PVC at `/paperclip`.

## Deployment and verification

```sh
make -C /Users/mac/Documents/GitHub/my-home-lab core-validate
make -C /Users/mac/Documents/GitHub/my-home-lab helm-lint
make -C /Users/mac/Documents/GitHub/my-home-lab helm-validate RELEASE=paperclip
make -C /Users/mac/Documents/GitHub/my-home-lab helm-apply RELEASE=paperclip
kubectl -n development-tools rollout status deployment/paperclip --timeout=15m
kubectl -n development-tools get infisicalsecret,pod,svc,ingress,pvc -l app.kubernetes.io/name=paperclip
curl --fail --silent --show-error https://paperclip.local.dev/health
```

Before deployment, verify the Paperclip InfisicalSecret reports `ReadyToSyncSecrets=True` and the shared `platform-storage/postgresql` StatefulSet is Ready. After deployment, verify the Paperclip server pod is Ready, the HTTPS Ingress uses `local-dev-tls`, DNS resolves through the local.dev workflow, the existing PostgreSQL service is reachable, and the health endpoint returns success. No load test is required for this local control-plane service.
