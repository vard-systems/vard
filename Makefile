# Loginer — run `make` for the list of targets.

SHELL   := bash
APPS    := console id
# Not ./...: that walks into web/*/node_modules.
PKGS    := ./cmd/... ./internal/... ./migrations/... ./i18n/... ./web
# The builds the binary embeds (web/embed.go).
WEB_BUILDS := $(foreach app,$(APPS),web/$(app)/build/index.html)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMPOSE := docker compose -f deploy/compose.yaml

.DEFAULT_GOAL := help
.PHONY: help setup full-start dev run build test bench test-integration check docs docs-check \
	migrate-up migrate-down migrate-status migrate-new db-create db-reset db-psql \
	web web-built web-check web-test deploy-build deploy-up deploy-down deploy-logs \
	deploy-local deploy-local-down deploy-local-logs clean

help: ## Show this help
	@awk -F ':.*## ' '/^## / {printf "\n\033[1m%s\033[0m\n", substr($$0, 4)} /^[a-z -]+:.*## / {printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## Setup

setup: ## Prepare a fresh checkout: .env, database, dependencies
	@scripts/setup.sh

## Run

dev full-start: ## Start the API and both apps (Vite, hot reload) — http://localhost:5173, :5174
	@scripts/start.sh

run: web-built ## Start only the server, with the apps as last built — http://localhost:8080, :8081
	go run ./cmd/loginer

build: web ## Build the server, both apps compiled in, into bin/loginer
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/loginer ./cmd/loginer

web: ## Build both apps into web/*/build, which the binary embeds
	@for app in $(APPS); do echo "==> $$app"; (cd web/$$app && bun install --frozen-lockfile && bun run build) || exit 1; done

# Anything that compiles cmd/ or ./web needs the builds to exist; whether they
# are current matters only for `make build`, which always makes them afresh.
web-built:
	@$(foreach build,$(WEB_BUILDS),[[ -f $(build) ]] &&) true || $(MAKE) --no-print-directory web

## Quality

check: web-built ## Format, vet and test the Go code
	go fmt $(PKGS) && go vet $(PKGS) && go test $(PKGS)

test: web-built ## Run the Go tests
	go test $(PKGS)

bench: ## Run the Go micro-benchmarks
	go test -run '^$$' -bench . -benchmem ./internal/jose/ ./internal/model/ ./internal/api/ratelimit/

test-integration: ## Run the tests that need Postgres and Redis (throwaway databases and key prefixes)
	@. scripts/lib.sh && load_env && LOGINER_TEST_DB_DSN="$${DB_URL:-$$LOGINER_DB_DSN}" \
		LOGINER_TEST_REDIS="$${LOGINER_TEST_REDIS:-$${LOGINER_REDIS_HOST:+$$LOGINER_REDIS_HOST:$${LOGINER_REDIS_PORT:-6379}}}" \
		go test -count=1 -run Live ./internal/...

docs: ## Write the API reference (the OpenAPI documents the server answers) from the Go source
	go run ./cmd/apidoc

docs-check: ## Fail if the API reference is behind the Go source
	go run ./cmd/apidoc -check

web-check: ## Lint and type-check the web apps
	@for app in $(APPS); do echo "==> $$app"; (cd web/$$app && bun run lint && bun run check) || exit 1; done

web-test: ## Run the web apps' unit tests
	@for app in $(APPS); do echo "==> $$app"; (cd web/$$app && bunx vitest --run) || exit 1; done


## Database

migrate-up: ## Apply pending migrations
	go run ./cmd/migrate up

migrate-down: ## Roll back the newest migration
	go run ./cmd/migrate down

migrate-status: ## Show applied migrations
	go run ./cmd/migrate status

migrate-new: ## Create a migration — make migrate-new name=add_x
	@test -n "$(name)" || { echo "usage: make migrate-new name=add_x"; exit 1; }
	go run ./cmd/migrate new $(name)

db-create db-reset db-psql: ## Create, reset or open the database in .env
	@scripts/db.sh $(@:db-%=%)

## Deploy

deploy-build: ## Build the production images
	VERSION=$(VERSION) $(COMPOSE) build

deploy-up: ## Start the production stack in the background
	VERSION=$(VERSION) $(COMPOSE) up -d --build

deploy-down: ## Stop the production stack
	$(COMPOSE) down

deploy-logs: ## Follow the API's logs in the production stack
	$(COMPOSE) logs -f api

deploy-local: ## Start the production stack on this machine — http://localhost:8080
	@VERSION=$(VERSION) scripts/deploy-local.sh

deploy-local-down: ## Stop the local stack and drop its database
	$(COMPOSE) down -v

deploy-local-logs: ## Follow the API's logs in the local stack
	$(COMPOSE) logs -f api

clean: ## Remove build output and logs
	rm -rf bin .logs $(foreach app,$(APPS),web/$(app)/build web/$(app)/.svelte-kit)
