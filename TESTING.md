# Test report

What was tested before release, what was found, and what was fixed.
Date: 3 October 2026. Environment: macOS (Apple Silicon), Go 1.27.1,
PostgreSQL 18, Redis, Bun 1.4, Docker 29.

## Summary

| Area                    | Result | Notes                                                        |
| ----------------------- | ------ | ------------------------------------------------------------ |
| Unit tests (Go)         | ✅ Pass | 219 tests                                                    |
| Unit tests (web apps)   | ✅ Pass | 100 tests (console 61, id 39)                                |
| Integration tests       | ✅ Pass | 101 tests + 44 subtests against real Postgres and Redis      |
| Codebase review         | ✅ Done | 12 fixes (below)                                             |
| Performance             | ✅ Done | Token endpoint 5× faster after a fix; benchmarks added       |
| Security                | ✅ Done | Dependency scans, static analysis, manual review; 3 fixes    |
| Scalability             | ✅ Pass | New tests run two server instances on one database          |
| Proxy                   | ✅ Pass | New tests put a real reverse proxy in front of the server    |
| Easy setup              | ✅ Pass | Fresh copy, `make setup` twice, build and start              |
| Easy deployment         | ✅ Pass | Full Docker stack built and started; 1 bug fixed             |
| Lint, types, formatting | ✅ Pass | `make check`, `make web-check`, `make docs-check`            |
| Code comments           | ✅ Done | 2,520 comment lines removed (25%)                            |

## How to run everything

```sh
make check              # Go: format, vet, unit tests
make test-integration   # Go: tests against Postgres and Redis
make web-check          # web apps: lint and type-check
cd web/console && bun run test   # web unit tests (also web/id)
make bench              # micro-benchmarks
make docs-check         # API reference matches the code
```

`make test-integration` creates a throwaway database per test and uses its own
Redis key prefix, so it never touches your data. Set
`LOGINER_LOAD_REQUESTS=20000` to run the load test harder.

## What was tested

### Unit tests
All existing Go and web unit tests pass. New ones added for the outbound
address filter, the issuer loopback check, the admin full-name format, and
the count wording helper in the console.

### Integration tests
The existing suite covers the whole OAuth/OIDC flow, refresh token rotation,
code replay, social sign-in, SSO (OIDC and SAML), MFA, emailed codes, login
flows, languages, mail, sessions and Redis cache invalidation. All 89 passed.
Twelve new top-level tests were added (below), bringing it to 101.

### Performance
Micro-benchmarks (one core, `make bench`):

| Operation                         | Time        |
| --------------------------------- | ----------- |
| Sign a token, RS256               | ~0.73 ms    |
| Sign a token, ES256               | ~0.02 ms    |
| Verify a token, RS256             | ~0.02 ms    |
| Decide what a token carries       | ~5.5 µs     |
| Hash a client secret / check PKCE | ~0.3 µs     |
| Rate limiter check (in memory)    | ~0.6 µs     |
| Hash a password (bcrypt)          | ~45 ms (by design) |

Load test (`TestLiveLoad`): two server instances, one Postgres, one Redis, 32
concurrent clients, zero errors:

| Endpoint              | Before fix                       | After fix                         |
| --------------------- | -------------------------------- | --------------------------------- |
| Client credentials    | 1,549 req/s, p50 20 ms, 12 queries | **8,007 req/s, p50 1.6 ms, 0 queries** |
| Userinfo              | 2,854 req/s, 7 queries           | **4,226 req/s, 3 queries**        |
| JWKS                  | ~95,000 req/s                    | ~105,000 req/s                    |

These numbers come from a laptop. Use them to compare runs, not as capacity
promises.

### Security
- **Go dependencies:** `govulncheck` found no vulnerability in code that is
  called. One advisory, for the unused `x/crypto/openpgp`, does not apply.
- **Web dependencies:** `bun audit` found high-severity advisories in `devalue`
  (used by SvelteKit at runtime) and `brace-expansion`. Both were upgraded.
- **Static analysis:** `staticcheck` was clean. `gosec` findings were reviewed
  and are expected: SHA-1 is required by TOTP (RFC 6238), cookies are Secure
  only over HTTPS, and the other hits are endpoint URLs and the docs generator.
- **Manual review** confirmed:
  - authorization codes are single-use, and a replay revokes what was issued;
  - refresh tokens rotate, and reuse detection revokes the whole family;
  - PKCE is enforced, and redirect URIs must match exactly;
  - an unknown email takes as long to refuse as a wrong password;
  - lockouts are atomic under parallel attempts;
  - TOTP codes cannot be replayed;
  - CSRF and CORS are strict;
  - cookies are HttpOnly and SameSite, and duplicate (planted) cookies are rejected;
  - the admin API is never on the public listener;
  - an attacker-chosen `X-Forwarded-For` is ignored;
  - CSV exports are safe from spreadsheet formula injection;
  - `?next=` cannot be used as an open redirect.

