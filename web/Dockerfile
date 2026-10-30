# syntax=docker/dockerfile:1
#
# A web app (web/console or web/id) as a Node server. Build context: the
# repository root, and APP says which app to build.
#
#   docker build -f web/Dockerfile --build-arg APP=web/id -t vard-id .
#
# The context is the repository and not the app's own directory because the
# apps import the shipped translations from i18n/ at the top of it
# (web/id/src/lib/i18n/messages.ts). The app therefore keeps its place in the
# tree inside the image — /src/web/id — so that relative import still points
# at /src/i18n.
#
# Runtime settings, all from compose.yaml: API_URL (the API inside the
# network), PORT (3000), and adapter-node's own — ADDRESS_HEADER, which header
# names the caller once a proxy is in front, and BODY_SIZE_LIMIT.

ARG APP=web/id

# Bun at the version the lockfiles were written with: --frozen-lockfile then
# installs exactly what they say.
FROM oven/bun:1.4.2-alpine AS build
ARG APP
WORKDIR /src/${APP}
COPY ${APP}/package.json ${APP}/bun.lock ./
RUN --mount=type=cache,target=/root/.bun/install/cache bun install --frozen-lockfile
COPY i18n /src/i18n
COPY ${APP} /src/${APP}
RUN bun run build \
	&& rm -rf node_modules \
	&& bun install --frozen-lockfile --production --ignore-scripts

FROM node:24-alpine
ARG APP
ARG VERSION=dev
LABEL org.opencontainers.image.title="vard-${APP#web/}" \
	org.opencontainers.image.version="${VERSION}" \
	org.opencontainers.image.source="https://github.com/vard-systems/vard"
WORKDIR /app
ENV NODE_ENV=production PORT=3000
COPY --from=build --chown=root:root /src/${APP}/package.json ./
COPY --from=build --chown=root:root /src/${APP}/node_modules ./node_modules
COPY --from=build --chown=root:root /src/${APP}/build ./build
# The files belong to root and the server runs as node, so it can read its
# code and change none of it.
USER node
EXPOSE 3000
CMD ["node", "build"]
