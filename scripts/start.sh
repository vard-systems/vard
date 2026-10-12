#!/usr/bin/env bash
#
# Run the whole stack: the Loginer API and the console, id and docs apps, as
# Vite dev servers with hot reload. The production builds run with
# `make deploy-local`, behind the same Caddy as in production.
#
# The API logs to this terminal; the apps log to .logs/<app>.log. Ctrl-C stops
# everything.

set -euo pipefail
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

need go "https://go.dev/dl"
need node "https://nodejs.org"
need bun "https://bun.sh"
load_env

api_port="${LOGINER_ADDR:-:8080}" api_port="${api_port##*:}"
admin_port="${LOGINER_ADMIN_ADDR:-:8081}" admin_port="${admin_port##*:}"

# name  port  API it calls
apps=(
	"id      5173 http://localhost:$api_port"
	"console 5174 http://localhost:$admin_port"
	"docs    5175 http://localhost:$api_port"
)

logs="$root/.logs"
mkdir -p "$logs" bin

# A port is taken when anything at all answers on it (curl exit 7: refused).
for port in "$api_port" "$admin_port" 5173 5174 5175; do
	code=0 && curl -s -o /dev/null --max-time 1 "http://localhost:$port" || code=$?
	[[ $code == 7 ]] || die "port $port is already in use"
done

for app in "${apps[@]}"; do
	read -r name _ <<<"$app"
	[[ -d web/$name/node_modules ]] || die "web/$name has no dependencies — run make setup"
done

# Background jobs ignore Ctrl-C, so stopping is done here, for all of them.
pids=()
trap 'trap - EXIT; kill ${pids[@]+"${pids[@]}"} 2>/dev/null; wait' EXIT
trap 'exit 0' INT TERM # stopping on purpose is not a failure

# ready <name> <port> <pid>: report once the app answers, or show why it died.
ready() {
	for _ in $(seq 240); do
		kill -0 "$3" 2>/dev/null || { printf '\n✗ %s stopped — .logs/%s.log:\n' "$1" "$1"; tail -n 20 "$logs/$1.log"; return; }
		curl -s -o /dev/null --max-time 1 "http://localhost:$2" && { echo "✓ $1 on http://localhost:$2"; return; }
		sleep 0.5
	done
}

echo "==> building"
go build -o bin/loginer ./cmd/loginer

for app in "${apps[@]}"; do
	read -r name port api <<<"$app"
	(cd "web/$name" && API_URL=$api exec node_modules/.bin/vite dev) >"$logs/$name.log" 2>&1 &
	pids+=($!)
	ready "$name" "$port" $! &
	pids+=($!)
done

echo "==> Loginer — app logs in .logs/"
bin/loginer &
pids+=($!)
wait $!
