#!/usr/bin/env bash

set -euo pipefail

readonly DOMAIN="https://infisical.local.dev"
readonly IDENTITY_NAME="local-dev-agents"
readonly KEYCHAIN_SERVICE="my-home-lab.infisical.agent.local-dev"
readonly ADMIN_KEYCHAIN_SERVICE="my-home-lab.infisical.admin"
readonly ADMIN_EMAIL="admin@local.dev"
readonly CLIENT_SECRET_DESCRIPTION="local-dev-env-mcp on this Mac"
readonly HOME_LAB_PROJECT_ID="e064fd84-b51e-4318-8442-8f30fec2b316"
readonly FINANCE_PROJECT_ID="4574398d-423d-49bc-90e0-1e6a57a20c23"
readonly CLIENT_SECRET_TTL="31536000"
readonly ACCESS_TOKEN_TTL="3600"
readonly LOCAL_DEV_PROJECT_IDS="${INFISICAL_LOCAL_DEV_PROJECT_IDS:-${HOME_LAB_PROJECT_ID},${FINANCE_PROJECT_ID}}"
readonly REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly PROJECT_ROOT="$(git -C "${REPO_ROOT}" worktree list --porcelain | awk '/^worktree / {sub(/^worktree /, ""); print; exit}')"

# shellcheck source=lib/local-dev-env-bootstrap-surfaces.sh
source "${REPO_ROOT}/scripts/lib/local-dev-env-bootstrap-surfaces.sh"

