#!/bin/sh
set -eu

data_dir="${KUBEDECK_DATA_DIR:-/data}"
port="${PORT:-3000}"
server_dir="/app/dist/server"
migration_file="/app/dist/.openai/drizzle/0000_fixed_forge.sql"

mkdir -p "${data_dir}"
cd "${server_dir}"

if ! wrangler d1 execute DB \
  --config wrangler.json \
  --local \
  --persist-to "${data_dir}" \
  --command "SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'admin_users';" \
  --json | grep -q '"admin_users"'; then
  wrangler d1 execute DB \
    --config wrangler.json \
    --local \
    --persist-to "${data_dir}" \
    --file "${migration_file}"
fi

set -- wrangler dev \
  --config wrangler.json \
  --local \
  --ip 0.0.0.0 \
  --port "${port}" \
  --persist-to "${data_dir}"

if [ -n "${KUBEDECK_ADMIN_FIRST_NAME:-}" ]; then
  set -- "$@" --var "KUBEDECK_ADMIN_FIRST_NAME:${KUBEDECK_ADMIN_FIRST_NAME}"
fi
if [ -n "${KUBEDECK_ADMIN_LAST_NAME:-}" ]; then
  set -- "$@" --var "KUBEDECK_ADMIN_LAST_NAME:${KUBEDECK_ADMIN_LAST_NAME}"
fi
if [ -n "${KUBEDECK_ADMIN_EMAIL:-}" ]; then
  set -- "$@" --var "KUBEDECK_ADMIN_EMAIL:${KUBEDECK_ADMIN_EMAIL}"
fi
if [ -n "${KUBEDECK_ADMIN_PASSWORD:-}" ]; then
  set -- "$@" --var "KUBEDECK_ADMIN_PASSWORD:${KUBEDECK_ADMIN_PASSWORD}"
fi
if [ -n "${KUBEDECK_AGENT_URL:-}" ]; then
  set -- "$@" --var "KUBEDECK_AGENT_URL:${KUBEDECK_AGENT_URL}"
fi
if [ -n "${KUBEDECK_AGENT_TOKEN:-}" ]; then
  set -- "$@" --var "KUBEDECK_AGENT_TOKEN:${KUBEDECK_AGENT_TOKEN}"
fi

exec "$@"
