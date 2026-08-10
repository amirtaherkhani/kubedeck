#!/usr/bin/env bash

set -euo pipefail

readonly DNS_SERVER="${HOME_LAB_DNS_SERVER:-192.168.64.14}"
readonly RESOLVER_DIR="/etc/resolver"

sudo mkdir -p "${RESOLVER_DIR}"
for domain in local.dev; do
  resolver_file="${RESOLVER_DIR}/${domain}"
  printf 'nameserver %s\n' "${DNS_SERVER}" | sudo tee "${resolver_file}" >/dev/null
  sudo chmod 0644 "${resolver_file}"
done
dscacheutil -flushcache
sudo killall -HUP mDNSResponder 2>/dev/null || true

echo "macOS resolver configured: local.dev -> ${DNS_SERVER}"
echo "Test with: dig grafana.local.dev +short"
