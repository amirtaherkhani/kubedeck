#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
EXPECTED_CONTEXT="${KUBE_CONTEXT:-rancher-desktop}"
NAMESPACE="platform-secrets"

cd "${REPO_ROOT}"

current_context="$(kubectl config current-context)"
[[ "${current_context}" == "${EXPECTED_CONTEXT}" ]] || {
  printf 'Error: current Kubernetes context is %s; expected %s\n' "${current_context}" "${EXPECTED_CONTEXT}" >&2
  exit 1
}

kubectl get namespace "${NAMESPACE}" >/dev/null

if kubectl get secret infisical-postgresql -n "${NAMESPACE}" >/dev/null 2>&1; then
  printf '%s\n' 'Keeping existing Infisical PostgreSQL bootstrap credentials.'
else
  postgres_password="$(openssl rand -hex 24)"
  kubectl create secret generic infisical-postgresql \
    --namespace "${NAMESPACE}" \
    --from-literal='POSTGRES_DB=infisical' \
    --from-literal='POSTGRES_USER=infisical' \
    --from-literal="POSTGRES_PASSWORD=${postgres_password}" \
    --dry-run=client -o yaml | kubectl apply -f -
  printf '%s\n' 'Created Infisical PostgreSQL bootstrap credentials.'
fi

kubectl apply -f apps/platform/infisical/manifests/postgresql.yaml

postgres_password="$(kubectl get secret infisical-postgresql -n "${NAMESPACE}" \
  -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -D)"
db_connection_uri="postgresql://infisical:${postgres_password}@infisical-postgresql:5432/infisical"

if kubectl get secret infisical-secrets -n "${NAMESPACE}" >/dev/null 2>&1; then
  auth_secret="$(kubectl get secret infisical-secrets -n "${NAMESPACE}" \
    -o jsonpath='{.data.AUTH_SECRET}' | base64 -D 2>/dev/null || true)"
  if [[ -z "${auth_secret}" ]]; then
    auth_secret="$(openssl rand -base64 32)"
  fi
  kubectl get secret infisical-secrets -n "${NAMESPACE}" -o json \
    | jq --arg auth "${auth_secret}" --arg db "${db_connection_uri}" \
      '.data.AUTH_SECRET=($auth|@base64) | .data.DB_CONNECTION_URI=($db|@base64) | .data.HOST=("0.0.0.0"|@base64) | .data.PORT=("8080"|@base64) | .data.INVITE_ONLY_SIGNUP=("false"|@base64)' \
    | kubectl apply -f -
  printf '%s\n' 'Updated Infisical backend bootstrap configuration.'
else
  kubectl create secret generic infisical-secrets \
    --namespace "${NAMESPACE}" \
    --from-literal="AUTH_SECRET=$(openssl rand -base64 32)" \
    --from-literal="ENCRYPTION_KEY=$(openssl rand -hex 16)" \
    --from-literal="DB_CONNECTION_URI=${db_connection_uri}" \
    --from-literal='REDIS_URL=redis://redis-master:6379' \
    --from-literal='SITE_URL=https://infisical.local.dev' \
    --from-literal='HOST=0.0.0.0' \
    --from-literal='PORT=8080' \
    --from-literal='INVITE_ONLY_SIGNUP=false' \
    --from-literal='TELEMETRY_ENABLED=false' \
    --dry-run=client -o yaml | kubectl apply -f -
  printf '%s\n' 'Created Infisical backend bootstrap credentials.'
fi

printf '%s\n' 'Bootstrap secrets are present. Import their values into Infisical after first login.'
