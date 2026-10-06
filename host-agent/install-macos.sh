#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != Darwin ]]; then
  echo "This installer requires macOS." >&2
  exit 1
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
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
mv "${bin}.new" "${bin}"

python3 - "${plist}.new" "${label}" "${bin}" "${log_dir}" "${HOME}" <<'PY'
import plistlib, sys
path, label, binary, log_dir, home = sys.argv[1:]
with open(path, 'wb') as file:
    plistlib.dump({
        'Label': label,
        'KubeDeckManaged': True,
        'ProgramArguments': [binary],
        'RunAtLoad': True,
        'StartInterval': 30,
        'EnvironmentVariables': {
            'HOME': home,
            'PATH': '/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin',
        },
        'StandardOutPath': log_dir + '/host-agent.log',
        'StandardErrorPath': log_dir + '/host-agent.error.log',
    }, file)
PY
chmod 0600 "${plist}.new"
if [[ -e "${plist}" ]]; then
  launchctl bootout "gui/$(id -u)/${label}"
fi
mv "${plist}.new" "${plist}"
launchctl bootstrap "gui/$(id -u)" "${plist}"
echo "Installed ${label}; checks the current LAN IP every 30 seconds."
