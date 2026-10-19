# The Loginer server: the API, with the id app and the console compiled in.
# Build context: the repository root.
#
#   docker build -f deploy/docker/api.Dockerfile -t loginer .

# The two apps, as static files. The id app imports the shipped translations
# from i18n/ at the top of the repository, so each app keeps its place in the
# tree here and that relative import still resolves.
FROM oven/bun:1-alpine AS web
WORKDIR /src
COPY web/id/package.json web/id/bun.lock web/id/
COPY web/console/package.json web/console/bun.lock web/console/
RUN cd web/id && bun install --frozen-lockfile && cd ../console && bun install --frozen-lockfile
COPY i18n i18n
COPY web web
RUN cd web/id && bun run build && cd ../console && bun run build

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
# The shipped translations are a Go package (i18n/i18n.go) that embeds
# the JSON beside it, and internal/api/respond imports it.
COPY i18n ./i18n
COPY web/embed.go ./web/
COPY --from=web /src/web/id/build ./web/id/build
COPY --from=web /src/web/console/build ./web/console/build
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/loginer ./cmd/loginer

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H -u 10001 loginer
WORKDIR /app
COPY --from=build /out/loginer ./loginer
# The migrations are compiled in and applied on start; goose only needs the
# directory it is pointed at to exist.
RUN mkdir migrations
USER loginer
ENV GIN_MODE=release LOGINER_ADDR=:8080 LOGINER_ADMIN_ADDR=:8081
# 8080 public, 8081 admin; deploy/compose.yaml publishes them.
EXPOSE 8080 8081
ENTRYPOINT ["/app/loginer"]