die() { printf 'Error: %s\n' "$*" >&2; exit 1; }
require_command() { command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"; }

keychain_optional() {
  local account="$1" value status
  if value="$(security find-generic-password -s "${KEYCHAIN_SERVICE}" -a "${account}" -w 2>/dev/null)"; then
    printf '%s\n' "${value}"
    return
  else
    status="$?"
  fi
  [[ "${status}" -eq 44 ]] || die "macOS Keychain lookup failed for account ${account}"
}

keychain_put() {
  local account="$1" value="$2"
  printf '%s\n%s\n' "${value}" "${value}" |
    security add-generic-password -s "${KEYCHAIN_SERVICE}" -a "${account}" -U -w >/dev/null 2>&1
}

authorization_config() {
  printf 'header = "Authorization: Bearer %s"\n' "$1"
}

curl_json() {
  local method="$1" url="$2" token="${3:-}" payload="${4:-}"
  if [[ -n "${token}" && -n "${payload}" ]]; then
    curl -fsS -X "${method}" "${url}" --config <(authorization_config "${token}") \
      -H 'Content-Type: application/json' --data-binary @- <<<"${payload}"
  elif [[ -n "${token}" ]]; then
    curl -fsS -X "${method}" "${url}" --config <(authorization_config "${token}")
  elif [[ -n "${payload}" ]]; then
    curl -fsS -X "${method}" "${url}" -H 'Content-Type: application/json' --data-binary @- <<<"${payload}"
  else
    curl -fsS -X "${method}" "${url}"
  fi
}

admin_token() {
  local password payload response initial_token organizations organization_id selected
  password="$(security find-generic-password -s "${ADMIN_KEYCHAIN_SERVICE}" -a "${ADMIN_EMAIL}" -w 2>/dev/null)" || \
    die "local administrator Keychain entry is unavailable for ${ADMIN_EMAIL}"
  payload="$(printf '%s\n%s\n' "${ADMIN_EMAIL}" "${password}" | jq -Rn '[inputs] | {email:.[0],password:.[1]}')"
  response="$(curl_json POST "${DOMAIN}/api/v3/auth/login" "" "${payload}")" || {
      unset password
      die 'administrator authentication failed'
    }
  unset password payload
  initial_token="$(jq -er '.accessToken' <<<"${response}")"
  organizations="$(curl_json GET "${DOMAIN}/api/v1/organization" "${initial_token}")"
  organization_id="$(jq -er '(.organizations // .)[0].id' <<<"${organizations}")"
  payload="$(jq -cn --arg id "${organization_id}" '{organizationId:$id,userAgent:"cli"}')"
  selected="$(curl_json POST "${DOMAIN}/api/v3/auth/select-organization" "${initial_token}" "${payload}")"
  unset initial_token
  printf '%s\t%s\n' "$(jq -er '.token' <<<"${selected}")" "${organization_id}"
}

admin_api() {
  local token="$1" method="$2" path="$3" payload="${4:-}" response
  if [[ -n "${payload}" ]]; then
    response="$(curl_json "${method}" "${DOMAIN}/api/v1${path}" "${token}" "${payload}")" || \
      die "Infisical admin API failed: ${method} /api/v1${path}"
  else
    response="$(curl_json "${method}" "${DOMAIN}/api/v1${path}" "${token}")" || \
      die "Infisical admin API failed: ${method} /api/v1${path}"
  fi
  printf '%s\n' "${response}"
}

ensure_membership() {
  local token="$1" project_id="$2" identity_id="$3" memberships member
  memberships="$(admin_api "${token}" GET "/projects/${project_id}/identity-memberships?limit=100")"
  member="$(jq -c --arg id "${identity_id}" '.identityMemberships[]? | select(.identityId == $id)' <<<"${memberships}")"
  if [[ -z "${member}" ]]; then
    admin_api "${token}" POST "/projects/${project_id}/identity-memberships/${identity_id}" \
      '{"roles":[{"role":"member","isTemporary":false}]}' >/dev/null
  elif ! jq -e '.roles | length == 1 and .[0].role == "member" and (.[0].isTemporary | not)' <<<"${member}" >/dev/null; then
    admin_api "${token}" PATCH "/projects/${project_id}/identity-memberships/${identity_id}" \
      '{"roles":[{"role":"member","isTemporary":false}]}' >/dev/null
  fi
}

ensure_folder_path() {
  local token="$1" project_id="$2" environment="$3" target_path="$4"
  local parent_path="/" segment existing
  local segments=()
  IFS='/' read -r -a segments <<<"${target_path#/}"
  for segment in "${segments[@]}"; do
    [[ -n "${segment}" ]] || continue
    existing="$(curl -fsS -G "${DOMAIN}/api/v1/folders" --config <(authorization_config "${token}") \
      --data-urlencode "workspaceId=${project_id}" --data-urlencode "environment=${environment}" \
      --data-urlencode "path=${parent_path}")"
    if ! jq -e --arg name "${segment}" '.folders[]? | select(.name == $name)' <<<"${existing}" >/dev/null; then
      admin_api "${token}" POST /folders "$(jq -cn --arg project "${project_id}" --arg environment "${environment}" \
        --arg name "${segment}" --arg parent "${parent_path}" \
        '{workspaceId:$project,environment:$environment,name:$name,path:$parent}')" >/dev/null
    fi
    if [[ "${parent_path}" == "/" ]]; then parent_path="/${segment}"; else parent_path="${parent_path}/${segment}"; fi
  done
}

credentials_are_valid() {
  local token="$1" expected_identity_id="$2" client_id client_secret client_secret_id response token_identity_id client_secret_prefix
  client_id="$(keychain_optional client-id)"
  client_secret="$(keychain_optional client-secret)"
  client_secret_id="$(keychain_optional client-secret-id)"
  [[ -n "${client_id}" && -n "${client_secret}" && -n "${client_secret_id}" ]] || return 1
  client_secret_prefix="$(admin_api "${token}" GET "/auth/universal-auth/identities/${expected_identity_id}/client-secrets" |
    jq -er --arg id "${client_secret_id}" --argjson ttl "${CLIENT_SECRET_TTL}" '.clientSecretData[] | select(.id == $id and (.isClientSecretRevoked | not) and .clientSecretTTL == $ttl) | .clientSecretPrefix')" || return 1
  [[ "${client_secret}" == "${client_secret_prefix}"* ]] || return 1
  response="$(printf '%s\n%s\n' "${client_id}" "${client_secret}" | jq -Rn '[inputs] | {clientId:.[0],clientSecret:.[1]}' |
    curl -fsS -X POST "${DOMAIN}/api/v1/auth/universal-auth/login" -H 'Content-Type: application/json' --data-binary @-)" || return 1
  token_identity_id="$(jq -er '.accessToken | split(".")[1] | @base64d | fromjson | .identityId' <<<"${response}")" || return 1
  [[ "${token_identity_id}" == "${expected_identity_id}" ]]
}

universal_auth_is_attached() {
  local token="$1" identity_id="$2" http_status
  http_status="$(curl -sS -o /dev/null -w '%{http_code}' --config <(authorization_config "${token}") \
    "${DOMAIN}/api/v1/auth/universal-auth/identities/${identity_id}")"
  case "${http_status}" in
    200) return 0 ;;
    404) return 1 ;;
    *) die "Infisical admin API failed: GET /api/v1/auth/universal-auth/identities/${identity_id} (HTTP ${http_status})" ;;
  esac
}

