#!/usr/bin/env bash
#
# Make compose.env for a server: scripts/docker-env.sh https://id.example.com
# https://admin.example.com (make docker-env public=… admin=…).
#
# The addresses are the two sites as browsers reach them, through the TLS
# proxy in front (DEPLOY.md). Every secret is generated here, once: an existing
# compose.env keeps its secrets — VARD_SECRET_KEY seals the signing keys and
# POSTGRES_PASSWORD is the one the database was made with — and only its
# addresses are changed.

set -euo pipefail
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

need openssl "install openssl"

[[ $# -eq 2 ]] || die "usage: scripts/docker-env.sh https://id.example.com https://admin.example.com"
public="${1%/}" admin="${2%/}"

# https, a host, and nothing after it: the issuer is compared exactly, and a
# session cookie is sent back only over https.
for address in "$public" "$admin"; do
	[[ $address =~ ^https://[a-z0-9]([a-z0-9.-]*[a-z0-9])?(:[0-9]+)?$ ]] ||
		die "$address is not an https address with a host and nothing after it"
done
[[ $public != "$admin" ]] || die "the two sites need two addresses: the sign-in pages and the panel are separate origins"

# set_value KEY VALUE — in compose.env, whether or not the key is there yet.
# The delimiter is | because a base64 secret can contain / but never that.
set_value() {
	grep -qE "^$1=" compose.env || echo "$1=" >>compose.env
	sed -i.bak "s|^$1=.*|$1=$2|" compose.env && rm -f compose.env.bak
}

umask 077
if [[ ! -f compose.env ]]; then
	cp compose.env.example compose.env
	set_value VARD_SECRET_KEY "$(openssl rand -base64 32)"
	set_value POSTGRES_PASSWORD "$(openssl rand -hex 32)"
	set_value POSTGRES_APP_PASSWORD "$(openssl rand -hex 32)"
	set_value REDIS_PASSWORD "$(openssl rand -hex 32)"
	echo "==> made compose.env with new secrets: back it up somewhere safe"
else
	echo "==> compose.env is there: its secrets are kept"
fi
chmod 600 compose.env

# A compose.env from before the server ran as its own role gets one; the
# server makes the role on its next start.
if ! grep -qE '^POSTGRES_APP_PASSWORD=.+' compose.env; then
	set_value POSTGRES_APP_PASSWORD "$(openssl rand -hex 32)"
fi

set_value PUBLIC_URL "$public"
set_value ADMIN_URL "$admin"

cat <<EOF
==> the sign-in pages: $public, the panel: $admin
    Next: fill in the SMTP settings in compose.env, then
      point the TLS proxy at the published ports (DEPLOY.md)
      make docker-up        build and start the stack
EOF
