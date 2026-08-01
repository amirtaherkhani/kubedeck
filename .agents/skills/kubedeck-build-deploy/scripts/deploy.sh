#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'
umask 077

usage() {
  cat <<'EOF'
Usage:
  deploy.sh [--branch <name>]

Build and deploy both KubeDeck and kubedeck-agent from the latest commit on an
origin branch. The default branch is main.

Environment overrides:
  KUBE_CONTEXT                  Kubernetes context (default: rancher-desktop)
  KUBEDECK_NAMESPACE           Namespace (default: kubedeck)
  KUBEDECK_REGISTRY            Registry (default: localhost:5001)
  KUBEDECK_APP_RELEASE         Dashboard Helm release (default: kubedeck)
  KUBEDECK_AGENT_RELEASE       Agent Helm release (default: kubedeck-agent)
  KUBEDECK_ADMIN_SECRET        Existing dashboard admin Secret
  KUBEDECK_AGENT_AUTH_SECRET   Shared agent token Secret
  KUBEDECK_VALUES_FILE         Optional dashboard Helm values file
  KUBEDECK_AGENT_VALUES_FILE   Optional agent Helm values file
  KUBEDECK_AGENT_URL           In-cluster agent URL
  KUBEDECK_HELM_TIMEOUT        Helm timeout (default: 10m)
  KUBEDECK_TARGET_PLATFORM     OCI platform; auto-detected for one node arch
EOF
}

log() {
  printf '\n==> %s\n' "$*"
}

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "Required command not found: $1"
}

