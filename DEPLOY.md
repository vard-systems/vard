# Deploying Vard on a server

One machine running Docker. The stack speaks plain HTTP; TLS and routing are
the infrastructure's, in front of it.

```
infrastructure ─▶ 127.0.0.1:3000  id       ─┐
               └▶ 127.0.0.1:3001  console  ─┤ edge network
                                   api ─────┘ (and out: mail, SSO, logout)
                                    │
                      postgres, redis   data network (internal, no way out)
```

- **Published:** only the two apps, on loopback. Postgres, Redis and the API
  publish nothing; the apps cannot reach Postgres or Redis.
- **The API runs as `vard_app`**, a database role that reads and writes rows
  and nothing more; it migrates as `vard`, the owner, and makes `vard_app`
  (with `POSTGRES_APP_PASSWORD`) on its first start.
- **The API believes only the two apps** about who is calling
  (`VARD_TRUSTED_PROXIES`, their fixed addresses).
- **Containers** run as unprivileged users with no capabilities, read-only
  filesystems (the apps and the API), `no-new-privileges`, memory and process
  limits, and capped logs.

## What you need

- A Linux server with Docker Engine and the Compose plugin, and git.
- Two DNS names pointing at it: one for the sign-in pages (the issuer, such as
  `id.example.com`) and one for the panel (`admin.example.com`).
- A firewall that lets in 22, 80 and 443 only:
  `sudo ufw allow OpenSSH && sudo ufw allow 80,443/tcp && sudo ufw enable`.

## First deployment

```sh
git clone https://github.com/vard-systems/vard.git && cd vard

# compose.env, with new secrets — back it up: VARD_SECRET_KEY unlocks the
# signing keys, and without it every token and session is lost.
make docker-env public=https://id.example.com admin=https://admin.example.com
$EDITOR compose.env          # the VARD_SMTP_* settings, so mail is sent

make docker-up               # build and start; waits until everything is healthy
```

Route `PUBLIC_URL` to `ID_PUBLISH` and `ADMIN_URL` to `CONSOLE_PUBLISH`,
then open
`https://admin.example.com/new-super-admin` and make the first administrator
straight away: until someone has, whoever opens that page first can.
Every administrator sets up an authenticator app at their first sign-in:
`compose.env` says `VARD_ADMIN_MFA=required`, which a super admin can change
later in Administrators › Two-factor sign-in.

The server refuses to start on plain http addresses other than localhost.
`VARD_ALLOW_INSECURE_HTTP` lets plain http through for a test setup, with a
warning at every start; never set it on a real server. It never lets a weak
`VARD_SECRET_KEY` through: a key of hex digits alone under 64 characters, or
any other key under 43 characters or with fewer than 16 distinct characters,
always stops the server.

## Updating

```sh
git pull
make db-backup               # first, always
make docker-up               # rebuilds what changed, migrates the database on start
```

### Upgrading to 1.1.0

Back up the database first (`make db-backup`). The server applies six
migrations as it starts, each with a down (`make migrate-down` steps back
one at a time):

| Migration                   | What it adds                                                                                      |
| --------------------------- | ------------------------------------------------------------------------------------------------- |
| `00005_dpop_jkt`            | `dpop_key` on authorization requests and codes, for a code bound to a DPoP key                    |
| `00006_step_up`             | a login flow's `require_second_factor` (off for every existing flow); a step-up's session         |
| `00007_remember_session`    | whether a session was "stay signed in" (existing sessions: no), so a step-up keeps its cookie     |
| `00008_sign_in_application` | the application a waiting sign-in with no authorization request is for (a device being allowed)   |
| `00009_sign_in_device`      | the device a step-up was started to allow, which "Not now" denies                                 |
| `00010_sso_second_factor`   | an SSO connection's `second_factor` (`local` for every existing one) and what its provider proves |

What changes for applications and the people signing in:

- **`amr` says `mfa` only after one of the account's own second factors**
  (an authenticator app, a recovery code or a passkey). A password and an
  emailed code is `pwd` and `otp`, no longer `mfa`: the emailed code proves
  only the mailbox a password reset also goes to. An application that
  reads `mfa` to decide anything gets it less often; one that needs it
  should sign in through a login flow that requires a second factor.
- **A login flow can require a second factor** (the Two-step verification
  step, "Require it"; off for every existing flow). Someone without one
  sets one up while signing in, once their address is confirmed; someone
  already signed in without one is asked for it in the session they have
  (`prompt=none` gets `interaction_required`), and may leave it with "Not
  now", which tells the application `access_denied`. Allowing a device is
  held to the same flow.
- **Refresh families end under a requiring flow.** Turning the requirement
  on for a flow in use refuses, at the next refresh, every refresh token
  whose sign-in never passed a second factor (`invalid_grant`, a retry
  within the 30-second grace included): the application signs the person
  in again. Nothing is revoked at once, and families that passed a factor
  carry on.
- **Single sign-on** under a requiring flow asks for the account's own
  factor (each connection's `local`, the default), or, for a connection set
  to trust its provider, takes the provider's own when its signed answer
  shows one of the values the connection lists (OpenID Connect `amr`,
  default `mfa`; SAML authentication contexts, which you list yourself).
  Otherwise it falls back to the account's own factor and the activity log
  records it.
- **Every second factor added** from the account page or while signing in
  emails its account a notice ("Two-step verification turned on" on the
  Mail page, where its text can be changed). With SMTP not configured it
  only reaches the log, like the other emails.
- **`dpop_jkt`** binds an authorization code to a DPoP key (RFC 9449 §10),
  as does pushing the request (PAR) with a DPoP proof. A bound code is
  exchanged only with a proof from that key; tried without it, the code is
  spent. `dpop_jkt` in a plain authorization request crosses the browser
  and can be stripped, so a client that needs the binding pushes its
  request with a proof.
- **RSA keys from outside are capped at 4096 bits.** A DPoP proof, or any
  key the server is handed, with a longer modulus is refused: verifying
  under a huge one costs the server. Keys this server makes are unchanged.
- **Social sign-in checks the provider's signature** on its id_token where
  the provider publishes keys (Apple, Google), fetched through the same
  guarded client as SSO. A provider's keys are fetched again only for a
  token naming a key not held, at most once a minute per key set, so a
  stream of bad tokens cannot make the server hammer the provider.
- The social and SSO callbacks are rate limited like their start routes,
  and a redirect URI's own query comes back byte for byte.

### Upgrading to 1.0.0

1.0.0 refuses to start with a weak `VARD_SECRET_KEY`, whatever else is set: a
key made only of hex digits needs at least 64 characters, and any other key at
least 43 characters and 16 distinct ones. A key made with `openssl rand -base64
32` (what `make docker-env` writes) or `openssl rand -hex 32` passes. The key
cannot be changed: under a new one nothing stored opens, so an installation
with a weak key starts again with a fresh database.

An installation from before the migrations were squashed into one setup
(`migrations/setup`) cannot be stepped forward: the server refuses its
database and says so. Back it up and start on a new, empty database; its
rows are not carried over.

## Backups

`make db-backup` writes `backups/vard-<time>.dump` and keeps the newest
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

Everything is in `compose.env`; `compose.env.example` explains each. The ones
a server must not change: `VARD_SECRET_KEY` and `POSTGRES_PASSWORD`, which
the stored keys and the database were made with, and `POSTGRES_APP_PASSWORD`,
which `vard_app` was made with. `ID_PUBLISH` and
`CONSOLE_PUBLISH` stay on loopback: Docker opens a published port past ufw.
