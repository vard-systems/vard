#!/usr/bin/env bash
#
# Run the whole stack: the Vard API and the id and console apps. Each app
# renders its pages and passes its API paths on to the API
# (src/hooks.server.ts), so the browser talks to one origin per app:
#
#   http://localhost:5173  the sign-in pages, the provider, the account API
#   http://localhost:5174  the panel and the admin API
#
#   scripts/start.sh dev   the apps as Vite dev servers, with hot reload
#   scripts/start.sh prod  the API and the apps built, and run as they are
#                          deployed: bin/vard and `node build`
#
# The production stack in containers runs with `make docker-local`.
#
# The API logs to this terminal; the apps log to .logs/<name>.log.
# Ctrl-C stops everything.

set -euo pipefail
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

mode="${1:-dev}"
[[ $mode == dev || $mode == prod ]] || die "usage: scripts/start.sh dev|prod"

need go "https://go.dev/dl"
need node "https://nodejs.org"
need bun "https://bun.sh"

# These are the stack's addresses whatever .env says: the API builds every URL
# it hands out, and decides which origins it believes, from them. The apps'
# servers, on this machine, are the one proxy it trusts: they send it the
# caller's address.
export VARD_ISSUER=http://localhost:5173
export VARD_ACCOUNT_URL=http://localhost:5173
export VARD_ADMIN_URL=http://localhost:5174
export VARD_TRUSTED_PROXIES=127.0.0.1/32,::1/128
load_env

api_port="${VARD_ADDR:-:8080}" api_port="${api_port##*:}"
admin_port="${VARD_ADMIN_ADDR:-:8081}" admin_port="${admin_port##*:}"

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

echo "==> building ($mode)"
if [[ $mode == prod ]]; then
	go build -trimpath -o bin/vard ./cmd/vard
	for app in "${apps[@]}"; do
		read -r name port _ <<<"$app"
		# The app's own address is built into it (vite.config.ts).
		(cd "web/$name" && ORIGIN="http://localhost:$port" bun run build) >"$logs/$name.log" 2>&1 || {
			tail -n 20 "$logs/$name.log"
			die "web/$name did not build — .logs/$name.log"
		}
	done
else
	go build -o bin/vard ./cmd/vard
fi

for app in "${apps[@]}"; do
	read -r name port api <<<"$app"
	if [[ $mode == prod ]]; then
		# adapter-node's server, on loopback like the dev server, taking a
		# body as large as the API does (a logo, a SAML response): its own
		# default is 512K.
		(cd "web/$name" && NODE_ENV=production HOST=127.0.0.1 PORT=$port \
			BODY_SIZE_LIMIT=1M API_URL=$api exec node build) >>"$logs/$name.log" 2>&1 &
	else
		(cd "web/$name" && API_URL=$api exec node_modules/.bin/vite dev) >"$logs/$name.log" 2>&1 &
	fi
	pids+=($!)
	ready "$name" "http://localhost:$port" $! &
	pids+=($!)
done

echo "==> Vard ($mode) — open http://localhost:5174/login — logs in .logs/"
bin/vard &
pids+=($!)
wait $!