### Scalability (new: `replicas_integration_test.go`)
Two server instances share one Postgres and one Redis, as they would behind a
load balancer. The tests check that:
- tokens, signing keys and browser sessions issued by one instance work on
  the other, and signing out on one ends the session on both;
- one authorization code sent to both instances at once is spent exactly once;
- one refresh token sent to both at once rotates exactly once, and the replays
  revoke the family;
- disabling an application or rotating its secret on one instance takes effect
  on the other at once;
- taking away API access or a role's scope on one instance takes effect on
  the other at once (this covers the new cache);
- wrong passwords sent to both at once lock the account on both;
- the rate limit is one shared budget across instances.

### Proxy (new: `proxy_integration_test.go`)
A real reverse proxy sits in front of the server and acts as the issuer. The
tests check that:
- discovery advertises the proxy's address, a browser can sign in through it,
  and the session records the real client IP;
- each client behind a trusted proxy gets its own rate-limit budget;
- a forged `X-Forwarded-For` is ignored, whether the proxy replaces the header
  (like Caddy) or appends to it (like nginx);
- a misconfigured, untrusted proxy fails closed: all clients share one budget
  rather than going unlimited;
- with no proxy configured, `X-Forwarded-For` is ignored.

### Easy setup
The project was copied into an empty directory (no `.env`, no `node_modules`).
Then:
- `make setup` created `.env` with a secret key, created the database, ran the
  migrations and installed every app's dependencies;
- a second `make setup` changed nothing and kept the secret key;
- `make build` produced a binary that started and served health and discovery.

### Easy deployment
`make deploy-local` built all the Docker images from scratch and started
Postgres, Redis, the API, both apps and Caddy. All six containers became
healthy. Then, through Caddy over HTTPS:
- the sign-in page and the console were served;
- discovery advertised the right issuer;
- the admin API was unreachable through the public host (404);
- the security headers were present;
- the first administrator could be created;
- the API logged the real client IP.

The stack was removed afterwards.

### UI
The console and the sign-in app were walked in a browser, in light and dark
themes: login, dashboard, applications, users, settings, sign-in, register,
forgot password, and a wrong-password error. No console errors appeared.

## What was fixed

| # | Problem | Fix |
|---|---------|-----|
| 1 | **API reference out of date:** the last security commit changed handlers without regenerating it, so `make check` failed. | Regenerated with `make docs`. |
| 2 | **Console header font missing:** the Poppins font file was never committed, and the theme CSS was never regenerated, so `make web-check` failed and the logo fell back to the system font. | Added the font file and its licence, and regenerated the CSS. |
| 3 | **Sign-in app failed lint:** a regular expression with control characters in the redirect guard. | Rewrote the check without a regex. Same behaviour, tested. |
| 4 | **Slow token endpoint:** each token request loaded every API in the installation, twice. Every user token reloaded the whole role graph. | Load only the requested API, and only once. The role graph and API access are now cached in Redis and invalidated on every write that affects them. Tests prove the invalidation works across instances. |
| 5 | **Internal requests (SSRF) on loopback:** outbound calls to identity providers allowed `127.0.0.1`, so in production a provider URL could reach the server's own admin port. `100.64.0.0/10` (some clouds' metadata range) was not blocked either. | Loopback is now allowed only when the server itself runs on loopback (development), and `100.64.0.0/10` is blocked. New tests added. |
| 6 | **Debug mode in production:** a bare binary (e.g. under systemd) ran Gin in debug mode. | Release mode is now the default unless `GIN_MODE` is set. |
| 7 | **`make deploy-local` broken on a fresh checkout:** it never generated `REDIS_PASSWORD`, which `compose.yaml` requires. | The script now generates one, including for an existing `deploy/.env`. |
| 8 | **Vulnerable web dependencies:** `devalue` and `brace-expansion`. | Upgraded within their version ranges. |
| 9 | **Wrong count wording in the UI:** "1 events" and similar plural mistakes in five places. | All use the existing `countOf` helper. |
| 10 | **Trailing space in names:** an administrator with no last name got the name `"Preview "`. | The name is trimmed; test added. |
| 11 | **Broken `/preview` default page:** it pointed at a page that had moved. | Updated the path. |
| 12 | **Shell lint warnings** in `scripts/lib.sh`. | Fixed. |

## Code comments

Comments are now short and only explain *why* (a security reason, a race, a
protocol rule). Long multi-paragraph explanations were cut to one or two lines,
comments that restated the code were removed, and decorative dividers were
deleted. Comment lines in non-test code went from 9,731 to 7,211.

