# Loginer

An OAuth 2.0 authorization server and OpenID Connect provider written in Go,
with two web apps:

| App           | What it is                                       |
| ------------- | ------------------------------------------------ |
| `web/id`      | Hosted sign-in pages and each user's own account |
| `web/console` | The admin panel                                  |

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
make setup    # .env with a secret key, the database, dependencies
make dev      # starts the API and both apps, with hot reload
```

Everything is plain http on localhost. Each app is on its own origin together
with the API paths it calls: under `make dev` the Vite dev server passes those
paths on to the API; otherwise the binary serves the app on the API's own
listener.

| What    | `make dev`                        | `make run`, `make deploy-local`   |
| ------- | --------------------------------- | --------------------------------- |
| id      | http://localhost:5173             | http://localhost:8080             |
| console | http://localhost:5174/admin/login | http://localhost:8081/admin/login |
| API     | `:8080` public, `:8081` admin     | the same ports as the apps        |

Open the console and create the first administrator. Run `make` to see every
target.

## Configuration

Every setting is an environment variable, read from `.env` (real environment
variables win). `.env.example` documents all of them. The ones that matter most:

| Variable                  | Purpose                                                                                  |
| ------------------------- | ---------------------------------------------------------------------------------------- |
| `LOGINER_SECRET_KEY`      | Encrypts signing keys and secrets at rest. **Back it up**: losing it signs everyone out. |
| `LOGINER_DB_DSN`          | PostgreSQL connection string.                                                            |
| `LOGINER_ISSUER`          | Public URL of the provider (the id app's origin).                                        |
| `LOGINER_ADMIN_URL`       | The console's URL; only this origin may change things via the admin API.                 |
| `LOGINER_REDIS_HOST`      | Redis for caching, sessions and shared rate limits. Empty runs without.                  |
| `LOGINER_TRUSTED_PROXIES` | Proxies whose `X-Forwarded-For` is believed. **Must be set behind a proxy.**             |
| `LOGINER_RATE_LIMIT`      | Attempts per minute per address on password and email endpoints.                         |
| `LOGINER_ADMIN_MFA`       | `required` or `optional`, seeded on first start.                                         |

Mail, admin MFA and one-time code settings are seeded from the environment on
the first start and are managed in the panel afterwards.

## Deploying

The server is one binary with both apps compiled in, PocketBase-style: the
id app is served on the public listener beside the provider, the console on
the admin listener beside the admin API, and no Node.js runs in production.
`make build` builds the apps and then `bin/loginer`; it applies its
migrations on start.

`deploy/` holds a production stack: Postgres, Redis and that server, over
plain http. The API publishes `8080` (id app and provider) and `8081` (console
and admin API) — the latter on the host's loopback only, so it is reached over
an SSH tunnel. There is no TLS yet: put a proxy in front for https, and list
it in `TRUSTED_PROXIES`.

```sh
cp deploy/.env.example deploy/.env   # URLs, secrets, SMTP
make deploy-up
```

To try the same stack on one machine (served on `http://localhost:8080` and
`http://localhost:8081`):

```sh
make deploy-local         # generates deploy/.env, builds and starts everything
make deploy-local-down    # stops it and drops its data
```

Without Docker, run `bin/loginer` directly, or behind a TLS proxy that sends
each host to its listener.

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
make web-check          # lint and type-check the two apps
make web-test           # the apps' unit tests
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