branch="main"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --branch)
      [[ $# -ge 2 ]] || die "--branch requires a value"
      branch="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "Unknown argument: $1"
      ;;
  esac
done

for command_name in git kubectl helm nerdctl rdctl node npm go curl openssl sort sed grep tr cut mktemp; do
  require_command "$command_name"
done

[[ -n "$branch" ]] || die "Branch name cannot be empty"
[[ "$branch" != -* ]] || die "Branch name cannot start with '-'"
git check-ref-format "refs/heads/${branch}" >/dev/null 2>&1 ||
  die "Invalid Git branch name: $branch"

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" ||
  die "Run this script from inside the KubeDeck Git repository"
cd "$repo_root"

for required_path in \
  Dockerfile \
  package.json \
  package-lock.json \
  kubedeck-agent/Dockerfile \
  kubedeck-agent/go.mod \
  charts/kubedeck/Chart.yaml \
  charts/kubedeck-agent/Chart.yaml; do
  [[ -f "$required_path" ]] || die "Missing required project file: $required_path"
done

context="${KUBE_CONTEXT:-rancher-desktop}"
namespace="${KUBEDECK_NAMESPACE:-kubedeck}"
registry="${KUBEDECK_REGISTRY:-localhost:5001}"
app_release="${KUBEDECK_APP_RELEASE:-kubedeck}"
agent_release="${KUBEDECK_AGENT_RELEASE:-kubedeck-agent}"
admin_secret="${KUBEDECK_ADMIN_SECRET:-kubedeck-admin}"
agent_auth_secret="${KUBEDECK_AGENT_AUTH_SECRET:-kubedeck-agent-auth}"
helm_timeout="${KUBEDECK_HELM_TIMEOUT:-10m}"
target_platform="${KUBEDECK_TARGET_PLATFORM:-}"
app_values_file="${KUBEDECK_VALUES_FILE:-}"
agent_values_file="${KUBEDECK_AGENT_VALUES_FILE:-}"
agent_url="${KUBEDECK_AGENT_URL:-http://${agent_release}:8080}"

current_context="$(kubectl config current-context)"
[[ "$current_context" == "$context" ]] ||
  die "Current Kubernetes context is '$current_context', expected '$context'. Set KUBE_CONTEXT only after the target context is explicitly approved."

kubectl --context "$context" cluster-info >/dev/null

if [[ -n "$(git status --porcelain=v1 --untracked-files=all)" ]]; then
  git status --short
  die "The checkout is dirty. Commit or remove the changes before deploying; this script will not stash or discard them."
fi

log "Updating origin/$branch"
git fetch --prune origin "+refs/heads/${branch}:refs/remotes/origin/${branch}" ||
  die "Could not fetch origin/$branch; check the branch name and remote access"
git show-ref --verify --quiet "refs/remotes/origin/${branch}" ||
  die "Remote branch origin/$branch does not exist"

if git show-ref --verify --quiet "refs/heads/${branch}"; then
  git switch "$branch"
else
  git switch --track -c "$branch" "origin/$branch"
fi

git pull --ff-only origin "$branch"

commit_sha="$(git rev-parse HEAD)"
remote_sha="$(git rev-parse "refs/remotes/origin/${branch}")"
[[ "$commit_sha" == "$remote_sha" ]] ||
  die "Local HEAD ($commit_sha) does not exactly match origin/$branch ($remote_sha)"
[[ -z "$(git status --porcelain=v1 --untracked-files=all)" ]] ||
  die "The checkout changed during branch preparation"

release_version="$(node -p "require('./package.json').version")"
[[ "$release_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)?$ ]] ||
  die "package.json version is not a supported release version: $release_version"

app_chart_version="$(sed -n 's/^version:[[:space:]]*//p' charts/kubedeck/Chart.yaml)"
app_chart_app_version="$(sed -n 's/^appVersion:[[:space:]]*//p' charts/kubedeck/Chart.yaml | tr -d '"')"
agent_chart_version="$(sed -n 's/^version:[[:space:]]*//p' charts/kubedeck-agent/Chart.yaml)"
agent_chart_app_version="$(sed -n 's/^appVersion:[[:space:]]*//p' charts/kubedeck-agent/Chart.yaml | tr -d '"')"
for chart_version in \
  "$app_chart_version" \
  "$app_chart_app_version" \
  "$agent_chart_version" \
  "$agent_chart_app_version"; do
  [[ "$chart_version" == "$release_version" ]] ||
    die "package.json and Helm chart versions must all match; expected $release_version, found $chart_version"
done

short_sha="$(git rev-parse --short=12 HEAD)"
branch_slug="$(
  printf '%s' "$branch" |
    tr '[:upper:]' '[:lower:]' |
    sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//' |
    cut -c1-48
)"
[[ -n "$branch_slug" ]] || branch_slug="branch"
timestamp="$(date -u +%Y%m%d%H%M%S)"
image_tag="${branch_slug}-${short_sha}-${timestamp}"
app_repository="${KUBEDECK_APP_REPOSITORY:-${registry%/}/kubedeck}"
agent_repository="${KUBEDECK_AGENT_REPOSITORY:-${registry%/}/kubedeck-agent}"
app_image="${app_repository}:${image_tag}"
agent_image="${agent_repository}:${image_tag}"

if [[ -n "$app_values_file" ]]; then
  [[ -f "$app_values_file" ]] ||
    die "Dashboard values file does not exist: $app_values_file"
  app_values_file="$(cd "$(dirname "$app_values_file")" && pwd)/$(basename "$app_values_file")"
fi
if [[ -n "$agent_values_file" ]]; then
  [[ -f "$agent_values_file" ]] ||
    die "Agent values file does not exist: $agent_values_file"
  agent_values_file="$(cd "$(dirname "$agent_values_file")" && pwd)/$(basename "$agent_values_file")"
fi

log "Validating dashboard at $commit_sha"
npm ci
npm run lint
npm test

log "Validating kubedeck-agent at $commit_sha"
(
  cd kubedeck-agent
  go mod download
  go test -race ./...
  go vet ./...
)

log "Linting and rendering Helm charts"
helm lint charts/kubedeck \
  --set-string image.tag=validation
helm template "$app_release" charts/kubedeck \
  --namespace "$namespace" \
  --set-string image.tag=validation >/dev/null
helm lint charts/kubedeck-agent \
  --set-string image.tag=validation
helm template "$agent_release" charts/kubedeck-agent \
  --namespace "$namespace" \
  --set-string image.tag=validation >/dev/null

curl -fsS --max-time 5 "http://${registry%/}/v2/" >/dev/null ||
  die "Registry is not reachable at http://${registry%/}/v2/"

if [[ -z "$target_platform" ]]; then
  node_arches="$(
    kubectl --context "$context" get nodes \
      -o jsonpath='{range .items[*]}{.metadata.labels.kubernetes\.io/arch}{"\n"}{end}' |
      sort -u |
      sed '/^$/d'
  )"
  arch_count="$(printf '%s\n' "$node_arches" | grep -c . || true)"
  [[ "$arch_count" -eq 1 ]] ||
    die "Could not select one cluster architecture. Set KUBEDECK_TARGET_PLATFORM explicitly."
  case "$node_arches" in
    amd64|arm64)
      target_platform="linux/${node_arches}"
      ;;
    *)
      die "Unsupported cluster node architecture: $node_arches"
      ;;
  esac
