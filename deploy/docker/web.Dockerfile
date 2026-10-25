# A web app (web/console or web/id) as a Node server. Build context: the
# repository root, and APP says which app to build.
#
#   docker build -f deploy/docker/web.Dockerfile --build-arg APP=web/id -t vard-id .
#
# The context is the repository and not the app's own directory because the
# apps import the shipped translations from i18n/ at the top of it
# (web/id/src/lib/i18n/messages.ts). The app therefore keeps its place in the
# tree inside the image — /src/web/id — so that relative import still points
# at /src/i18n.
#
# Runtime settings: API_URL (the API inside the network) and PORT (3000).
# Nothing sits in front of it yet, so the address it is reached at and the
# caller's address come from the connection itself.

ARG APP=web/id

FROM oven/bun:1-alpine AS build
ARG APP
WORKDIR /src/${APP}
COPY ${APP}/package.json ${APP}/bun.lock ./
RUN bun install --frozen-lockfile
COPY i18n /src/i18n
COPY ${APP} /src/${APP}
RUN bun run build && rm -rf node_modules && bun install --frozen-lockfile --production --ignore-scripts

FROM node:24-alpine
ARG APP
WORKDIR /app
ENV NODE_ENV=production PORT=3000
COPY --from=build /src/${APP}/package.json ./
COPY --from=build /src/${APP}/node_modules ./node_modules
COPY --from=build /src/${APP}/build ./build
USER node
EXPOSE 3000
CMD ["node", "build"]
