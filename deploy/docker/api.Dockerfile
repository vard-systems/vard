# The Vard API server.
# Build context: the repository root.
#
#   docker build -f deploy/docker/api.Dockerfile -t vard .

FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
# The shipped translations are a Go package (i18n/i18n.go) that embeds
# the JSON beside it, and internal/api/respond imports it.
COPY i18n ./i18n
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/vard ./cmd/vard

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H -u 10001 vard
WORKDIR /app
COPY --from=build /out/vard ./vard
# The migrations are compiled in and applied on start; goose only needs the
# directory it is pointed at to exist.
RUN mkdir migrations
USER vard
ENV GIN_MODE=release VARD_ADDR=:8080 VARD_ADMIN_ADDR=:8081
# 8080 public, 8081 admin: neither is published; the apps reach them.
EXPOSE 8080 8081
ENTRYPOINT ["/app/vard"]
