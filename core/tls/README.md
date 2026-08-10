# Local HTTPS

Every web service uses Traefik with HTTPS and the shared `local-dev-tls`
certificate. Plain HTTP is permitted only as a redirect to the HTTPS route.

Service-specific certificates are allowed only when a service requires a
different trust boundary and the exception is documented in its README.

The lab uses one cert-manager-generated local CA in `platform-system/local-dev-ca`.
Every namespace-local `local-dev-tls` certificate is signed by that CA. macOS
trusts the CA once, so adding a new
service does not require another per-service trust operation.

Canonical and supported service URLs use `*.local.dev`.

For a new application namespace, issue `local-dev-tls` with the
`local-dev-ca` ClusterIssuer, or add the cert-manager ingress-shim annotation
`cert-manager.io/cluster-issuer: local-dev-ca` to an Ingress that uses the
`local-dev-tls` secret.

Run `make macos-trust-tls` to import the shared CA into the macOS System
keychain. Restart browsers after importing it.
