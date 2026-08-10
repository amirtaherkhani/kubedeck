# Local DNS

All new home-lab routes use the `*.local.dev` suffix, for example
`https://grafana.local.dev` and `https://kubedeck.local.dev`.
The old `*.dev.local` suffix is supported only to redirect old bookmarks to
the canonical `*.local.dev` URL.

Kubernetes pods use service DNS such as
`grafana.observability.svc.cluster.local`. macOS applications must use the
local DNS/hosts integration and must not use Kubernetes service names.

CoreDNS aliases and any macOS resolver setup must be generated from the same
hostname inventory. New services are not complete until their DNS entry,
Traefik Ingress, and TLS host are documented together.

The cluster wildcard is configured by `coredns-custom.yaml` and resolves
`*.local.dev` to the Rancher Desktop Traefik address `192.168.64.14`. The
legacy `*.dev.local` wildcard resolves to the same address for the redirect.
`home-lab-dns` LoadBalancer exposes CoreDNS on `192.168.64.14:53` so macOS can
query the same records through a local resolver.

Configure the macOS scoped resolver with `make macos-dns`. This requires the
user's macOS administrator password because it writes `/etc/resolver/local.dev`
and the compatibility `/etc/resolver/dev.local` file.
