#!/usr/bin/env bash

set -euo pipefail

readonly PF_CONF="/etc/pf.conf"
readonly HOST_IP="${HOME_LAB_HOST_IP:-192.168.1.100}"
readonly KUBE_CONTEXT="${KUBE_CONTEXT:-rancher-desktop}"
readonly HOMEBREW_PREFIX="${HOMEBREW_PREFIX:-/opt/homebrew}"
readonly DNSMASQ_BIN="${HOME_LAB_DNSMASQ_BIN:-${HOMEBREW_PREFIX}/sbin/dnsmasq}"
readonly DNSMASQ_CONF_DIR="${HOMEBREW_PREFIX}/etc/dnsmasq.d"
readonly DNSMASQ_CONF="${DNSMASQ_CONF_DIR}/home-lab-local-dev.conf"
readonly DNSMASQ_MAIN_CONF="${HOMEBREW_PREFIX}/etc/dnsmasq.conf"
readonly DNSMASQ_SERVICE="homebrew.mxcl.dnsmasq"
readonly PF_RDR_ANCHOR='rdr-anchor "home-lab-dns"'
readonly PF_LOAD_ANCHOR='load anchor "home-lab-dns" from "/etc/pf.anchors/home-lab-dns"'

readonly PF_TMP="$(mktemp -t home-lab-pf)"

cluster_dns_ip="${HOME_LAB_CLUSTER_DNS_IP:-}"
if [[ -z "${cluster_dns_ip}" ]] && command -v kubectl >/dev/null 2>&1; then
  cluster_dns_ip="$(kubectl --context "${KUBE_CONTEXT}" -n kube-system get service home-lab-dns \
    -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || true)"
fi
readonly CLUSTER_DNS_IP="${cluster_dns_ip:-192.168.1.183}"

trap 'rm -f "${PF_TMP}"' EXIT

if [[ ! -x "${DNSMASQ_BIN}" ]]; then
  echo "dnsmasq is required at ${DNSMASQ_BIN}; install it with Homebrew or set HOME_LAB_DNSMASQ_BIN." >&2
  exit 1
fi

sudo mkdir -p "${DNSMASQ_CONF_DIR}"
sudo tee "${DNSMASQ_CONF}" >/dev/null <<EOF
listen-address=127.0.0.1,${HOST_IP}
bind-interfaces
server=/local.dev/${CLUSTER_DNS_IP}
EOF
sudo chmod 0644 "${DNSMASQ_CONF}"

/usr/bin/awk -v rdr="${PF_RDR_ANCHOR}" -v load="${PF_LOAD_ANCHOR}" '
  $0 == rdr || $0 == load { next }
  { print }
' "${PF_CONF}" > "${PF_TMP}"

sudo /sbin/pfctl -nf "${PF_TMP}"
sudo /usr/bin/install -o root -g wheel -m 0644 "${PF_TMP}" "${PF_CONF}"
sudo /sbin/pfctl -f "${PF_CONF}"
sudo /sbin/pfctl -E >/dev/null

"${DNSMASQ_BIN}" --test -C "${DNSMASQ_MAIN_CONF}" -7 "${DNSMASQ_CONF_DIR},*.conf"
sudo /bin/launchctl kickstart -k "system/${DNSMASQ_SERVICE}"

echo "macOS DNS listener configured: ${HOST_IP}:53 -> ${CLUSTER_DNS_IP}:53 for local.dev"
