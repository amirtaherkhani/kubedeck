#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
lab_root="$(cd "${script_dir}/.." && pwd)"
chart="${lab_root}/apps/platform/storage"
action="${1:-}"

case "${action}" in
  validate|apply) ;;
  *) printf 'Usage: %s validate|apply\n' "${0}" >&2; exit 2 ;;
esac

for tool in helm kubectl jq; do
  command -v "${tool}" >/dev/null 2>&1 || {
    printf 'Required command missing: %s\n' "${tool}" >&2
    exit 1
  }
done
[[ "$(kubectl config current-context)" == docker-desktop ]] || {
  printf 'Select the docker-desktop Kubernetes context first.\n' >&2
  exit 1
}

source "${script_dir}/lib/docker-desktop-scope.sh"
scope="$(resolve_docker_desktop_scope)"
IFS=$'\t' read -r project_slug environment <<<"${scope}"

values=(
  -f "${chart}/values.yaml"
  -f "${chart}/values.docker-desktop.yaml"
  -f "${chart}/values.docker-desktop-core.yaml"
  --set-string "infisical.projectSlug=${project_slug}"
  --set-string "infisical.envSlug=${environment}"
)

if [[ "${action}" == validate ]]; then
  helm lint "${chart}" "${values[@]}"
  helm template platform-storage "${chart}" -n platform-storage "${values[@]}" >/dev/null
  printf 'Docker Desktop core chart rendered for project %s / %s.\n' "${project_slug}" "${environment}"
  exit 0
fi

helm upgrade --install platform-storage "${chart}" \
  -n platform-storage --create-namespace "${values[@]}" \
  --wait --timeout 15m --history-max 10
