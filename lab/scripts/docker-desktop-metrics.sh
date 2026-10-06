#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
action="${1:-}"
case "${action}" in
  validate|apply) ;;
  *) printf 'Usage: %s validate|apply\n' "${0}" >&2; exit 2 ;;
esac
[[ "$(kubectl config current-context)" == docker-desktop ]] || {
  printf 'Select the docker-desktop Kubernetes context first.\n' >&2
  exit 1
}

helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/ --force-update >/dev/null
args=(
  upgrade --install metrics-server metrics-server/metrics-server
  -n kube-system --version 3.14.0
  -f "${script_dir}/../apps/observability/metrics-server/values.docker-desktop.yaml"
)
if [[ "${action}" == validate ]]; then
  helm "${args[@]}" --dry-run=server --hide-secret >/dev/null
  printf 'Docker Desktop Metrics Server chart validated.\n'
else
  helm "${args[@]}" --wait --timeout 5m
fi
