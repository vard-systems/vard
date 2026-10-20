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
| `LOGINER_TRUSTED_PROXIES` | Who may say who the caller is: the apps' servers, and nothing else.                      |
| `LOGINER_RATE_LIMIT`      | Attempts per minute per address on password and email endpoints.                         |
| `LOGINER_ADMIN_MFA`       | `required` or `optional`, seeded on first start.                                         |

Mail, admin MFA and one-time code settings are seeded from the environment on
the first start and are managed in the panel afterwards.

## Deploying

The API is one binary (`make build-api` → `bin/loginer`); it applies its
migrations on start. Each app is a Node server (`make build-web`, then `node build`
in its directory, with `API_URL` pointing at its listener).

`deploy/` holds a production stack: Postgres, Redis, the API and the two apps,
over plain http. Only the apps publish ports: the id app on `8080`, the
console on `8081` on the host's loopback only, so it is reached over an SSH
tunnel. The API is not published. There is no TLS yet: put a proxy in front
of the apps for https.

```sh
cp deploy/.env.example deploy/.env   # URLs, secrets, SMTP
make docker-up
```

To try the same stack on one machine (served on `http://localhost:8080` and
`http://localhost:8081`):

```sh
make docker-local    # generates deploy/.env, builds and starts everything
make docker-reset    # stops it and drops its data
```

Without Docker, run `bin/loginer` and the two `node build` servers, with the
API's `LOGINER_TRUSTED_PROXIES` set to the apps' addresses.

### Before going live

- [ ] `PUBLIC_URL` and `ADMIN_URL` are `https` (session cookies become Secure).
- [ ] The admin host is reachable only from staff networks or a VPN.
- [ ] `LOGINER_SECRET_KEY` is stored in a secret manager.
- [ ] SMTP is configured, or reset emails only reach the log.
- [ ] Postgres is backed up.
- [ ] A TLS proxy is in front of the apps, and only it publishes a port.
- [ ] The apps' Node servers read the caller from that proxy (`ADDRESS_HEADER`,
      `XFF_DEPTH`), and the API trusts only the apps (`LOGINER_TRUSTED_PROXIES`).
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
cmd/loginer, cmd/migrate   the server, and the migration runner
internal/                  the Go code: api (HTTP handlers), oidc (the provider),
                           store (every query), model (the schema), and the rest
migrations/                the schema and every change to it, in order
i18n/                      the sign-in pages' translations, the API's messages
web/id, web/console        the two SvelteKit apps
deploy/                    the Docker stack
scripts/                   what the Makefile runs
```

## Third-party assets

- The sign-in pages bundle Roboto and Roboto Mono, and the console Google Sans
  Code and Poppins, all under the SIL Open Font License 1.1 (the licence is
  beside each font).
- The console's icons are [Remix Icon](https://remixicon.com), under the
  Apache License 2.0; `make generate` copies the ones it uses from the
  `remixicon` package.
- **The console also bundles Chirp, X's proprietary typeface, which is not open
  source**: replace it, or confirm you have a licence, before distributing the
  console publicly.

## License

[MIT](LICENSE)
