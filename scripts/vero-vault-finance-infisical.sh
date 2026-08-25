#!/usr/bin/env bash

set -euo pipefail

# Finance-only Infisical access. This script intentionally never prints or
# writes credentials, access tokens, or secret values to disk.
readonly DOMAIN="${INFISICAL_DOMAIN:-https://infisical.local.dev}"
readonly PROJECT_ID="4574398d-423d-49bc-90e0-1e6a57a20c23"
readonly ENVIRONMENT="development"
readonly SECRET_PATH="/finance"
readonly IDENTITY_NAME="vero-vault-finance-agent"
readonly AUTH_SECRET_NAMESPACE="platform-secrets"
readonly AUTH_SECRET_NAME="vero-vault-finance-infisical-auth"
readonly ADMIN_KEYCHAIN_SERVICE="my-home-lab.infisical.admin"
readonly ADMIN_EMAIL="admin@local.dev"
readonly FINANCE_PROJECT_DIR="${VERO_FINANCE_PROJECT_DIR:-/Users/mac/Documents/GitHub/verovault-finance}"

die() { printf 'Error: %s\n' "$*" >&2; exit 1; }

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

usage() {
  cat <<'EOF'
Usage:
  vero-vault-finance-infisical.sh provision-agent
  vero-vault-finance-infisical.sh status
  vero-vault-finance-infisical.sh list
  vero-vault-finance-infisical.sh get KEY
  vero-vault-finance-infisical.sh set KEY < secret-value-on-stdin
  vero-vault-finance-infisical.sh delete KEY
  vero-vault-finance-infisical.sh run -- COMMAND [ARG...]

`provision-agent` is a one-time local bootstrap. It creates the Finance-only
machine identity `vero-vault-finance-agent`, grants it Member access to the
`vero-finance` project, configures Universal Auth, stores credentials only in
macOS Keychain and platform-secrets/vero-vault-finance-infisical-auth, then
configures the local Finance profile.

`get` intentionally prints the requested secret. `set` reads one value from
standard input so it is not recorded in shell history. `delete` changes a
shared secret and should be used only for an intended Finance key.
EOF
}

check_cluster() {
  [[ "$(kubectl config current-context)" == "rancher-desktop" ]] || \
    die 'current Kubernetes context is not rancher-desktop'
  kubectl get nodes --no-headers | awk '$2 == "Ready" { found=1 } END { exit(found ? 0 : 1) }' || \
    die 'no Ready Rancher Desktop node found'
}

admin_token() {
  local password response initial_token organizations organization_id selected
  password="$(security find-generic-password -s "${ADMIN_KEYCHAIN_SERVICE}" -a "${ADMIN_EMAIL}" -w 2>/dev/null)" || \
    die "local administrator Keychain entry is unavailable for ${ADMIN_EMAIL}"
  response="$(curl -ksSf -X POST "${DOMAIN}/api/v3/auth/login" \
    -H 'Content-Type: application/json' \
    -H 'User-Agent: my-home-lab-finance-bootstrap/1.0' \
    --data "$(jq -cn --arg email "${ADMIN_EMAIL}" --arg password "${password}" '{email:$email,password:$password}')")" || {
      unset password
      die 'administrator authentication failed'
    }
  unset password
  initial_token="$(jq -er '.accessToken' <<<"${response}")"
  organizations="$(curl -ksSf -H "Authorization: Bearer ${initial_token}" "${DOMAIN}/api/v1/organization")" || {
    unset initial_token
    die 'could not list administrator organizations'
  }
  organization_id="$(jq -er '(.organizations // .)[0].id' <<<"${organizations}")"
  selected="$(curl -ksSf -X POST "${DOMAIN}/api/v3/auth/select-organization" \
    -H "Authorization: Bearer ${initial_token}" \
    -H 'Content-Type: application/json' \
    -H 'User-Agent: my-home-lab-finance-bootstrap/1.0' \
    --data "$(jq -cn --arg id "${organization_id}" '{organizationId:$id,userAgent:"cli"}')")" || {
      unset initial_token
      die 'could not select the administrator organization'
    }
  unset initial_token
  printf '%s\t%s\n' "$(jq -er '.token' <<<"${selected}")" "${organization_id}"
}

