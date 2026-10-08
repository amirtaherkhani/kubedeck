#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
RELEASES_FILE="${REPO_ROOT}/core/helm/releases.conf"
REPOSITORIES_FILE="${REPO_ROOT}/core/helm/repositories.conf"
EXPECTED_CONTEXT="${KUBE_CONTEXT:-docker-desktop}"

cd "${REPO_ROOT}"

usage() {
  cat <<'EOF'
Usage: scripts/platform-helm.sh <command> [release]

Commands:
  inventory          Show the pinned repository release inventory.
  repos              Add and update required Helm repositories.
  status             Show live Helm releases in all namespaces.
  validate [release] Validate one release or all releases against the cluster.
  apply [release]    Install or upgrade one release or all releases.
  values <release>   Show the live user-supplied values for a release.
  remove <release>   Uninstall one release with explicit confirmation.

The default release for validate and apply is "all".
To uninstall, set CONFIRM_UNINSTALL to the exact release name.
Set HELM_AUTO_ROLLBACK=false for a custom-image replacement whose previous
image has been removed. Automatic rollback defaults to true.
EOF
}

die() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

check_context() {
  local current_context
  current_context="$(kubectl config current-context)"
  [[ "${current_context}" == "${EXPECTED_CONTEXT}" ]] || die \
    "current Kubernetes context is ${current_context}; expected ${EXPECTED_CONTEXT}"
}

