#!/usr/bin/env bash
set -euo pipefail

[[ "$(uname -s)" == Darwin ]] || { echo "macOS is required" >&2; exit 1; }
[[ "$(kubectl config current-context)" == docker-desktop ]] || { echo "Select docker-desktop context first" >&2; exit 1; }

interface="$(route -n get default | awk '$1 == "interface:" {print $2; exit}')"
[[ -n "${interface}" && "${interface}" != utun* ]] || { echo "No physical default-route interface" >&2; exit 1; }
host_ip="$(ipconfig getifaddr "${interface}")"
answer="$(dig +time=2 +tries=1 @127.0.0.1 infisical.local.dev A +short)"
[[ "${answer}" == "${host_ip}" ]] || { echo "Technitium localhost DNS does not return the current host IP" >&2; exit 1; }

resolver=/etc/resolver/local.dev
if [[ -f "${resolver}" && "$(cat "${resolver}")" == 'nameserver 127.0.0.1' ]]; then
  echo "Mac local.dev resolver already uses localhost."
  exit 0
fi

backup_dir="${HOME}/Library/Application Support/KubeDeck/Backups"
mkdir -p "${backup_dir}"
if [[ -f "${resolver}" && ! -e "${backup_dir}/local.dev.pre-docker-desktop" ]]; then
  cp "${resolver}" "${backup_dir}/local.dev.pre-docker-desktop"
  chmod 0600 "${backup_dir}/local.dev.pre-docker-desktop"
fi

# This one-time macOS administrator prompt replaces only the scoped resolver.
osascript -e 'do shell script "/bin/echo '\''nameserver 127.0.0.1'\'' > /etc/resolver/local.dev && /usr/bin/dscacheutil -flushcache && /usr/bin/killall -HUP mDNSResponder" with administrator privileges'
[[ "$(dscacheutil -q host -a name infisical.local.dev | awk '$1 == "ip_address:" {print $2; exit}')" == "${host_ip}" ]] || {
  echo "macOS resolver verification failed" >&2
  exit 1
}
echo "Mac local.dev resolver now follows localhost Technitium."
