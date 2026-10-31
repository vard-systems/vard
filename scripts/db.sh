#!/usr/bin/env bash
#
# The database named by VARD_DB_DSN (.env), or by DB_URL when set.
#
#   scripts/db.sh create    create it if it is missing
#   scripts/db.sh reset     every migration down and up again: a new installation (asks first)
#   scripts/db.sh psql      open a psql session

set -euo pipefail
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

need psql "install the PostgreSQL client tools"
load_env

url="${DB_URL:-${VARD_DB_DSN:-}}"
[[ -n $url ]] || die "set VARD_DB_DSN in .env, or pass DB_URL"
name="$(sed -E 's|.*/([^/?]+)(\?.*)?$|\1|' <<<"$url")"
server="$(sed -E 's|/[^/?]+(\?.*)?$|/postgres\1|' <<<"$url")" # creating needs another database

case "${1:-}" in
create)
	if [[ "$(psql "$server" -tAc "SELECT 1 FROM pg_database WHERE datname = '$name'")" == 1 ]]; then
		echo "database $name exists"
	else
		psql "$server" -qc "CREATE DATABASE \"$name\"" && echo "created database $name"
	fi
	;;
reset)
	read -rp "Delete every row in \"$name\"? Type yes: " answer
	[[ $answer == yes ]] || die "cancelled"
	VARD_DB_DSN="$url" go run ./cmd/vard migrate reset -yes
	;;
psql) exec psql "$url" ;;
*) die "usage: scripts/db.sh create|reset|psql" ;;
esac
