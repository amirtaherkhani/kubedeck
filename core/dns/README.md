# Local DNS

All new home-lab routes use the `*.local.dev` suffix, for example
`https://grafana.local.dev` and `https://kubedeck.local.dev`.

Kubernetes pods use service DNS such as
`grafana.observability.svc.cluster.local`. macOS applications must use the
local DNS/hosts integration and must not use Kubernetes service names.

CoreDNS aliases and any macOS resolver setup must be generated from the same
hostname inventory. New services are not complete until their DNS entry,
Traefik Ingress, and TLS host are documented together.

The cluster wildcard is configured by `coredns-custom.yaml` and resolves
`*.local.dev` to the macOS host address `192.168.1.100`, where Rancher Desktop
forwards the Traefik HTTP and HTTPS ports. The `home-lab-dns` LoadBalancer
exposes CoreDNS on the Rancher Desktop node address `192.168.1.183:53`; the
Homebrew `dnsmasq` listener presents the `local.dev` zone on
`192.168.1.100:53` and forwards that zone to CoreDNS.

Configure the macOS DNS listener and scoped resolver with `make macos-dns`.
This requires the user's macOS administrator password because it updates the
Homebrew `dnsmasq` configuration, reloads its launch service, and writes
`/etc/resolver/local.dev`.