fi

build_git_tree() {
  local treeish="$1"
  local image="$2"

  git archive --format=tar "$treeish" |
    (
      cd /tmp
      rdctl shell sh -c '
        set -eu
        platform="$1"
        image="$2"
        revision="$3"
        version="$4"
        build_dir="$(mktemp -d /tmp/kubedeck-build.XXXXXX)"

        cleanup() {
          case "$build_dir" in
            /tmp/kubedeck-build.*)
              rm -r -- "$build_dir"
              ;;
          esac
        }
        trap cleanup EXIT HUP INT TERM

        tar -xf - -C "$build_dir"
        sudo nerdctl build \
          --platform "$platform" \
          --label "org.opencontainers.image.revision=${revision}" \
          --label "org.opencontainers.image.version=${version}" \
          --tag "$image" \
          "$build_dir"
      ' sh "$target_platform" "$image" "$commit_sha" "$release_version"
    )
}

log "Building $app_image for $target_platform"
build_git_tree "$commit_sha" "$app_image"

log "Building $agent_image for $target_platform"
build_git_tree "${commit_sha}:kubedeck-agent" "$agent_image"

log "Pushing immutable images"
(
  cd /tmp
  nerdctl push "$app_image"
  nerdctl push "$agent_image"
)

log "Preparing namespace and Secret references"
kubectl --context "$context" create namespace "$namespace" \
  --dry-run=client \
  -o yaml |
  kubectl --context "$context" apply -f -

kubectl --context "$context" --namespace "$namespace" get secret "$admin_secret" >/dev/null 2>&1 ||
  die "Required dashboard admin Secret '$namespace/$admin_secret' does not exist"

admin_keys="$(
  kubectl --context "$context" --namespace "$namespace" get secret "$admin_secret" \
    -o go-template='{{range $key, $value := .data}}{{$key}}{{"\n"}}{{end}}'
)"
for admin_key in \
  KUBEDECK_ADMIN_FIRST_NAME \
  KUBEDECK_ADMIN_LAST_NAME \
  KUBEDECK_ADMIN_EMAIL \
  KUBEDECK_ADMIN_PASSWORD; do
  printf '%s\n' "$admin_keys" | grep -Fxq "$admin_key" ||
    die "Admin Secret '$namespace/$admin_secret' is missing key '$admin_key'"
done

agent_secret_action="reused"
if kubectl --context "$context" --namespace "$namespace" get secret "$agent_auth_secret" >/dev/null 2>&1; then
  agent_keys="$(
    kubectl --context "$context" --namespace "$namespace" get secret "$agent_auth_secret" \
      -o go-template='{{range $key, $value := .data}}{{$key}}{{"\n"}}{{end}}'
  )"
  printf '%s\n' "$agent_keys" | grep -Fxq token ||
    die "Agent auth Secret '$namespace/$agent_auth_secret' is missing key 'token'"
else
  agent_token="$(openssl rand -hex 32)"
  kubectl --context "$context" --namespace "$namespace" create secret generic "$agent_auth_secret" \
    --from-literal="token=${agent_token}" >/dev/null
  unset agent_token
  agent_secret_action="created"
fi

temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/kubedeck-deploy.XXXXXX")"
cleanup() {
  if [[ -n "${temp_dir:-}" && -d "$temp_dir" ]]; then
    rm -r -- "$temp_dir"
  fi
}
trap cleanup EXIT

