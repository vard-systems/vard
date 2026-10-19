# CLAUDE.md

## Overview

An OAuth 2.0 authorization server and OpenID Connect provider in Go, with
two SvelteKit apps under `web/`:

| App           | `make dev`            | Built (`make run`)    | What it is                                       |
| ------------- | --------------------- | --------------------- | ------------------------------------------------ |
| `web/id`      | http://localhost:5173 | http://localhost:8080 | Hosted sign-in pages and each user's own account |
| `web/console` | http://localhost:5174 | http://localhost:8081 | The admin panel                                  |

One Go binary is the whole server, PocketBase-style: it listens on `:8080`
(provider and account API, and the id app) and `:8081` (admin API, and the
console), with both apps built to static files and compiled in
(`web/embed.go`). No Node.js runs in production, and no proxy: each app is
served on its own origin together with the API paths it calls. Everything is
plain http on localhost for now; TLS will come from a proxy in front.
Postgres holds everything; Redis is an optional cache. `README.md` is the
overview, `TESTING.md` the test report — update either when a change makes a
sentence in it untrue.

## Commands

| What                                 | Command                                                                   |
| ------------------------------------ | ------------------------------------------------------------------------- |
| Run everything (hot reload)          | `make dev` — API, id (Vite :5173), console (Vite :5174); logs in `.logs/` |
| Run only the API                     | `make run`                                                                |
| Go: format, vet, unit tests          | `make check`                                                              |
| Go: tests against Postgres           | `make test-integration` (every `TestLive…`, throwaway DBs)                |
| Apps: lint and type-check            | `make web-check` (or `bun run lint` / `bun run check` in an app)          |
| Apps: unit tests                     | `make web-test`                                                           |
| Apps: build (embedded by the binary) | `make web` — `make build` runs it first                                   |
| API reference (OpenAPI)              | `make docs` writes it, `make docs-check` verifies it                      |
| Migrations                           | `make migrate-new name=x`, `migrate-up`, `migrate-down`, `migrate-status` |
| Benchmarks                           | `make bench`                                                              |
| Production stack on this machine     | `make deploy-local` → http://localhost:8080, http://localhost:8081        |
| Everything else                      | `make` lists every target                                                 |

Project commands in `.claude/commands/`: `/check` (every check), `/preview`
(the app in a browser on spare ports), `/endpoint` (add an admin endpoint),
`/migration` (write a migration).

## Architecture

```
cmd/loginer/main.go      startup, in order, in one function
cmd/migrate, cmd/apidoc  the migration runner; the OpenAPI generator
internal/brand           the project's name and every name built from it
internal/config          reads .env
internal/model           one file per table; every model listed in model.All
internal/store           every query in the project, one file per subject
internal/cache           Redis: the cache and session databases, rate-limit counts
internal/auth            signing administrators in, and what they may do
internal/oidc            the provider: authorize, tokens, userinfo, logout, social, SSO
internal/api/server.go   the engine and the route table (registerPublicRoutes,
                         registerAdminRoutes)
internal/api/<subject>/  handler.go, request.go, response.go, validation.go
internal/api/respond     answers and the problem catalog (shared.go: problems
                         several packages use)
internal/api/ui.go       WithPublicUI / WithAdminUI: the app on every path the API
                         does not answer; unknown API paths stay JSON 404s
internal/webui           SPA: a built app from an fs.FS, index.html for routes
internal/apidoc          writes internal/api/reference/*.json (the OpenAPI the
                         server answers) from the source
migrations/              the schema, then one file per change, applied in order
i18n/                    translations: id/<locale>/ for the sign-in pages,
                         console/en/ for the admin API's problems
web/embed.go             go:embed of web/id/build and web/console/build
web/console/src/lib/     api/, query/, components/ui/, components/<feature>/
deploy/                  compose.yaml (make deploy-up, make deploy-local), Dockerfiles
```

**Layering.** Nothing above `store` writes a query; nothing below `api` knows
about HTTP. A handler reads a request, asks the store, and answers. Rules live
in `validation.go` and the model.

