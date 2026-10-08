# Local image registry

The registry stores custom images for this Mac's Docker Desktop Kubernetes
node. Its Deployment and PVC live in `platform-system`; the node reads images
through its own `localhost:5001`. The Mac uses a localhost-only port-forward
on the same fixed port for image pushes. There is no external domain or TLS
endpoint for this development registry.

Install the persistent Mac listener alongside the Technitium admin listener:

```sh
bash lab/core/host/macos/install-local-service-forwarders.sh
curl -fsS http://127.0.0.1:5001/v2/
```

The LaunchAgent reconnects after the registry pod or Kubernetes restarts. The
Docker Desktop image script reuses this listener when it is available and
starts a temporary forward only when it is not.
