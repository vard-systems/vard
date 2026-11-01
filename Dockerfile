# syntax=docker/dockerfile:1
#
# The Vard API server. Build context: the repository root.
#
#   docker build -t vard-api .
#
# A static binary on Alpine, run as an unprivileged user with nothing else in
# the image it could use: no compiler, no sources, no shell tools beyond
# BusyBox's (whose wget the healthcheck uses).

FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd ./cmd
COPY api ./api
COPY core ./core
COPY tools ./tools
COPY migrations ./migrations
# The shipped translations are a Go package (i18n/i18n.go) that embeds the JSON
# beside it, and api/kit/respond imports it.
COPY i18n ./i18n
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/vard ./cmd/vard

FROM alpine:3.22
ARG VERSION=dev
LABEL org.opencontainers.image.title="vard-api" \
	org.opencontainers.image.version="${VERSION}" \
	org.opencontainers.image.source="https://github.com/vard-systems/vard"
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H -u 10001 vard
WORKDIR /app
COPY --from=build /out/vard ./vard
USER 10001:10001
ENV GIN_MODE=release VARD_ADDR=:8080 VARD_ADMIN_ADDR=:8081
# 8080 public, 8081 admin: neither is published; the apps reach them.
EXPOSE 8080 8081
ENTRYPOINT ["/app/vard"]
