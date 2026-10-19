#!/usr/bin/env bash
#
# Start the production stack on this machine: deploy/compose.yaml, served on
# http://localhost:8080 and http://localhost:8081. Safe to run again.

set -euo pipefail
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

need docker "https://docs.docker.com/engine/install"
need openssl "install openssl"

compose=(docker compose -f deploy/compose.yaml)

# set_value KEY VALUE — in deploy/.env, whether or not the key is there yet.
# The delimiter is | because a base64 secret can contain / but never that.
set_value() {
	grep -qE "^$1=" deploy/.env || echo "$1=" >>deploy/.env
	sed -i.bak "s|^$1=.*|$1=$2|" deploy/.env && rm -f deploy/.env.bak
}

# An existing deploy/.env is left alone: its LOGINER_SECRET_KEY is what the
# stored signing keys are sealed with, and its POSTGRES_PASSWORD is the one
# the database was initialised with — Postgres reads that only once, on an
# empty data directory, so replacing it here would lock the API out.
if [[ ! -f deploy/.env ]]; then
	umask 077
	cp deploy/.env.example deploy/.env

	set_value LOGINER_SECRET_KEY "$(openssl rand -base64 32)"
	set_value POSTGRES_PASSWORD "$(openssl rand -hex 32)"
	# So the first administrator can be created without an authenticator app.
	# A server runs this as required; deploy/compose.yaml asks for that.
	set_value LOGINER_ADMIN_MFA optional

	echo "created deploy/.env for http://localhost:8080 — it is gitignored"
fi

# The addresses are this machine's whatever deploy/.env says, including one
# made when the stack was served on https://id.localhost.
set_value PUBLIC_URL http://localhost:8080
set_value ADMIN_URL http://localhost:8081

# Redis keeps nothing on disk and reads its password at every start, so one
# can be filled in at any time, including into a deploy/.env made before it
# was required.
if ! grep -qE '^REDIS_PASSWORD=.+' deploy/.env; then
	set_value REDIS_PASSWORD "$(openssl rand -hex 32)"
fi

"${compose[@]}" up -d --build --wait

cat <<'TEXT'

Ready.

  http://localhost:8081/admin/login   the panel — makes the first administrator
  http://localhost:8080/login         the sign-in pages

  make deploy-local-logs   follow the API
  make deploy-local-down   stop it and drop its data
TEXT
