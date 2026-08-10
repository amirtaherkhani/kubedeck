#!/usr/bin/env bash

set -euo pipefail

readonly KUBE_CONTEXT="${KUBE_CONTEXT:-rancher-desktop}"
readonly CERT_FILE="$(mktemp -t home-lab-local-dev-ca)"

cleanup() {
  rm -f "${CERT_FILE}"
}
trap cleanup EXIT

kubectl --context "${KUBE_CONTEXT}" -n platform-system get secret local-dev-ca \
  -o jsonpath='{.data.ca\.crt}' | base64 -D >"${CERT_FILE}"

sudo security add-trusted-cert \
  -d \
  -r trustRoot \
  -k /Library/Keychains/System.keychain \
  "${CERT_FILE}"

echo "macOS now trusts the shared Home Lab CA for *.local.dev and legacy *.dev.local URLs."
echo "Restart Chrome, Safari, or other browsers to reload the trust store."
