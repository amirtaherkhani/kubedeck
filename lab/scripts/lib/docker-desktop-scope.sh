#!/usr/bin/env bash

resolve_docker_desktop_scope() {
  local environment="${INFISICAL_ENV_SLUG:-dev}"
  local project_slug="${INFISICAL_PROJECT_SLUG:-}"
  local helper_dir token projects project_name project_id project

  if [[ -z "${project_slug}" ]]; then
    helper_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    # The existing helper authenticates through this Mac's Keychain-backed
    # Infisical administrator entry; neither token nor values are printed.
    source "${helper_dir}/../local-dev-env-bootstrap.sh"
    IFS=$'\t' read -r token _ <<<"$(admin_token)"
    projects="$(admin_api "${token}" GET /projects)"
    project_name="${INFISICAL_HOME_PROJECT_NAME:-home-lab}"
    project_slug="$(jq -er --arg name "${project_name}" \
      '[.projects[] | select(.name == $name)] | if length == 1 then .[0].slug else error("expected exactly one project") end' \
      <<<"${projects}")"
    project_id="$(jq -er --arg name "${project_name}" \
      '.projects[] | select(.name == $name) | .id' <<<"${projects}")"
    project="$(admin_api "${token}" GET "/projects/${project_id}")"
    jq -e --arg env "${environment}" \
      '.project.environments[] | select(.slug == $env)' <<<"${project}" >/dev/null || {
        printf 'Infisical environment %s does not exist in %s.\n' "${environment}" "${project_name}" >&2
        return 1
      }
    unset token projects project project_id
  fi

  printf '%s\t%s\n' "${project_slug}" "${environment}"
}
