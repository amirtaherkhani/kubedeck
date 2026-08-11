#!/usr/bin/env bash

set -euo pipefail

readonly EXPECTED_CONTEXT="${KUBE_CONTEXT:-rancher-desktop}"
readonly DEFAULT_DOMAIN="${INFISICAL_DOMAIN:-https://infisical.local.dev}"
readonly DEFAULT_NAMESPACE="${INFISICAL_AGENT_SECRET_NAMESPACE:-platform-secrets}"
readonly KEYCHAIN_PREFIX="${INFISICAL_AGENT_KEYCHAIN_PREFIX:-my-home-lab.infisical.agent}"

action=""
profile=""
project_id=""
environment=""
secret_path="/"
project_dir="$(pwd)"
kube_secret=""
secret_namespace="${DEFAULT_NAMESPACE}"
domain="${DEFAULT_DOMAIN}"
format="dotenv"
command_args=()

usage() {
  cat <<'EOF'
Usage:
  infisical-agent-access.sh install
  infisical-agent-access.sh configure --profile NAME --project-id ID --environment SLUG [options]
  infisical-agent-access.sh status --profile NAME
  infisical-agent-access.sh export --profile NAME [--format FORMAT]
  infisical-agent-access.sh run --profile NAME -- COMMAND [ARG...]

Configure options:
  --path PATH                 Infisical folder, default /
  --project-dir DIR           External project directory, default current directory
  --kube-secret NAME          Secret containing clientId/clientSecret
  --secret-namespace NAME     Kubernetes namespace, default platform-secrets
  --domain URL                Infisical URL, default https://infisical.local.dev

If --kube-secret is omitted, configure accepts INFISICAL_AGENT_CLIENT_ID and
INFISICAL_AGENT_CLIENT_SECRET for one-time setup.
EOF
}

die() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

service_name() {
  printf 'my-home-lab.infisical.agent.%s' "${profile}"
}

keychain_get() {
  security find-generic-password -a "$1" -s "$(service_name)" -w 2>/dev/null || true
}

keychain_put() {
  security add-generic-password -a "$1" -s "$(service_name)" -w "$2" -U >/dev/null
}

install_cli() {
  require_command brew
  if command -v infisical >/dev/null 2>&1; then
    printf 'Infisical CLI already installed.\n'
    return 0
  fi
  brew install infisical/get-cli/infisical
  printf 'Infisical CLI installed.\n'
}

check_cluster() {
  require_command kubectl
  [[ "$(kubectl config current-context)" == "${EXPECTED_CONTEXT}" ]] || \
    die "current Kubernetes context is not ${EXPECTED_CONTEXT}"
  kubectl get nodes --no-headers | awk '$2 == "Ready" { found=1 } END { exit(found ? 0 : 1) }' || \
    die "no Ready node found in ${EXPECTED_CONTEXT}"
}

load_profile() {
  local config
  config="$(keychain_get config)"
  [[ -n "${config}" ]] || die "profile not configured: ${profile}"
  project_id="$(printf '%s' "${config}" | jq -er '.projectId')"
  environment="$(printf '%s' "${config}" | jq -er '.environment')"
  secret_path="$(printf '%s' "${config}" | jq -er '.path')"
  domain="$(printf '%s' "${config}" | jq -er '.domain')"
  client_id="$(keychain_get client-id)"
  client_secret="$(keychain_get client-secret)"
  [[ -n "${client_id}" && -n "${client_secret}" ]] || die "incomplete Keychain profile: ${profile}"
}

load_kubernetes_credentials() {
  [[ -n "${kube_secret}" ]] || return 0
  check_cluster
  client_id="$(kubectl get secret "${kube_secret}" -n "${secret_namespace}" -o jsonpath='{.data.clientId}' | base64 -D)"
  client_secret="$(kubectl get secret "${kube_secret}" -n "${secret_namespace}" -o jsonpath='{.data.clientSecret}' | base64 -D)"
  [[ -n "${client_id}" && -n "${client_secret}" ]] || \
    die "${secret_namespace}/${kube_secret} must contain clientId and clientSecret"
}

