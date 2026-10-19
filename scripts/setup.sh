#!/usr/bin/env bash
#
# Prepare a fresh checkout: .env with its own secret key, the database and its
# migrations, and the web apps' dependencies. Safe to run again.

set -euo pipefail
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

need go "https://go.dev/dl"
need bun "https://bun.sh"
need node "https://nodejs.org"
need psql "install the PostgreSQL client tools"
need openssl "install openssl"

[[ -f .env ]] || { cp .env.example .env && echo "created .env — review the database settings"; }

# A key that already exists stays: the stored signing keys are sealed with it.
if ! grep -qE '^LOGINER_SECRET_KEY=.{32,}' .env; then
	sed -i.bak '/^LOGINER_SECRET_KEY=/d' .env && rm .env.bak
	echo "LOGINER_SECRET_KEY=$(openssl rand -base64 32)" >>.env
	echo "wrote a new LOGINER_SECRET_KEY to .env"
fi

# A .env from before the server had Redis gets the Redis settings, copied from
# .env.example; one that has them keeps its own, even an empty host.
if ! grep -qE '^LOGINER_REDIS_HOST=' .env; then
	{ echo; sed -n '/^# Redis:/,/^LOGINER_REDIS_PREFIX=/p' .env.example; } >>.env
	echo "added the Redis settings to .env"
fi

# A Redis that is configured has to answer, or the server will not start.
redis_host="$(sed -n 's/^LOGINER_REDIS_HOST=//p' .env | tail -n 1)"
redis_port="$(sed -n 's/^LOGINER_REDIS_PORT=//p' .env | tail -n 1)"
if [[ -n $redis_host ]] && command -v redis-cli >/dev/null; then
	redis-cli -h "$redis_host" -p "${redis_port:-6379}" ping >/dev/null 2>&1 ||
		echo "warning: Redis at $redis_host:${redis_port:-6379} is not answering — start it (brew services start redis), or empty LOGINER_REDIS_HOST in .env"
fi

go mod download
scripts/db.sh create
go run ./cmd/migrate up

# The apps' dependencies.
for app in console id; do (cd "web/$app" && bun install --frozen-lockfile); done

echo "Ready. Start everything with: make dev"
