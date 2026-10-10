# Infisical to Helm probe

This opt-in chart proves that an Infisical secret written through the scoped
KuchDesk host bridge is synced by the existing Operator into a Kubernetes
Secret, then consumed by a Helm Deployment through `secretKeyRef`. It is
disabled by default and contains no secret value or Universal Auth bootstrap
credential. The value must be generated and sent through the authenticated
agent/host bridge only after its separate live setup is complete.

The dedicated test path is `/apps/development-tools/kuchdesk-helm-probe` in
the approved `home-lab-nb0-k` project and `dev` environment. The test secret
name is `KUCHDESK_HELM_PROBE`. The Operator reads its existing credential
reference in `platform-secrets`; the application Pod receives only the
managed test Secret. No Service or public network port is created.

Render without values to confirm the default creates no resources. For a
live test, set `enabled=true`, `infisical.projectSlug`, and
`infisical.envSlug` explicitly and install in `development-tools`. Wait for
the `InfisicalSecret` Ready condition and the Deployment to become 1/1 Ready.
Verify the Kubernetes Secret contains the expected key by checking key names
only. Do not print Secret data or pass the value through Helm values. Remove
the Helm release and the dedicated test secret from Infisical when the test is
complete, after confirming no other workload uses them.
