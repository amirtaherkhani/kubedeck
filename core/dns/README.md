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
exposes CoreDNS on the current Rancher Desktop node address at port 53; the
Homebrew `dnsmasq` listener presents the `local.dev` zone on
`192.168.1.100:53` and forwards that zone to the discovered CoreDNS address.
The forwarding helper queries the `home-lab-dns` ServiceLB address on each run,
with `HOME_LAB_CLUSTER_DNS_IP` available as an explicit override.

Configure the macOS DNS listener and scoped resolver with `make macos-dns`.
This requires the user's macOS administrator password because it updates the
Homebrew `dnsmasq` configuration, reloads its launch service, and writes
`/etc/resolver/local.dev`.

## macOS resolver repair

The CoreDNS LoadBalancer address and the `local.dev` answer are deliberately
different:

- `home-lab-dns` is the DNS **server** address. Rancher Desktop can assign it a
  new ServiceLB address after a restart.
- CoreDNS answers `*.local.dev` with `192.168.1.100`, the macOS host address
  that accepts the HTTPS ingress traffic.

If `grafana.local.dev` or another home-lab URL resolves to public DNS instead of
the local address, do not write `192.168.1.100` into
`/etc/resolver/local.dev`. Discover and verify the live DNS server, then repair
the scoped resolver:

```zsh
make macos-dns-check
./core/host/macos/configure-local-dev-resolver.sh
```

The script refuses to change the resolver unless the `rancher-desktop` context
has a Ready node and `home-lab-dns` resolves `grafana.local.dev` to the expected
macOS ingress address. It updates only `/etc/resolver/local.dev`, flushes the
macOS DNS cache, and reloads `mDNSResponder`.

Use these checks after the repair:

```zsh
dscacheutil -q host -a name grafana.local.dev
curl --fail --head https://grafana.local.dev/login
```

Use `dscacheutil`, not a bare `dig`, to check the macOS scoped resolver. A
direct authoritative check remains useful when diagnosing the cluster DNS
service:

```zsh
DNS_SERVER="$(kubectl --context rancher-desktop -n kube-system get service home-lab-dns -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
dig "@${DNS_SERVER}" grafana.local.dev +short
```
