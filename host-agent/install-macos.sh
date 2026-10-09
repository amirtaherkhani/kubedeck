#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != Darwin ]]; then
  echo "This installer requires macOS." >&2
  exit 1
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
kubectl_bin="$(command -v kubectl || true)"
if [[ -z "${kubectl_bin}" || ! -x "${kubectl_bin}" ]]; then
  echo "kubectl must be installed before the host agent." >&2
  exit 1
fi
kube_context="${KUBEDECK_HOST_AGENT_KUBE_CONTEXT:-}"
if [[ -z "${kube_context}" ]]; then
  kube_context="$(kubectl config current-context)"
fi
if [[ -z "${kube_context}" ]]; then
  echo "Select a Kubernetes context or set KUBEDECK_HOST_AGENT_KUBE_CONTEXT." >&2
  exit 1
fi
zone="${KUBEDECK_HOST_AGENT_ZONE:-}"
if [[ -z "${zone}" ]]; then
  echo "Set KUBEDECK_HOST_AGENT_ZONE to the DNS zone this agent may change." >&2
  exit 1
fi
bin_dir="${HOME}/.local/bin"
agent_dir="${HOME}/Library/LaunchAgents"
log_dir="${HOME}/Library/Logs/KubeDeck"
label="dev.kubedeck.host-agent"
plist="${agent_dir}/${label}.plist"
bin="${bin_dir}/kubedeck-host-agent"

mkdir -p "${bin_dir}" "${agent_dir}" "${log_dir}"
if [[ -e "${plist}" ]] && ! /usr/libexec/PlistBuddy -c 'Print :KubeDeckManaged' "${plist}" 2>/dev/null | grep -qx true; then
  echo "Refusing to replace an existing unmanaged LaunchAgent: ${plist}" >&2
  exit 1
fi

(cd "${script_dir}" && go build -o "${bin}.new" ./cmd/kubedeck-host-agent)
chmod 0700 "${bin}.new"

python3 - "${plist}.new" "${label}" "${bin}" "${log_dir}" "${HOME}" "${kube_context}" "${zone}" "$(dirname "${kubectl_bin}")" <<'PY'
import os
import plistlib
import sys

path, label, binary, log_dir, home, kube_context, zone, kubectl_dir = sys.argv[1:]
arguments = [
    binary,
    '-kube-context', kube_context,
    '-zone', zone,
]
for variable, option in (
    ('KUBEDECK_HOST_AGENT_INTERFACE', '-interface'),
    ('KUBEDECK_HOST_AGENT_TARGET_IP', '-target-ip'),
    ('KUBEDECK_HOST_AGENT_NAMESPACE', '-namespace'),
    ('KUBEDECK_HOST_AGENT_SERVICE', '-service'),
    ('KUBEDECK_HOST_AGENT_ADMIN_SECRET', '-admin-secret'),
    ('KUBEDECK_HOST_AGENT_PASSWORD_KEY', '-password-key'),
    ('KUBEDECK_HOST_AGENT_ADMIN_USER', '-admin-user'),
    ('KUBEDECK_HOST_AGENT_API_PORT', '-api-port'),
):
    if value := os.environ.get(variable):
        arguments.extend((option, value))
environment = {
    'HOME': home,
    'PATH': ':'.join((kubectl_dir, '/usr/bin', '/bin', '/usr/sbin', '/sbin')),
}
if kubeconfig := os.environ.get('KUBECONFIG'):
    environment['KUBECONFIG'] = os.pathsep.join(
        os.path.abspath(os.path.expanduser(entry)) for entry in kubeconfig.split(os.pathsep) if entry
    )
with open(path, 'wb') as file:
    plistlib.dump({
        'Label': label,
        'KubeDeckManaged': True,
        'ProgramArguments': arguments,
        'RunAtLoad': True,
        'StartInterval': 30,
        'EnvironmentVariables': environment,
        'StandardOutPath': log_dir + '/host-agent.log',
        'StandardErrorPath': log_dir + '/host-agent.error.log',
    }, file)
PY
chmod 0600 "${plist}.new"
plutil -lint "${plist}.new" >/dev/null
if [[ -e "${plist}" ]]; then
  launchctl bootout "gui/$(id -u)/${label}"
fi
mv "${bin}.new" "${bin}"
mv "${plist}.new" "${plist}"
launchctl bootstrap "gui/$(id -u)" "${plist}"
echo "Installed ${label}; checks the current LAN IP every 30 seconds."
