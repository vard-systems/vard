#!/usr/bin/env bash
#
# Dump the stack's database into deploy/backups (make db-backup), keeping the
# newest $KEEP dumps (14 by default). Safe while the stack serves: pg_dump
# reads one consistent snapshot.
#
# Restore one into a stopped API with:
#   docker compose -f deploy/compose.yaml stop api
#   docker compose -f deploy/compose.yaml exec -T postgres \
#     pg_restore -U vard -d vard --clean --if-exists <deploy/backups/vard-….dump
#   docker compose -f deploy/compose.yaml start api
#
# A dump holds every user and every sealed secret: keep it as carefully as
# deploy/.env, and away from this machine too.

set -euo pipefail
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

need docker "https://docs.docker.com/engine/install"

keep="${KEEP:-14}"
[[ $keep =~ ^[1-9][0-9]*$ ]] || die "KEEP must be a whole number of dumps, at least 1"

umask 077
mkdir -p deploy/backups

dump="deploy/backups/vard-$(date -u +%Y%m%dT%H%M%SZ).dump"
partial="$dump.partial"

# Written under another name first, so an interrupted dump is never mistaken
# for a whole one, nor counted among those kept.
docker compose -f deploy/compose.yaml exec -T postgres \
	pg_dump -U vard -d vard --format=custom --no-owner >"$partial"
mv "$partial" "$dump"

echo "==> $dump ($(du -h "$dump" | cut -f1))"

# The oldest beyond $keep go.
ls -1t deploy/backups/vard-*.dump | tail -n +"$((keep + 1))" | while IFS= read -r old; do
	rm -f -- "$old"
	echo "    removed $old"
done
