#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
RELEASES_FILE="${REPO_ROOT}/core/helm/releases.conf"
REPOSITORIES_FILE="${REPO_ROOT}/core/helm/repositories.conf"
RANCHER_MANAGED_FILE="${REPO_ROOT}/core/helm/rancher-managed.conf"
EXPECTED_CONTEXT="${KUBE_CONTEXT:-rancher-desktop}"

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

assert_rancher_boundary() {
  local rancher_release rancher_namespace owner
  local managed_release managed_namespace chart version values timeout

  while IFS='|' read -r rancher_release rancher_namespace owner; do
    if [[ -z "${rancher_release}" || "${rancher_release}" == \#* ]]; then
      continue
    fi

    while IFS='|' read -r managed_release managed_namespace chart version values timeout; do
      if [[ -z "${managed_release}" || "${managed_release}" == \#* ]]; then
        continue
      fi
      if [[ "${managed_release}" == "${rancher_release}" && "${managed_namespace}" == "${rancher_namespace}" ]]; then
        die "${managed_release} in ${managed_namespace} is owned by ${owner} and cannot be repository-managed"
      fi
    done < "${RELEASES_FILE}"
  done < "${RANCHER_MANAGED_FILE}"
}

reject_rancher_target() {
  local target="$1"
  local rancher_release rancher_namespace owner

  [[ "${target}" != "all" ]] || return 0

  while IFS='|' read -r rancher_release rancher_namespace owner; do
    if [[ -z "${rancher_release}" || "${rancher_release}" == \#* ]]; then
      continue
    fi
    if [[ "${target}" == "${rancher_release}" ]]; then
      die "${target} is owned by ${owner}; manage it through Rancher Desktop"
    fi
  done < "${RANCHER_MANAGED_FILE}"
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

  HELM_ARGS=(
    upgrade
    --install
    "${release}"
    "${chart}"
    --namespace "${namespace}"
    --create-namespace
    --values "${values}"
  )

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
  local rollback_flag="--atomic"

  if [[ "${release}" == "infisical" ]]; then
    "${REPO_ROOT}/scripts/infisical-bootstrap.sh"
  fi

  if [[ "$(helm upgrade --help)" == *"--rollback-on-failure"* ]]; then
    rollback_flag="--rollback-on-failure"
  fi

  printf 'Applying   %-18s namespace=%-12s chart=%s\n' "${release}" "${namespace}" "${chart}"
  helm_base_args "${release}" "${namespace}" "${chart}" "${version}" "${values}"
  helm "${HELM_ARGS[@]}" \
    --timeout "${timeout}" \
    --wait \
    --wait-for-jobs \
    "${rollback_flag}" \
    --cleanup-on-fail \
    --history-max 10
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

  if [[ "${target}" == "all" || "${target}" == "grafana" ]]; then
    kubectl apply --dry-run=server --kustomize apps/observability/grafana/manifests/dashboard >/dev/null
  fi
  if [[ "${target}" == "all" || "${target}" == "k6-operator" ]]; then
    kubectl apply --dry-run=server --kustomize apps/observability/k6/dashboard >/dev/null
  fi
  if [[ "${target}" == "all" || "${target}" == "n8n" ]]; then
    kubectl apply --dry-run=server -f apps/dev/n8n/manifests/https-redirect.yaml >/dev/null
  fi
  if [[ "${target}" == "all" || "${target}" == "plane" ]]; then
    kubectl apply --dry-run=server -f apps/dev/plane/manifests/infisicalsecrets.yaml >/dev/null
    kubectl apply --dry-run=server -f apps/dev/plane/manifests/https-redirect.yaml >/dev/null
    kubectl apply --dry-run=server -f apps/dev/plane/manifests/god-mode-redirect.yaml >/dev/null
  fi
  if [[ "${target}" == "all" || "${target}" == "temporal" ]]; then
    kubectl apply --dry-run=server -f apps/dev/temporal/manifests/https-redirect.yaml >/dev/null
    kubectl apply --dry-run=server -f apps/dev/temporal/manifests/infisicalsecret.yaml >/dev/null
  fi
  if [[ "${target}" == "all" || "${target}" == "infisical" ]]; then
    kubectl apply --dry-run=server -f apps/platform/infisical/manifests/https-redirect.yaml >/dev/null
  fi
  if [[ "${target}" == "all" || "${target}" == "radar" ]]; then
    kubectl apply --dry-run=server -f apps/observability/radar/manifests/https-redirect.yaml >/dev/null
  fi
}

apply_manifests() {
  local target="$1"

  if [[ "${target}" == "all" ]]; then
    kubectl apply -f core/dns/coredns-custom.yaml
    kubectl apply -f core/dns/service.yaml
    kubectl apply -f core/ingress/tlsstore.yaml
  fi

  if [[ "${target}" == "all" || "${target}" == "monitoring" ]]; then
    scripts/patch-macos-dashboard.sh apply
  fi
  if [[ "${target}" == "all" || "${target}" == "grafana" ]]; then
    kubectl apply --kustomize apps/observability/grafana/manifests/dashboard
  fi
  if [[ "${target}" == "all" || "${target}" == "k6-operator" ]]; then
    kubectl apply --kustomize apps/observability/k6/dashboard
  fi
  if [[ "${target}" == "all" || "${target}" == "n8n" ]]; then
    kubectl apply -f apps/dev/n8n/manifests/https-redirect.yaml
  fi
  if [[ "${target}" == "all" || "${target}" == "plane" ]]; then
    kubectl apply -f apps/dev/plane/manifests/infisicalsecrets.yaml
    kubectl apply -f apps/dev/plane/manifests/https-redirect.yaml
    kubectl apply -f apps/dev/plane/manifests/god-mode-redirect.yaml
  fi
  if [[ "${target}" == "all" || "${target}" == "temporal" ]]; then
    kubectl apply -f apps/dev/temporal/manifests/https-redirect.yaml
    kubectl apply -f apps/dev/temporal/manifests/infisicalsecret.yaml
  fi
  if [[ "${target}" == "all" || "${target}" == "infisical" ]]; then
    kubectl apply -f apps/platform/infisical/manifests/https-redirect.yaml
  fi
  if [[ "${target}" == "all" || "${target}" == "radar" ]]; then
    kubectl apply -f apps/observability/radar/manifests/https-redirect.yaml
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

print_rancher_inventory() {
  local release namespace owner

  printf '%-18s %-12s %s\n' RELEASE NAMESPACE OWNER
  while IFS='|' read -r release namespace owner; do
    if [[ -z "${release}" || "${release}" == \#* ]]; then
      continue
    fi
    printf '%-18s %-12s %s\n' "${release}" "${namespace}" "${owner}"
  done < "${RANCHER_MANAGED_FILE}"
}

require_command helm
require_command kubectl
assert_rancher_boundary

command_name="${1:-help}"
target_release="${2:-all}"

case "${command_name}" in
  inventory)
    print_inventory
    printf '\nRancher-owned Helm releases excluded from repository management:\n'
    print_rancher_inventory
    ;;
  repos)
    sync_repositories
    ;;
  status)
    printf 'Kubernetes context: %s\n\n' "$(kubectl config current-context)"
    print_inventory
    printf '\nRancher-owned Helm releases excluded from repository management:\n'
    print_rancher_inventory
    printf '\nLive Helm releases:\n'
    helm list --all-namespaces
    ;;
  validate)
    reject_rancher_target "${target_release}"
    check_context
    sync_repositories_for_target "${target_release}"
    run_selected "${target_release}" validate_release
    validate_manifests "${target_release}"
    printf 'Validation completed for %s.\n' "${target_release}"
    ;;
  apply)
    reject_rancher_target "${target_release}"
    check_context
    sync_repositories_for_target "${target_release}"
    run_selected "${target_release}" apply_release
    apply_manifests "${target_release}"
    printf 'Apply completed for %s.\n' "${target_release}"
    ;;
  values)
    [[ "${target_release}" != "all" ]] || die "values requires a release name"
    reject_rancher_target "${target_release}"
    run_selected "${target_release}" show_values
    ;;
  remove)
    [[ "${target_release}" != "all" ]] || die "remove requires a release name"
    reject_rancher_target "${target_release}"
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
