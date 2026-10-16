# CLAUDE.md

An OAuth 2.0 / OpenID Connect server in Go, with three SvelteKit apps under
`web/`: `console` (the admin panel), `id` (sign-in pages and a user's own
account) and `docs` (documentation for developers integrating with it).
`README.md` is the overview and `TESTING.md` the test report; update either
when a change makes a sentence in it untrue.

## Commands

| What                         | Command                                                                   |
| ---------------------------- | ------------------------------------------------------------------------- |
| Format, vet and test Go      | `make check`                                                              |
| Tests that need Postgres     | `make test-integration` (every `TestLive…`, throwaway DBs)                |
| Lint and type-check the apps | `make web-check` (or `bun run lint` / `bun run check` in `web/<app>`)     |
| Test the apps                | `make web-test`                                                           |
| API reference                | `make docs` writes it, `make docs-check` verifies it                      |
| Migrations                   | `make migrate-new name=x`, `migrate-up`, `migrate-down`, `migrate-status` |
| Run everything               | `make dev` — API :8080/:8081, id :5173, console :5174, docs :5175         |
| Benchmarks                   | `make bench`                                                              |

Before calling a change done: `make check`, and `make web-check` if an app
changed. Run `make test-integration` after touching a migration, the store,
hand-written SQL, or the route table — unit tests never see a real database.
Formatting is enforced; the hook in `.claude/` formats each file as it is
written, and `bunx prettier --write <path>` fixes what `bun run lint` refuses.

Never read `.env` (it holds the secret key); `.env.example` documents every
variable. The apps log to `.logs/<app>.log`. If a stack is already running on
8080/8081/5173/5174, do not restart it — `/preview` starts a throwaway one.

## Layout

```
cmd/loginer/main.go      startup, in order, in one function
internal/brand           the project's name and every name built from it
internal/config          reads .env
internal/model           one file per table, every model listed in model.All
internal/store           every query in the project, one file per subject
internal/cache           Redis: the cache and session databases, rate-limit counts
internal/auth            signing administrators in, and what they may do
internal/oidc            the provider: authorize, tokens, userinfo, logout, social, SSO
internal/api/server.go   the engine and the table of every route
internal/api/<subject>/  handler.go, request.go, response.go, validation.go
internal/api/respond     answers and the problem catalog (respond/shared.go: problems
                         several handlers use)
internal/apidoc          writes the API reference from the source
migrations/              the schema, then one file per change, applied in order
web/console/src/lib/     api/, query/, components/ui/, components/<feature>/
web/docs/src/content/    the docs: guides by hand, reference/ by make docs
```

Layering: nothing above `store` writes a query, and nothing below `api` knows
about HTTP. A handler reads a request, asks the store, and answers. Rules live
in `validation.go` and the model.

## The apps (SvelteKit 3)

- Imports from `src/lib` use the `#lib` subpath import and name the file:
  `import { BRAND } from '#lib/brand.js'` (the `.js` resolves to the `.ts`).
  There is no `$lib` and no `svelte.config.js`; configuration is the
  `sveltekit({...})` plugin in `vite.config.ts`.
- `resolve()` reads a leading slash as a route id, groups included
  (`resolve('/admin/(panel)/dashboard/users')`); a plain path has none
  (`resolve('login')`).
- `page.url` is read-only: copy it (`new URL(page.url.href)`) before changing it.
- Reload data with `refreshAll()`, not `invalidateAll()`. `goto` takes
  `replace`, `refreshAll` and `reset: false`, and rejects a path with no route.
- Server environment comes from `src/env.ts` (`defineEnvVars`, where each
  default lives) and is read from `$app/env/private`.
- Behind the proxy, adapter-node learns the public address from
  `PROTOCOL_HEADER`/`HOST_HEADER` (`deploy/compose.yaml`); there is no `ORIGIN`.
- TypeScript stays on 6 until SvelteKit, `svelte-check` and `typescript-eslint`
  support 7.

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
- **Session cookies are read through `session.Token`, `session.UserToken`
  or `session.SignInState`**, which treat a duplicated cookie (one planted
  from a sibling subdomain) as none. Never `c.Cookie(...)` for a session.
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
  `middleware.SecurityHeaders`, each app's from its `hooks.server.ts`.

