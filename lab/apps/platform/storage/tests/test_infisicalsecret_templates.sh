#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
lab_root="$(cd "${script_dir}/../../../../" && pwd)"
chart="${lab_root}/apps/platform/storage"

for tool in helm yq; do
  command -v "${tool}" >/dev/null 2>&1 || {
    printf 'Required command missing: %s\n' "${tool}" >&2
    exit 1
  }
done

rendered="$(helm template platform-storage "${chart}" -n platform-storage \
  -f "${chart}/values.yaml" \
  -f "${chart}/values.docker-desktop.yaml" \
  -f "${chart}/values.docker-desktop-core.yaml" \
  --set-string infisical.projectSlug=test-project \
  --set-string infisical.envSlug=dev)"

nats_monitoring="$(yq 'select(.kind == "InfisicalSecret" and .metadata.name == "infisical-nats-monitoring")' <<<"${rendered}")"
nats="$(yq 'select(.kind == "InfisicalSecret" and .metadata.name == "infisical-nats")' <<<"${rendered}")"

[[ "$(yq '.spec.managedSecretReference.template.includeAllSecrets' <<<"${nats_monitoring}")" == false ]]
[[ "$(yq '.spec.managedSecretReference.template.data | keys | join(",")' <<<"${nats_monitoring}")" == users ]]
[[ "$(yq '.spec.managedSecretReference.template.includeAllSecrets' <<<"${nats}")" == true ]]

printf '%s\n' 'PASS: templated Infisical Secrets contain only rendered data; standard Secrets sync all fields.'
