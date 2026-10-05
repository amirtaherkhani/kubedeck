#!/usr/bin/env bash

set -euo pipefail

readonly NAMESPACE="platform-storage"
readonly STATEFULSET="nats"
readonly EXPECTED_CONTEXT="${KUBE_CONTEXT:-rancher-desktop}"
readonly CURRENT_CONTEXT="$(kubectl config current-context)"
readonly TARGET_SERVICE_NAME="nats-headless"

if [[ "${CURRENT_CONTEXT}" != "${EXPECTED_CONTEXT}" ]]; then
  echo "Current Kubernetes context is ${CURRENT_CONTEXT}; expected ${EXPECTED_CONTEXT}." >&2
  exit 1
fi

current_service_name="$(
  kubectl get statefulset "${STATEFULSET}" --namespace "${NAMESPACE}" \
    --output jsonpath='{.spec.serviceName}' 2>/dev/null || true
)"

if [[ -n "${current_service_name}" && "${current_service_name}" != "nats" && "${current_service_name}" != "${TARGET_SERVICE_NAME}" ]]; then
  echo "Unexpected NATS governing Service: ${current_service_name}" >&2
  exit 1
fi

if [[ "${current_service_name}" == "${TARGET_SERVICE_NAME}" ]]; then
  echo "NATS StatefulSet already uses ${TARGET_SERVICE_NAME}; applying the chart normally."
  ./scripts/platform-helm.sh apply platform-storage
  exit 0
fi

nats_pod="$(
  kubectl get pod --namespace "${NAMESPACE}" \
    --selector 'app.kubernetes.io/name=nats,app.kubernetes.io/instance=platform-storage' \
    --output jsonpath='{.items[0].metadata.name}' 2>/dev/null || true
)"

if [[ -n "${nats_pod}" ]]; then
  varz="$(kubectl exec --namespace "${NAMESPACE}" "${nats_pod}" -- wget -qO- http://127.0.0.1:8222/varz)"
  jsz="$(kubectl exec --namespace "${NAMESPACE}" "${nats_pod}" -- wget -qO- http://127.0.0.1:8222/jsz)"
  connections="$(jq -r '.connections // 0' <<<"${varz}")"
  streams="$(jq -r '.streams // 0' <<<"${jsz}")"

  if [[ "${connections}" -ne 0 || "${streams}" -ne 0 ]]; then
    echo "Refusing NATS HA migration with active state." >&2
    echo "Connections: ${connections}; JetStream streams: ${streams}" >&2
    exit 1
  fi
fi

if [[ -n "${current_service_name}" ]]; then
  echo "Scaling only ${NAMESPACE}/${STATEFULSET} to zero for the immutable Service migration."
  kubectl scale statefulset "${STATEFULSET}" --namespace "${NAMESPACE}" --replicas=0
  kubectl wait --namespace "${NAMESPACE}" \
    --for=delete pod \
    --selector 'app.kubernetes.io/name=nats,app.kubernetes.io/instance=platform-storage' \
    --timeout=180s
  kubectl delete statefulset "${STATEFULSET}" --namespace "${NAMESPACE}" \
    --cascade=orphan --wait=true
fi

./scripts/platform-helm.sh apply platform-storage
