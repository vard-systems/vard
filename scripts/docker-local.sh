#!/usr/bin/env bash
#
# Start the production stack on this machine: compose.yaml, served on
# http://localhost:8080 and http://localhost:8081. Safe to run again.

set -euo pipefail
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

need docker "https://docs.docker.com/engine/install"
need openssl "install openssl"

compose=(docker compose -f compose.yaml --env-file compose.env)

# set_value KEY VALUE — in compose.env, whether or not the key is there yet.
# The delimiter is | because a base64 secret can contain / but never that.
set_value() {
	grep -qE "^$1=" compose.env || echo "$1=" >>compose.env
	sed -i.bak "s|^$1=.*|$1=$2|" compose.env && rm -f compose.env.bak
}

# An existing compose.env is left alone: its VARD_SECRET_KEY is what the
# stored signing keys are sealed with, and its POSTGRES_PASSWORD is the one
# the database was initialised with — Postgres reads that only once, on an
# empty data directory, so replacing it here would lock the API out.
if [[ ! -f compose.env ]]; then
	umask 077
	cp compose.env.example compose.env

	set_value VARD_SECRET_KEY "$(openssl rand -base64 32)"
	set_value POSTGRES_PASSWORD "$(openssl rand -hex 32)"
	set_value POSTGRES_APP_PASSWORD "$(openssl rand -hex 32)"
	# Each administrator sets up an authenticator from their profile, if at all.
	set_value VARD_ADMIN_MFA optional

	echo "created compose.env for http://localhost:8080 — it is gitignored"
fi

# The addresses are this machine's whatever compose.env says, including one
# made when the stack was served on https://id.localhost.
set_value PUBLIC_URL http://localhost:8080
set_value ADMIN_URL http://localhost:8081
# The apps publish straight onto this machine's ports.
set_value ID_PUBLISH 127.0.0.1:8080
set_value CONSOLE_PUBLISH 127.0.0.1:8081

# Redis keeps nothing on disk and reads its password at every start, so one
# can be filled in at any time, including into a compose.env made before it
# was required.
if ! grep -qE '^REDIS_PASSWORD=.+' compose.env; then
	set_value REDIS_PASSWORD "$(openssl rand -hex 32)"
fi
# Likewise the runtime role's, which the server makes on its next start.
if ! grep -qE '^POSTGRES_APP_PASSWORD=.+' compose.env; then
	set_value POSTGRES_APP_PASSWORD "$(openssl rand -hex 32)"
fi

"${compose[@]}" up -d --build --wait

cat <<'TEXT'

Ready.

  http://localhost:8081/login         the panel — makes the first administrator
  http://localhost:8080/login         the sign-in pages

  make docker-logs service=api   follow the API
  make docker-reset              stop it and drop its data
TEXT
