# Core

Base home-lab configuration shared by all applications.

- `helm/` — release inventory, repositories, and Rancher ownership boundary.
- `dns/` — `*.local.dev` naming and CoreDNS/macOS resolution rules.
- `tls/` — shared local HTTPS certificate contract.
- `host/` — macOS host exporters and host integration.

Rancher Desktop owns the `kube-system` Traefik, CoreDNS, local-path-provisioner,
metrics-server, Flannel, and ServiceLB components. They are observed and
configured here only where the ownership boundary permits it.