## Adding things

**An endpoint.** Write the method in its package under `internal/api/`,
keeping the four files, and mount it in `registerRoutes` in
`internal/api/server.go` behind `session.Require` and the permission it needs
— a route guarded by a permission under `managed` also takes an admin-cli
token; a route for a person alone goes under `signedIn`. The handler's doc
comment and its request fields' comments are its documentation: run
`make docs` (`TestTheReferenceIsCurrent` fails until you do). `/endpoint`
walks through it.

**An error.** Never a sentence in Go. Define the problem beside the handler
— `var taken = respond.Define(http.StatusConflict, "language_code_taken", respond.Admin)`
— or in `respond/shared.go` if more than one package answers with it. Answer
with `respond.Fail(c, taken)` or return `taken.With("code", code)`, and add
`error.language_code_taken` to `i18n/<app>/<locale>/server.json` for each app
it is for (`Public` → `id`, `Admin` → `console`, `Both`). A parameter in the
sentence is listed in `TestErrorParametersAreSent`. A malformed id is answered
like a missing one: `*_not_found`. `TestErrorCodesMatchTheCatalogs` fails
until code and catalog agree. The one exception is `respond.Invalid`, which
passes on a model's own validation sentence.

**A model.** A struct in `internal/model` in a file of its own, embedding
`Base`, registered in `model.All()` — bump the count in
`TestAllListsEveryModel`. Columns say what the Go type can hold: a plain
`string` column is `not null;default:''`, a `bool` or `int` is `not null`,
and anything that may be absent is a pointer. Nothing is soft-deleted:
deletes remove the row.

**A migration.** `migrations/20260927000000_schema.go` builds a fresh
database from `model.All()`, so a new model's table follows — but existing
databases are stepped forward, never rebuilt, so every change after it gets
its own migration (`make migrate-new name=x`; `/migration` walks through it).
`up` must also be a no-op on a fresh database (the schema step already built
the new shape), `down` undoes it, and lists of tables or columns are written
out rather than read from the models, so the file means the same thing
forever. No migration that has run anywhere is edited. Test the upgrade of an
existing database, not just a fresh one.

**A permission.** A constant and catalog entry in
`internal/model/admin_permission.go`, routes guarded with `session.Can` or
`session.CanAnywhere`, and the name added to `AdminPermissionName` in
`web/console/src/lib/api/types.ts`. The panel builds its permission UI from
the catalog, and `store.EnsureSystemAPIs` makes it an admin API scope on the
next start.

**An activity log entry.** `audit.Record` / `RecordWith` in the handler, then
a sentence, icon and category in
`web/console/src/lib/components/activity/actions.ts`. If the target is a
record the log can name, say who may see that name in `maySeeName` in
`internal/api/activity/handler.go`.

**A setting the outside world sees.** Follow `internal/api/organization/`:
the panel writes it, the discovery document publishes the agreements
(`internal/oidc/provider.go`), and the sign-in pages read the rest from
`GET /api/v1/account/organization`, which takes no session — so only what a
stranger may see goes in `PublicOrganization`. A setting nothing reads is a
note, not a setting.

**A setting that decides how people sign in.** Copy the organisation's or
`admin_security`: a singleton row seeded from configuration on first start,
owned by the panel afterwards, and read where it matters on each request.

**A social provider.** One entry in `model.SocialSpecs` — endpoints, scopes,
where the identity is in the answer, and quirks. The flow, panel and sign-in
pages all read it. Only an installation's credentials go in the database.

**A login step.** One entry in `model.LoginStepSpecs`; `Implemented` says
whether the server runs it yet. Implementing one is flipping that flag and
writing the step.

