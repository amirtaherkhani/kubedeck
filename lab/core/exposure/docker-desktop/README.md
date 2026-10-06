# Docker Desktop LAN exposure

This additive overlay publishes the existing Traefik and Technitium pods through Docker Desktop `LoadBalancer` Services. It does not replace the Helm-owned Traefik Service or the existing Technitium ClusterIP Service. No host IP, network interface, or router address is committed.

Apply only to the verified Docker Desktop cluster:

```sh
kubectl --context docker-desktop apply --server-side --dry-run=server -k lab/core/exposure/docker-desktop
kubectl --context docker-desktop apply -k lab/core/exposure/docker-desktop
kubectl --context docker-desktop -n platform-system get service traefik-lan
kubectl --context docker-desktop -n technitium get service technitium-lan
```

Check TCP 80/443 and both UDP/TCP 53 from the Mac and another LAN client before changing any resolver or DHCP setting. Keep the Technitium admin UI on the existing internal Service. Docker Desktop may expose only localhost or fail to allocate one protocol; record the actual listener and response rather than inferring reachability from `EXTERNAL-IP`.

This overlay only publishes traffic. Technitium still needs an authoritative `local.dev` zone and an automatic record update for the current Mac ingress IP. LAN clients also need a stable DNS-server discovery path through router DHCP/DNS or a reserved resolver endpoint. Neither is established by this overlay.

The V1 [macOS Host Agent](../../../../host-agent/README.md) reconciles the wildcard A record. Once Technitium answers on localhost port 53, run `lab/core/host/macos/configure-docker-desktop-resolver.sh` to replace the old fixed-IP macOS scoped resolver with `127.0.0.1`. It backs up the prior resolver file and asks for macOS administrator authentication only when a change is needed. Router DHCP/DNS remains a separate LAN-client gate.