sync_repositories() {
  local name url

  while IFS='|' read -r name url; do
    if [[ -z "${name}" || "${name}" == \#* ]]; then
      continue
    fi
    helm repo add "${name}" "${url}" --force-update >/dev/null
  done < "${REPOSITORIES_FILE}"

  helm repo update
}

sync_repositories_for_target() {
  local target="$1"
  local release namespace chart version values timeout repository repository_url

  if [[ "${target}" == "all" ]]; then
    sync_repositories
    return
  fi

  while IFS='|' read -r release namespace chart version values timeout; do
    if [[ -z "${release}" || "${release}" == \#* || "${release}" != "${target}" ]]; then
      continue
    fi
    if [[ "${chart}" == ./* ]]; then
      printf 'Skipping repository sync for local chart %s.\n' "${chart}"
    else
      repository="${chart%%/*}"
      repository_url="$(awk -F'|' -v name="${repository}" '$1 == name {print $2; exit}' "${REPOSITORIES_FILE}")"
      [[ -n "${repository_url}" ]] || die "Helm repository is not registered: ${repository}"
      helm repo add "${repository}" "${repository_url}" --force-update >/dev/null
      helm repo update "${repository}"
    fi
    return
  done < "${RELEASES_FILE}"

  die "release is not managed by this repository: ${target}"
}

run_selected() {
  local target="$1"
  local callback="$2"
  local release namespace chart version values timeout
  local found=0

  while IFS='|' read -r release namespace chart version values timeout; do
    if [[ -z "${release}" || "${release}" == \#* ]]; then
      continue
    fi
    if [[ "${target}" != "all" && "${target}" != "${release}" ]]; then
      continue
    fi

    found=1
    "${callback}" "${release}" "${namespace}" "${chart}" "${version}" "${values}" "${timeout}"
  done < "${RELEASES_FILE}"

  [[ "${found}" -eq 1 ]] || die "release is not managed by this repository: ${target}"
}

helm_base_args() {
  local release="$1"
  local namespace="$2"
  local chart="$3"
  local version="$4"
  local values="$5"
  local scope project_slug environment

  HELM_ARGS=(
    upgrade
    --install
    "${release}"
    "${chart}"
    --namespace "${namespace}"
    --create-namespace
    --values "${values}"
  )

  if [[ "${release}" == "platform-storage" || "${release}" == "grafana" ]]; then
    require_command jq
    source "${SCRIPT_DIR}/lib/docker-desktop-scope.sh"
    scope="$(resolve_docker_desktop_scope)"
    IFS=$'\t' read -r project_slug environment <<<"${scope}"
    [[ -n "${project_slug}" && -n "${environment}" ]] || die "Infisical project scope is incomplete"
    if [[ "${release}" == "platform-storage" ]]; then
      HELM_ARGS+=(--reuse-values --values apps/platform/storage/values.docker-desktop.yaml --values apps/platform/storage/values.docker-desktop-core.yaml)
    fi
    HELM_ARGS+=(--set-string "infisical.projectSlug=${project_slug}" --set-string "infisical.envSlug=${environment}")
  fi

  if [[ "${version}" != "-" ]]; then
    HELM_ARGS+=(--version "${version}")
  fi

}

validate_release() {
  local release="$1"
  local namespace="$2"
  local chart="$3"
  local version="$4"
  local values="$5"
  local timeout="$6"

  printf 'Validating %-18s namespace=%-12s chart=%s\n' "${release}" "${namespace}" "${chart}"
  if [[ "${release}" == "infisical" ]]; then
    kubectl apply --dry-run=server -f apps/platform/infisical/manifests/postgresql.yaml >/dev/null
  fi
  helm_base_args "${release}" "${namespace}" "${chart}" "${version}" "${values}"
  helm "${HELM_ARGS[@]}" \
    --timeout "${timeout}" \
    --dry-run=server \
    --hide-secret >/dev/null
}

apply_release() {
  local release="$1"
  local namespace="$2"
  local chart="$3"
  local version="$4"
  local values="$5"
  local timeout="$6"
  local auto_rollback="${HELM_AUTO_ROLLBACK:-true}"
  local rollback_flag=""

  case "${auto_rollback}" in
    true)
      rollback_flag="--atomic"
      if [[ "$(helm upgrade --help)" == *"--rollback-on-failure"* ]]; then
        rollback_flag="--rollback-on-failure"
      fi
      ;;
    false)
      ;;
    *)
      die "HELM_AUTO_ROLLBACK must be true or false"
      ;;
  esac

  if [[ "${release}" == "infisical" ]]; then
    "${REPO_ROOT}/scripts/infisical-bootstrap.sh"
  fi

  printf 'Applying   %-18s namespace=%-12s chart=%s\n' "${release}" "${namespace}" "${chart}"
  helm_base_args "${release}" "${namespace}" "${chart}" "${version}" "${values}"
  HELM_ARGS+=(
    --timeout "${timeout}"
    --wait
    --wait-for-jobs
  )
  if [[ -n "${rollback_flag}" ]]; then
    HELM_ARGS+=("${rollback_flag}")
  fi
  HELM_ARGS+=(
    --cleanup-on-fail
    --history-max 10
  )
  helm "${HELM_ARGS[@]}"
}

show_values() {
  local release="$1"
  local namespace="$2"

  helm get values "${release}" --namespace "${namespace}" --output yaml
}

remove_release() {
  local release="$1"
  local namespace="$2"

  [[ "${CONFIRM_UNINSTALL:-}" == "${release}" ]] || die \
    "set CONFIRM_UNINSTALL=${release} to confirm this uninstall"
  helm uninstall "${release}" --namespace "${namespace}" --wait
}

validate_manifests() {
  local target="$1"

  if [[ "${target}" == "all" ]]; then
    kubectl apply --dry-run=server -f core/dns/coredns-custom.yaml >/dev/null
    kubectl apply --dry-run=server -f core/dns/service.yaml >/dev/null
    kubectl apply --dry-run=server -f core/ingress/tlsstore.yaml >/dev/null
  fi

  if [[ "${target}" == "all" || "${target}" == "infisical" ]]; then
    kubectl apply --dry-run=server -f apps/platform/infisical/manifests/https-redirect.yaml >/dev/null
  fi
}

apply_manifests() {
  local target="$1"

  if [[ "${target}" == "all" ]]; then
    kubectl apply -f core/dns/coredns-custom.yaml
    kubectl apply -f core/dns/service.yaml
    kubectl apply -f core/ingress/tlsstore.yaml
  fi

  if [[ "${target}" == "all" || "${target}" == "infisical" ]]; then
    kubectl apply -f apps/platform/infisical/manifests/https-redirect.yaml
  fi
}

print_inventory() {
  local release namespace chart version values timeout

  printf '%-18s %-12s %-47s %-10s %s\n' RELEASE NAMESPACE CHART VERSION VALUES
  while IFS='|' read -r release namespace chart version values timeout; do
    if [[ -z "${release}" || "${release}" == \#* ]]; then
      continue
    fi
    printf '%-18s %-12s %-47s %-10s %s\n' \
      "${release}" "${namespace}" "${chart}" "${version}" "${values}"
  done < "${RELEASES_FILE}"
}

require_command helm
require_command kubectl

command_name="${1:-help}"
target_release="${2:-all}"

case "${command_name}" in
  inventory)
    print_inventory
    ;;
  repos)
    sync_repositories
    ;;
  status)
    printf 'Kubernetes context: %s\n\n' "$(kubectl config current-context)"
    print_inventory
    printf '\nLive Helm releases:\n'
    helm list --all-namespaces
    ;;
  validate)
    check_context
    sync_repositories_for_target "${target_release}"
    run_selected "${target_release}" validate_release
    validate_manifests "${target_release}"
    printf 'Validation completed for %s.\n' "${target_release}"
    ;;
  apply)
    check_context
    sync_repositories_for_target "${target_release}"
    run_selected "${target_release}" apply_release
    apply_manifests "${target_release}"
    printf 'Apply completed for %s.\n' "${target_release}"
    ;;
  values)
    [[ "${target_release}" != "all" ]] || die "values requires a release name"
    run_selected "${target_release}" show_values
    ;;
  remove)
    [[ "${target_release}" != "all" ]] || die "remove requires a release name"
    check_context
    run_selected "${target_release}" remove_release
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    usage
    die "unknown command: ${command_name}"
    ;;
esac
