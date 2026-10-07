#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
lab_root="$(cd "${script_dir}/.." && pwd)"
crd_manifest="${lab_root}/core/ingress/traefik-provider-crds.yaml"

for tool in kubectl grep; do
  command -v "${tool}" >/dev/null 2>&1 || {
    printf 'Required command missing: %s\n' "${tool}" >&2
    exit 1
  }
done

[[ "$(kubectl config current-context)" == docker-desktop ]] || {
  printf 'Select the docker-desktop Kubernetes context first.\n' >&2
  exit 1
}

ready_nodes="$(kubectl get nodes -o jsonpath='{range .items[*]}{.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}')"
grep -qx 'True' <<<"${ready_nodes}" || {
  printf 'No Ready Kubernetes node; refusing to apply Traefik CRDs.\n' >&2
  exit 1
}

kubectl apply -f "${crd_manifest}"

for crd in \
  ingressroutes.traefik.io \
  ingressroutetcps.traefik.io \
  ingressrouteudps.traefik.io \
  middlewares.traefik.io \
  middlewaretcps.traefik.io \
  serverstransports.traefik.io \
  serverstransporttcps.traefik.io \
  tlsoptions.traefik.io \
  tlsstores.traefik.io \
  traefikservices.traefik.io; do
  kubectl wait --for=condition=Established "crd/${crd}" --timeout=120s
done

printf 'Traefik v3.6.12 provider CRDs are established on Docker Desktop.\n'
