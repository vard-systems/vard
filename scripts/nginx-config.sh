#!/usr/bin/env bash
#
# Make deploy/nginx/vard.conf from deploy/nginx/vard.conf.template, for the
# addresses and published ports in deploy/.env (make nginx-config).

set -euo pipefail
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

[[ -f deploy/.env ]] || die "no deploy/.env: make docker-env public=… admin=… first"

# value KEY [DEFAULT] — a setting's value in deploy/.env, read without
# running the file: it holds secrets and is not a script.
value() {
	local found
	found="$(sed -n "s/^$1=//p" deploy/.env | tail -n 1)"
	echo "${found:-${2:-}}"
}

# host_of URL — the host, without a port: what nginx's server_name and the
# certificate name.
host_of() {
	local rest="${1#https://}"
	echo "${rest%%[:/]*}"
}

public="$(value PUBLIC_URL)" admin="$(value ADMIN_URL)"
for address in "$public" "$admin"; do
	[[ $address == https://* ]] || die "deploy/.env: $address is not https; the proxy is what serves https"
done

id_host="$(host_of "$public")"
console_host="$(host_of "$admin")"
id_upstream="$(value ID_PUBLISH 127.0.0.1:3000)"
console_upstream="$(value CONSOLE_PUBLISH 127.0.0.1:3001)"

for upstream in "$id_upstream" "$console_upstream"; do
	[[ $upstream == 127.0.0.1:* || $upstream == "[::1]:"* ]] ||
		die "deploy/.env publishes an app on $upstream: anything but loopback is reached around the proxy"
done

sed \
	-e "s|{{ID_HOST}}|$id_host|g" \
	-e "s|{{CONSOLE_HOST}}|$console_host|g" \
	-e "s|{{CERT_NAME}}|$id_host|g" \
	-e "s|{{ID_UPSTREAM}}|$id_upstream|g" \
	-e "s|{{CONSOLE_UPSTREAM}}|$console_upstream|g" \
	deploy/nginx/vard.conf.template >deploy/nginx/vard.conf

cat <<EOF
==> deploy/nginx/vard.conf: https://$id_host → $id_upstream, https://$console_host → $console_upstream
    One certificate, named $id_host, covers both:
      sudo certbot certonly --nginx --cert-name $id_host -d $id_host -d $console_host
    Then install the configuration:
      sudo cp deploy/nginx/vard-proxy.conf /etc/nginx/vard-proxy.conf
      sudo cp deploy/nginx/vard.conf /etc/nginx/conf.d/vard.conf
      sudo nginx -t && sudo systemctl reload nginx
EOF
