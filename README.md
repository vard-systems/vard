# Loginer

An OAuth 2.0 authorization server and OpenID Connect provider written in Go,
with three web apps:

| App            | What it is                                         |
| -------------- | -------------------------------------------------- |
| `web/id`       | Hosted sign-in pages and each user's own account   |
| `web/console`  | The admin panel                                    |
| `web/docs`     | Documentation for developers integrating with it   |

**Features:** authorization code with PKCE, refresh token rotation with reuse
detection, client credentials, userinfo, logout, revocation, introspection,
social sign-in (Google, Apple, Facebook, Yandex, VK, any OIDC/OAuth provider),
enterprise SSO (OpenID Connect and SAML 2.0), configurable login flows, emailed
one-time codes, TOTP for administrators, roles and API scopes, an activity log,
and translations editable from the panel.

## Quick start

Requirements: Go 1.27+, PostgreSQL 14+, Redis 6.2+ (optional but recommended),
[Bun](https://bun.sh) and Node.js 24+.

```sh
make setup    # creates .env with a secret key, the database, and installs dependencies
make dev      # starts the API and all three apps
```

| What    | URL                                 |
| ------- | ----------------------------------- |
| id      | http://localhost:5173               |
| console | http://localhost:5174/admin/login   |
| docs    | http://localhost:5175               |
| API     | `:8080` public, `:8081` admin       |

Open the console and create the first administrator. Run `make` to see every
target.

## Configuration

Every setting is an environment variable, read from `.env` (real environment
variables win). `.env.example` documents all of them. The ones that matter most:

| Variable                     | Purpose                                                                 |
| ---------------------------- | ----------------------------------------------------------------------- |
| `LOGINER_SECRET_KEY`         | Encrypts signing keys and secrets at rest. **Back it up**: losing it signs everyone out. |
| `LOGINER_DB_DSN`             | PostgreSQL connection string.                                           |
| `LOGINER_ISSUER`             | Public URL of the provider (the id app's origin).                       |
| `LOGINER_ADMIN_URL`          | The console's URL; only this origin may change things via the admin API. |
| `LOGINER_REDIS_HOST`         | Redis for caching, sessions and shared rate limits. Empty runs without. |
| `LOGINER_TRUSTED_PROXIES`    | Proxies whose `X-Forwarded-For` is believed. **Must be set behind a proxy.** |
| `LOGINER_RATE_LIMIT`         | Attempts per minute per address on password and email endpoints.        |
| `LOGINER_ADMIN_MFA`          | `required` or `optional`, seeded on first start.                        |

Mail, admin MFA and one-time code settings are seeded from the environment on
the first start and are managed in the panel afterwards.

## Deploying

`deploy/` holds a production stack: Postgres, Redis, the API, both apps, and
Caddy in front with automatic TLS. Only Caddy publishes ports. The production
`deploy/Caddyfile` is not in the repository: the server provides its own, and
`compose.yaml` mounts it from there. `deploy/Caddyfile.local` shows the routing
it needs.

```sh
cp deploy/.env.example deploy/.env   # URLs, secret key, database and Redis passwords, SMTP
# put the server's Caddyfile at deploy/Caddyfile
make deploy-up
```

To try the exact same stack on one machine (served on `https://id.localhost`
and `https://admin-id.localhost`):

```sh
make deploy-local         # generates deploy/.env, builds and starts everything
make deploy-local-down    # stops it and drops its data
```

A single binary is also fine: `make build` produces `bin/loginer`, which applies
its migrations on start.

### Before going live

- [ ] `PUBLIC_URL` and `ADMIN_URL` are `https` (session cookies become Secure).
- [ ] The admin host is reachable only from staff networks or a VPN.
- [ ] `LOGINER_SECRET_KEY` is stored in a secret manager.
- [ ] SMTP is configured, or reset emails only reach the log.
- [ ] Postgres is backed up.
- [ ] Only the reverse proxy publishes a port.
- [ ] `LOGINER_TRUSTED_PROXIES` names your proxies, and nothing else.
- [ ] `LOGINER_ADMIN_MFA=required`, and at least two super admins exist.
- [ ] Applications register `https` redirect URIs.

## Security model in brief

- Every secret handed out (codes, refresh tokens, session cookies, reset links)
  is 256 random bits stored only as a hash.
- Signing keys and stored secrets are sealed with `LOGINER_SECRET_KEY`.
- Signing keys rotate automatically and stay published until their tokens expire.
- Passwords use bcrypt; unknown accounts take as long to refuse as wrong passwords.
- Lockouts and one-time-code guesses are counted atomically in the database, so
  parallel guessing cannot slip past them.
- The admin API runs on its own listener and is never served on the public one.
- Outbound calls to identity providers refuse private, link-local and cloud
  metadata addresses.
- **There is no consent screen.** Every application registered in the panel is
  trusted to sign users in, which is right for an organisation's own apps.
  Registering an application is therefore an admin permission.

## Development

```sh
make check              # gofmt, go vet, Go unit tests
make test-integration   # tests against a real Postgres and Redis (throwaway databases)
make web-check          # lint and type-check the three apps
make bench              # Go micro-benchmarks
make docs               # regenerate the API reference after changing handlers
```

[`CLAUDE.md`](CLAUDE.md) is the contributor guide: where things live, how each
kind of change is made, and the security invariants every change keeps. What was tested before this release, and what was fixed, is in
[`TESTING.md`](TESTING.md).

## Third-party assets

The console bundles three typefaces. Google Sans Code and Poppins are under
the SIL Open Font License 1.1 (see `OFL.txt` beside each). **Chirp is X's
proprietary typeface and is not open source**: replace it, or confirm you have
a licence, before distributing the console publicly.

## License

[MIT](LICENSE)