admin_api() {
  local token="$1" method="$2" path="$3" payload="${4:-}"
  if [[ -n "${payload}" ]]; then
    curl -ksSf -X "${method}" "${DOMAIN}/api/v1${path}" \
      -H "Authorization: Bearer ${token}" \
      -H 'Content-Type: application/json' \
      --data "${payload}"
  else
    curl -ksSf -X "${method}" "${DOMAIN}/api/v1${path}" \
      -H "Authorization: Bearer ${token}"
  fi
}

finance_token() {
  local client_id client_secret response
  client_id="$(security find-generic-password -s 'my-home-lab.infisical.agent.vero-finance' -a client-id -w 2>/dev/null)" || \
    die 'Finance profile is not configured; run provision-agent first'
  client_secret="$(security find-generic-password -s 'my-home-lab.infisical.agent.vero-finance' -a client-secret -w 2>/dev/null)" || \
    die 'Finance profile has no client secret; rotate it through the Finance workflow'
  response="$(curl -ksSf -X POST "${DOMAIN}/api/v1/auth/universal-auth/login" \
    -H 'Content-Type: application/json' \
    --data "$(jq -cn --arg id "${client_id}" --arg secret "${client_secret}" '{clientId:$id,clientSecret:$secret}')")" || {
      unset client_id client_secret
      die 'Finance Universal Auth login failed'
    }
  unset client_id client_secret
  jq -er '.accessToken' <<<"${response}"
}

with_finance_cli() {
  local token
  token="$(finance_token)"
  INFISICAL_TOKEN="${token}" infisical secrets "$@" \
    --env="${ENVIRONMENT}" --path="${SECRET_PATH}" --projectId="${PROJECT_ID}" --domain="${DOMAIN}" --silent
  unset token
}

list_finance_keys() {
  local token
  token="$(finance_token)"
  INFISICAL_TOKEN="${token}" infisical export \
    --env="${ENVIRONMENT}" --path="${SECRET_PATH}" --projectId="${PROJECT_ID}" \
    --format=json --domain="${DOMAIN}" --silent | jq -r 'if type == "array" then .[].key else keys[]? end' | sort
  unset token
}

