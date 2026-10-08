#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != Darwin ]]; then
  echo "This installer requires macOS." >&2
  exit 1
fi

if [[ -x /Applications/Docker.app/Contents/Resources/bin/kubectl ]]; then
  kubectl_bin=/Applications/Docker.app/Contents/Resources/bin/kubectl
else
  kubectl_bin="$(command -v kubectl)"
fi

"${kubectl_bin}" --context docker-desktop wait --for=condition=Ready nodes --all --timeout=10s >/dev/null
"${kubectl_bin}" --context docker-desktop -n technitium get service technitium >/dev/null
"${kubectl_bin}" --context docker-desktop -n platform-system get deployment kubedeck-local-registry >/dev/null

umask 077
agent_dir="${HOME}/Library/LaunchAgents"
log_dir="${HOME}/Library/Logs/KubeDeck"
mkdir -p "${agent_dir}" "${log_dir}"

services=(
  'technitium-admin|technitium|service/technitium|5380:5380|5380'
  'local-registry|platform-system|deployment/kubedeck-local-registry|5001:5001|5001'
)

# Refuse to take over an existing listener or an unmanaged LaunchAgent.
for service in "${services[@]}"; do
  IFS='|' read -r name namespace resource mapping port <<<"${service}"
  plist="${agent_dir}/dev.kubedeck.${name}.plist"
  if [[ -e "${plist}" ]]; then
    if ! /usr/libexec/PlistBuddy -c 'Print :KubeDeckManaged' "${plist}" 2>/dev/null | grep -qx true; then
      echo "Refusing to replace an unmanaged LaunchAgent: ${plist}" >&2
      exit 1
    fi
  elif lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "Port ${port} already has a listener; refusing to replace it." >&2
    exit 1
  fi
done

for service in "${services[@]}"; do
  IFS='|' read -r name namespace resource mapping port <<<"${service}"
  label="dev.kubedeck.${name}"
  plist="${agent_dir}/${label}.plist"

  python3 - "${plist}.new" "${label}" "${kubectl_bin}" "${namespace}" "${resource}" "${mapping}" "${log_dir}" "${HOME}" <<'PY'
import plistlib
import sys

path, label, kubectl, namespace, resource, mapping, log_dir, home = sys.argv[1:]
with open(path, 'wb') as file:
    plistlib.dump({
        'Label': label,
        'KubeDeckManaged': True,
        'ProgramArguments': [
            kubectl, '--context', 'docker-desktop', '-n', namespace,
            'port-forward', '--address', '127.0.0.1', resource, mapping,
        ],
        'RunAtLoad': True,
        'KeepAlive': True,
        'ThrottleInterval': 10,
        'EnvironmentVariables': {
            'HOME': home,
            'PATH': '/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin',
        },
        'StandardOutPath': log_dir + '/' + label + '.log',
        'StandardErrorPath': log_dir + '/' + label + '.error.log',
    }, file)
PY
  plutil -lint "${plist}.new" >/dev/null

  if [[ -e "${plist}" ]]; then
    launchctl bootout "gui/$(id -u)/${label}" 2>/dev/null || true
  fi
  mv "${plist}.new" "${plist}"
  launchctl bootstrap "gui/$(id -u)" "${plist}"
done

for service in "${services[@]}"; do
  IFS='|' read -r name namespace resource mapping port <<<"${service}"
  path=/
  if [[ "${name}" == local-registry ]]; then
    path=/v2/
  fi
  ready=false
  for _ in {1..15}; do
    if curl -fsS --max-time 2 "http://127.0.0.1:${port}${path}" >/dev/null 2>&1; then
      ready=true
      break
    fi
    sleep 1
  done
  if [[ "${ready}" != true ]]; then
    echo "${name} did not answer on localhost:${port}; inspect ${log_dir}/dev.kubedeck.${name}.error.log" >&2
    exit 1
  fi
  echo "${name} available on 127.0.0.1:${port}"
done