**The origins.** Built, the binary serves `web/id` on `:8080` beside
`/oauth2/*`, `/.well-known/*` and `/api/v1/account/*`, and `web/console` on
`:8081` beside `/api/v1/admin/*` (`internal/api/ui.go`). Under `make dev` each
Vite dev server stands in: it serves its app and passes the paths in its
`apiPaths` (`vite.config.ts`) on to the API (`API_URL`).
`TestEveryRouteIsReachableThroughItsApp` holds every route to those lists. In
`deploy/compose.yaml` the API publishes both ports itself, `8081` on the
host's loopback only. Cookies are host-only and nothing is cross-origin.

**The apps.** Single-page apps (`adapter-static`, `ssr = false` in each root
layout): the binary serves one `index.html` for every route and the browser
draws the page from the API's answers. Files under `_app/immutable/` are
cached for a year; everything else, `index.html` first, is revalidated. Under
`make dev` (`LOGINER_WEB_DEV=true`) the binary serves neither app and Vite
does, with hot reload.

## Security invariants

This is an identity server; these are not style. Keep each one when changing
code near it, and add a test when adding a case.

- **Secrets are stored hashed or sealed, never plain.** Codes, refresh tokens,
  sessions and reset links are 256 random bits kept as a SHA-256
  (`model.NewSecret` / `HashSecret`). Provider secrets, SMTP passwords and TOTP
  seeds are sealed with `jose.Sealer`. Recovery codes are hashed.
- **Single-use means one atomic `UPDATE … WHERE used_at IS NULL`** checked by
  `RowsAffected`, never read-then-write. A replayed code revokes what the
  first use issued; a replayed refresh token revokes its family.
- **Session cookies are read through `session.Token`, `session.UserToken` or
  `session.SignInState`**, which treat a duplicated cookie (planted from a
  sibling subdomain) as none. Never `c.Cookie(...)` for a session.
- **Outbound requests to admin- or user-supplied URLs** (social, SSO, SAML
  metadata) go through `newFederationClient` in `internal/oidc/outbound.go`,
  which refuses private, link-local and metadata addresses at dial time.
- **Redirect targets** are exact-matched redirect URIs, or a path checked by
  `safePath` (Go) / `safeNext` (`web/id`). Never redirect to a raw parameter.
- **Unknown accounts cost the same as wrong passwords** (`CompareDummyPassword`),
  and sign-in failures give one answer, so timing and wording reveal nothing.
- **Browsers are held to exact origin lists** (`api/cors`, `api/csrf`): no
  wildcards, no suffix matching. State-changing requests must be JSON.
- **Nothing internal reaches a browser.** Every answer is a defined problem;
  anything else goes through `respond.Failure`, which logs it and answers 500.
  The request log records the path, never the query string.
- **Every response carries security headers** — the API's from
  `middleware.SecurityHeaders`, each app's (`Permissions-Policy` among them)
  from `internal/webui`. HSTS comes with the TLS proxy, later. The
  page shell holds no one's data; what is personal comes from API answers.
- **The apps run no script they did not ship.** Their Content Security Policy
  is SvelteKit's `csp` option in each `vite.config.ts`, written into
  `index.html` as a `<meta>`: `script-src 'self'` plus the hash of SvelteKit's
  bootstrap script. `internal/webui` adds `frame-ancestors 'none'` as a header,
  which a `<meta>` cannot carry. Never add `'unsafe-inline'` or
  `'unsafe-eval'` to scripts, an inline `<script>` to `app.html`, or an outside
  origin to `connect-src`; a new external resource is a deliberate change to
  that list.

## How things are added

**An endpoint.** The method in its package under `internal/api/`, keeping the
four files; mounted in `internal/api/server.go` behind `session.Require` and
the permission it needs. A route under `managed` (`session.RequireAny`) also
takes an admin-cli token; a route for a person alone goes under `signedIn`.
The handler's doc comment and its request fields' comments are the API
reference: run `make docs`. `/endpoint` walks through it.

**An error.** Never a sentence in Go. Define the problem beside the handler —
`var taken = respond.Define(http.StatusConflict, "language_code_taken", respond.Admin)`
— or in `respond/shared.go` if several packages use it. Answer with
`respond.Fail(c, taken)` or return `taken.With("code", code)`, and add
`error.language_code_taken` to `i18n/<app>/<locale>/server.json` for each app
it is for (`Public` → `id`, `Admin` → `console`, `Both`). A parameter in the
sentence is listed in `TestErrorParametersAreSent`. A malformed id is answered
like a missing one: `*_not_found`. The one exception is `respond.Invalid`,
which passes on a model's own validation sentence.

