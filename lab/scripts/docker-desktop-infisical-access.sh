#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Reuse the existing Keychain-backed Infisical API and identity operations.
# Sourcing defines functions but does not run its legacy bootstrap main.
source "${script_dir}/local-dev-env-bootstrap.sh"

for command in curl git jq kubectl npm python3 security; do require_command "${command}"; done
[[ "$(kubectl config current-context)" == docker-desktop ]] || die "select the docker-desktop Kubernetes context"
if ! command -v kubedeck-env-mcp >/dev/null 2>&1; then
  npm install --global "${script_dir}/../tools/kubedeck-env-mcp" >/dev/null
fi

admin_context="$(admin_token)"
IFS=$'\t' read -r token organization_id <<<"${admin_context}"
projects="$(admin_api "${token}" GET /projects)"

project_field() {
  local project_name="$1" field="$2" count
  count="$(jq --arg name "${project_name}" '[.projects[] | select(.name == $name)] | length' <<<"${projects}")"
  [[ "${count}" == 1 ]] || die "expected exactly one Infisical project named ${project_name}"
  jq -er --arg name "${project_name}" --arg field "${field}" \
    '.projects[] | select(.name == $name) | .[$field]' <<<"${projects}"
}

home_id="$(project_field home-lab id)"
home_slug="$(project_field home-lab slug)"
finance_id="$(project_field vero-finance id)"
for project_id in "${home_id}" "${finance_id}"; do
  project="$(admin_api "${token}" GET "/projects/${project_id}")"
  jq -e '.project.environments[] | select(.slug == "dev")' <<<"${project}" >/dev/null || \
    die "project ${project_id} has no dev environment"
done

search="$(admin_api "${token}" POST /identities/search \
  "$(jq -cn --arg name "${IDENTITY_NAME}" '{limit:100,offset:0,search:{name:{"$eq":$name}}}')")"
identity_id="$(jq -r '.identities[0].identity.id // empty' <<<"${search}")"
if [[ -z "${identity_id}" ]]; then
  identity="$(admin_api "${token}" POST /identities \
    "$(jq -cn --arg name "${IDENTITY_NAME}" --arg org "${organization_id}" \
      '{name:$name,organizationId:$org,role:"no-access",hasDeleteProtection:false,metadata:[{key:"owner",value:"kubedeck"},{key:"scope",value:"approved-development-projects-only"}]}')")"
  identity_id="$(jq -er '.identity.id' <<<"${identity}")"
fi
ensure_membership "${token}" "${home_id}" "${identity_id}"
ensure_membership "${token}" "${finance_id}" "${identity_id}"
ensure_connector_credentials "${token}" "${identity_id}"
unset token admin_context

repo_root="$(git -C "${script_dir}/../.." rev-parse --show-toplevel)"
primary_root="$(git -C "${repo_root}" worktree list --porcelain | awk '/^worktree / {sub(/^worktree /, ""); print; exit}')"
for root in "${primary_root}" "${repo_root}"; do
  kubedeck-env-mcp register --project-id "${home_id}" --environment dev \
    --path /apps/development-tools/kubedeck --root "${root}" >/dev/null
  kubedeck-env-mcp register --project-id "${home_id}" --environment dev \
    --path /apps/development-tools/kubedeck-agent --root "${root}/kubedeck-agent" >/dev/null
done
finance_root="${VERO_FINANCE_PROJECT_DIR:-/Users/mac/Documents/GitHub/verovault-finance}"
if [[ -d "${finance_root}" ]]; then
  kubedeck-env-mcp register --project-id "${finance_id}" --environment dev \
    --path /finance --root "${finance_root}" >/dev/null
fi

# Never pass the client credential in a process argument or write a manifest to disk.
python3 - "${KEYCHAIN_SERVICE}" <<'PY'
import base64, json, subprocess, sys
service = sys.argv[1]
data = {}
for key, account in [('clientId', 'client-id'), ('clientSecret', 'client-secret')]:
    value = subprocess.check_output(['security', 'find-generic-password', '-s', service, '-a', account, '-w']).strip()
    data[key] = base64.b64encode(value).decode()
manifest = {
    'apiVersion': 'v1', 'kind': 'Secret',
    'metadata': {'name': 'infisical-universal-auth', 'namespace': 'platform-secrets',
                 'labels': {'app.kubernetes.io/part-of': 'kubedeck-platform'}},
    'type': 'Opaque', 'data': data,
}
result = subprocess.run(['kubectl', '--context', 'docker-desktop', 'apply', '-f', '-'],
                        input=json.dumps(manifest), capture_output=True, text=True)
if result.returncode:
    raise RuntimeError('unable to apply Kubernetes bootstrap credential; output withheld')
PY

NODE_USE_SYSTEM_CA=1 kubedeck-env-mcp doctor --root "${repo_root}"
printf 'Infisical projectSlug for Helm: %s; environment: dev. Values withheld.\n' "${home_slug}"