if helm upgrade --help | grep -q -- '--rollback-on-failure'; then
  helm_failure_flag="--rollback-on-failure"
else
  helm_failure_flag="--atomic"
fi

app_helm_command=(
  --kube-context "$context"
  upgrade
  --install
  "$app_release"
  charts/kubedeck
  --namespace "$namespace"
  --create-namespace
)
agent_helm_command=(
  --kube-context "$context"
  upgrade
  --install
  "$agent_release"
  charts/kubedeck-agent
  --namespace "$namespace"
  --create-namespace
)

# Preserve computed installed values as well as user overrides. This keeps
# immutable resources such as an existing PVC's storageClassName stable when
# chart defaults differ from the originally installed release.
if helm --kube-context "$context" status "$app_release" --namespace "$namespace" >/dev/null 2>&1; then
  helm --kube-context "$context" get values "$app_release" \
    --namespace "$namespace" \
    --all \
    --output yaml >"$temp_dir/app-existing-values.yaml"
  app_helm_command+=(-f "$temp_dir/app-existing-values.yaml")
fi
if helm --kube-context "$context" status "$agent_release" --namespace "$namespace" >/dev/null 2>&1; then
  helm --kube-context "$context" get values "$agent_release" \
    --namespace "$namespace" \
    --all \
    --output yaml >"$temp_dir/agent-existing-values.yaml"
  agent_helm_command+=(-f "$temp_dir/agent-existing-values.yaml")
fi
if [[ -n "$app_values_file" ]]; then
  app_helm_command+=(-f "$app_values_file")
fi
if [[ -n "$agent_values_file" ]]; then
  agent_helm_command+=(-f "$agent_values_file")
fi

log "Deploying $agent_release"
helm "${agent_helm_command[@]}" \
  --set-string "image.repository=${agent_repository}" \
  --set-string "image.tag=${image_tag}" \
  --set-string "auth.existingSecret=${agent_auth_secret}" \
  "$helm_failure_flag" \
  --wait \
  --timeout "$helm_timeout"

log "Deploying $app_release"
helm "${app_helm_command[@]}" \
  --set-string "image.repository=${app_repository}" \
  --set-string "image.tag=${image_tag}" \
  --set-string "admin.existingSecret=${admin_secret}" \
  --set-string "agent.url=${agent_url}" \
  --set-string "agent.existingSecret=${agent_auth_secret}" \
  "$helm_failure_flag" \
  --wait \
  --timeout "$helm_timeout"

verify_release() {
  local release_name="$1"
  local deployments
  local deployment_name
  deployments="$(
    kubectl --context "$context" --namespace "$namespace" get deployment \
      -l "app.kubernetes.io/instance=${release_name}" \
      -o name
  )"
  [[ -n "$deployments" ]] ||
    die "No Deployment found for Helm release $release_name"

  while IFS= read -r deployment_name; do
    [[ -n "$deployment_name" ]] || continue
    kubectl --context "$context" --namespace "$namespace" rollout status \
      "$deployment_name" \
      --timeout="$helm_timeout"
  done <<<"$deployments"

  kubectl --context "$context" --namespace "$namespace" wait \
    --for=condition=Ready pod \
    -l "app.kubernetes.io/instance=${release_name}" \
      --timeout="$helm_timeout"
}

verify_endpoints() {
  local release_name="$1"
  local services
  local service_name
  local endpoint_slices
  services="$(
    kubectl --context "$context" --namespace "$namespace" get service \
      -l "app.kubernetes.io/instance=${release_name}" \
      -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
  )"
  [[ -n "$services" ]] ||
    die "No Service found for Helm release $release_name"

  while IFS= read -r service_name; do
    [[ -n "$service_name" ]] || continue
    endpoint_slices="$(
      kubectl --context "$context" --namespace "$namespace" get endpointslice \
        -l "kubernetes.io/service-name=${service_name}" \
        -o name
    )"
    [[ -n "$endpoint_slices" ]] ||
      die "No EndpointSlice found for Service $namespace/$service_name"
    kubectl --context "$context" --namespace "$namespace" get endpointslice \
      -l "kubernetes.io/service-name=${service_name}" \
      -o wide
  done <<<"$services"
}

