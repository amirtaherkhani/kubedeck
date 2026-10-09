# Observability runtime

This is the checked-in local deployment profile. Its `local.dev` domains, namespace names, and Infisical references describe this installation; they are not defaults required by either KubeDeck agent. Copy or override the Helm values and manifests for another cluster instead of changing the agent code.

The agent code and chart are portable; this profile is deliberately site-specific. Before using it in another cluster, choose that cluster's Kubernetes context, namespaces, DNS zone and trusted certificate issuer, ingress class, storage classes, Secret names and Infisical scope. Keep those selections in a separate values/manifest overlay and render the result before applying. The observability components use Kubernetes search-domain Service names so their internal connections do not assume `cluster.local`; Grafana's Infisical endpoint, credential Secret and path are configurable in `values.homelab.yaml`. The raw TLS and PostgreSQL manifests still describe this site and must be replaced or patched for a different site. No host filesystem paths are required by the agents or this profile.

This directory keeps the Grafana observability stack, k6 Operator, and the platform configuration required for local HTTPS and Infisical-backed Grafana credentials. It does not manage the KubeDeck dashboard, shared application databases, or unrelated home-lab services.

The runtime on Docker Desktop Kubernetes consists of Grafana and its image renderer, Prometheus Stack, Loki, Tempo, Alloy, and k6 Operator. cert-manager and Traefik provide `https://grafana.local.dev`; Infisical and its operator supply `observability/grafana-admin`. The existing Grafana, Loki, Tempo, and Infisical PVCs are reused. Loki's chart values retain its PVC when the StatefulSet scales down or is removed.

## Prerequisites

- The `docker-desktop` Kubernetes context is selected and its node is Ready.
- Technitium/local DNS resolves `grafana.local.dev` to this Mac, and the local development CA is trusted on this Mac.
- `platform-secrets/infisical-secrets`, `platform-secrets/infisical-postgresql`, and `platform-secrets/infisical-universal-auth` exist. Provision credentials through the established secret workflow; do not place them in Helm values or Git.
- Set `INFISICAL_PROJECT_SLUG` and `INFISICAL_ENV_SLUG` to the active Infisical project and environment. Override the chart's sample scope for every deployment.

## Deployment order

The pinned versions below match the verified local stack. Run these commands from `lab/`.

```bash
helm repo add cert-manager https://charts.jetstack.io
helm repo add traefik https://traefik.github.io/charts
helm repo add infisical-helm-charts https://dl.cloudsmith.io/public/infisical/helm-charts/helm/charts/
helm repo add grafana https://grafana.github.io/helm-charts
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
kubectl create namespace observability --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace observability-tests --dry-run=client -o yaml | kubectl apply -f -
helm upgrade --install cert-manager cert-manager/cert-manager -n platform-system --create-namespace --version v1.21.0 -f apps/platform/cert-manager/values.yaml --wait
kubectl apply -f core/tls/clusterissuer.yaml -f core/tls/certificates.yaml
helm upgrade --install traefik traefik/traefik -n platform-system --version 39.0.7 -f core/ingress/traefik-docker-desktop-values.yaml --wait
kubectl apply -f core/ingress/tlsstore.yaml -f core/exposure/docker-desktop/traefik-lan.yaml
kubectl apply -f apps/platform/infisical/manifests/postgresql.yaml
helm upgrade --install infisical infisical-helm-charts/infisical -n platform-secrets --version 0.4.2 -f apps/platform/infisical/values.yaml --wait
kubectl apply -f apps/platform/infisical/manifests/https-redirect.yaml
helm upgrade --install infisical-operator infisical-helm-charts/secrets-operator -n platform-secrets --version 0.11.11 -f apps/platform/infisical-operator/values.yaml --wait
helm upgrade --install loki grafana/loki -n observability --version 7.0.0 -f apps/observability/loki/values.yaml --wait
helm upgrade --install monitoring prometheus-community/kube-prometheus-stack -n observability --version 87.15.1 -f apps/observability/kube-prometheus-stack/values.yaml --wait
helm upgrade --install tempo grafana/tempo -n observability --version 1.24.4 -f apps/observability/tempo/values.yaml --wait
helm upgrade --install alloy grafana/alloy -n observability --version 1.10.1 -f apps/observability/alloy/values.yaml --wait
helm upgrade --install grafana ./apps/observability/grafana -n observability -f apps/observability/grafana/values.homelab.yaml --set-string infisical.projectSlug="${INFISICAL_PROJECT_SLUG}" --set-string infisical.envSlug="${INFISICAL_ENV_SLUG}" --wait
helm upgrade --install k6-operator grafana/k6-operator -n observability-tests --version 4.5.0 -f apps/observability/k6/values.yaml --wait
kubectl apply -k apps/observability/k6/dashboard
```

After installation, confirm `observability/grafana-admin` reports `ReadyToSyncSecrets=True` and the Grafana Deployment is Ready. The image renderer is part of the Grafana chart. For changes to a live release, render and server-side dry-run before applying.

The k6 Operator uses Prometheus's enabled remote-write receiver. The dashboard ConfigMaps are discovered by Grafana's sidecar. The one-iteration smoke TestRun can be started with `kubectl apply -f apps/observability/k6/tests/smoke-test.yaml`.

## Verification

```bash
kubectl -n observability get pods,pvc
kubectl -n observability get infisicalsecret grafana-admin
kubectl -n observability-tests get deploy,servicemonitor,testruns.k6.io
curl --fail https://grafana.local.dev/api/health
kubectl get --raw '/api/v1/namespaces/observability/services/http:loki:3100/proxy/ready'
kubectl get --raw '/api/v1/namespaces/observability/services/http:tempo:3200/proxy/ready'
kubectl get --raw '/api/v1/namespaces/observability/services/http:monitoring-kube-prometheus-prometheus:9090/proxy/-/ready'
```
