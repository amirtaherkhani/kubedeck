---
name: infisical-project-access
description: Use when configuring Infisical access for a home-lab service, external project, human, AI agent, Kubernetes workload, or project-scoped secret synchronization.
---

# Infisical Project Access

`my-home-lab` administers the Infisical organization and the shared `home-lab`
project. Applications deployed by this repository use that project with
service-specific paths and the home-lab Kubernetes identity. Projects
deployed outside this repository retain their own environments, paths,
machine identities, client secrets, and access audit.

Use named human accounts for people. Use one Universal Auth machine identity
per agent or workload. Never share the administrator token, a human password,
or another project's client secret.

## Roles

- `viewer`: read-only agent or observer.
- `member`: agent that must create, edit, or delete project secrets.
- `admin`: designated project owner only.

Prefer the narrowest path and environment scope. Home-lab services use the
`home-lab` project and a service path such as
`/apps/development-tools/<service>`. External deployments use their own
project; a folder is not a substitute for project membership.

## Required workflow

1. Identify project, environment, folder path, principal type, role, reason,
   and expiry/rotation plan.
2. For local development, create or reuse the central `local-dev-agents`
   machine identity and add it only to explicitly approved development
   projects. Kubernetes workloads still use their own project identity.
3. Store client credentials in macOS Keychain or a Kubernetes Secret in
   `platform-secrets`, never in Git or chat.
4. Use the globally installed `kubedeck-env-mcp` connector for local agent
   access. It selects project and path from the workspace registry, reads the
   central Keychain credential, and renews short-lived tokens.
5. For Kubernetes, use an Infisical Secrets Operator resource with the
   home-lab `credentialsRef` for services deployed by this repository, or the
   external project's credential for deployments outside it.
6. Verify login, project access, `ReadyToSyncSecrets=True`, and the target
   workload before reporting success.
7. Record the principal, role, scope, and verification; revoke and rotate when
   the agent, human, or task ends.

## Communication contract

An access request states: project ID/slug, environment, path, read/write need,
identity name, duration, and verification command. The home-lab administrator
returns only the connector scope, never a secret. Agents must not request
organization-admin access to solve a project-level problem.

## Safe commands

```bash
make local-dev-env-doctor
kubedeck-env-mcp register --project-id ID --environment development --root /absolute/project/path
```

Do not redirect secrets to files or print them in logs.

## Home-lab application contract

For a service deployed by `my-home-lab`, use the shared `home-lab` project,
the `local` environment, and a service path such as
`/apps/development-tools/<service>`. Store secret and non-secret runtime
configuration there. The service's `InfisicalSecret` must reference the
existing `platform-secrets/infisical-universal-auth` credential and map only
the keys the workload needs.

If a service needs a first administrator, keep bootstrap-only email/password
keys in Infisical and use the vendor-supported one-shot flow after the workload
is Ready. Use `admin`, `admin`, `admin`, and `admin@local.dev` as the standard
identity; never print the generated password or mount it into the long-running
pod unless required.

For Codex ACP or another server-side agent adapter, an interactive login in a
separate Codex session is not sufficient. The server must receive its adapter
credential through its own Infisical-backed environment, for example
`OPENAI_API_KEY`. If it is absent, report the adapter unavailable while
keeping the base application healthy; never copy a credential from another
service path.
