#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../.." && pwd)"
action="${1:-}"
case "${action}" in
  validate|images|apply) ;;
  *) printf 'Usage: %s validate|images|apply\n' "${0}" >&2; exit 2 ;;
esac

for tool in helm kubectl jq git; do
  command -v "${tool}" >/dev/null 2>&1 || {
    printf 'Required command missing: %s\n' "${tool}" >&2
    exit 1
  }
done
[[ "$(kubectl config current-context)" == docker-desktop ]] || {
  printf 'Select the docker-desktop Kubernetes context first.\n' >&2
  exit 1
}

nodes="$(kubectl get nodes -o json)"
node_count="$(jq '.items | length' <<<"${nodes}")"
ready_count="$(jq '[.items[] | select(any(.status.conditions[]; .type == "Ready" and .status == "True"))] | length' <<<"${nodes}")"
[[ "${node_count}" == 1 && "${ready_count}" == 1 ]] || {
  printf 'The localhost registry profile requires exactly one Ready Docker Desktop node.\n' >&2
  exit 1
}
architecture="$(jq -r '.items[0].status.nodeInfo.architecture' <<<"${nodes}")"
case "${architecture}" in
  arm64|amd64) ;;
  *) printf 'Unsupported node architecture: %s\n' "${architecture}" >&2; exit 1 ;;
esac

tag="${KUBEDECK_IMAGE_TAG:-dev-$(git -C "${repo_root}" rev-parse --short HEAD)}"
[[ "${tag}" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]*$ ]] || {
  printf 'Invalid image tag.\n' >&2
  exit 1
}
registry=localhost:5001
agent_image="${registry}/homelab/dev/kubedeck-agent:${tag}"
web_image="${registry}/homelab/dev/kubedeck:${tag}"

source "${script_dir}/lib/docker-desktop-scope.sh"
scope="$(resolve_docker_desktop_scope)"
IFS=$'\t' read -r project_slug environment <<<"${scope}"
scope_args=(--set-string "infisical.projectSlug=${project_slug}" --set-string "infisical.envSlug=${environment}")
agent_values=(-f "${repo_root}/lab/apps/dev/kubedeck-agent/values.yaml" -f "${repo_root}/lab/apps/dev/kubedeck-agent/values.docker-desktop.yaml" --set-string "image.tag=${tag}" "${scope_args[@]}")
web_values=(-f "${repo_root}/lab/apps/dev/kubedeck/values.yaml" -f "${repo_root}/lab/apps/dev/kubedeck/values.docker-desktop.yaml" --set-string "image.tag=${tag}" "${scope_args[@]}")

if [[ "${action}" == validate ]]; then
  helm lint "${repo_root}/lab/apps/dev/kubedeck-agent" "${agent_values[@]}"
  helm lint "${repo_root}/lab/apps/dev/kubedeck" "${web_values[@]}"
  helm template kubedeck-agent "${repo_root}/lab/apps/dev/kubedeck-agent" -n development-tools "${agent_values[@]}" >/dev/null
  helm template kubedeck "${repo_root}/lab/apps/dev/kubedeck" -n development-tools "${web_values[@]}" >/dev/null
  kubectl apply --dry-run=server -f "${repo_root}/lab/apps/platform/local-registry/registry.yaml" >/dev/null
  printf 'KubeDeck Docker Desktop charts rendered for %s / %s, image tag %s.\n' "${project_slug}" "${environment}" "${tag}"
  exit 0
fi

for tool in crane curl; do
  command -v "${tool}" >/dev/null 2>&1 || {
    printf 'Required command missing: %s\n' "${tool}" >&2
    exit 1
  }
done
if [[ "${KUBEDECK_SKIP_BUILD:-0}" != 1 ]]; then
  command -v docker >/dev/null 2>&1 || {
    printf 'Required command missing: docker\n' >&2
    exit 1
  }
fi

kubectl apply -f "${repo_root}/lab/apps/platform/local-registry/registry.yaml"
kubectl rollout status deployment/kubedeck-local-registry -n platform-system --timeout=120s
[[ "$(kubectl get pvc kubedeck-local-registry -n platform-system -o jsonpath='{.status.phase}')" == Bound ]] || {
  printf 'Local registry PVC is not Bound.\n' >&2
  exit 1
}

temporary_dir="$(mktemp -d "${TMPDIR:-/tmp}/kubedeck-images.XXXXXX")"
forward_pid=""
cleanup() {
  if [[ -n "${forward_pid}" ]]; then
    kill "${forward_pid}" 2>/dev/null || true
    wait "${forward_pid}" 2>/dev/null || true
  fi
  rm -rf "${temporary_dir}"
}
trap cleanup EXIT
kubectl port-forward -n platform-system deployment/kubedeck-local-registry 5001:5001 --address 127.0.0.1 \
  >"${temporary_dir}/port-forward.log" 2>&1 &
forward_pid=$!
for _ in {1..30}; do
  if ! kill -0 "${forward_pid}" 2>/dev/null; then
    printf 'Local registry port-forward could not start; check port 5001.\n' >&2
    exit 1
  fi
  if curl -fsS --max-time 1 http://127.0.0.1:5001/v2/ >/dev/null 2>&1; then break; fi
  sleep 1
done
curl -fsS --max-time 2 http://127.0.0.1:5001/v2/ >/dev/null

if [[ "${KUBEDECK_SKIP_BUILD:-0}" != 1 ]]; then
  docker build --platform "linux/${architecture}" -t "${agent_image}" "${repo_root}/kubedeck-agent"
  docker image save --platform "linux/${architecture}" -o "${temporary_dir}/agent.tar" "${agent_image}"
  crane --insecure push "${temporary_dir}/agent.tar" "${agent_image}"
  rm "${temporary_dir}/agent.tar"

  docker build --platform "linux/${architecture}" -t "${web_image}" "${repo_root}"
  docker image save --platform "linux/${architecture}" -o "${temporary_dir}/web.tar" "${web_image}"
  crane --insecure push "${temporary_dir}/web.tar" "${web_image}"
  rm "${temporary_dir}/web.tar"
fi

crane --insecure digest "${agent_image}" >/dev/null
crane --insecure digest "${web_image}" >/dev/null
printf 'Local registry contains KubeDeck images tagged %s.\n' "${tag}"
[[ "${action}" == apply ]] || exit 0

helm upgrade --install kubedeck-agent "${repo_root}/lab/apps/dev/kubedeck-agent" \
  -n development-tools --create-namespace "${agent_values[@]}" --wait --timeout 10m
helm upgrade --install kubedeck "${repo_root}/lab/apps/dev/kubedeck" \
  -n development-tools --create-namespace "${web_values[@]}" --wait --timeout 10m
