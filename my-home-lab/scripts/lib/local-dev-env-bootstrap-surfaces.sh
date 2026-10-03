#!/usr/bin/env bash

# Project discovery and local agent-surface installation helpers. This file is
# sourced by local-dev-env-bootstrap.sh after its immutable context is defined.

extract_secret_paths() {
  awk '
    $1 == "secretsPath:" && $2 ~ /^\// {print $2}
    {
      for (i = 1; i < NF; i++) {
        if ($i == "\"path\"") {
          value = $(i + 1)
          gsub(/[\")]/, "", value)
          if (value ~ /^\//) print value
        }
      }
    }
  ' "$1" | sort -u
}

registration_path_for_file() {
  local first_path path_count
  first_path="$(extract_secret_paths "$1" | head -n 1)"
  path_count="$(extract_secret_paths "$1" | wc -l | tr -d ' ')"
  if [[ "${path_count}" -gt 1 ]]; then printf '%s\n' "${first_path%/*}"; else printf '%s\n' "${first_path}"; fi
}

register_projects() {
  local token="$1" metadata service_path specific_path service_dir canonical_service_dir finance_dir
  local command=(kubedeck-env-mcp register --project-id "${HOME_LAB_PROJECT_ID}" --environment local --domain "${DOMAIN}")
  kubedeck-env-mcp unregister --root "${PROJECT_ROOT}" >/dev/null
  if [[ "${REPO_ROOT}" != "${PROJECT_ROOT}" ]]; then kubedeck-env-mcp unregister --root "${REPO_ROOT}" >/dev/null; fi
  "${command[@]}" --path / --root "${PROJECT_ROOT}" >/dev/null

  while IFS= read -r metadata; do
    while IFS= read -r specific_path; do
      ensure_folder_path "${token}" "${HOME_LAB_PROJECT_ID}" local "${specific_path}"
    done < <(extract_secret_paths "${metadata}")
    service_path="$(registration_path_for_file "${metadata}")"
    [[ -n "${service_path}" ]] || continue
    service_dir="$(dirname "$(dirname "${metadata}")")"
    canonical_service_dir="${PROJECT_ROOT}${service_dir#"${REPO_ROOT}"}"
    "${command[@]}" --path "${service_path}" --root "${canonical_service_dir}" >/dev/null
  done < <(find "${REPO_ROOT}/apps" -type f \( -name 'infisicalsecret.yaml' -o -name 'infisicalsecrets.yaml' \) | sort)

  finance_dir="${VERO_FINANCE_PROJECT_DIR:-/Users/mac/Documents/GitHub/verovault-finance}"
  if [[ -d "${finance_dir}" ]]; then
    kubedeck-env-mcp register --project-id "${FINANCE_PROJECT_ID}" --environment development \
      --path /finance --domain "${DOMAIN}" --root "${finance_dir}" >/dev/null
  fi
}

install_agent_surfaces() {
  local executable codex_config opencode_config opencode_temp
  npm install --global "${REPO_ROOT}/tools/kubedeck-env-mcp" >/dev/null
  mkdir -p /Users/mac/.codex/skills/local-dev-env /Users/mac/.config/opencode/skills/local-dev-env
  install -m 0644 "${REPO_ROOT}/.codex/skills/local-dev-env/SKILL.md" /Users/mac/.codex/skills/local-dev-env/SKILL.md
  install -m 0644 "${REPO_ROOT}/.codex/skills/local-dev-env/SKILL.md" /Users/mac/.config/opencode/skills/local-dev-env/SKILL.md
  executable="$(command -v kubedeck-env-mcp)"

  codex_config="${LOCAL_DEV_ENV_CODEX_CONFIG:-/Users/mac/.codex/config.toml}"
  mkdir -p "$(dirname "${codex_config}")"
  touch "${codex_config}"
  chmod 0600 "${codex_config}"
  opencode_temp="$(mktemp)"
  awk '
    /^\[mcp_servers\.local-dev-env\]$/ { skip=1; next }
    skip && /^\[/ { skip=0 }
    !skip { print }
  ' "${codex_config}" >"${opencode_temp}"
  printf '\n%s\n%s\n%s\n%s\n%s\n%s\n' \
    '[mcp_servers.local-dev-env]' \
    "command = \"${executable}\"" \
    'args = ["serve"]' \
    'enabled = true' \
    'startup_timeout_sec = 30.0' \
    'tool_timeout_sec = 30.0' \
    '' \
    '[mcp_servers.local-dev-env.env]' \
    'NODE_USE_SYSTEM_CA = "1"' >>"${opencode_temp}"
  install -m 0600 "${opencode_temp}" "${codex_config}"
  rm -f "${opencode_temp}"

  opencode_config="${LOCAL_DEV_ENV_OPENCODE_CONFIG:-/Users/mac/.config/opencode/opencode.json}"
  mkdir -p "$(dirname "${opencode_config}")"
  [[ -f "${opencode_config}" ]] || printf '{}\n' >"${opencode_config}"
  opencode_temp="$(mktemp)"
  jq --arg executable "${executable}" \
    '.mcp["local-dev-env"] = {type:"local",command:[$executable,"serve"],environment:{NODE_USE_SYSTEM_CA:"1"},enabled:true,timeout:30000}' \
    "${opencode_config}" >"${opencode_temp}"
  install -m 0600 "${opencode_temp}" "${opencode_config}"
  rm -f "${opencode_temp}"
}
