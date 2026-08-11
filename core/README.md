# Core

Base home-lab configuration shared by all applications.

- `helm/` — release inventory, repositories, and Rancher ownership boundary.
- `nodes/` — Linux node configuration required by cluster workloads.
- `dns/` — `*.local.dev` naming and CoreDNS/macOS resolution rules.
- `tls/` — shared local HTTPS certificate contract.
- `host/` — macOS host exporters and host integration.

Rancher Desktop owns the `kube-system` Traefik, CoreDNS, local-path-provisioner,
metrics-server, Flannel, and ServiceLB components. They are observed and
configured here only where the ownership boundary permits it.

## Node inotify capacity

`nodes/inotify-capacity.yaml` raises `fs.inotify.max_user_instances` to at least
`1024` on every Linux node. Rancher Desktop's Kubernetes and container runtime
use inotify while following container logs; its lower default can otherwise
produce `failed to create fsnotify watcher: too many open files` even when the
application processes have few open files.

Only the DaemonSet init container is privileged and mounts the host `/proc/sys`.
The long-running container is unprivileged, drops all capabilities, uses a
read-only root filesystem, and does not receive a Kubernetes API token. The
init container never lowers a node that is already configured above `1024`.

Verify the live setting after `make core-apply`:

```bash
kubectl rollout status daemonset/node-inotify-capacity -n platform-system
rdctl shell cat /proc/sys/fs/inotify/max_user_instances
```
