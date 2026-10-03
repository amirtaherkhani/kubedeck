# n8n

This local Helm chart deploys n8n with PostgreSQL persistence, a persistent
`/home/node/.n8n` volume for the encryption key/settings file, a Traefik
Ingress, and a Prometheus `ServiceMonitor`.

The chart intentionally does not create secret values. Bootstrap the `n8n`
secret and database first, then install the release through the repository
release manager.

The default image tag is `2.30.7-arm64` for the local Mac/Rancher Desktop
cluster. Use the matching `2.30.7-amd64` tag on x86 nodes.
