# Deploying Vard on a server

One machine running Docker. The stack speaks plain HTTP; TLS ends at the
proxy or load balancer the platform puts in front — Caddy, Traefik, a cloud
load balancer, a Kubernetes ingress — which is the only thing the internet
reaches.

```
internet ─443─▶ TLS proxy ─▶ 127.0.0.1:3000  id       ─┐
                          └▶ 127.0.0.1:3001  console  ─┤ edge network
                                              api ─────┘ (and out: mail, SSO, logout)
                                               │
                                 postgres, redis   data network (internal, no way out)
```

- **Published:** only the two apps, on loopback. Postgres, Redis and the API
  publish nothing; the apps cannot reach Postgres or Redis.
- **The API believes only the two apps** about who is calling
  (`VARD_TRUSTED_PROXIES`), and the apps believe only the proxy
  (`APP_ADDRESS_HEADER`, which it must overwrite).
- **Containers** run as unprivileged users with no capabilities, read-only
  filesystems (the apps and the API), `no-new-privileges`, memory and process
  limits, and capped logs.

## What the TLS proxy must do

- Serve `PUBLIC_URL` and `ADMIN_URL` over https, and pass them to
  `ID_PUBLISH` (127.0.0.1:3000) and `CONSOLE_PUBLISH` (127.0.0.1:3001).
- Redirect plain http to https. HSTS is sent by the apps and the API
  themselves once their URLs are https.
- **Overwrite** `X-Real-IP` (`APP_ADDRESS_HEADER`) with the connecting
  address, never pass on one a client sent: it is who the rate limits, the
  sign-in throttle and the activity log believe is calling.
- Take request bodies of at least 1 MiB (a logo, a SAML response).
- Only for mutual TLS (below): on its own host alone, ask for a client
  certificate and pass it in `APP_CLIENT_CERT_HEADER`, URL-escaped PEM;
  everywhere else, **remove that header** from the request, or anyone could
  claim any certificate.

With Caddy, for example (`/etc/caddy/Caddyfile`):

```
id.example.com {
	reverse_proxy 127.0.0.1:3000 {
		header_up X-Real-IP {remote_host}
		header_up -X-Client-Certificate
	}
}
admin.example.com {
	reverse_proxy 127.0.0.1:3001 {
		header_up X-Real-IP {remote_host}
		header_up -X-Client-Certificate
	}
}
```

## What you need

- A Linux server with Docker Engine and the Compose plugin, git, and the TLS
  proxy.
- Two DNS names pointing at it: one for the sign-in pages (the issuer, such as
  `id.example.com`) and one for the panel (`admin.example.com`).
- A firewall that lets in 22, 80 and 443 only:
  `sudo ufw allow OpenSSH && sudo ufw allow 80,443/tcp && sudo ufw enable`.

## First deployment

```sh
git clone https://github.com/vard-systems/vard.git && cd vard

# deploy/.env, with new secrets — back it up: VARD_SECRET_KEY unlocks the
# signing keys, and without it every token and session is lost.
make docker-env public=https://id.example.com admin=https://admin.example.com
$EDITOR deploy/.env          # the VARD_SMTP_* settings, so mail is sent

make docker-up               # build and start; waits until everything is healthy
```

Point the TLS proxy at the two published ports, then open
`https://admin.example.com/login` and make the first administrator with the
setup token the API logged (`make docker-logs service=api`, `setup_token=…`).
They are asked to set up an authenticator app straight away.

The panel can be closed to everyone but the staff: list their networks in
`VARD_ADMIN_NETWORKS`.

## Mutual TLS

For clients that authenticate with a certificate (`tls_client_auth`,
`self_signed_tls_client_auth`) or have their tokens bound to one (RFC 8705).
It needs a third DNS name, such as `mtls.id.example.com`: the proxy asks for a
client certificate there and only there, so no browser on the sign-in pages
is ever asked to pick one, and passes on only the token, PAR, revocation,
introspection, device and userinfo endpoints, with the certificate in
`APP_CLIENT_CERT_HEADER`.

```sh
# deploy/.env
MTLS_URL=https://mtls.id.example.com
# Only for tls_client_auth: the CAs clients' certificates chain to,
# intermediates too, as a PEM file in deploy/mtls/.
MTLS_CA_FILE=/etc/vard/mtls/clients-ca.pem

make docker-up
```

A `tls_client_auth` client is best named by a SAN (DNS name, URI, IP or
email). Its subject DN works too, but is compared as text in the form RFC
4514 writes it (`CN=app,O=Example`), not by full distinguished-name
matching, so a certificate that encodes or spaces it otherwise does not
match.

The discovery document then lists the endpoints there as
`mtls_endpoint_aliases`, and the panel's Credentials section offers the two
certificate methods and binding tokens to the certificate.

## Updating

```sh
git pull
make db-backup               # first, always
make docker-up               # rebuilds what changed, migrates the database on start
```

## Backups

`make db-backup` writes `deploy/backups/vard-<time>.dump` and keeps the newest
14 (`KEEP=30 make db-backup` for more). Run it from cron, and copy the dumps
off the machine: they hold every user and every sealed secret.

```
15 3 * * * cd /srv/vard && make db-backup >>/var/log/vard-backup.log 2>&1
```

Restoring is in the comment at the top of `scripts/db-backup.sh`.

## Checking it

```sh
make docker-ps                         # every container healthy
make docker-logs service=api           # the API's log
curl -sI https://id.example.com/.well-known/openid-configuration   # 200, with HSTS
```

## Settings

Everything is in `deploy/.env`; `deploy/.env.example` explains each. The ones
a server must not change: `VARD_SECRET_KEY` and `POSTGRES_PASSWORD`, which
the stored keys and the database were made with. `ID_PUBLISH` and
`CONSOLE_PUBLISH` stay on loopback: Docker opens a published port past ufw.
