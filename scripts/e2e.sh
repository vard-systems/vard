#!/usr/bin/env bash
#
# Run the end-to-end tests (web/e2e) against a stack of their own: the API
# built from this tree, and both apps, on spare ports, with a database and a
# Redis prefix made for the run and dropped after it. A stack already running
# on the usual ports is left alone.
#
#   scripts/e2e.sh [playwright arguments]
#
# Only the database and Redis addresses are taken from .env (as
# make test-integration takes them). The API runs in an empty directory, so
# it reads no .env: the developer's mail settings and keys never reach it.

set -euo pipefail
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

need go "https://go.dev/dl"
need bun "https://bun.sh"
need psql "install the PostgreSQL client tools"

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

api_port=8290 admin_port=8291 id_port=5293 console_port=5294
for port in $api_port $admin_port $id_port $console_port; do
	if lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
		die "port $port is in use; the end-to-end stack needs 8290, 8291, 5293 and 5294"
	fi
done

server="$(load_env && echo "${VARD_TEST_DB_DSN:-${DB_URL:-${VARD_DB_DSN:-}}}")"
redis="$(load_env && echo "${VARD_REDIS_HOST:-}")"
[[ -n $server ]] || die "set VARD_TEST_DB_DSN (or VARD_DB_DSN in .env) to a Postgres server the tests may create a database on"

name="vard_e2e_$(openssl rand -hex 4)"
admin_dsn="$(sed -E "s|/[^/?]+(\?.*)?$|/postgres\1|" <<<"$server")"
dsn="$(sed -E "s|/[^/?]+(\?.*)?$|/$name\1|" <<<"$server")"
prefix="vard-e2e-$name:"
logs="$root/.logs" && mkdir -p "$logs"
work="$(mktemp -d)"
pids=()

cleanup() {
	for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; done
	wait 2>/dev/null || true
	psql "$admin_dsn" -qc "DROP DATABASE IF EXISTS \"$name\" WITH (FORCE)" >/dev/null 2>&1 || true
	if [[ -n $redis ]] && command -v redis-cli >/dev/null; then
		for db in 0 1; do
			redis-cli -h "$redis" -n "$db" --scan --pattern "$prefix*" | xargs -r redis-cli -h "$redis" -n "$db" del >/dev/null
		done
	fi
	rm -rf "$work"
}
trap cleanup EXIT

psql "$admin_dsn" -qc "CREATE DATABASE \"$name\""
go build -o "$work/vard" ./cmd/vard
# The migrations are compiled in; goose only needs the directory to exist.
mkdir "$work/migrations"

(
	cd "$work"
	exec env VARD_ADDR=":$api_port" VARD_ADMIN_ADDR=":$admin_port" \
		VARD_DB_DSN="$dsn" VARD_DB_MIGRATE=true \
		VARD_ISSUER="http://localhost:$id_port" VARD_ACCOUNT_URL="http://localhost:$id_port" \
		VARD_ADMIN_URL="http://localhost:$console_port" VARD_ADMIN_MFA=optional \
		VARD_TRUSTED_PROXIES=127.0.0.1/32,::1/128 \
		VARD_SECRET_KEY="$(openssl rand -base64 32)" \
		VARD_REDIS_HOST="$redis" VARD_REDIS_PREFIX="$prefix" \
		VARD_RATE_LIMIT=1000 \
		./vard
) >"$logs/e2e-api.log" 2>&1 &
pids+=($!)

(cd web/id && API_URL="http://localhost:$api_port" exec ./node_modules/.bin/vite dev --port $id_port --strictPort) >"$logs/e2e-id.log" 2>&1 &
pids+=($!)
(cd web/console && API_URL="http://localhost:$admin_port" exec ./node_modules/.bin/vite dev --port $console_port --strictPort) >"$logs/e2e-console.log" 2>&1 &
pids+=($!)

for url in "http://localhost:$api_port/readyz" "http://localhost:$id_port/robots.txt" "http://localhost:$console_port/robots.txt"; do
	for _ in $(seq 1 60); do
		curl -sf -o /dev/null "$url" && break
		sleep 1
	done
	curl -sf -o /dev/null "$url" || die "$url did not come up; see .logs/e2e-*.log"
done

token="$(grep -o 'setup_token=[^ ]*' "$logs/e2e-api.log" | head -1 | cut -d= -f2)"
[[ -n $token ]] || die "the API logged no setup token; see .logs/e2e-api.log"

cd web/e2e
E2E_ID_URL="http://localhost:$id_port" E2E_CONSOLE_URL="http://localhost:$console_port" \
	E2E_SETUP_TOKEN="$token" ./node_modules/.bin/playwright test "$@"
