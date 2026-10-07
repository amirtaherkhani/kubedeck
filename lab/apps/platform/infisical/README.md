# Infisical

Web-based source of truth for all home-lab secret and non-secret environment
configuration.

| Field | Value |
|---|---|
| Namespace | `platform-secrets` |
| UI | Yes: `https://infisical.local.dev` |
| Storage | Dedicated PostgreSQL and Redis in `platform-secrets` (Infisical exception to shared data services) |
| HTTPS | Traefik and `local-dev-tls` |
| Integration | Infisical Secrets Operator |

The project/environment convention is `home-lab` / `local`. Each app will
have a clear path for runtime values, and synced Kubernetes Secret names will
be documented beside the app. Credentials are intentionally absent from the
repository.

Run `make infisical-bootstrap` before the first Helm deployment. It creates
one-time, generated Kubernetes bootstrap secrets without writing them to Git.

Run `make infisical-admin-bootstrap` after the deployment. This creates the
first administrator and the `home-lab` organization through Infisical's
supported bootstrap endpoint. The administrator password is generated only
when no password exists, stored in macOS Keychain under
`my-home-lab.infisical.admin`, and never written to Git. The generated machine
identity token is stored in the Kubernetes Secret
`platform-secrets/infisical-bootstrap-admin` for later automation.

The standard administrator identity is `admin@local.dev`. Application
configuration is migrated into Infisical before application releases are
deployed, then synchronized with the Infisical Secrets Operator.
