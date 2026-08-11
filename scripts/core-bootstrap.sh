#!/usr/bin/env bash

set -euo pipefail

readonly EXPECTED_CONTEXT="${KUBE_CONTEXT:-rancher-desktop}"
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${REPO_ROOT}"

current_context="$(kubectl config current-context)"
[[ "${current_context}" == "${EXPECTED_CONTEXT}" ]] || {
  echo "Current Kubernetes context is ${current_context}; expected ${EXPECTED_CONTEXT}." >&2
  exit 1
}

node_status="$(kubectl get nodes -o jsonpath='{range .items[*]}{.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}')"
grep -qx 'True' <<<"${node_status}" || {
  echo "Kubernetes node is not Ready; refusing core bootstrap." >&2
  exit 1
}

kubectl apply -f core/namespaces/namespaces.yaml
kubectl apply -f core/nodes/inotify-capacity.yaml
kubectl rollout status daemonset/node-inotify-capacity \
  --namespace platform-system --timeout=180s
kubectl apply -f core/dns/coredns-custom.yaml
kubectl apply -f core/dns/service.yaml

./scripts/platform-helm.sh apply cert-manager

kubectl apply -f core/tls/clusterissuer.yaml
kubectl apply -f core/tls/certificates.yaml

kubectl wait --for=condition=Ready certificate/local-dev-ca \
  --namespace platform-system --timeout=180s

for namespace in kube-system platform-system platform-storage platform-secrets observability observability-tests development-tools ai-tools; do
  kubectl wait --for=condition=Ready "certificate/local-dev-tls" \
    --namespace "${namespace}" --timeout=180s
done

kubectl apply -f core/ingress/tlsstore.yaml
kubectl apply -f core/ingress/traefik-helmchartconfig.yaml
kubectl rollout status deployment/traefik --namespace kube-system --timeout=180s

echo "Core Rancher node capacity, CoreDNS, cert-manager, Traefik, and local HTTPS configuration is ready."
