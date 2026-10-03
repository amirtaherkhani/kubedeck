#!/usr/bin/env bash

set -euo pipefail

expected_context="${KUBE_CONTEXT:-rancher-desktop}"
current_context="$(kubectl config current-context)"

if [[ "${current_context}" != "${expected_context}" ]]; then
  printf 'Error: current Kubernetes context is %s; expected %s\n' \
    "${current_context}" "${expected_context}" >&2
  exit 1
fi

# --rm prevents this check from leaving a failed or completed diagnostic Pod.
kubectl -n observability run tempo-smoke \
  --attach \
  --rm \
  --restart=Never \
  --image=curlimages/curl:8.12.1 \
  --command \
  -- sh -ec 'curl --fail --show-error --silent --max-time 10 http://tempo.observability.svc.cluster.local:3200/ready'
