# Deploying Vard on a server

One machine running Docker, with nginx on the host in front: nginx ends TLS
and is the only thing the internet reaches.

```
internet ─443─▶ nginx (host) ─▶ 127.0.0.1:3000  id       ─┐
                             └▶ 127.0.0.1:3001  console  ─┤ edge network
                                                 api ─────┘ (and out: mail, SSO, logout)
                                                  │
                                    postgres, redis   data network (internal, no way out)
```

- **Published:** only the two apps, on loopback. Postgres, Redis and the API
  publish nothing; the apps cannot reach Postgres or Redis.
- **The API believes only the two apps** about who is calling
  (`VARD_TRUSTED_PROXIES`), and the apps believe only nginx (`X-Real-IP`,
  which nginx overwrites).
- **Containers** run as unprivileged users with no capabilities, read-only
  filesystems (the apps and the API), `no-new-privileges`, memory and process
  limits, and capped logs.

## What you need

- A Linux server with Docker Engine and the Compose plugin, nginx, certbot
  and git.
- Two DNS names pointing at it: one for the sign-in pages (the issuer, such as
  `id.example.com`) and one for the panel (`admin.example.com`).
- A firewall that lets in 22, 80 and 443 only:
  `sudo ufw allow OpenSSH && sudo ufw allow 'Nginx Full' && sudo ufw enable`.

## First deployment

```sh
git clone https://github.com/vard-systems/vard.git && cd vard

# deploy/.env, with new secrets — back it up: VARD_SECRET_KEY unlocks the
# signing keys, and without it every token and session is lost.
make docker-env public=https://id.example.com admin=https://admin.example.com
$EDITOR deploy/.env          # the VARD_SMTP_* settings, so mail is sent

make docker-up               # build and start; waits until everything is healthy

# nginx: one certificate for both names, then the sites.
make nginx-config
sudo certbot certonly --nginx --cert-name id.example.com -d id.example.com -d admin.example.com
sudo cp deploy/nginx/vard-proxy.conf /etc/nginx/vard-proxy.conf
sudo cp deploy/nginx/vard.conf /etc/nginx/conf.d/vard.conf
sudo nginx -t && sudo systemctl reload nginx
```

Then open `https://admin.example.com/login`: the first visit makes the first
administrator, who is asked to set up an authenticator app straight away
(`VARD_ADMIN_MFA=required`).

The panel can be closed to everyone but the staff: list their addresses in
the `allow` lines of `deploy/nginx/vard.conf`'s second site.

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