**A cached read.** Only in the store, with `cached(ctx, s, group, field, load)`,
only for something every request asks for and almost nothing changes. A new
group is a constant in `internal/cache` listed in `cache.Groups` under `cache`
(anybody may see it) or `sessions` (decides who is signed in). Every write
calls `s.forget(ctx, group)` — or returns `s.forgetting(ctx, err, group)` —
after it commits. Cache a projection, never a model carrying a secret, and add
it to `TestCachedTypesSurviveJSON`. Sessions are put after each write
(`putAdminSessions` / `putUserSessions`). Never cache users, codes or refresh
tokens. Redis is optional: everything must work with a nil cache.

**A language** (sign-in pages only; the panel is English). Installations add
their own in the panel. To ship one, add `i18n/id/<tag>/` with the same groups
as `en` — `common`, `auth`, `account`, `server`, `validation`, `email` — and
`$name`/`$native` in `common.json`. `TestTranslationsAreComplete` fails when a
shipped language falls behind `en`. `i18n/console/en/` holds only the English
of the admin API's problems.

**Text in the sign-in pages.** Never a literal: `const t = useTranslator()`
and `t('area.thing', { param })`, with the key added to every group under
`i18n/id/`. The product's own name comes from `#lib/brand.js`, not a dictionary.

**Text in the panel.** The opposite: English, in the markup. An error shown to
an administrator is the sentence the API sent (`messageOf` in `#lib/api/index.js`).

**A colour, size or font in the panel.** `web/console/src/lib/theme/theme.ts`
decides the look; `bun run theme` writes `lib/styles/tokens.css` and
`fonts.css` from it (never edit those by hand). Components name tokens —
`var(--color-brand)`. `web/docs/src/lib/styles/tokens.css` copies the
console's tokens, so a change to `theme.ts` is copied there too.

**A panel page.** `+page.server.ts` checks the permission
(`requirePermission` / `requireAnywhere`) and fetches with `apiGet`;
`+page.svelte` seeds a TanStack query from it; query options live in
`lib/query/` with their key in `lib/query/keys.ts`, the client call in
`lib/api/admin.ts`, components in `lib/components/<feature>/`. Add the route
to `sections.ts` with an `allowed` check — the sidebar and command palette
read it. Pages render on the server: fetch in `+page.server.ts`, never in
`onMount`. Build from `#lib/components/ui`; a page never writes a colour, a
height or a hover state itself.

**The docs.** Markdown in `web/docs/src/content/`, where a file's path is its
address. Front matter places a guide in the sidebar: `section:` (one of
`SECTIONS` in `web/docs/src/lib/server/content.ts`), `order:`, `nav:`. Write
`{{name}}` and the other `PLACEHOLDERS` of `web/docs/src/lib/brand.ts`, never
the project's name. Examples are consecutive fenced blocks titled `"cURL"`,
`"Java"`, `"Go"`, `"Python"`, in that order. `content/reference/` belongs to
`make docs` and is never edited.

**The project's own name.** Never spelled out. `internal/brand` holds it for
the server and `web/<app>/src/lib/brand.ts` for each app.
`TestNothingElseSpellsTheName` fails on a hand-written one, and
`TestTheAppsSpellTheSameNames` checks the cookies the server and apps share.
A rename is those constants, the module path in `go.mod`, the three
`brand.ts`, and a `git mv` of the command's directory under `cmd/` — plus
`.env.example`, `deploy/`, the `Makefile` and `scripts/`.

## Style

Code is written to be read. Comments are few and short and say _why_ — a
security reason, a race, a protocol rule — never what the next line does.
Names are spelled out; identifiers use American spelling (`Normalize`,
`Organization`), prose may be British. Handler and request-field doc comments
stay: they are the API reference.

Tests are table-driven and named for what they prove ("a slug with spaces",
not "TestCase3"). Tests needing Postgres are named `TestLive…` and skip
without `LOGINER_TEST_DB_DSN`.

## `.claude/`

```
settings.json        the permission allowlist and the formatting hook
hooks/format.sh      gofmt or the app's Prettier, on every file written
commands/check.md    /check      every check this repository has
commands/preview.md  /preview    the app in a browser, on spare ports
commands/endpoint.md /endpoint   add an admin API endpoint
commands/migration.md /migration write a migration
```

`settings.local.json` is for personal overrides and is not committed.
