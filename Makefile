# Loginer — run `make` for the list of targets.

SHELL   := bash
APPS    := console id docs
# Not ./...: that walks into web/*/node_modules.
PKGS    := ./cmd/... ./internal/... ./migrations/... ./i18n/...
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMPOSE := docker compose -f deploy/compose.yaml
# The same stack on one machine: .localhost hostnames, Caddy's own CA. An
# explicit -f means compose picks up no override on its own, so it is named.
COMPOSE_LOCAL := $(COMPOSE) -f deploy/compose.local.yaml

.DEFAULT_GOAL := help
.PHONY: help setup full-start dev run build test bench test-integration check docs docs-check \
	migrate-up migrate-down migrate-status migrate-new db-create db-reset db-psql \
	web-check web-build deploy-build deploy-up deploy-down deploy-logs \
	deploy-local deploy-local-down deploy-local-logs clean

help: ## Show this help
	@awk -F ':.*## ' '/^## / {printf "\n\033[1m%s\033[0m\n", substr($$0, 4)} /^[a-z -]+:.*## / {printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## Setup

setup: ## Prepare a fresh checkout: .env, database, dependencies
	@scripts/setup.sh

## Run

dev full-start: ## Start the API and all web apps, with hot reload
	@scripts/start.sh

run: ## Start only the API
	go run ./cmd/loginer

build: ## Build the API into bin/loginer
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/loginer ./cmd/loginer

## Quality

check: ## Format, vet and test the Go code
	go fmt $(PKGS) && go vet $(PKGS) && go test $(PKGS)

test: ## Run the Go tests
	go test $(PKGS)

bench: ## Run the Go micro-benchmarks
	go test -run '^$$' -bench . -benchmem ./internal/jose/ ./internal/model/ ./internal/api/ratelimit/

test-integration: ## Run the tests that need Postgres and Redis (throwaway databases and key prefixes)
	@. scripts/lib.sh && load_env && LOGINER_TEST_DB_DSN="$${DB_URL:-$$LOGINER_DB_DSN}" \
		LOGINER_TEST_REDIS="$${LOGINER_TEST_REDIS:-$${LOGINER_REDIS_HOST:+$$LOGINER_REDIS_HOST:$${LOGINER_REDIS_PORT:-6379}}}" \
		go test -count=1 -run Live ./internal/...

docs: ## Write the API reference (OpenAPI and web/docs pages) from the Go source
	go run ./cmd/apidoc

docs-check: ## Fail if the API reference is behind the Go source
	go run ./cmd/apidoc -check

web-check: ## Lint and type-check the web apps
	@for app in $(APPS); do echo "==> $$app"; (cd web/$$app && bun run lint && bun run check) || exit 1; done

web-build: ## Build the web apps
	@for app in $(APPS); do echo "==> $$app"; (cd web/$$app && bun run build) || exit 1; done

## Database

migrate-up: ## Apply pending migrations
	go run ./cmd/migrate up

migrate-down: ## Roll back the newest migration
	go run ./cmd/migrate down

migrate-status: ## Show applied migrations
	go run ./cmd/migrate status

migrate-new: ## Create a migration — make migrate-new name=add_x
	@test -n "$(name)" || { echo "usage: make migrate-new name=add_x"; exit 1; }
	go tool goose -dir migrations create $(name) go

db-create db-reset db-psql: ## Create, reset or open the database in .env
	@scripts/db.sh $(@:db-%=%)

## Deploy

deploy-build: ## Build the production images
	VERSION=$(VERSION) $(COMPOSE) build

deploy-up: deploy/Caddyfile ## Start the production stack in the background
	VERSION=$(VERSION) $(COMPOSE) up -d --build

# Not in the repository: the server provides its own (see the README).
deploy/Caddyfile:
	@echo "deploy/Caddyfile is missing: put this server's Caddyfile there first" >&2; exit 1

deploy-down: ## Stop the production stack
	$(COMPOSE) down

deploy-logs: ## Follow the API's logs in the production stack
	$(COMPOSE) logs -f api

deploy-local: ## Start the whole stack on this machine — https://id.localhost
	@VERSION=$(VERSION) scripts/deploy-local.sh

deploy-local-down: ## Stop the local stack and drop its database
	$(COMPOSE_LOCAL) down -v

deploy-local-logs: ## Follow the API's logs in the local stack
	$(COMPOSE_LOCAL) logs -f api

clean: ## Remove build output and logs
	rm -rf bin .logs $(foreach app,$(APPS),web/$(app)/build web/$(app)/.svelte-kit)
