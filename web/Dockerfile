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
WORKDIR /src/web

# The apps are one Bun workspace with the design they share (web/ui): its
# manifests first, so the install is cached until one of them changes.
COPY web/package.json web/bun.lock ./
COPY web/console/package.json console/
COPY web/id/package.json id/
COPY web/ui/package.json ui/
COPY web/e2e/package.json e2e/
RUN --mount=type=cache,target=/root/.bun/install/cache bun install --frozen-lockfile

COPY i18n /src/i18n
COPY web/ui ui
COPY ${APP} /src/${APP}
RUN cd /src/${APP} && bun run build

# What the server needs at run time is the app's own dependencies: the rest,
# web/ui among them, is in the build. Hoisted into one plain tree, so it can be
# copied as it is.
RUN rm -rf node_modules */node_modules \
	&& bun install --frozen-lockfile --production --ignore-scripts --linker hoisted --filter "./${APP#web/}"

FROM node:24-alpine
ARG APP
ARG VERSION=dev
LABEL org.opencontainers.image.title="vard-${APP#web/}" \
	org.opencontainers.image.version="${VERSION}" \
	org.opencontainers.image.source="https://github.com/vard-systems/vard"
WORKDIR /app
ENV NODE_ENV=production PORT=3000
COPY --from=build --chown=root:root /src/${APP}/package.json ./
COPY --from=build --chown=root:root /src/web/node_modules ./node_modules
COPY --from=build --chown=root:root /src/${APP}/build ./build
# The files belong to root and the server runs as node, so it can read its
# code and change none of it.
USER node
EXPOSE 3000
CMD ["node", "build"]
