# Vard

An OAuth 2.0 authorization server and OpenID Connect provider written in Go,
with two web apps:

| App           | What it is                                       |
| ------------- | ------------------------------------------------ |
| `web/id`      | Hosted sign-in pages and each user's own account |
| `web/console` | The admin panel                                  |

**Features:** authorization code with PKCE (always, for clients without
credentials), pushed authorization requests (PAR), the device authorization
grant (sign-in on TVs and command lines), refresh token rotation with reuse
detection and a short grace for a retried refresh, client credentials,
DPoP-bound tokens, native apps'
redirects (private-use schemes, `127.0.0.1` / `[::1]` loopback on any port,
Expo Go's `exp://` for an app marked as in development),
dynamic client registration, userinfo, introspection, revocation, ID tokens
that say how the person signed in (`amr`: `pwd`, `otp`, `hwk`, `user`, `mfa`),
logout (RP-initiated, back-channel and front-channel), a consent page, terms
acceptance, social sign-in (Google, Apple, Facebook, Yandex, VK, any
OIDC/OAuth provider), enterprise SSO (OpenID Connect and SAML 2.0),
configurable login flows, emailed one-time codes, TOTP for administrators,
optional two-step verification (authenticator app or passkeys) for users,
passwordless sign-in with a passkey, roles and API scopes, an activity log, and
the sign-in pages in English, with more languages added, translated and made the
default from the panel.

## Quick start

Requirements: Go 1.27.3+, PostgreSQL 14+, Redis 6.2+ (optional but recommended),
[Bun](https://bun.sh) and Node.js 24+.

```sh
make setup    # .env with a secret key, the database, dependencies
make dev      # starts the API and both apps, with hot reload
make start    # the same from production builds (`make start env=dev` is `make dev`)
```

Everything is plain http on localhost. Each app is a SvelteKit server that
renders its pages and passes the API paths it calls on to the API, so each
app is one origin and the browser never talks to the API directly.

| What    | `make dev`                    | `make docker-local`             |
| ------- | ----------------------------- | ------------------------------- |
| id      | http://localhost:5173         | http://localhost:8080           |
| console | http://localhost:5174/login   | http://localhost:8081/login     |
| API     | `:8080` public, `:8081` admin | inside the stack, not published |

Open the console and create the first administrator with the one-time setup
token the API writes to its log at startup (`setup_token=…`, at WARN), or, on a
server without a browser at hand, `bin/vard admin create -email you@example.com
-first-name You` with the password on standard input. Run `make` to see every
target.

## Configuration

Every setting is an environment variable, read from `.env` (real environment
variables win). `.env.example` documents all of them. The ones that matter most:

| Variable                   | Purpose                                                                                  |
| -------------------------- | ---------------------------------------------------------------------------------------- |
| `VARD_SECRET_KEY`          | Encrypts signing keys and secrets at rest. **Back it up**: losing it signs everyone out. |
| `VARD_SECRET_KEY_PREVIOUS` | The key `VARD_SECRET_KEY` replaced, while rotating; it opens and never seals.            |
| `VARD_PWNED_PASSWORDS_URL` | The Pwned Passwords range API new passwords are checked against; empty for none.         |
| `VARD_DB_DSN`              | PostgreSQL connection string.                                                            |
| `VARD_ISSUER`              | Public URL of the provider (the id app's origin).                                        |
| `VARD_ADMIN_URL`           | The console's URL; only this origin may change things via the admin API.                 |
| `VARD_REDIS_HOST`          | Redis for caching, sessions and shared rate limits. Empty runs without.                  |
| `VARD_TRUSTED_PROXIES`     | Who may say who the caller is: the apps' servers, and nothing else.                      |
| `VARD_RATE_LIMIT`          | Attempts per minute per address on password and email endpoints.                         |
| `VARD_ADMIN_MFA`           | `required` (the default) or `optional`, seeded on first start.                           |

Mail, admin MFA and one-time code settings are seeded from the environment on
the first start and are managed in the panel afterwards.

## Deploying

The API is one binary (`make build-api` → `bin/vard`); it applies its
migrations on start. Each app is a Node server (`make build-web`, then `node build`
in its directory, with `API_URL` pointing at its listener).

`compose.yaml` is the production stack: Postgres, Redis, the API and the two
apps, speaking plain HTTP; TLS and routing are the infrastructure's.
Only the apps publish ports,
on the host's loopback; Postgres and Redis are on an internal network the
apps cannot reach. [`DEPLOY.md`](DEPLOY.md) walks through a
server from DNS to backups:

```sh
make docker-env public=https://id.example.com admin=https://admin.example.com
make docker-up       # build and start; waits until everything is healthy
```

Then route the two sites to the two published ports.

To try the same stack on one machine (served on `http://localhost:8080` and
`http://localhost:8081`):

```sh
make docker-local    # generates compose.env, builds and starts everything
make docker-reset    # stops it and drops its data
```

Without Docker, run `bin/vard` and the two `node build` servers, with the
API's `VARD_TRUSTED_PROXIES` set to the apps' addresses. Each API listener
answers `/healthz` (the process is up) and `/readyz` (its database answers),
for whatever starts and routes to it.

### Before going live

- [ ] `PUBLIC_URL` and `ADMIN_URL` are `https` (session cookies become Secure
      `__Host-` cookies).
- [ ] The admin host is reachable only from staff networks or a VPN.
- [ ] `VARD_SECRET_KEY` is stored in a secret manager.
- [ ] SMTP is configured, or reset emails only reach the log.
- [ ] Postgres is backed up (`make db-backup` from cron), and the dumps leave
      the machine. A database reached over a network is `sslmode=verify-full`
      (the server warns at start about one that is not encrypted).
- [ ] Both sites are served over https, and the apps publish on loopback only (`ID_PUBLISH`, `CONSOLE_PUBLISH`).
- [ ] The firewall lets in 22, 80 and 443 only.
- [ ] `VARD_ADMIN_MFA=required`, and at least two super admins exist.
- [ ] Web applications register `https` redirect URIs; native apps a
      private-use scheme (`com.example.app:/callback`) or loopback.

## Security model in brief

- Every secret handed out (codes, refresh tokens, session cookies, reset links)
  is 256 random bits stored only as a hash — with one exception: a refresh
  token that has just replaced another is also kept sealed with
  `VARD_SECRET_KEY`, for the 30-second grace below and cleared within another
  30 seconds.
- Signing keys and stored secrets are sealed with a key derived from
  `VARD_SECRET_KEY` (HKDF, AES-256-GCM), each bound to the table, column and
  row it is kept in, and marked with its key, so the key can be rotated.
- Signing keys rotate automatically and stay published until their tokens expire.
- Access tokens are short-lived JWTs that name the session and refresh family
  they were issued in; ending either ends them at this server's endpoints.
- Passwords use Argon2id (OWASP's parameters), at least 12 characters (14 for
  administrators) and at most 128; unknown accounts take as long to refuse
  as wrong passwords. New ones can be checked against known breaches.
- Sign-in attempts and one-time-code guesses are counted atomically in the
  database, so parallel guessing cannot slip past them. Sign-ins, users'
  and administrators' alike, are throttled — each attempt past a few
  failures waits longer — rather than locked.
- The admin API runs on its own listener and is never served on the public one.
- Outbound calls to identity providers refuse private, link-local and cloud
  metadata addresses.
- A refresh token presented again after it was replaced revokes its whole
  family — unless it comes within 30 seconds, when it is a retry (a lost
  answer, two tabs at once) and is given the same token that replaced it.
  Every copy a client holds stays good, the retry is held to the same client
  authentication and DPoP binding as any refresh, and the
  family still has exactly one live token.
- A refresh token is issued only for `offline_access`, and only to an
  application an administrator gave the refresh-token grant, for APIs that
  allow offline access. OpenID Connect Core §11 asks for `prompt=consent`
  instead; here the consent step is what asks the person, so a login flow
  without it grants `offline_access` on the administrator's say alone — add
  the consent step to a flow used by applications you do not run yourself.
- Setting up a fresh panel takes the one-time token the server logged, not just
  reaching the console first; it works once, and only while nobody is an
  administrator.
- An address is proved before an account is used: new login flows require a
  verified address, social sign-in makes no account for an address the
  provider has not verified, and the first proof of an address (a reset or
  verification link) ends everything the account held before it — linked
  identities, second factors, recovery codes, consents, sessions and refresh
  tokens — so whoever registered somebody else's address first keeps nothing.
- An SSO connection links to an existing account only at a domain it lists
  and has proved with a DNS TXT record, and the first time it is linked to an
  account with a second factor, that factor is asked for.
- **Consent is a login-flow step.** Without it, every application registered
  in the panel is trusted to sign users in, which is right for an
  organisation's own apps; a flow with the Consent step asks each user once
  for what an application wants. Registering an application — in the panel or
  through `/oauth2/register` with an initial access token — is an admin
  permission either way.

## Development

```sh
make check              # lint (formatting, vet, staticcheck, both apps) and every unit test
make fmt                # format everything in place
make test-integration   # tests against a real Postgres and Redis (throwaway databases)
make ci                 # all of the above, and the Docker stack's config
make bench              # Go benchmarks
make vuln               # known vulnerabilities in the Go code and its modules
```

`.github/workflows/ci.yml` runs what `make ci` does on every push to `main`
and every pull request.

[`CLAUDE.md`](CLAUDE.md) is the contributor guide: where things live, how each
kind of change is made, and the security invariants every change keeps.

## Project layout

```
cmd/vard                the server, the migration runner (vard migrate), and
                           the first administrator from the shell (vard admin create)
internal/                  the Go code, in layers: domain (the models), infrastructure
                           (every query, Redis, mail), application (the provider,
                           administrators' sign-in) and interfaces (the HTTP API)
migrations/                the schema and every change to it, in order
i18n/                      the English that ships: sign-in pages, API messages
web/id, web/console        the two SvelteKit apps; each keeps its own brand in
                           src/lib/brand.ts, the server in internal/brand, and
                           `make rename` renames all three
compose.yaml, Dockerfile   the Docker stack (web/Dockerfile for the apps);
                           DEPLOY.md, the server guide
scripts/                   what the Makefile runs
```

## Third-party assets

- Both apps bundle two typefaces and no others: Chirp for text and Google Sans
  Code for code. Google Sans Code is under the SIL Open Font License 1.1 (the
  licence is beside the files).
- Both apps' icons are the [Remix Icon](https://remixicon.com) font, version
  4.8.0, under the Apache License 2.0 (later versions use the Remix Icon
  License); the sign-in providers' marks in it remain their owners'
  trademarks.
- The favicons (`internal/brand/favicon.svg`, `web/*/src/lib/assets/favicon.svg`)
  are the Svelte logo, a placeholder until the project has its own mark.
- **Chirp is X's proprietary typeface, not an open font**, and both apps serve
  it, the sign-in pages to anyone: replace it, or confirm you have a licence,
  before distributing either publicly.

## License

[MIT](LICENSE)