Doc comments on handlers and request fields were kept, because `make docs`
builds the public API reference from them. Migration files were left
untouched: a migration that has run anywhere is never edited.

## Refactor and security pass (14 October 2026)

Go 1.27.2. Every check passes: `make check`, `make test-integration` (36
packages), `make web-check`, `make docs-check`, the apps' unit tests (console
61, id 39), `staticcheck`, and `govulncheck` (no reachable vulnerability).

| # | Problem | Fix |
|---|---------|-----|
| 13 | **Go standard library and `x/net` vulnerabilities:** `govulncheck` found 9 reachable advisories in Go 1.27.1 and `x/net` v0.59.0 (HTTP/2 crash and flow-control bugs, HTTP/1 desync, MIME header memory limit, TLS ECH). | Go 1.27.2 and `x/net` v0.61.0. Every Go dependency is on its latest release. |
| 14 | **About 50 modules never run:** the `goose` CLI, a `tool` in `go.mod`, linked every database driver it supports (ClickHouse, YDB, Azure SDK, gRPC), including three deprecated modules. | `make migrate-new` calls the goose library through `cmd/migrate new`; the tool is gone, and with it every deprecated module. |
| 15 | **The admin session cookie accepted a planted duplicate.** The user cookies already refused two cookies of one name (a sibling subdomain can set one); the admin cookie was read raw in seven places. | All read through `session.Token`; test added. |
| 16 | **The API sent no security headers of its own.** It relied on the proxy in front of it. | `middleware.SecurityHeaders`: `nosniff`, `no-referrer`, no framing, `default-src 'none'`. The console also sends `Referrer-Policy` and `nosniff` itself. |
| 17 | **About 70 errors were English sentences written in Go**, against the repository's own rule: no stable code, missing from the API reference. | Each is a defined problem with a code and a catalog sentence (shared ones in `respond/shared.go`). A malformed id now answers `*_not_found`, as the newer handlers did. The API reference documents about 50 errors it did not list before. |
| 18 | **Columns that could hold what the code cannot read:** 74 text columns were nullable while read into a plain `string` (one NULL fails every read of that row), every `created_at`/`updated_at` was nullable, and 30 tables kept GORM's soft-delete column although nothing is soft-deleted. | Migration `tighten_columns`: text columns `NOT NULL DEFAULT ''` (existing NULLs filled), timestamps `NOT NULL`, `deleted_at` dropped. Rehearsed on a database built by the previous release with NULLs planted: up, down and up again. |
| 19 | **Files holding many subjects:** `model/oauth.go` held seven tables and `store/oauth.go` seven subjects; `store/sessions.go` held admin sessions beside user ones. | One file per table or subject; admin session methods are named for admins (`CreateAdminSession`, …). |
| 20 | **Older idioms** where Go now has a standard form (pointer helpers, `strings.SplitSeq`, `maps.Copy`, `errors.AsType`). | Applied Go's `modernize` fixes. Migrations that have run were left untouched. |
| 21 | **The apps were a major version behind:** SvelteKit 2, adapter-node 5, Vitest 4. | SvelteKit 3, adapter-node 6, Vitest 5 in all three apps, with the official codemod and the manual steps after it: `#lib` imports, route-id `resolve()`, read-only `page.url`, `refreshAll`, `goto`'s renamed options (`replace` still keeps a reset token out of history), and server environment in `src/env.ts`. The codemod left `API_URL` empty when unset, which would have broken every API call in development; the default was restored. A sign-in `next` path with no route now loads plainly instead of leaving the user on the sign-in page. adapter-node dropped `ORIGIN`, so the containers read the public address from Caddy's forwarded headers instead (checked against a stub API). `bun audit` is clean in every app: SvelteKit 3 ships `cookie` v2, and an override replaces the unused old `esbuild` in `svelte-i18n`. |

Static analysis: `gosec` findings were reviewed again and are the expected ones
listed above, plus the migration's table names (constants) being formatted
into SQL.

## Known items (not changed)

- **TypeScript 7** is not taken: SvelteKit 3 requires TypeScript 6, and
  `svelte-check` and `typescript-eslint` do not support 7 yet (its compiler API
  arrives in 7.1). `@types/node` stays on 24, the Node version the images run.

- **Chirp font licence:** the console bundles Chirp, X's proprietary typeface.
  Replace it, or confirm you have a licence, before publishing the console as
  open source.
- **Platform coverage:** the Docker stack was tested on Apple Silicon only.
  Building on linux/amd64 in CI is recommended.
- **Single test machine:** load numbers come from a laptop running the
  database, Redis and both instances. Measure on production hardware before
  capacity planning.