verify_dashboard_agent_proxy() {
  log "Verifying authenticated dashboard-to-agent snapshot proxy"
  kubectl --context "$context" --namespace "$namespace" exec -i \
    "deployment/${app_release}" -- node --input-type=module - <<'NODE'
const baseURL = "http://127.0.0.1:3000";
const email = process.env.KUBEDECK_ADMIN_EMAIL;
const password = process.env.KUBEDECK_ADMIN_PASSWORD;

if (!email || !password) {
  throw new Error("dashboard admin credentials are unavailable in the runtime");
}

const login = await fetch(`${baseURL}/api/auth/login`, {
  method: "POST",
  headers: { "content-type": "application/json" },
  body: JSON.stringify({ email, password }),
});
if (!login.ok) {
  throw new Error(`dashboard login failed with HTTP ${login.status}`);
}

const cookie = login.headers.get("set-cookie")?.split(";", 1)[0];
if (!cookie) {
  throw new Error("dashboard login did not return a session cookie");
}

const response = await fetch(`${baseURL}/api/cluster/snapshot`, {
  headers: { accept: "application/json", cookie },
});
if (!response.ok) {
  throw new Error(`dashboard agent proxy failed with HTTP ${response.status}`);
}

const snapshot = await response.json();
if (snapshot.schemaVersion !== "kubedeck.io/v1alpha1") {
  throw new Error(`unexpected snapshot schema: ${snapshot.schemaVersion}`);
}

console.log(JSON.stringify({
  dashboardAgentProxy: "ok",
  clusterId: snapshot.cluster?.id,
  nodes: snapshot.nodes?.length ?? 0,
  services: snapshot.services?.length ?? 0,
  workloads: snapshot.workloads?.length ?? 0,
}));
NODE
}

log "Verifying both releases"
verify_release "$agent_release"
verify_release "$app_release"
verify_endpoints "$agent_release"
verify_endpoints "$app_release"

app_runtime_image="$(
  kubectl --context "$context" --namespace "$namespace" get deployment \
    -l "app.kubernetes.io/instance=${app_release}" \
    -o jsonpath='{.items[0].spec.template.spec.containers[0].image}'
)"
agent_runtime_image="$(
  kubectl --context "$context" --namespace "$namespace" get deployment \
    -l "app.kubernetes.io/instance=${agent_release}" \
    -o jsonpath='{.items[0].spec.template.spec.containers[0].image}'
)"
[[ "$app_runtime_image" == "$app_image" ]] ||
  die "Dashboard runtime image mismatch: expected $app_image, found $app_runtime_image"
[[ "$agent_runtime_image" == "$agent_image" ]] ||
  die "Agent runtime image mismatch: expected $agent_image, found $agent_runtime_image"

verify_dashboard_agent_proxy

log "Recent kubedeck-agent logs"
kubectl --context "$context" --namespace "$namespace" logs \
  -l "app.kubernetes.io/instance=${agent_release}" \
  --all-containers \
  --tail=50

log "Recent KubeDeck logs"
kubectl --context "$context" --namespace "$namespace" logs \
  -l "app.kubernetes.io/instance=${app_release}" \
  --all-containers \
  --tail=50

log "Deployment complete"
printf 'Branch: %s\n' "$branch"
printf 'Commit: %s\n' "$commit_sha"
printf 'Release version: %s\n' "$release_version"
printf 'Context: %s\n' "$context"
printf 'Namespace: %s\n' "$namespace"
printf 'Dashboard image: %s\n' "$app_image"
printf 'Agent image: %s\n' "$agent_image"
printf 'Agent auth Secret: %s (%s)\n' "$agent_auth_secret" "$agent_secret_action"
helm --kube-context "$context" list \
  --namespace "$namespace" \
  --filter "^(${app_release}|${agent_release})$"
kubectl --context "$context" --namespace "$namespace" get pods \
  -l "app.kubernetes.io/instance in (${app_release},${agent_release})" \
  -o wide
