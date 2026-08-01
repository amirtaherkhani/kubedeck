<p align="center">
  <img src="public/brand/kubedeck-mark.svg" width="112" height="112" alt="KubeDeck logo">
</p>

<h1 align="center">KubeDeck</h1>

<p align="center">
  <strong>A Kubernetes-native homepage for your homelab.</strong>
</p>

<p align="center">
  One private launchpad for clusters, nodes, workloads, web interfaces,
  services, DNS names, ports, health, deployment history, and notifications.
</p>

![KubeDeck — your Kubernetes ecosystem, one click away](public/kubedeck-banner.png)

## What is KubeDeck?

KubeDeck is a focused home dashboard for Kubernetes homelabs. It takes the
one-click service-launcher idea popularized by
[Homepage](https://gethomepage.dev/) and applies it specifically to the
Kubernetes ecosystem.

Instead of maintaining a general bookmark page, KubeDeck is designed around
the objects and operational questions that matter inside a cluster:

- Which nodes and workloads are healthy?
- Which services expose a web interface?
- What is the internal cluster DNS name?
- Which external hostname and ports should I use?
- When was the workload last deployed?
- Is the service ready, degraded, or unavailable?
- Which services belong to storage, observability, automation, AI, or another
  platform category?

KubeDeck is distribution-independent. It is intended for K3s, RKE2, Rancher,
Talos, MicroK8s, kind, managed Kubernetes, and other conformant clusters.

## Project status

KubeDeck `v0.1.1` is an early, Kubernetes-native foundation.

The repository includes the dashboard, responsive liquid-glass interface,
authentication, Kubernetes catalog, multi-node review, notifications, settings,
Go cluster agent, container runtimes, and separate Helm charts for the app and
agent. The agent provides live Kubernetes API discovery, resource metrics,
reconnecting SSE updates, and opt-in CoreDNS service-alias management. The
dashboard now server-renders the current authenticated agent snapshot and then
keeps the overview, nodes, catalog, DNS, settings, charts, and notifications
current through `/api/cluster/events`. An explicitly labeled illustrative
fallback is used only while the agent is unavailable.

## Highlights

- Kubernetes-only service and application homepage
- One-click access to web UIs and external routes
- Internal service DNS, ClusterIP, protocol, port, namespace, workload, and pod
  readiness details
- Intelligent catalog categories:
  - Web Applications
  - Databases & Storage
  - Observability & Metrics
  - Automation & Workflows
  - Deployments & Testing
  - AI & MCP Services
  - Messaging & Events
  - Developer Tools
  - Platform & Security
- Multi-node cluster review with live CPU, memory, ephemeral-storage request,
  and pod-allocation charts collected during the current browser session
- Graphical readiness, uptime, endpoint, and last-deployment indicators
- Service and node status notification center
- Search, status filters, resource-kind filters, and sorting
- Current-user, Kubernetes, application, and user-management settings
- First-admin bootstrap from a Kubernetes Secret
- PBKDF2-SHA-256 password hashing and signed, secure, HTTP-only sessions
- Responsive KubeDeck banner, live topology animation, reduced-motion support,
  and multi-size brand assets
- OCI container build and Helm deployment with persistent local D1 storage
- Go `client-go` cluster agent with informer caches and SSE streaming
- Protected CoreDNS alias editor with validation, dry-run preview, and
  optimistic concurrency

## Interface

The dashboard is organized as an operational homepage rather than a generic
intranet portal:

1. **Cluster overview** summarizes nodes, pods, namespaces, workloads, routes,
   DNS, and current attention items.
2. **Node review** compares every node and charts four resource properties over
   time.
3. **Catalog** groups Kubernetes web applications and internal services by
   platform function.
4. **Service details** expose cluster DNS, external domain, ClusterIP, ports,
   workload source, endpoints, uptime, and last deployment.
5. **Notifications** separate service and node status changes.
6. **Settings** prepare the application for users, roles, cluster connections,
   discovery rules, and application preferences.

## Architecture

```mermaid
flowchart LR
    Browser["Browser"] --> Ingress["Kubernetes Ingress / TLS"]
    Ingress --> App["KubeDeck Worker runtime"]
    App --> Auth["Admin authentication"]
    Auth --> D1["Persistent local D1 database"]
    App --> Proxy["Authenticated cluster proxy"]
    App --> UI["Dashboard, graphs, notifications, settings"]
    Proxy --> Agent["KubeDeck Go agent"]
    Agent --> Kubernetes["Kubernetes API and Metrics API"]
    Agent --> SSE["Snapshot and SSE stream"]
    Agent -. "opt-in scoped write" .-> CoreDNS["CoreDNS custom ConfigMap"]
```

The current Helm release runs one application replica because its embedded D1
database uses a single-writer persistent volume. A future external database
backend can enable horizontally scaled application replicas.

## Technology

- React 19 and Next.js-compatible App Router
- vinext and Vite
- Cloudflare Workers local runtime
- D1, SQLite, Drizzle ORM
- shadcn/ui and Base UI primitives
- Motion and Recharts
- TypeScript and Tailwind CSS
- Docker/OCI and Helm
- Go 1.26, `client-go`, Kubernetes Metrics API, and CoreDNS Caddyfile parser

## Repository layout

```text
kubedeck-agent/           Go Kubernetes discovery and CoreDNS agent
app/                      KubeDeck App Router pages and API proxies
charts/kubedeck/          Dashboard Helm chart
charts/kubedeck-agent/    Agent, RBAC, SSE, and CoreDNS Helm chart
components/               Dashboard and shadcn UI components
tests/                    Rendered app and authenticated proxy tests
```

## Local development

Requirements:

- Node.js `>=22.13.0`
- npm

```bash
git clone https://github.com/amirtaherkhani/kubedeck.git
cd kubedeck
npm ci
npm run dev
```

Open `http://localhost:3000`.

Useful commands:

```bash
npm run dev
npm run build
npm run lint
npm test
npm run db:generate
```

Agent checks:

```bash
cd kubedeck-agent
go test -race ./...
go vet ./...
```

The agent package choices, environment variables, API contract, and CoreDNS
safety model are documented in [`kubedeck-agent/README.md`](kubedeck-agent/README.md).

## Administrator setup

KubeDeck supports two first-run paths.

### Backend bootstrap

Set all four values together:

```dotenv
KUBEDECK_ADMIN_FIRST_NAME=Homelab
KUBEDECK_ADMIN_LAST_NAME=Admin
KUBEDECK_ADMIN_EMAIL=admin@example.com
KUBEDECK_ADMIN_PASSWORD=use-a-unique-password-with-12-or-more-characters
```

The first request validates the configuration and stores only a salted
PBKDF2-SHA-256 password hash in D1. Environment values never overwrite an
existing administrator.

### Browser setup

When all four variables are absent, KubeDeck displays **Create the admin
account**. The hosted private setup route additionally expects the authenticated
Sites identity header. For a Kubernetes deployment, backend bootstrap through a
Secret is the recommended path.

Sessions use a signed `__Host-` cookie with `Secure`, `HttpOnly`, and
`SameSite=Strict`, and expire after 12 hours.

## Build the container

The Dockerfile supports Linux `amd64` and `arm64`.

```bash
docker build -t kubedeck:local .
docker run --rm \
  --name kubedeck \
  --publish 3000:3000 \
  --volume kubedeck-data:/data \
  --env-file .env \
  kubedeck:local
```

For Kubernetes, push an immutable image tag to a registry accessible from the
cluster:

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag ghcr.io/your-user/kubedeck:0.1.1 \
  --push .
```

## Verified release deployment

The repository release workflow validates the dashboard, Go agent, and both
Helm charts; builds immutable images from the verified remote commit; deploys
the agent before the dashboard; and runs an authenticated dashboard-to-agent
snapshot smoke test before reporting success:

```bash
bash .agents/skills/kubedeck-build-deploy/scripts/deploy.sh --branch main
```

The runtime forwards `KUBEDECK_AGENT_URL` and `KUBEDECK_AGENT_TOKEN` into the
local Wrangler Worker as server-only bindings. The bearer token is never sent
to browser code.

## Install with Helm

The app and agent charts are located at `charts/kubedeck` and
`charts/kubedeck-agent`.

Create the namespace and administrator Secret:

```bash
kubectl create namespace kubedeck
kubectl -n kubedeck create secret generic kubedeck-admin \
  --from-env-file=.env.admin
kubectl -n kubedeck create secret generic kubedeck-agent-auth \
  --from-literal=token='replace-with-a-long-random-token'
```

Example `.env.admin`:

```dotenv
KUBEDECK_ADMIN_FIRST_NAME=Homelab
KUBEDECK_ADMIN_LAST_NAME=Admin
KUBEDECK_ADMIN_EMAIL=admin@example.com
KUBEDECK_ADMIN_PASSWORD=replace-with-a-long-random-password
```

Install without an Ingress:

```bash
helm upgrade --install kubedeck-agent ./charts/kubedeck-agent \
  --namespace kubedeck \
  --set image.repository=ghcr.io/your-user/kubedeck-agent \
  --set image.tag=0.1.1 \
  --set auth.existingSecret=kubedeck-agent-auth \
  --wait
```

Then install the app:

```bash
helm upgrade --install kubedeck ./charts/kubedeck \
  --namespace kubedeck \
  --set image.repository=ghcr.io/your-user/kubedeck \
  --set image.tag=0.1.1 \
  --set agent.existingSecret=kubedeck-agent-auth \
  --set ingress.enabled=false \
  --wait
```

Then forward the service:

```bash
kubectl -n kubedeck port-forward service/kubedeck 3000:80
```

### Traefik and TLS

Create a values file for your domain:

```yaml
image:
  repository: ghcr.io/your-user/kubedeck
  tag: 0.1.1

ingress:
  enabled: true
  className: traefik
  forceHttps: true
  hosts:
    - host: kubedeck.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: kubedeck-tls
      hosts:
        - kubedeck.example.com
```

Install it:

```bash
helm upgrade --install kubedeck ./charts/kubedeck \
  --namespace kubedeck \
  --values kubedeck-values.yaml \
  --wait
```

`ingress.forceHttps` creates a Traefik `Middleware` and a dedicated HTTP
redirect Ingress. Leave it disabled when using another ingress controller and
configure that controller's HTTPS redirect mechanism instead.

### CoreDNS service aliases

CoreDNS writes are disabled by default. On K3s, enable the dedicated custom
ConfigMap integration when installing the agent:

```bash
helm upgrade --install kubedeck-agent ./charts/kubedeck-agent \
  --namespace kubedeck \
  --set image.repository=ghcr.io/your-user/kubedeck-agent \
  --set image.tag=0.1.1 \
  --set auth.existingSecret=kubedeck-agent-auth \
  --set dnsManagement.enabled=true \
  --wait
```

The optional Role can update only `kube-system/coredns-custom`; it cannot write
other ConfigMaps, workloads, Secrets, or node proxy APIs. KubeDeck manages only
the `kubedeck.override` key and refuses to overwrite unrecognized content.

## Helm behavior

- Requires an immutable `image.tag`
- Uses a `ClusterIP` Service
- Creates a persistent `ReadWriteOnce` volume for local D1 data
- Uses `Recreate` deployment strategy to protect the single-writer database
- Runs as a non-root user with dropped Linux capabilities
- Disables automatic ServiceAccount token mounting
- Includes startup, readiness, and liveness probes
- Loads administrator values from an existing Kubernetes Secret
- Connects to the agent through an internal URL and server-side bearer token
- Supports custom storage classes, resources, node selectors, affinity,
  tolerations, labels, annotations, and extra environment values

## Security model

- KubeDeck does not store plaintext passwords in D1.
- The example Helm chart never contains administrator credentials.
- Kubernetes credentials should be supplied through a Secret.
- The current ServiceAccount has no Kubernetes API token.
- The separate agent ServiceAccount has read-only discovery permissions.
- Optional CoreDNS management uses a namespaced Role restricted to one custom
  ConfigMap and requires authenticated agent requests.
- HTTPS should be enabled before using authentication outside local development.
- Do not grant write access to workloads, Secrets, or cluster administration
  APIs solely for dashboard discovery.

## Roadmap

- [x] Read-only Service, Ingress, EndpointSlice, Deployment, StatefulSet, Pod,
      Namespace, and Node discovery agent
- [x] Kubernetes watch-based snapshot and SSE updates
- [x] Opt-in CoreDNS Service alias management
- [x] Bind dashboard, catalog, DNS, settings, notifications, and session charts
      to live agent snapshots
- [ ] Prometheus and metrics-server resource histories
- [ ] Multi-cluster connection management
- [ ] Configurable classification rules and annotations
- [ ] Viewer, editor, and administrator roles
- [ ] User management and external identity providers
- [ ] Pluggable database backend for horizontal scaling
- [ ] Helm repository and signed multi-architecture container releases
- [ ] Import/export and declarative catalog configuration
- [ ] Notification delivery integrations

## Inspiration

KubeDeck is inspired by the usability of
[Homepage](https://gethomepage.dev/), but it intentionally narrows the product
to Kubernetes homelabs: cluster-aware navigation, service discovery, DNS and
port details, node health, workload status, and operational context.

KubeDeck is an independent project and is not affiliated with Homepage.