login_token() {
  require_command curl
  require_command jq
  curl -ksS -X POST "${domain}/api/v1/auth/universal-auth/login" \
    -H 'Content-Type: application/json' \
    --data "$(jq -cn --arg id "${client_id}" --arg secret "${client_secret}" '{clientId:$id,clientSecret:$secret}')" \
    | jq -er '.accessToken'
}

configure_profile() {
  require_command infisical
  require_command jq
  require_command security
  [[ -n "${profile}" && -n "${project_id}" && -n "${environment}" ]] || \
    die '--profile, --project-id, and --environment are required'
  [[ -d "${project_dir}" ]] || die "project directory does not exist: ${project_dir}"

  client_id="${INFISICAL_AGENT_CLIENT_ID:-}"
  client_secret="${INFISICAL_AGENT_CLIENT_SECRET:-}"
  load_kubernetes_credentials
  [[ -n "${client_id}" && -n "${client_secret}" ]] || \
    die 'provide --kube-secret or INFISICAL_AGENT_CLIENT_ID/INFISICAL_AGENT_CLIENT_SECRET'

  keychain_put client-id "${client_id}"
  keychain_put client-secret "${client_secret}"
  keychain_put config "$(jq -cn --arg p "${project_id}" --arg e "${environment}" --arg s "${secret_path}" --arg d "${domain}" '{projectId:$p,environment:$e,path:$s,domain:$d}')"

  jq -n --arg p "${project_id}" --arg e "${environment}" --arg d "${domain}" \
    '{workspaceId:$p,defaultEnvironment:$e,domain:$d}' > "${project_dir}/.infisical.json"

  token="$(login_token)"
  [[ -n "${token}" ]] || die 'Universal Auth login returned no token'
  INFISICAL_TOKEN="${token}" infisical login status --domain="${domain}" --json >/dev/null 2>&1
  printf 'Configured profile %s.\n' "${profile}"
  printf 'Wrote non-secret metadata to %s/.infisical.json.\n' "${project_dir}"
  printf 'Credentials stored in Keychain service %s.\n' "$(service_name)"
}

run_status() {
  require_command infisical
  require_command jq
  load_profile
  token="$(login_token)"
  INFISICAL_TOKEN="${token}" infisical login status --domain="${domain}" --json 2>/dev/null \
    | jq '.sessions[] | select(.principalType == "machine-identity") | {status,domain,identityId:.identity.id,verification}'
}

run_export() {
  require_command infisical
  load_profile
  token="$(login_token)"
  INFISICAL_TOKEN="${token}" INFISICAL_DOMAIN="${domain}" infisical export \
    --projectId="${project_id}" --env="${environment}" --path="${secret_path}" --format="${format}"
}

run_command() {
  require_command infisical
  load_profile
  [[ "${#command_args[@]}" -gt 0 ]] || die 'run requires a command after --'
  token="$(login_token)"
  INFISICAL_TOKEN="${token}" INFISICAL_DOMAIN="${domain}" infisical run \
    --projectId="${project_id}" --env="${environment}" \
    --path="${secret_path}" -- "${command_args[@]}"
}

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    install|configure|status|export|run) action="$1"; shift ;;
    --profile) profile="$2"; shift 2 ;;
    --project-id) project_id="$2"; shift 2 ;;
    --environment) environment="$2"; shift 2 ;;
    --path) secret_path="$2"; shift 2 ;;
    --project-dir) project_dir="$2"; shift 2 ;;
    --kube-secret) kube_secret="$2"; shift 2 ;;
    --secret-namespace) secret_namespace="$2"; shift 2 ;;
    --domain) domain="$2"; shift 2 ;;
    --format) format="$2"; shift 2 ;;
    --) shift; command_args=("$@"); break ;;
    --help|-h) usage; exit 0 ;;
    *) die "unknown option or action: $1" ;;
  esac
done

case "${action}" in
  install) install_cli ;;
  configure) configure_profile ;;
  status) run_status ;;
  export) run_export ;;
  run) run_command ;;
  *) usage; exit 1 ;;
esac