configure_universal_auth() {
  local token="$1" identity_id="$2" method="POST"
  if universal_auth_is_attached "${token}" "${identity_id}"; then method="PATCH"; fi
  # This self-hosted edition rejects custom IP ranges. The endpoint remains local-only,
  # while the one-year client credential is stored only in Keychain and access tokens
  # remain short-lived and are renewed by the connector.
  admin_api "${token}" "${method}" "/auth/universal-auth/identities/${identity_id}" \
    "$(jq -cn --argjson ttl "${ACCESS_TOKEN_TTL}" '{clientSecretTrustedIps:[{ipAddress:"0.0.0.0/0"},{ipAddress:"::/0"}],accessTokenTrustedIps:[{ipAddress:"0.0.0.0/0"},{ipAddress:"::/0"}],accessTokenTTL:$ttl,accessTokenMaxTTL:$ttl,accessTokenNumUsesLimit:0,accessTokenPeriod:0,lockoutEnabled:true,lockoutThreshold:5,lockoutDurationSeconds:60,lockoutCounterResetSeconds:30}')"
}

revoke_superseded_client_secrets() {
  local token="$1" identity_id="$2" keep_id="$3" secret_id inventory
  inventory="$(admin_api "${token}" GET "/auth/universal-auth/identities/${identity_id}/client-secrets")"
  while IFS= read -r secret_id; do
    admin_api "${token}" POST "/auth/universal-auth/identities/${identity_id}/client-secrets/${secret_id}/revoke" >/dev/null
  done < <(jq -r --arg keep "${keep_id}" --arg description "${CLIENT_SECRET_DESCRIPTION}" \
    '.clientSecretData[] | select(.id != $keep and .description == $description and (.isClientSecretRevoked | not)) | .id' <<<"${inventory}")
}

ensure_connector_credentials() {
  local token="$1" identity_id="$2" identity client_id client_secret client_secret_id
  identity="$(configure_universal_auth "${token}" "${identity_id}")"
  if ! credentials_are_valid "${token}" "${identity_id}"; then
    client_id="$(jq -er '.identityUniversalAuth.clientId' <<<"${identity}")"
    identity="$(admin_api "${token}" POST "/auth/universal-auth/identities/${identity_id}/client-secrets" \
      "$(jq -cn --arg description "${CLIENT_SECRET_DESCRIPTION}" --argjson ttl "${CLIENT_SECRET_TTL}" '{description:$description,numUsesLimit:0,ttl:$ttl}')")"
    client_secret="$(jq -er '.clientSecret' <<<"${identity}")"
    client_secret_id="$(jq -er '.clientSecretData.id' <<<"${identity}")"
    keychain_put client-id "${client_id}"
    keychain_put client-secret "${client_secret}"
    keychain_put client-secret-id "${client_secret_id}"
    unset client_id client_secret
  fi
  client_secret_id="$(security find-generic-password -s "${KEYCHAIN_SERVICE}" -a client-secret-id -w)"
  revoke_superseded_client_secrets "${token}" "${identity_id}" "${client_secret_id}"
}

main() {
  local admin_context token organization_id search identity_id identity project_id
  for command in curl jq security npm node; do require_command "${command}"; done
  admin_context="$(admin_token)"
  IFS=$'\t' read -r token organization_id <<<"${admin_context}"
  search="$(admin_api "${token}" POST /identities/search "$(jq -cn --arg name "${IDENTITY_NAME}" '{limit:100,offset:0,search:{name:{"$eq":$name}}}')")"
  identity_id="$(jq -r '.identities[0].identity.id // empty' <<<"${search}")"
  if [[ -z "${identity_id}" ]]; then
    identity="$(admin_api "${token}" POST /identities "$(jq -cn --arg name "${IDENTITY_NAME}" --arg org "${organization_id}" \
      '{name:$name,organizationId:$org,role:"no-access",hasDeleteProtection:false,metadata:[{key:"owner",value:"my-home-lab"},{key:"scope",value:"approved-development-projects-only"},{key:"rotation",value:"365-days"}]}')")"
    identity_id="$(jq -er '.identity.id' <<<"${identity}")"
  fi

  local project_ids=()
  IFS=',' read -r -a project_ids <<<"${LOCAL_DEV_PROJECT_IDS}"
  for project_id in "${project_ids[@]}"; do
    [[ -n "${project_id}" ]] || continue
    admin_api "${token}" GET "/projects/${project_id}" >/dev/null
    ensure_membership "${token}" "${project_id}" "${identity_id}"
  done

  ensure_connector_credentials "${token}" "${identity_id}"
  install_agent_surfaces
  register_projects "${token}"
  unset token
  (cd "${REPO_ROOT}" && NODE_USE_SYSTEM_CA=1 kubedeck-env-mcp doctor >/dev/null)
  printf 'kubedeck-env-mcp is installed, registered, and authenticated for local development.\n'
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then main "$@"; fi
