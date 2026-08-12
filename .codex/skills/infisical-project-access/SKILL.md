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
`/apps/development-tools/paperclip`. External deployments use their own
project; a folder is not a substitute for project membership.

## Required workflow

1. Identify project, environment, folder path, principal type, role, reason,
   and expiry/rotation plan.
2. Create or reuse only the project's machine identity and add it to that
   project.
3. Store client credentials in macOS Keychain or a Kubernetes Secret in
   `platform-secrets`, never in Git or chat.
4. Use `scripts/infisical-agent-access.sh` for external repositories. It writes
   only non-secret `.infisical.json` metadata and runs commands with a
   short-lived token.
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
returns only the profile name and scope, never a secret. Agents must not request
organization-admin access to solve a project-level problem.

## Safe commands

```bash
./scripts/infisical-agent-access.sh status --profile <project-profile>
./scripts/infisical-agent-access.sh run --profile <project-profile> -- <command>
```

Do not redirect secrets to files or print them in logs.
