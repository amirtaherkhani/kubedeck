#!/usr/bin/env bash

set -euo pipefail

readonly NAMESPACE="${MONITORING_NAMESPACE:-observability}"
readonly CONFIGMAP="monitoring-kube-prometheus-nodes-darwin"
readonly DASHBOARD_KEY="nodes-darwin.json"
readonly MODE="${1:-apply}"

if [[ "${MODE}" != "apply" && "${MODE}" != "check" ]]; then
  echo "Usage: $0 [apply|check]" >&2
  exit 1
fi

for command_name in jq kubectl; do
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "Required command not found: ${command_name}" >&2
    exit 1
  fi
done

dashboard="$(
  kubectl get configmap "${CONFIGMAP}" --namespace "${NAMESPACE}" --output json \
    | jq --raw-output --arg key "${DASHBOARD_KEY}" '.data[$key] // empty'
)"

if [[ -z "${dashboard}" ]]; then
  echo "Dashboard ${DASHBOARD_KEY} was not found in ${NAMESPACE}/${CONFIGMAP}." >&2
  exit 1
fi

patched_dashboard="$(
  jq '
    walk(
      if type == "object" and (.expr? | type == "string") then
        .expr |= (
          gsub("\\(mmcblk"; "(disk.+|mmcblk")
          | gsub("device!=\"lo\""; "device!~\"lo.*\"")
        )
      else
        .
      end
    )
  ' <<<"${dashboard}"
)"

disk_query_count="$(
  jq '[.. | objects | .expr? | strings | select(contains("disk.+|mmcblk"))] | length' \
    <<<"${patched_dashboard}"
)"
network_query_count="$(
  jq '[.. | objects | .expr? | strings | select(contains("device!~\"lo.*\""))] | length' \
    <<<"${patched_dashboard}"
)"

if [[ "${disk_query_count}" -ne 3 || "${network_query_count}" -ne 2 ]]; then
  echo "Unexpected Darwin dashboard query layout; refusing to apply a partial patch." >&2
  echo "Disk queries: ${disk_query_count}; network queries: ${network_query_count}" >&2
  exit 1
fi

patch_payload="$(
  jq --null-input \
    --arg key "${DASHBOARD_KEY}" \
    --arg dashboard "${patched_dashboard}" \
    '{data: {($key): $dashboard}}'
)"

if [[ "${MODE}" == "check" ]]; then
  kubectl patch configmap "${CONFIGMAP}" --namespace "${NAMESPACE}" \
    --type merge --patch "${patch_payload}" --dry-run=server --output name >/dev/null
  echo "macOS dashboard patch validation passed."
  exit 0
fi

if [[ "${dashboard}" == "${patched_dashboard}" ]]; then
  echo "macOS dashboard queries are already patched."
  exit 0
fi

kubectl patch configmap "${CONFIGMAP}" --namespace "${NAMESPACE}" \
  --type merge --patch "${patch_payload}" --output name
echo "Patched macOS disk and loopback queries."
