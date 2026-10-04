#!/usr/bin/env bash

set -euo pipefail

readonly VERSION="1.11.1"
readonly ARCHIVE="node_exporter-${VERSION}.darwin-arm64.tar.gz"
readonly SHA256="e987428618362c2d2540a68b722bd982ef1486c9961631298f20ea8fd57d3be4"
readonly DOWNLOAD_URL="https://github.com/prometheus/node_exporter/releases/download/v${VERSION}/${ARCHIVE}"
readonly LABEL="com.prometheus.node-exporter"
readonly PORT="${NODE_EXPORTER_PORT:-9101}"

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly INSTALL_DIR="${HOME}/.local/opt/node_exporter-${VERSION}"
readonly BIN_LINK="${HOME}/.local/bin/node_exporter"
readonly LAUNCH_AGENT="${HOME}/Library/LaunchAgents/${LABEL}.plist"
readonly LAUNCH_DOMAIN="gui/$(id -u)"

if [[ "$(uname -s)" != "Darwin" || "$(uname -m)" != "arm64" ]]; then
  echo "This installer supports Darwin arm64 only." >&2
  exit 1
fi

if lsof -nP -iTCP:"${PORT}" -sTCP:LISTEN >/dev/null 2>&1; then
  listener="$(lsof -nP -iTCP:"${PORT}" -sTCP:LISTEN -t | head -n 1)"
  if ! ps -p "${listener}" -o command= | grep -q "node_exporter"; then
    echo "TCP port ${PORT} is already used by another process." >&2
    lsof -nP -iTCP:"${PORT}" -sTCP:LISTEN >&2
    exit 1
  fi
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

curl --fail --location --retry 3 --retry-delay 2 \
  --output "${tmp_dir}/${ARCHIVE}" "${DOWNLOAD_URL}"

actual_sha="$(shasum -a 256 "${tmp_dir}/${ARCHIVE}" | awk '{print $1}')"
if [[ "${actual_sha}" != "${SHA256}" ]]; then
  echo "Checksum mismatch for ${ARCHIVE}." >&2
  echo "Expected: ${SHA256}" >&2
  echo "Actual:   ${actual_sha}" >&2
  exit 1
fi

tar -xzf "${tmp_dir}/${ARCHIVE}" -C "${tmp_dir}"
install -d "${INSTALL_DIR}" "$(dirname "${BIN_LINK}")" \
  "$(dirname "${LAUNCH_AGENT}")" "${HOME}/Library/Logs"
install -m 0755 \
  "${tmp_dir}/node_exporter-${VERSION}.darwin-arm64/node_exporter" \
  "${INSTALL_DIR}/node_exporter"
ln -sfn "${INSTALL_DIR}/node_exporter" "${BIN_LINK}"

sed \
  -e "s|__HOME__|${HOME}|g" \
  -e "s|__PORT__|${PORT}|g" \
  "${SCRIPT_DIR}/${LABEL}.plist.template" > "${tmp_dir}/${LABEL}.plist"
plutil -lint "${tmp_dir}/${LABEL}.plist" >/dev/null
install -m 0644 "${tmp_dir}/${LABEL}.plist" "${LAUNCH_AGENT}"

launchctl bootout "${LAUNCH_DOMAIN}" "${LAUNCH_AGENT}" >/dev/null 2>&1 || true
launchctl bootstrap "${LAUNCH_DOMAIN}" "${LAUNCH_AGENT}"
launchctl enable "${LAUNCH_DOMAIN}/${LABEL}"
launchctl kickstart -k "${LAUNCH_DOMAIN}/${LABEL}"

metrics_file="${tmp_dir}/metrics"
for _ in $(seq 1 20); do
  if curl --silent --show-error --fail \
    "http://127.0.0.1:${PORT}/metrics" > "${metrics_file}" 2>/dev/null \
    && grep -q 'node_exporter_build_info.*goos="darwin"' "${metrics_file}" \
    && grep -q 'node_uname_info.*sysname="Darwin"' "${metrics_file}"; then
    echo "node_exporter ${VERSION} is running on http://127.0.0.1:${PORT}/metrics"
    exit 0
  fi
  sleep 1
done

echo "node_exporter did not become healthy. Check:" >&2
echo "  ${HOME}/Library/Logs/node-exporter.err.log" >&2
exit 1
