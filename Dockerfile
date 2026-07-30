# syntax=docker/dockerfile:1.7

FROM node:22.19.0-bookworm-slim AS builder

ARG TARGETARCH

ENV NPM_CONFIG_FETCH_RETRIES=5 \
    NPM_CONFIG_FETCH_RETRY_MINTIMEOUT=20000 \
    NPM_CONFIG_FETCH_RETRY_MAXTIMEOUT=120000

WORKDIR /app

COPY package.json package-lock.json ./
RUN --mount=type=cache,id=kubedeck-npm-cache,target=/root/.npm \
    npm ci --prefer-offline --no-audit --no-fund

# npm can omit platform-specific optional packages when package-lock.json was
# generated on macOS. Install the native build binding for the target platform.
RUN --mount=type=cache,id=kubedeck-npm-cache,target=/root/.npm \
    case "${TARGETARCH}" in \
      amd64) binding="@rolldown/binding-linux-x64-gnu@1.0.1" ;; \
      arm64) binding="@rolldown/binding-linux-arm64-gnu@1.0.1" ;; \
      *) echo "Unsupported target architecture: ${TARGETARCH}" >&2; exit 1 ;; \
    esac \
    && npm install --no-save --prefer-offline --no-audit --no-fund "${binding}"

COPY . .
RUN npm run build

FROM node:22.19.0-bookworm-slim AS runtime

ENV NODE_ENV=production \
    PORT=3000 \
    KUBEDECK_DATA_DIR=/data \
    NPM_CONFIG_FETCH_RETRIES=5 \
    NPM_CONFIG_FETCH_RETRY_MINTIMEOUT=20000 \
    NPM_CONFIG_FETCH_RETRY_MAXTIMEOUT=120000

RUN --mount=type=cache,id=kubedeck-npm-cache,target=/root/.npm \
    npm install --global --prefer-offline --no-audit --no-fund wrangler@4.92.0 \
    && npm cache clean --force

WORKDIR /app

COPY --from=builder --chown=node:node /app/dist ./dist
COPY --chown=node:node docker-entrypoint.sh /usr/local/bin/kubedeck-entrypoint

RUN chmod 0555 /usr/local/bin/kubedeck-entrypoint \
    && mkdir -p /data \
    && chown node:node /data

USER node

EXPOSE 3000
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/kubedeck-entrypoint"]