provision_agent() {
  local admin_context token organization_id search_response identity_response identity_id membership_response ua_response client_id client_secret
  require_command curl
  require_command jq
  require_command kubectl
  require_command security
  require_command infisical
  [[ -d "${FINANCE_PROJECT_DIR}" ]] || die "Finance project directory does not exist: ${FINANCE_PROJECT_DIR}"
  check_cluster

  if kubectl get secret "${AUTH_SECRET_NAME}" -n "${AUTH_SECRET_NAMESPACE}" >/dev/null 2>&1; then
    die "${AUTH_SECRET_NAMESPACE}/${AUTH_SECRET_NAME} already exists; refusing to overwrite credentials"
  fi

  admin_context="$(admin_token)"
  IFS=$'\t' read -r token organization_id <<<"${admin_context}"
  admin_api "${token}" GET "/projects/${PROJECT_ID}" >/dev/null || \
    die "Finance project ${PROJECT_ID} is not accessible in the selected Infisical organization; restore access or explicitly create a replacement project before provisioning"
  search_response="$(admin_api "${token}" POST '/identities/search' "$(jq -cn --arg name "${IDENTITY_NAME}" \
    '{limit:100,offset:0,search:{name:{"$eq":$name}}}')")" || die 'could not search Finance machine identities'
  identity_id="$(jq -r '.identities[0].identity.id // empty' <<<"${search_response}")"
  if [[ -z "${identity_id}" ]]; then
    identity_response="$(admin_api "${token}" POST '/identities' "$(jq -cn \
      --arg name "${IDENTITY_NAME}" \
      --arg org "${organization_id}" \
      '{name:$name,organizationId:$org,role:"no-access",hasDeleteProtection:false,metadata:[{key:"owner",value:"vero-vault-finance"},{key:"scope",value:"development:/finance"},{key:"rotation",value:"90-days"}]}'
    )")" || die 'could not create the Finance machine identity'
    identity_id="$(jq -er '.identity.id' <<<"${identity_response}")"
  fi

  membership_response="$(admin_api "${token}" GET "/projects/${PROJECT_ID}/identity-memberships?limit=100")" || \
    die 'could not inspect Finance identity memberships'
  if ! jq -e --arg id "${identity_id}" '.identityMemberships[]? | select(.identityId == $id)' <<<"${membership_response}" >/dev/null; then
    admin_api "${token}" POST "/projects/${PROJECT_ID}/identity-memberships/${identity_id}" \
      '{"roles":[{"role":"member","isTemporary":false}]}' >/dev/null || \
      die 'Finance identity was created but could not receive project Member access'
  fi

  ua_response="$(admin_api "${token}" POST "/auth/universal-auth/identities/${identity_id}" \
    '{"clientSecretTrustedIps":[{"ipAddress":"0.0.0.0/0"},{"ipAddress":"::/0"}],"accessTokenTrustedIps":[{"ipAddress":"0.0.0.0/0"},{"ipAddress":"::/0"}],"accessTokenTTL":3600,"accessTokenMaxTTL":3600,"accessTokenNumUsesLimit":0,"accessTokenPeriod":0,"lockoutEnabled":true,"lockoutThreshold":3,"lockoutDurationSeconds":300,"lockoutCounterResetSeconds":30}')" || \
    die 'Finance identity membership exists but Universal Auth could not be configured'
  client_id="$(jq -er '.identityUniversalAuth.clientId' <<<"${ua_response}")"
  client_secret="$(admin_api "${token}" POST "/auth/universal-auth/identities/${identity_id}/client-secrets" \
    '{"description":"vero-vault-finance-agent local Rancher Desktop","numUsesLimit":0,"ttl":7776000}' | jq -er '.clientSecret')" || \
    die 'Finance Universal Auth exists but its client secret could not be created'
  unset token

  kubectl create secret generic "${AUTH_SECRET_NAME}" -n "${AUTH_SECRET_NAMESPACE}" \
    --from-literal=clientId="${client_id}" \
    --from-literal=clientSecret="${client_secret}" \
    --dry-run=client -o yaml | kubectl apply -f - >/dev/null

  INFISICAL_AGENT_CLIENT_ID="${client_id}" INFISICAL_AGENT_CLIENT_SECRET="${client_secret}" \
    ./scripts/infisical-agent-access.sh configure \
      --profile vero-finance \
      --project-id "${PROJECT_ID}" \
      --environment "${ENVIRONMENT}" \
      --path "${SECRET_PATH}" \
      --project-dir "${FINANCE_PROJECT_DIR}" >/dev/null
  unset client_id client_secret

  token="$(finance_token)"
  curl -ksSf -X POST "${DOMAIN}/api/v1/folders" \
    -H "Authorization: Bearer ${token}" \
    -H 'Content-Type: application/json' \
    --data "$(jq -cn --arg project "${PROJECT_ID}" --arg environment "${ENVIRONMENT}" \
      '{workspaceId:$project,environment:$environment,name:"finance",path:"/"}')" >/dev/null
  unset token

  ./scripts/infisical-agent-access.sh status --profile vero-finance >/dev/null
  printf 'Provisioned Finance machine identity %s with Member access to %s:%s.\n' \
    "${IDENTITY_NAME}" "${ENVIRONMENT}" "${SECRET_PATH}"
}

main() {
  local action="${1:-}"
  case "${action}" in
    provision-agent) provision_agent ;;
    status) ./scripts/infisical-agent-access.sh status --profile vero-finance ;;
    list) list_finance_keys ;;
    get) [[ $# -eq 2 ]] || die 'get requires KEY'; with_finance_cli get "$2" --plain ;;
    set)
      [[ $# -eq 2 ]] || die 'set requires KEY and reads the value from standard input'
      local value
      IFS= read -r value || die 'set requires one value on standard input'
      with_finance_cli set "$2=$value"
      unset value
      ;;
    delete) [[ $# -eq 2 ]] || die 'delete requires KEY'; with_finance_cli delete "$2" --type shared ;;
    run) shift; [[ "${1:-}" == '--' ]] || die 'run requires -- COMMAND [ARG...]'; shift; ./scripts/infisical-agent-access.sh run --profile vero-finance -- "$@" ;;
    --help|-h|help|'') usage ;;
    *) die "unknown action: ${action}" ;;
  esac
}

main "$@"
