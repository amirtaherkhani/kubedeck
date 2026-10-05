#!/usr/bin/env bash

set -euo pipefail

readonly NAMESPACE="${NATS_NAMESPACE:-platform-storage}"
readonly EXPECTED_CONTEXT="${KUBE_CONTEXT:-rancher-desktop}"
readonly EXPECTED_REPLICAS="${NATS_REPLICAS:-3}"
readonly NATS_BOX_IMAGE="${NATS_BOX_IMAGE:-natsio/nats-box:0.19.7-nonroot}"
readonly CLIENT_POD="nats-ha-test"
readonly STREAM="CODEX_HA_SMOKE"
readonly CONSUMER="WORKERS"
readonly CORE_SUBJECT="codex.ha.smoke.core"
readonly JS_SUBJECT="codex.ha.smoke.js"

die() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

extract_result() {
  local key="$1"
  local result="$2"

  awk -F= -v key="${key}" '$1 == key { print $2 }' <<<"${result}"
}

cleanup() {
  if kubectl get pod "${CLIENT_POD}" --namespace "${NAMESPACE}" >/dev/null 2>&1; then
    kubectl exec --namespace "${NAMESPACE}" "${CLIENT_POD}" -- \
      nats stream rm "${STREAM}" --force >/dev/null 2>&1 || true
  fi

  kubectl delete pod "${CLIENT_POD}" --namespace "${NAMESPACE}" \
    --ignore-not-found --grace-period=0 --force >/dev/null 2>&1 || true
}

require_command kubectl
require_command jq

current_context="$(kubectl config current-context)"
[[ "${current_context}" == "${EXPECTED_CONTEXT}" ]] || die \
  "current Kubernetes context is ${current_context}; expected ${EXPECTED_CONTEXT}"

trap cleanup EXIT INT TERM
cleanup

kubectl wait --namespace "${NAMESPACE}" \
  --for=condition=Ready pod \
  --selector 'app.kubernetes.io/name=nats,app.kubernetes.io/instance=platform-storage' \
  --timeout=180s >/dev/null

ready_replicas="$(
  kubectl get pods --namespace "${NAMESPACE}" \
    --selector 'app.kubernetes.io/name=nats,app.kubernetes.io/instance=platform-storage' \
    --output json |
    jq '[.items[] | select(.status.containerStatuses[0].ready == true)] | length'
)"
[[ "${ready_replicas}" -eq "${EXPECTED_REPLICAS}" ]] || die \
  "expected ${EXPECTED_REPLICAS} ready NATS pods; found ${ready_replicas}"

expected_peers=$((EXPECTED_REPLICAS - 1))
while IFS= read -r pod; do
  route_data="$(
    kubectl exec --namespace "${NAMESPACE}" "${pod}" -- \
      wget -qO- http://127.0.0.1:8222/routez
  )"
  peer_count="$(jq '[.routes[]?.remote_name] | unique | length' <<<"${route_data}")"
  [[ "${peer_count}" -eq "${expected_peers}" ]] || die \
    "${pod} has ${peer_count} unique route peers; expected ${expected_peers}"
done < <(
  kubectl get pods --namespace "${NAMESPACE}" \
    --selector 'app.kubernetes.io/name=nats,app.kubernetes.io/instance=platform-storage' \
    --output jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
)

pod_overrides="$(
  jq -nc \
    --arg image "${NATS_BOX_IMAGE}" \
    --arg nats_url "nats://nats.${NAMESPACE}.svc.cluster.local:4222" \
    '{
      apiVersion: "v1",
      spec: {
        automountServiceAccountToken: false,
        enableServiceLinks: false,
        terminationGracePeriodSeconds: 0,
        securityContext: {
          runAsNonRoot: true,
          seccompProfile: {type: "RuntimeDefault"}
        },
        containers: [{
          name: "nats-ha-test",
          image: $image,
          imagePullPolicy: "IfNotPresent",
          command: ["sleep", "3600"],
          env: [
            {name: "NATS_URL", value: $nats_url},
            {
              name: "NATS_USER",
              valueFrom: {
                secretKeyRef: {name: "nats", key: "NATS_USERNAME"}
              }
            },
            {
              name: "NATS_PASSWORD",
              valueFrom: {
                secretKeyRef: {name: "nats", key: "NATS_PASSWORD"}
              }
            }
          ],
          securityContext: {
            allowPrivilegeEscalation: false,
            capabilities: {drop: ["ALL"]}
          }
        }]
      }
    }'
)"

