#!/usr/bin/env bash

set -euo pipefail

readonly KUBE_CONTEXT="${KUBE_CONTEXT:-rancher-desktop}"
readonly DNS_NAMESPACE="${HOME_LAB_DNS_NAMESPACE:-kube-system}"
readonly DNS_SERVICE="${HOME_LAB_DNS_SERVICE:-home-lab-dns}"
readonly EXPECTED_HOST_IP="${HOME_LAB_HOST_IP:-192.168.1.100}"
readonly RESOLVER_DIR="/etc/resolver"

usage() {
  cat <<'EOF'
Usage: configure-local-dev-resolver.sh [--dry-run]

Discovers the current home-lab CoreDNS LoadBalancer address and configures the
macOS scoped resolver for local.dev. The DNS server address is intentionally
different from the local.dev answer: CoreDNS returns the macOS ingress address.

Overrides: KUBE_CONTEXT, HOME_LAB_DNS_NAMESPACE, HOME_LAB_DNS_SERVICE,
HOME_LAB_DNS_SERVER, and HOME_LAB_HOST_IP.
EOF
}

dry_run=false
if [[ "${1:-}" == "--dry-run" ]]; then
  dry_run=true
  shift
fi
if [[ "$#" -ne 0 ]]; then
  usage >&2
  exit 2
fi

if [[ -n "${HOME_LAB_DNS_SERVER:-}" ]]; then
  dns_server="${HOME_LAB_DNS_SERVER}"
else
  if ! command -v kubectl >/dev/null 2>&1; then
    echo "kubectl is required to discover ${DNS_NAMESPACE}/${DNS_SERVICE}; set HOME_LAB_DNS_SERVER only for an explicit emergency override." >&2
    exit 1
  fi

  if ! kubectl --context "${KUBE_CONTEXT}" get node --no-headers 2>/dev/null | awk '$2 == "Ready" { found = 1 } END { exit !found }'; then
    echo "Kubernetes context ${KUBE_CONTEXT} has no Ready node; refusing to replace the macOS resolver with an unverified DNS target." >&2
    exit 1
  fi

  dns_server="$(kubectl --context "${KUBE_CONTEXT}" -n "${DNS_NAMESPACE}" get service "${DNS_SERVICE}" \
    -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
fi

if [[ -z "${dns_server}" ]]; then
  echo "${DNS_NAMESPACE}/${DNS_SERVICE} has no LoadBalancer IP; refusing to replace the macOS resolver." >&2
  exit 1
fi

if ! command -v dig >/dev/null 2>&1; then
  echo "dig is required to validate the discovered DNS server." >&2
  exit 1
fi

if ! dig +time=2 +tries=1 "@${dns_server}" grafana.local.dev A +short | grep -qx "${EXPECTED_HOST_IP}"; then
  echo "${dns_server} did not resolve grafana.local.dev to ${EXPECTED_HOST_IP}; refusing to replace the macOS resolver." >&2
  exit 1
fi

echo "Verified local.dev DNS: ${dns_server} -> ${EXPECTED_HOST_IP}"
if [[ "${dry_run}" == true ]]; then
  echo "Dry run: /etc/resolver/local.dev was not changed."
  exit 0
fi

sudo mkdir -p "${RESOLVER_DIR}"
for domain in local.dev; do
  resolver_file="${RESOLVER_DIR}/${domain}"
  printf 'nameserver %s\n' "${dns_server}" | sudo tee "${resolver_file}" >/dev/null
  sudo chmod 0644 "${resolver_file}"
done
dscacheutil -flushcache
sudo killall -HUP mDNSResponder 2>/dev/null || true

echo "macOS resolver configured: local.dev -> ${dns_server}"
echo "Verify with: dscacheutil -q host -a name grafana.local.dev"
