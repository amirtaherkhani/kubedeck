#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
EXPECTED_CONTEXT="${KUBE_CONTEXT:-rancher-desktop}"
NAMESPACE="platform-secrets"
API_URL="${INFISICAL_API_URL:-https://infisical.local.dev}"
ADMIN_EMAIL="${INFISICAL_ADMIN_EMAIL:-admin@local.dev}"
ADMIN_ORGANIZATION="${INFISICAL_ADMIN_ORGANIZATION:-home-lab}"
KEYCHAIN_SERVICE="${INFISICAL_KEYCHAIN_SERVICE:-my-home-lab.infisical.admin}"
BOOTSTRAP_SECRET="${INFISICAL_BOOTSTRAP_SECRET:-infisical-bootstrap-admin}"

cd "${REPO_ROOT}"

for command_name in kubectl curl jq openssl security; do
  command -v "${command_name}" >/dev/null 2>&1 || {
    printf 'Error: required command not found: %s\n' "${command_name}" >&2
    exit 1
  }
done

[[ "$(kubectl config current-context)" == "${EXPECTED_CONTEXT}" ]] || {
  printf 'Error: current Kubernetes context is not %s\n' "${EXPECTED_CONTEXT}" >&2
  exit 1
}

kubectl get namespace "${NAMESPACE}" >/dev/null

if kubectl get secret "${BOOTSTRAP_SECRET}" -n "${NAMESPACE}" >/dev/null 2>&1; then
  printf 'Infisical bootstrap credentials already exist in Kubernetes Secret %s/%s.\n' "${NAMESPACE}" "${BOOTSTRAP_SECRET}"
  exit 0
fi

admin_password="${INFISICAL_ADMIN_PASSWORD:-}"
if [[ -z "${admin_password}" ]]; then
  admin_password="$(security find-generic-password -a "${ADMIN_EMAIL}" -s "${KEYCHAIN_SERVICE}" -w 2>/dev/null || true)"
fi
if [[ -z "${admin_password}" ]]; then
  admin_password="$(openssl rand -hex 32)"
  security add-generic-password \
    -a "${ADMIN_EMAIL}" \
    -s "${KEYCHAIN_SERVICE}" \
    -w "${admin_password}" \
    -U >/dev/null
  printf 'Generated and stored the Infisical administrator password in macOS Keychain (%s).\n' "${KEYCHAIN_SERVICE}"
else
  printf 'Using the existing Infisical administrator password from macOS Keychain.\n'
fi

payload="$(jq -cn \
  --arg email "${ADMIN_EMAIL}" \
  --arg password "${admin_password}" \
  --arg organization "${ADMIN_ORGANIZATION}" \
  '{email: $email, password: $password, organization: $organization}')"

response_file="$(mktemp)"
trap 'rm -f "${response_file}"' EXIT

http_status="$(curl -ksS \
  -o "${response_file}" \
  -w '%{http_code}' \
  -X POST "${API_URL}/api/v1/admin/bootstrap" \
  -H 'Content-Type: application/json' \
  --data "${payload}")"

if [[ "${http_status}" != 2* ]]; then
  printf 'Error: Infisical admin bootstrap returned HTTP %s.\n' "${http_status}" >&2
  jq -r 'if type == "object" then (.message // .error // "bootstrap failed") else "bootstrap failed" end' "${response_file}" >&2 || true
  exit 1
fi

token="$(jq -er '.identity.credentials.token' "${response_file}")"
organization_id="$(jq -er '.organization.id' "${response_file}")"
user_id="$(jq -er '.user.id' "${response_file}")"

kubectl create secret generic "${BOOTSTRAP_SECRET}" \
  --namespace "${NAMESPACE}" \
  --from-literal="token=${token}" \
  --from-literal="organization-id=${organization_id}" \
  --from-literal="user-id=${user_id}" \
  --from-literal="email=${ADMIN_EMAIL}" \
  --dry-run=client -o yaml \
  | kubectl apply -f - >/dev/null

printf 'Infisical administrator bootstrapped: %s\n' "${ADMIN_EMAIL}"
printf 'Organization created: %s\n' "${ADMIN_ORGANIZATION}"
printf 'Machine identity token stored in Kubernetes Secret %s/%s.\n' "${NAMESPACE}" "${BOOTSTRAP_SECRET}"
