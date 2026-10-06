#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
chart="${script_dir}/../apps/platform/object-storage"
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

kubectl get secret minio -n platform-storage -o json | jq -e \
  '.data | has("MINIO_ACCESS_KEY") and has("MINIO_SECRET_KEY")' >/dev/null || {
    printf 'The platform-storage S3 credential Secret is not ready.\n' >&2
    exit 1
  }

helm lint "${chart}"
if [[ "${action}" == validate ]]; then
  helm template object-storage "${chart}" -n platform-storage | kubectl apply --dry-run=server -f - >/dev/null
  printf 'Docker Desktop object storage chart validated.\n'
else
  helm upgrade --install object-storage "${chart}" \
    -n platform-storage --wait --timeout 10m --history-max 10
  version_status="$(printf 's3.bucket.versioning -name default -enable\ns3.bucket.versioning -name default\nexit\n' |
    kubectl exec -i -n platform-storage object-storage-0 -- weed shell -master=localhost:9333)"
  [[ "${version_status}" == *'Versioning: Enabled'* ]] || {
    printf 'The default S3 bucket did not report versioning Enabled.\n' >&2
    exit 1
  }
  printf 'The default S3 bucket has versioning Enabled.\n'
fi