kubectl run "${CLIENT_POD}" --namespace "${NAMESPACE}" \
  --image "${NATS_BOX_IMAGE}" --restart=Never \
  --overrides "${pod_overrides}" >/dev/null
kubectl wait --namespace "${NAMESPACE}" \
  --for=condition=Ready "pod/${CLIENT_POD}" --timeout=240s >/dev/null

core_result="$(
  kubectl exec --namespace "${NAMESPACE}" "${CLIENT_POD}" -- \
    env CORE_SUBJECT="${CORE_SUBJECT}" sh -ec '
      rm -f /tmp/core-worker-*.out
      nats sub "${CORE_SUBJECT}" --queue codex-smoke-workers --raw --wait=2s > /tmp/core-worker-1.out &
      worker_1=$!
      nats sub "${CORE_SUBJECT}" --queue codex-smoke-workers --raw --wait=2s > /tmp/core-worker-2.out &
      worker_2=$!
      nats sub "${CORE_SUBJECT}" --queue codex-smoke-workers --raw --wait=2s > /tmp/core-worker-3.out &
      worker_3=$!
      sleep 1
      nats pub "${CORE_SUBJECT}" --count=300 "core-{{Count}}" --quiet >/dev/null 2>&1
      wait "${worker_1}"
      wait "${worker_2}"
      wait "${worker_3}"
      printf "worker_1="; wc -l < /tmp/core-worker-1.out
      printf "worker_2="; wc -l < /tmp/core-worker-2.out
      printf "worker_3="; wc -l < /tmp/core-worker-3.out
      printf "total="; cat /tmp/core-worker-*.out | wc -l
      printf "unique="; cat /tmp/core-worker-*.out | sort -u | wc -l
    '
)"

core_total="$(extract_result total "${core_result}")"
core_unique="$(extract_result unique "${core_result}")"
[[ "${core_total}" -eq 300 && "${core_unique}" -eq 300 ]] || die \
  "Core queue test delivered total=${core_total}, unique=${core_unique}; expected 300/300"

kubectl exec --namespace "${NAMESPACE}" "${CLIENT_POD}" -- \
  nats stream rm "${STREAM}" --force >/dev/null 2>&1 || true
kubectl exec --namespace "${NAMESPACE}" "${CLIENT_POD}" -- \
  nats stream add "${STREAM}" \
  --subjects "${JS_SUBJECT}" \
  --storage file \
  --replicas "${EXPECTED_REPLICAS}" \
  --retention work \
  --max-age 10m \
  --max-msgs 1000 \
  --discard old \
  --defaults >/dev/null
kubectl exec --namespace "${NAMESPACE}" "${CLIENT_POD}" -- \
  nats consumer add "${STREAM}" "${CONSUMER}" \
  --pull \
  --ack explicit \
  --deliver all \
  --filter "${JS_SUBJECT}" \
  --max-pending 300 \
  --max-deliver 5 \
  --wait 30s \
  --replicas "${EXPECTED_REPLICAS}" \
  --defaults >/dev/null

stream_followers=0
consumer_followers=0
for _ in $(seq 1 30); do
  stream_info="$(
    kubectl exec --namespace "${NAMESPACE}" "${CLIENT_POD}" -- \
      nats stream info "${STREAM}" --json
  )"
  consumer_info="$(
    kubectl exec --namespace "${NAMESPACE}" "${CLIENT_POD}" -- \
      nats consumer info "${STREAM}" "${CONSUMER}" --json
  )"
  stream_followers="$(
    jq '[.cluster.replicas[]? | select(.current == true)] | length' <<<"${stream_info}"
  )"
  consumer_followers="$(
    jq '[.cluster.replicas[]? | select(.current == true)] | length' <<<"${consumer_info}"
  )"
  if [[ "${stream_followers}" -eq "${expected_peers}" &&
        "${consumer_followers}" -eq "${expected_peers}" ]]; then
    break
  fi
  sleep 1