**A model.** A struct in its own file in `internal/model`, embedding `Base`,
registered in `model.All()` — bump the count in `TestAllListsEveryModel`. A
plain `string` column is `not null;default:''`, a `bool` or `int` is
`not null`, anything that may be absent is a pointer. Nothing is soft-deleted.

**A migration.** `migrations/20260927000000_schema.go` builds a fresh database
from `model.All()`; existing databases are stepped forward, so every change
after it gets its own file (`make migrate-new name=x`; `/migration`). `up` is a
no-op on a fresh database, `down` undoes it, and table and column lists are
written out, never read from the models. Test the upgrade of an existing
database, not just a fresh one.

**A permission.** A constant and catalog entry in
`internal/model/admin_permission.go`, routes guarded with `session.Can` or
`session.CanAnywhere`, and the name added to `AdminPermissionName` in
`web/console/src/lib/api/types.ts`. `store.EnsureSystemAPIs` makes it an
admin API scope on the next start.

**An activity log entry.** `h.audit.Record` / `RecordWith` in the handler, then
a sentence, icon and category in
`web/console/src/lib/components/activity/actions.ts`. If the log can name the
target, say who may see that name in `maySeeName`
(`internal/api/activity/handler.go`).

**A setting the outside world sees.** Follow `internal/api/organization/`: the
panel writes it, the discovery document publishes the agreements
(`internal/oidc/provider.go`), and the sign-in pages read the rest from
`GET /api/v1/account/organization`, which takes no session — so only what a
stranger may see goes in `PublicOrganization`.

**A setting that decides how people sign in.** Like the organisation's or
`admin_security`: a singleton row seeded from configuration on first start,
owned by the panel afterwards, read where it matters on each request.

**A social provider.** One entry in `model.SocialSpecs`. **A login step.** One
entry in `model.LoginStepSpecs`; `Implemented` says whether the server runs it.

**A cached read.** Only in the store, with `cached(ctx, s, group, field, load)`,
only for what every request asks for and almost nothing changes. A new group
is a constant in `internal/cache` listed in `cache.Groups`. Every write calls
`s.forget(ctx, group)` (or returns `s.forgetting(ctx, err, group)`) after it
commits. Cache a projection, never a model carrying a secret, and add it to
`TestCachedTypesSurviveJSON`. Never cache users, codes or refresh tokens.
Everything must work with a nil cache.

**A language** (sign-in pages only; the panel is English). Add `i18n/id/<tag>/`
with the same groups as `en` and `$name`/`$native` in `common.json`.

**A panel page.** `+page.ts` checks the permission (`requirePermission` /
`requireAnywhere`, from `#lib/api/load.js`) and fetches with `apiGet`; the API
enforces the same permission on every call. `+page.svelte` seeds a TanStack
query from it. Query options in `lib/query/` (key in `lib/query/keys.ts`), the
client call in `lib/api/admin.ts`, components in `lib/components/<feature>/`.
Add the route to `lib/components/layout/sidebar/sections.ts` with an `allowed`
check. Build from `#lib/components/ui`; a page never writes a colour, a height
or a hover state itself.

**A colour, size or font in the panel.** `web/console/src/lib/theme/theme.ts`
decides; `bun run theme` writes `lib/styles/tokens.css` and `fonts.css` (never
edit those by hand).

## Conventions

- **Text.** Sign-in pages: never a literal — `const t = useTranslator()` and
  `t('area.thing', { param })`, the key added to every group under `i18n/id/`.
  Panel: the opposite — English in the markup; an error shown is the sentence
  the API sent (`messageOf` from `#lib/api/index.js`).
- **The project's own name is never spelled out.** `internal/brand` holds it
  for the server and `web/<app>/src/lib/brand.ts` for each app;
  `TestNothingElseSpellsTheName` fails on a hand-written one.
- **SvelteKit 3.** Imports from `src/lib` use `#lib` and name the file
  (`import { BRAND } from '#lib/brand.js'`); there is no `$lib` and no
  `svelte.config.js` — configuration is `sveltekit({...})` in `vite.config.ts`.
  `resolve()` reads a leading slash as a route id, groups included
  (`resolve('/admin/(panel)/dashboard/users')`). `page.url` is read-only. Reload
  with `refreshAll()`, not `invalidateAll()`. Pages are drawn in the browser:
  fetch in the page's `load` (`+page.ts`), never in `onMount`. There are no
  server files — no `+page.server.ts`, no `hooks.server.ts`, no form actions —
  since a static build cannot run them; what needs a server is an API
  endpoint. The theme is applied before the first paint by `static/theme.js`.
