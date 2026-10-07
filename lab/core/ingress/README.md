# Traefik ingress

The `nats-monitor.local.dev` Ingress routes to the NATS monitoring HTTP
service, `platform-storage/nats-monitoring:8222`, and protects it with the
`nats-monitoring-basic-auth` Traefik middleware. It is separate from the NATS
client/broker endpoint on port 4222.

Docker Desktop's Traefik v3.6.12 provider watches the Traefik CRD API types.
Install the version-matched CRDs with:

```bash
make -C lab docker-desktop-traefik-crds
```

The pinned CRDs are sourced from Traefik Helm chart `traefik/traefik` version
39.0.7. Keep `traefik-provider-crds.yaml` aligned with the installed chart
when upgrading Traefik. Verify the route with:

```bash
curl -I https://nats-monitor.local.dev/
```

An HTTP 401 response confirms the route reaches the NATS monitor and that
Basic Auth is active; do not remove the middleware to make the URL respond.