done

[[ "${stream_followers}" -eq "${expected_peers}" ]] || die \
  "JetStream stream has ${stream_followers} current followers; expected ${expected_peers}"
[[ "${consumer_followers}" -eq "${expected_peers}" ]] || die \
  "JetStream consumer has ${consumer_followers} current followers; expected ${expected_peers}"

js_result="$(
  kubectl exec --namespace "${NAMESPACE}" "${CLIENT_POD}" -- \
    env STREAM="${STREAM}" CONSUMER="${CONSUMER}" JS_SUBJECT="${JS_SUBJECT}" sh -ec '
      rm -f /tmp/js-worker-*.out
      nats pub "${JS_SUBJECT}" --jetstream --count=300 "js-{{Count}}" --quiet >/dev/null 2>&1
      nats consumer next "${STREAM}" "${CONSUMER}" --count=100 --ack --raw > /tmp/js-worker-1.out &
      worker_1=$!
      nats consumer next "${STREAM}" "${CONSUMER}" --count=100 --ack --raw > /tmp/js-worker-2.out &
      worker_2=$!
      nats consumer next "${STREAM}" "${CONSUMER}" --count=100 --ack --raw > /tmp/js-worker-3.out &
      worker_3=$!
      wait "${worker_1}"
      wait "${worker_2}"
      wait "${worker_3}"
      printf "worker_1="; wc -l < /tmp/js-worker-1.out
      printf "worker_2="; wc -l < /tmp/js-worker-2.out
      printf "worker_3="; wc -l < /tmp/js-worker-3.out
      printf "total="; cat /tmp/js-worker-*.out | wc -l
      printf "unique="; cat /tmp/js-worker-*.out | sort -u | wc -l
    '
)"

js_total="$(extract_result total "${js_result}")"
js_unique="$(extract_result unique "${js_result}")"
[[ "${js_total}" -eq 300 && "${js_unique}" -eq 300 ]] || die \
  "JetStream test delivered total=${js_total}, unique=${js_unique}; expected 300/300"

consumer_info="$(
  kubectl exec --namespace "${NAMESPACE}" "${CLIENT_POD}" -- \
    nats consumer info "${STREAM}" "${CONSUMER}" --json
)"
ack_pending="$(jq '.num_ack_pending' <<<"${consumer_info}")"
redelivered="$(jq '.num_redelivered' <<<"${consumer_info}")"
pending="$(jq '.num_pending' <<<"${consumer_info}")"
[[ "${ack_pending}" -eq 0 && "${redelivered}" -eq 0 && "${pending}" -eq 0 ]] || die \
  "JetStream consumer state ack_pending=${ack_pending}, redelivered=${redelivered}, pending=${pending}"

printf 'NATS HA smoke test passed.\n'
printf '  ready servers: %s\n' "${ready_replicas}"
printf '  Core queue workers: %s / %s / %s; total=%s; unique=%s\n' \
  "$(extract_result worker_1 "${core_result}")" \
  "$(extract_result worker_2 "${core_result}")" \
  "$(extract_result worker_3 "${core_result}")" \
  "${core_total}" "${core_unique}"
printf '  JetStream workers: %s / %s / %s; total=%s; unique=%s\n' \
  "$(extract_result worker_1 "${js_result}")" \
  "$(extract_result worker_2 "${js_result}")" \
  "$(extract_result worker_3 "${js_result}")" \
  "${js_total}" "${js_unique}"
printf '  JetStream state: ack_pending=%s; redelivered=%s; pending=%s\n' \
  "${ack_pending}" "${redelivered}" "${pending}"
