# Infisical project onboarding

At the 2026-10-10 local check, the host bridge permitted only the existing
`home-lab-nb0-k` project. Its machine identity has a permanent Project Admin
membership there and a separate organization Admin role. No cross-project
automatic grant or dynamic bridge allowlist is enabled.

Infisical's [machine identity guide](https://infisical.com/blog/introducing-machine-identities)
distinguishes the organization role from adding an identity to a project.

## Why an all-project reconciler is not ready

The installed Infisical v0.151.0 OpenAPI describes
`GET /api/v2/organizations/{organizationId}/workspaces` as returning projects
the caller belongs to. The live machine identity listed its one known project;
that response cannot prove there are no other organization projects or discover
future projects created elsewhere. The normal identity-membership write API is
project-scoped. We have not verified that this identity can add itself to a
project where it has no membership, and there is no safe basis for retrying a
403 with a user token or widening the host bridge automatically.

For a project created outside this host control plane, a project administrator
must first add the existing machine identity with the intended project role
using Infisical's project access controls. After that, verify the identity's
membership and the project ID, then explicitly add that ID to the host bridge's
allowlist and restart only the bridge process. Do not put the Universal Auth
client secret in Kubernetes, Helm values, an agent prompt, or a second
credential store to avoid this step.

Projects created through the host control plane may have different creator
permissions. Verify the resulting machine-identity membership before allowing
the new project through the bridge. Project creation is currently disabled in
the live bridge, so this path has not been tested.

An automatic all-project policy needs a supported, version-pinned Infisical
organization-level discovery and grant mechanism, plus an explicit decision
to expand access. Until both exist, keep the home-lab allowlist and fail closed
for other project IDs.