- **Comments** are few and short and say _why_ — a security reason, a race, a
  protocol rule — never what the next line does. Handler and request-field doc
  comments stay: they are the API reference.
- **Names** are spelled out; identifiers use American spelling (`Normalize`,
  `Organization`), prose may be British.
- **Tests** are table-driven and named for what they prove ("a slug with
  spaces", not "TestCase3"). Tests needing Postgres are named `TestLive…` and
  skip without `LOGINER_TEST_DB_DSN`.
- **Formatting is enforced.** The hook in `.claude/hooks/format.sh` runs gofmt
  or the app's Prettier on every file written; `bunx prettier --write <path>`
  fixes what `bun run lint` refuses.

## Gotchas and warnings

- **Never read `.env`** — it holds the secret key. `.env.example` documents
  every variable.
- **Do not restart a running stack.** If anything answers on 8080, 8081, 5173
  or 5174, leave it alone; `/preview` raises a throwaway one on spare ports
  with its own database and Redis prefix, and takes it down afterwards.
- **Never edit a migration that has run anywhere**, and never edit
  `internal/api/reference/*.json` by hand — `make docs` owns it.
- **`make dev` is slow on the first load** (several seconds for the console):
  the dev server pre-bundles all of `svelte-remixicon`. Production builds
  tree-shake it and are fast. This is accepted; do not rewrite the imports.
- **The binary needs the builds to compile.** `web/embed.go` fails with
  "pattern all:id/build: no matching files found" until `make web` has run.
  `make check`, `test`, `run` and `dev` build them when missing; `make build`
  always rebuilds them. They are gitignored: to commit them like PocketBase,
  drop `/build` from each app's `.gitignore` and run `make web` before
  committing.
- **Under `make dev` the browser opens Vite's ports, not the API's.** The
  API's URLs (issuer, admin URL) are Vite's then, and a path the API answers
  reaches it only if it is in that app's `apiPaths`.
- **The dev URLs are fixed by `scripts/start.sh`**, which exports
  `LOGINER_ISSUER`, `LOGINER_ACCOUNT_URL`, `LOGINER_ADMIN_URL`,
  `LOGINER_TRUSTED_PROXIES` and `LOGINER_WEB_DEV` over whatever `.env` says.
- **No proxy is trusted until there is one.** `LOGINER_TRUSTED_PROXIES` is
  empty, so `X-Forwarded-For` is ignored. When a TLS proxy goes in front, list
  its address alone — never a whole private range — and keep the admin port
  off the public network.
- **Redis is optional.** Every path must work with a nil cache, and Redis is
  never the only copy of anything.
- **The shell is zsh.** An unquoted glob such as `--include=*.go` fails with
  "no matches found" — quote it.
- **TypeScript stays on 6** until SvelteKit, `svelte-check` and
  `typescript-eslint` support 7.

## Definition of done

A change is finished when all of this holds, and the summary says which checks
ran and their result — or which were skipped and why:

1. `make check` passes (gofmt, go vet, Go unit tests).
2. If an app changed: `make web-check` is clean and `make web-test` passes.
   If anything under `deploy/` changed: `docker compose -f deploy/compose.yaml config -q` passes.
3. If a migration, the store, hand-written SQL or a route changed:
   `make test-integration` passes.
4. If a handler's doc comment, a request field or a route changed: `make docs`
   was run and `make docs-check` passes.
5. Every new problem code is in the catalogs, every new key in every language,
   every new permission in `types.ts`.
6. New behaviour has a test; a security case has one without exception.
7. A user-visible change was looked at in a browser (`/preview`).
8. `README.md`, `TESTING.md` and this file say nothing the change made untrue.

## Git

- **Never commit, push, branch, tag or open a pull request unless asked** in
  this conversation. Leave changes in the working tree and say what changed.
- When asked to commit: never `--amend`, `--force` or `--no-verify`, and never
  rewrite history that has been pushed. Branch off `main` first if on it.
- `.claude/settings.local.json` is personal and is not committed.
