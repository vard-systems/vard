#!/usr/bin/env bash
#
# Run the whole stack: the Loginer API, and the id and console apps as Vite
# dev servers with hot reload. Each dev server passes its API paths on to the
# API, so the browser talks to one origin per app:
#
#   http://localhost:5173  the sign-in pages, the provider, the account API
#   http://localhost:5174  the panel and the admin API
#
# The production builds, served by the binary itself, run with `make run` or
# `make deploy-local`.
#
# The API logs to this terminal; the apps log to .logs/<name>.log.
# Ctrl-C stops everything.

set -euo pipefail
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

need go "https://go.dev/dl"
need node "https://nodejs.org"
need bun "https://bun.sh"

# These are the stack's addresses whatever .env says: the API builds every URL
# it hands out, and decides which origins it believes, from them. Vite's proxy
# is on this machine and vouches for no one, so no proxy is trusted.
export LOGINER_ISSUER=http://localhost:5173
export LOGINER_ACCOUNT_URL=http://localhost:5173
export LOGINER_ADMIN_URL=http://localhost:5174
export LOGINER_TRUSTED_PROXIES=
# The apps come from Vite here, with hot reload, not from the builds compiled
# into the binary.
export LOGINER_WEB_DEV=true
load_env

api_port="${LOGINER_ADDR:-:8080}" api_port="${api_port##*:}"
admin_port="${LOGINER_ADMIN_ADDR:-:8081}" admin_port="${admin_port##*:}"

# name  port  API it calls
apps=(
	"id      5173 http://localhost:$api_port"
	"console 5174 http://localhost:$admin_port"
)

logs="$root/.logs"
mkdir -p "$logs" bin

# A port is taken when anything at all answers on it (curl exit 7: refused).
for port in "$api_port" "$admin_port" 5173 5174; do
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

# ready <name> <url> <pid>: report once it answers, or show why it died.
ready() {
	for _ in $(seq 240); do
		kill -0 "$3" 2>/dev/null || { printf '\n✗ %s stopped — .logs/%s.log:\n' "$1" "$1"; tail -n 20 "$logs/$1.log"; return; }
		curl -s -o /dev/null --max-time 2 "$2" && { echo "✓ $1 on $2"; return; }
		sleep 0.5
	done
}

echo "==> building"
# The binary embeds the apps' builds, so it does not compile without them;
# they are not served here, so any build will do.
make --no-print-directory web-built
go build -o bin/loginer ./cmd/loginer

for app in "${apps[@]}"; do
	read -r name port api <<<"$app"
	(cd "web/$name" && API_URL=$api exec node_modules/.bin/vite dev) >"$logs/$name.log" 2>&1 &
	pids+=($!)
	ready "$name" "http://localhost:$port" $! &
	pids+=($!)
done

echo "==> Loginer — open http://localhost:5174/admin/login — logs in .logs/"
bin/loginer &
pids+=($!)
wait $!
