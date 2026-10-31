# CLAUDE.md

## Overview

An OAuth 2.0 authorization server and OpenID Connect provider in Go, with
two SvelteKit apps under `web/`:

| App           | `make dev`            | `make docker-local`   | What it is                                       |
| ------------- | --------------------- | --------------------- | ------------------------------------------------ |
| `web/id`      | http://localhost:5173 | http://localhost:8080 | Hosted sign-in pages and each user's own account |
| `web/console` | http://localhost:5174 | http://localhost:8081 | The admin panel                                  |

One Go binary is the API: `:8080` (provider and account API) and `:8081`
(admin API). Each app is a SvelteKit Node server in front of it that renders
its pages and passes the API paths it calls on to it, so each app is one
origin and the browser never reaches the API directly. Locally everything is
plain http; on a server, TLS and routing are the infrastructure's, in front
of the apps (`DEPLOY.md`).
Postgres holds everything; Redis is an optional cache. `README.md` is the
overview for people; update it when a change makes a sentence in it untrue.

## Commands

| What                               | Command                                                                                   |
| ---------------------------------- | ----------------------------------------------------------------------------------------- |
| Run everything (hot reload)        | `make dev` — API, id (Vite :5173), console (Vite :5174); logs in `.logs/`                 |
| Run everything (production builds) | `make start` — the API and both apps built, on the same ports; `env=dev` is `make dev`    |
| Run only the API                   | `make run`                                                                                |
| Lint and unit-test everything      | `make check` (`lint` + `test`; each also as `-go` / `-web`)                               |
| The slow, type-aware lint rules    | `make lint-types` (floating promises, deprecated APIs; CI runs it, `make check` does not) |
| Format everything                  | `make fmt` (`lint` only reports, never rewrites)                                          |
| Go: tests against Postgres         | `make test-integration` (every `TestLive…`, throwaway DBs; fails with no database)        |
| End-to-end, in a browser           | `make e2e` (web/e2e, Playwright: a stack of its own on 8290/8291/5293/5294)               |
| Everything CI runs                 | `make ci` (`check`, `test-integration`, `docker-config`)                                  |
| Build                              | `make build` (`build-api` → `bin/vard`, `build-web` → `node build`)                       |
| Console's generated files          | `make generate` (theme tokens, fonts)                                                     |
| Migrations                         | `make migrate-new name=x`, `migrate-up`, `migrate-down`, `migrate-status`, `db-reset`     |
| Benchmarks, vulnerabilities        | `make bench`, `make vuln`                                                                 |
| Rename the project                 | `make rename name="New Name" slug=newslug` (see A brand, below)                           |
| Production stack on this machine   | `make docker-local` → http://localhost:8080, http://localhost:8081                        |
| A server                           | `make docker-env`, `make docker-up`, `make db-backup` (DEPLOY.md)                         |
| Everything else                    | `make` lists every target                                                                 |

Project commands in `.claude/commands/`: `/check` (every check), `/preview`
(the app in a browser on spare ports), `/endpoint` (add an admin endpoint),
`/migration` (write a migration).

## Architecture

```
cmd/vard/main.go         the commands: serve (the default), migrate, and admin
                         create (the first administrator from the shell)
cmd/vard/serve.go        startup, in order, in one function
cmd/vard/migrate.go      the migration runner (up, down, status, new)
internal/                in layers, each importing only those below it
                         (internal/layers_test.go):
  config, brand, jose      shared by all: settings from .env; the brand (name,
                           slug, favicon, every name built from them; each app
                           has its own, lib/brand.ts); JWT signing and
                           verifying, and Sealer (secrets at rest)
  domain/                  what the server is about, with no database or HTTP:
    model                  one file per table; every model listed in model.All
    totp, recovery         time-based one-time codes; recovery codes
  infrastructure/          what it is kept in and talks to:
    store                  every query in the project, one file per subject
    cache                  Redis: the cache and session databases, rate-limit counts
    database               the Postgres connection and the migration runner
    mail, passkey          sending email; the WebAuthn relying party
  application/             the use cases, on the domain and the infrastructure:
    oidc                   the provider (authorize, tokens, userinfo, logout,
                           social, SSO) and how users sign themselves in:
                           session.go, register.go, reset.go, verify.go, account.go
    auth                   signing administrators in, and what they may do
  interfaces/api/          HTTP: server.go, the engine and the route table
                           (registerPublicRoutes, registerAdminRoutes);
                           <subject>/, one package per subject the API answers
                           about (handler.go, request.go, response.go,
                           validation.go); kit/, what every handler shares:
                           respond (answers and the problem catalog; shared.go:
                           problems several packages use), validate, query,
                           session (who is calling, route guards), audit, cors,
                           csrf, ratelimit, middleware; signedin (what account,
                           profile, steps and twostep share); tstypes (holds the
                           panel's types.ts to the answers)
migrations/              All(): every migration, compiled in, run by goose's
                         Provider (database/migrate.go). setup/ is version 1
                         (schema/up.sql, frozen, and seed.go); changelogs/ one
                         file per change after it, 00002_x.go is version 2;
                         reset/ takes them all down and up again (make db-reset)
i18n/                    the English that ships, read with go-i18n: id/*.en.json
                         for the sign-in pages, console/*.en.json for the admin
                         API's problems; Catalog formats any language's text
web/<app>/src/hooks.server.ts  the app's server: API paths passed on
                         (lib/server/api.ts), page headers (lib/server/headers.ts)
web/console/src/lib/     api/, query/, components/ui/, components/<feature>/,
                         styles/tokens.css (generated: make generate)
web/id/src/lib/          api/, i18n/, state/, components/ui/ (primitives),
                         components/auth|account|layout/, constants.ts (the
                         API's field limits)
web/<app>/src/lib/testing/  expectAccessible: axe on a mounted component
web/<app>/src/lib/brand.ts  the app's brand (name, slug, mark, favicon from
                         lib/assets/, cookie names; the console's links and
                         pageTitle)
web/console/scripts/     theme.ts, the generator
web/e2e/                 Playwright tests of both apps through a real stack
                         (scripts/e2e.sh raises it with its own database, Redis
                         prefix and no .env); tests/stack.ts, their helpers
scripts/                 what the Makefile runs: setup, start, docker-local,
                         docker-env, db, db-backup, rename;
                         lib.sh (load_env) is shared by all of them
compose.yaml             the Docker stack (make docker-up, make docker-local), read
                         with compose.env (compose.env.example); never .env, the
                         server's own development settings
Dockerfile, web/Dockerfile  the API's image and the apps'
DEPLOY.md                a server, from DNS to backups
```

**Layering.** `domain` → `infrastructure` → `application` → `interfaces`, each
importing only what is to its left (and the shared `brand`, `config`, `jose`);
`TestEachLayerImportsOnlyWhatIsBelowIt` fails otherwise. Nothing but `store`
writes a query; nothing but `interfaces` answers a request. A handler reads a
request, asks the application or the store, and answers. Rules live in
`validation.go` and the model. The layers are folders and the import rule,
not interfaces: the application calls the store it is given, with no
repository abstraction between them.

**The origins.** Each app's `hooks.server.ts` passes the paths in its
`API_PATHS` (`src/lib/server/api.ts`) on to `API_URL` — `/oauth2`,
`/.well-known` and `/api/v1/account` for `web/id`, `/api/v1/admin` for
`web/console` — with the caller's address in `X-Forwarded-For` and anything a
client sent there dropped. The same hook carries the browser's requests and
the ones a load makes while a page is rendered, in `make dev` and in
production alike. `TestEveryRouteIsReachableThroughItsApp` holds every route
to those lists. In `compose.yaml` only the apps publish ports, on the
host's loopback; Postgres and Redis are on an internal network the
apps are not on. Cookies are host-only and nothing is
cross-origin.

**The apps.** Rendered on the server (`adapter-node`, `node build`): every
page arrives with its content and its `<title>`, then hydrates. Loads call
the API with the load's `fetch` on the app's own origin, which works on the
server and in the browser. A load that needs the request itself (cookies,
`Accept-Language`) is a `+layout.server.ts`.

## Security invariants

This is an identity server; these are not style. Keep each one when changing
code near it, and add a test when adding a case.

- **Secrets are stored hashed or sealed, never plain.** Codes, refresh tokens,
  sessions and reset links are 256 random bits kept as a SHA-256
  (`model.NewSecret` / `HashSecret`). Provider secrets, SMTP passwords and TOTP
  seeds are sealed with `jose.Sealer`, always bound to where they are kept —
  `SealFor(model.X.YWhere(), …)`, table, column and row, as AEAD additional
  data — so a value moved to another column or row does not open; the row's
  id is given before the value is sealed. A new sealed column gets its
  `…Where` method in `internal/domain/model/sealed.go` and a line in
  `store.sealedColumns`, which `secrets reseal` walks — and the server walks
  as it starts while `VARD_SECRET_KEY_PREVIOUS` is set. Recovery codes, device
  user codes and throttle keys are keyed hashes (`Sealer.Tag`) under the
  installation's own tag key (`server_secrets`, sealed like the rest), so
  they survive a rotation of `VARD_SECRET_KEY`. The one
  secret also kept recoverable is a refresh token that has just replaced
  another: sealed on the replaced row (`successor_sealed`) for the grace
  below, and cleared within one more — when read past it, at the family's
  next rotation, and by `KeepSwept` every `model.RefreshReuseGrace`. Nothing
  else may keep a token it hands out.
- **Single-use means one atomic `UPDATE … WHERE used_at IS NULL`** checked by
  `RowsAffected`, never read-then-write. A replayed code revokes what the
  first use issued; a replayed refresh token revokes its family — unless it
  was replaced less than `model.RefreshReuseGrace` (30 s) before, when it is
  answered with the very token that replaced it, so every copy a client holds
  stays good and a family never has two live tokens. The retry is checked
  like any refresh — client authentication, then DPoP key —
  before it is given anything.
- **Session cookies are read through `session.Token`, `session.UserToken` or
  `session.SignInState`**, which treat a duplicated cookie (planted from a
  sibling subdomain) as none. Never `c.Cookie(...)` for a session. Over https
  every session and sign-in cookie is written by `writeCookie` as a
  `brand.HostPrefix` (`__Host-`) one — Secure, `Path=/`, no Domain — which no
  sibling can set at all, and is read under that name alone
  (`session.CookieMode`, set per listener); plain-http development uses the
  bare names, which a browser requires there.
- **Outbound requests to admin- or user-supplied URLs** (social, SSO, SAML
  metadata, back-channel logout) go through `newFederationClient` in
  `internal/application/oidc/outbound.go`, which refuses private, link-local and metadata
  addresses at dial time. A logout token follows no redirect. A provider's
  keys are fetched again because a token did not verify at most once a
  `keyRefetchEvery` (a minute) per key set (`providerKeys`), and a social
  id_token is checked against its kind's published keys
  (`SocialSpec.JWKSURL`) wherever the kind names them.
- **A key from outside is at most `jose.MaxRSABits` (4096) long.** A DPoP
  proof brings a key of the sender's choosing, and verifying under a large
  modulus costs the server; `ParseJWK` refuses a longer one.
- **Redirect targets** are exact-matched redirect URIs
  (`Application.AllowsRedirect`: a native app's loopback URI on any port, RFC
  8252 §7.3, and nothing else loosened; Expo Go's `exp://` only for a native
  app with `allow_development_redirects`), or a path checked by
  `safePath` (Go) / `safeNext` (`web/id`). Never redirect to a raw parameter.
- **A user's sign-in is throttled, never locked, and a second factor is
  guessed against the same count.** Every answer — password, second factor,
  current password — is counted before it is checked (`ReserveSignIn`, rows
  locked in key order) against the caller's network and the address typed,
  keyed by a keyed hash so an address nobody has counts the same; past the
  free failures each attempt waits longer (`model.ThrottleWait`). A right
  password does not clear the
  count while a second factor is still to come; only the right second factor
  does, and a success only gives the network its one attempt back. The
  waiting sign-in is named by an httpOnly cookie, never by anything the page
  or the address holds. Administrators are throttled the same way, on a keyed
  hash of the username typed, counted before it is looked up
  (`internal/application/auth/throttle.go`); nothing is ever locked.
- **Only the person, in their own browser, accepts terms or allows an
  application.** The Agreements, Consent and device answers, and every change
  to two-step verification, refuse an application's access token.
- **Every one-time thing is taken with one conditional UPDATE**: a pushed
  request's `request_uri`, a device's approval and its tokens, a passkey
  sign-in's ceremony, a front-channel logout page; a DPoP proof's jti is
  recorded once. Device user codes are short, so they are kept as a keyed hash
  and entering them is rate-limited.
- **A DPoP-bound access token is never taken as a bearer token.** Userinfo
  checks its proof; the account and admin APIs, which do not, refuse it.
- **A client without credentials always uses PKCE** (`Application.MustUsePKCE`),
  whatever its `RequirePKCE` says, and a code asked for with a challenge is
  never exchanged without its verifier.
- **Introspection answers a token's own client, or the API's.** A
  confidential client may introspect its own tokens; another's access token
  only when an administrator listed it for an API in the token's `aud`
  (`api_introspectors`, never inferred from being allowed to request tokens
  for that API). A refresh token only its own client. A public client is
  refused introspection and cannot be listed.
- **Registering a client takes an initial access token** (an admin API token
  with `applications.write`); `/oauth2/register` is never open.
- **Passwords are Argon2id** (`model.HashPassword`; `VerifyPassword` takes
  bcrypt too and asks for a rehash, done at sign-in with a conditional
  UPDATE). Lengths are `model.MinUserPasswordLength`,
  `MinAdminPasswordLength` and `MaxPasswordLength`, mirrored in both apps'
  `constants.ts` (`TestThePagesKnowThePasswordLengths`).
- **Unknown accounts cost the same as wrong passwords** (`CompareDummyPassword`),
  and sign-in failures give one answer, so timing and wording reveal nothing.
- **Browsers are held to exact origin lists** (`api/kit/cors`, `api/kit/csrf`): no
  wildcards, no suffix matching. State-changing requests must be JSON.
  `web/id` turns SvelteKit's own CSRF check off (`csrf.trustedOrigins: ['*']`)
  because the provider takes form posts from other origins and from none
  (token endpoint, SAML ACS, Apple); so it never gets a form action or a
  `+server` route (`TestTheIDAppLetsTheProviderTakeFormPosts`).
- **A logout without `id_token_hint` asks first.** Anyone can link to
  `/oauth2/logout`, so a hint-less one from a signed-in browser goes to the
  `/logout` confirmation page, and only its JSON call to
  `/api/v1/account/logout` (`Confirmed`) signs out.
- **A SAML response signs in only the browser that started it, once.** The
  ACS POST comes cross-site without cookies, so it only holds the response
  on the sign-in its `RelayState` names (never the state cookie's value) and
  answers 303; the GET that follows brings the state cookie and does every
  check. Only the HTTP-POST binding (a `SAMLart` would make the library fetch
  from the metadata outside `newFederationClient`), no SHA-1 signatures, no
  RSA-1_5 key transport, a bearer confirmation, and each assertion ID once
  (`saml_assertions`).
- **An SSO connection signs in only addresses it may**: its own domains, or —
  without domains — any address at a domain no other connection holds.
- **An SSO connection takes over an existing account only at a proved
  domain.** Linking needs `matching = link`, listed domains (validation and
  `ssoAccountFor` both refuse it without), and an `sso_domain_verifications`
  row proved by its DNS TXT record.
  The first link to an account with a second factor waits for that factor:
  the identity is held on the pending sign-in (`link_connection_id`) and
  made only by `SubmitSecondFactor`, and the provider syncs nothing until
  then.
- **The first proof of an address ends what came before it.** A reset or
  verification link that first marks an account verified also deletes, in
  the same transaction (`clearBeforeProofIn`), its social and SSO
  identities, factors, recovery codes, consents and waiting sign-ins, and
  ends its sessions, refresh tokens, login codes and devices — whoever
  registered the address first may not have been its owner. Social sign-in
  never makes an account for an address the provider has not verified.
- **An access token ends with what it was issued in.** It carries `sid`
  and `fid`; whatever ends a session, a refresh family or a grant writes a
  `token_revocations` row in the same transaction (`revokeTokensIn`,
  `revokeUserRefreshIn`, `familiesRevokedIn`), and `verifyAccessToken` —
  userinfo, introspection, the account and admin APIs — refuses a token any
  of whose keys was revoked at or after its `iat`. Read from the database,
  never a cache. `aud` is never empty.
- **An administrator's session is new once its second factor passes.**
  `PassAdminSessionMFA` ends the session the password opened and starts its
  successor in one transaction, and the handler sets the new cookie, so a
  token seen before the second factor is worth nothing after it. A session
  unused for `AdminSecurity.IdleTimeoutMinutes` is refused (`last_seen_at`,
  written at most once a `model.SeenEvery`). A passkey answers only the
  ceremony its session was offered (`passkey_ceremony`), spent on the first
  answer, and is counted against the lockout as a code is.
- **The panel is set up once.** While no administrator exists, the setup
  page makes the first, a super admin; `CreateFirstAdmin` counts them under
  an advisory lock in the transaction that makes it, so racing setups make
  one and any after is refused. Whoever reaches a fresh panel first sets it
  up, so an installation is set up as soon as it is up.
- **Nothing internal reaches a browser.** Every answer is a defined problem;
  anything else goes through `respond.Failure`, which logs it and answers 500.
  The request log records the path, never the query string.
- **An uploaded picture is what its bytes are.** The organisation's logo is
  decoded by `model.NewOrganizationLogo` — PNG or JPEG only, never SVG, which
  would run script on the apps' origin; at most 512 KB and 4096 pixels a
  side — and served with its stored type and `nosniff`.
- **Every response carries security headers** — the API's from
  `middleware.SecurityHeaders`, each app's (`Permissions-Policy` among them)
  from `src/lib/server/headers.ts`. HSTS is sent by both over https
  (`SecurityHeaders(https)`, `setPageHeaders(headers, url)`).
  A rendered page holds the reader's data, so every page is `no-store`.
- **The apps run no script they did not ship.** Their Content Security Policy
  is SvelteKit's `csp` option in each `vite.config.ts`, sent as a header:
  `script-src 'self'` plus a fresh nonce for SvelteKit's own scripts, and
  `frame-ancestors 'none'`. Never add `'unsafe-inline'` or
  `'unsafe-eval'` to scripts, an inline `<script>` to `app.html`, or an outside
  origin to `connect-src`; a new external resource is a deliberate change to
  that list.

## How things are added

**An endpoint.** The method in its package under `internal/interfaces/api/`, keeping the
four files; mounted in `internal/interfaces/api/server.go` behind `session.Require` and
the permission it needs. A route under `managed` (`session.RequireAny`) also
takes an admin-cli token; a route for a person alone goes under `signedIn`.
`/endpoint` walks through it.

**An error.** Never a sentence in Go. Define the problem beside the handler —
`var taken = respond.Define(http.StatusConflict, "language_code_taken", respond.Admin)`
— or in `kit/respond/shared.go` if several packages use it. Answer with
`respond.Fail(c, taken)` or return `taken.With("code", code)` (a write that
may hit a unique value: `respond.Taken`), and add
`error.language_code_taken` to `i18n/<app>/server.en.json` for each app
it is for (`Public` → `id`, `Admin` → `console`, `Both`). A parameter in the
sentence is listed in `TestErrorParametersAreSent`. A malformed id is answered
like a missing one: `*_not_found`. The one exception is `respond.Invalid`,
which passes on a model's own validation sentence.

**A model.** A struct in its own file in `internal/domain/model`, embedding `Base`,
registered in `model.All()` — bump the count in `TestAllListsEveryModel`. A
plain `string` column is `not null;default:''`, a `bool` or `int` is
`not null`, anything that may be absent is a pointer. Nothing is soft-deleted.
A column holding one of a set of values carries a named
`check:chk_<table>_<column>,<column> IN (…)` listing exactly the Go constants
(`TestEachCheckListsWhatGoAccepts`); a column naming another row has its
relation with a deliberate `constraint:OnDelete:` and an index; JSON is
`type:jsonb;serializer:json`. Anything with an `ExpiresAt` goes in the
sweep's `expiring` list (`TestEverythingThatExpiresIsSwept`).

**A migration.** `migrations/setup` builds a fresh database from
`schema/up.sql`, written out once and frozen, and seeds it with named
columns (`seed.go`). Every change after it is a changelog, its own file in
`migrations/changelogs` (`make migrate-new name=x`; `/migration`), whose
statements are written out, never read from the models, and whose `down`
undoes it. `TestLiveTheMigrationsMakeWhatTheModelsDescribe`
fails when a model and the migrations disagree — a column, a constraint or an
index — so a model change and its migration land together. Test the upgrade
of an existing database, not just a fresh one.

**A permission.** A constant and catalog entry in
`internal/domain/model/admin_permission.go`, routes guarded with `session.Can` or
`session.CanAnywhere`, and the name added to `AdminPermissionName` in
`web/console/src/lib/api/types.ts`. `store.EnsureSystemAPIs` makes it an
admin API scope on the next start.

**An activity log entry.** `h.audit.Record` / `RecordWith` in the handler, then
a sentence, icon and category in
`web/console/src/lib/components/activity/actions.ts`. If the log can name the
target, say who may see that name in `maySeeName`
(`internal/interfaces/api/activity/handler.go`).

**A setting the outside world sees.** Follow `internal/interfaces/api/organization/`: the
panel writes it, the discovery document publishes the agreements
(`internal/application/oidc/provider.go`), and the sign-in pages read the rest from
`GET /api/v1/account/organization`, which takes no session — so only what a
stranger may see goes in `PublicOrganization`.

**A setting that decides how people sign in.** Like the organisation's or
`admin_security`: a singleton row seeded from configuration on first start,
owned by the panel afterwards, read where it matters on each request.

**A social provider.** One entry in `model.SocialSpecs`. **A login step.** One
entry in `model.LoginStepSpecs`; `Implemented` says whether the server runs it.

**A cached read.** Only in the store, with `cached(ctx, s, group, field, load)`,
only for what every request asks for and almost nothing changes. A new group
is a constant in `internal/infrastructure/cache` listed in `cache.Groups`. Every write calls
`s.forget(ctx, group)` (or returns `s.forgetting(ctx, err, group)`) after it
commits. Cache a projection, never a model carrying a secret, and add it to
`TestCachedTypesSurviveJSON`. Never cache users, codes or refresh tokens.
Everything must work with a nil cache.

**A brand.** Each application has its own: the server's in `internal/brand`
(Go constants and `favicon.svg`), each app's in `src/lib/brand.ts` (`BRAND`,
the cookie names built from `BRAND.slug`; the console's Help links and
`pageTitle` too) with its favicon in `src/lib/assets/favicon.svg`. Each may
name the product, draw its mark and pick its favicon as it likes; the slug
must be the same in all three, because the server reads back cookies the apps
name, which `TestTheAppsBuildTheSameNames` checks. A new field goes in the
app's `BRAND` and is read as `BRAND.field`; a new asset goes in
`src/lib/assets/` and is imported there, so a build fingerprints it. A new
name for everything is `make rename name="New Name" slug=newslug`, which also
rewrites what has to spell it (go.mod, `cmd/<slug>`, the environment prefix,
scripts, the compose and Docker files, CI, docs). An installation's own name, app logo, terms
and privacy links are not a brand: an administrator sets them in Settings
(the logo is an uploaded file, `model.OrganizationLogo`, never an address),
and they take the brand's place in the console's header and on the sign-in
pages. Where the apps and the API are reached is
configuration (`VARD_ISSUER`, `VARD_ACCOUNT_URL`, `VARD_ADMIN_URL`).

**A language** (sign-in pages only; the panel is English). English is the
only one that ships, and the default on a fresh installation. Every other is
added on the panel's Languages page (Settings), translated there, and offered
when it is turned on; one of them may be made the default the sign-in pages
show before somebody chooses. Nothing but `*.en.json` goes in `i18n/`. The
server formats text with go-i18n (`i18n.Catalog`, `{name}` placeholders, as
svelte-i18n does on the pages); a group may not hold only keys go-i18n
reserves (`other`, `description`, `one`, …), which `TestEveryKeyIsRead` catches.

**A panel page.** A folder under `src/routes/(panel)/`, whose layout signs
the administrator in and draws the header and sidebar. `+page.ts` checks the
permission (`requirePermission` / `requireAnywhere`, from `#lib/api/load.js`)
and fetches through the typed client in `lib/api/admin.ts`, wrapped in
`fromApi` — an endpoint's path and answer are written there once, never in a
load; the answer's type in `lib/api/types.ts` is held to the Go struct by the
package's `types_test.go` (`tstypes.Match`), so a new answer adds its pair there. The API enforces the same permission on every call. `+page.svelte` seeds
a TanStack query from it. Query options in `lib/query/` (key in
`lib/query/keys.ts`), components in `lib/components/<feature>/`. A list's
filters live in its address: `setSearchParams` and `debounce` from
`#lib/utils/url.js`. Add the route to `lib/components/layout/sidebar/sections.ts`
with an `allowed` check. Build from `#lib/components/ui`; a page never writes a
colour, a height or a hover state itself.

**A form in the panel.** It posts to a form action, enhanced with `use:enhance`.
Actions live by subject in `src/routes/actions/<subject>/+page.server.ts`, so a
dialog posts to the same one wherever it is mounted, and each is a method of
the typed client — `update: action(usersApi.update)` — or, where one save is
several calls, `act(event, …)` (`#lib/server/actions.js`). The form sends the
method's arguments as its `input` field, and `submit()` from `#lib/forms.js`
builds the `use:enhance` callback that sends them and applies the answer to
the page's state and its queries. `act` refuses a form from another origin
before it calls (SvelteKit's own check is off under its dev server); the call
goes through the request's fetch, with the administrator's cookies and this
origin, and the cookies the API sets come back with the action's answer. A
change made with one click (a remove, a toggle) and a search box stay as they
are: a call from the browser, and the address's filters. The sign-in pages have
no form actions, ever: SvelteKit's origin check is off there.

**A colour, size or font in the panel.** `web/console/src/lib/theme/theme.ts`
decides; `bun run theme` writes `lib/styles/tokens.css` and `fonts.css` (never
edit those by hand). Both apps use two faces and no others: Chirp for text,
Google Sans Code for code; the sign-in pages' `lib/styles/fonts.css` declares
the same two.

**An icon** (both apps). The Remix Icon font, as PocketBase uses it: the root
layout loads `remixicon/fonts/remixicon.css`, and an icon is its class on an
`<i>` — `<i class="ri-add-line" aria-hidden="true"></i>` (an icon that means
something on its own takes `role="img"` and an `aria-label` instead). Names are
on remixicon.com. Passed as a value an icon is that class, a string typed
`IconName` (`icon="ri-add-line"`, drawn `<i class={icon} …></i>`). Its size is
its font size: 16px in the panel and 18px on the sign-in pages (`base.css`);
any other is `style:font-size`. `icons.spec.ts` fails on a class the font
lacks. `remixicon` stays pinned at 4.8.0, the last Apache-2.0 release: from
4.9.0 its licence forbids using an icon as a logo, which the sign-in pages'
shield mark does.

## Conventions

- **Text.** Sign-in pages: never a literal — `const t = useTranslator()` and
  `t('area.thing', { param })`, the key added to its group in `i18n/id/*.en.json`.
  Panel: the opposite — English in the markup; an error shown is the sentence
  the API sent (`messageOf` from `#lib/api/index.js`).
- **The project's own name is never spelled out.** It is written only in the
  brands: `internal/brand` and each app's `src/lib/brand.ts`
  (`import { BRAND, pageTitle } from '#lib/brand.js'`).
  `TestNothingElseSpellsTheName` fails on it anywhere else in the code.
- **SvelteKit 3.** Imports from `src/lib` use `#lib` and name the file
  (`import { COOKIES } from '#lib/brand.js'`); there is no `$lib` and no
  `svelte.config.js` — configuration is `sveltekit({...})` in `vite.config.ts`.
  `resolve()` reads a leading slash as a route id, groups included
  (`resolve('/(panel)/users')`). `page.url` is read-only. Reload
  with `refreshAll()`, not `invalidateAll()`. Fetch in the page's `load`
  (`+page.ts`), never in `onMount`. Code that runs while a page renders runs
  on the server too: no `document`, `window` or `localStorage` outside an
  effect, an event handler or a `browser` check. What the server needs to do
  for the API is the API's job: a console form action only passes the form
  to it (A form in the panel). The theme is the
  reader's cookie, which `hooks.server.ts` writes into `<html data-theme>` as
  the page is rendered; no script applies it. A choice the reader makes (theme,
  language, the sidebar) is written and read through `#lib/utils/cookies.js`
  (`rememberChoice`, `readChoice`, `setChoice`): over https a `__Host-` cookie,
  read by its whole name, never matched by a pattern.
- **Comments** are few and short and say _why_ — a security reason, a race, a
  protocol rule — never what the next line does.
- **Names** are spelled out; identifiers use American spelling (`Normalize`,
  `Organization`), prose may be British.
- **Tests** are table-driven and named for what they prove ("a slug with
  spaces", not "TestCase3"). Tests needing Postgres are named `TestLive…` and
  skip without `VARD_TEST_DB_DSN`. A new component that people use goes in
  the app's `components/a11y.spec.ts`, which runs axe on it in jsdom
  (`expectAccessible` from `#lib/testing/a11y.js`).
- **Formatting is enforced.** The hook in `.claude/hooks/format.sh` runs gofmt
  or the app's Prettier on every file written; `bunx prettier --write <path>`
  fixes what `bun run lint` refuses.

## Gotchas and warnings

- **Never read `.env` or `compose.env`** — they hold the secret key and the
  stack's passwords. `.env.example` documents
  every variable.
- **Do not restart a running stack.** If anything answers on 8080, 8081, 5173
  or 5174, leave it alone; `/preview` raises a throwaway one on spare ports
  with its own database and Redis prefix, and takes it down afterwards.
- **Never edit a migration that has run anywhere.**
- **The browser opens the apps' ports, never the API's.** The API's URLs
  (issuer, admin URL) are the apps', and a path the API answers reaches it
  only if it is in that app's `API_PATHS`.
- **The local URLs are fixed by `scripts/start.sh`** (`make dev`, `make start`),
  which exports `VARD_ISSUER`, `VARD_ACCOUNT_URL`, `VARD_ADMIN_URL`,
  and `VARD_TRUSTED_PROXIES` over whatever `.env` says.
- **The API trusts the apps' servers and nobody else** to say who the
  caller is: loopback under `make dev`, the two containers' fixed addresses
  in `compose.yaml` — never a whole private range. The apps read the
  caller from the connection, or from `ADDRESS_HEADER` when the
  infrastructure in front sets one; the API's ports are never published. adapter-node's `BODY_SIZE_LIMIT` is the API's
  1 MiB, or it refuses a logo or a SAML response first.
- **The scripts read `.env` with `load_env` (`scripts/lib.sh`), the server
  with viper,** and the scripts export what they read, which the server then
  believes. A difference between the two is a different value depending on
  how the server was started — for `VARD_SECRET_KEY`, signing keys one of
  them cannot open. `TestTheScriptsReadDotEnvLikeTheServer` holds them
  together; change both or neither.
- **Postgres bounds every connection**: a statement past
  `VARD_DB_STATEMENT_TIMEOUT` (15 s) or a transaction idle past
  `VARD_DB_IDLE_IN_TRANSACTION_TIMEOUT` (60 s) is ended by the database.
  Migrations run under a session advisory lock, one instance at a time.
  `/readyz` is the orchestrator's probe (the database answers); `/healthz`
  says only that the process is up. With `VARD_DB_MIGRATE_DSN` (the
  schema owner's, as in `compose.yaml`) the server migrates as the
  owner and runs as `VARD_DB_DSN`'s role, which `MigrateAsOwner` makes and
  grants rows only — no DDL, not goose's table. A store query that needs more
  than SELECT, INSERT, UPDATE or DELETE fails there; it belongs in a
  migration.
- **Each app's address is built into it.** adapter-node no longer reads
  `ORIGIN` at run time: `vite.config.ts` takes `paths.origin` from `ORIGIN`
  when the app is built (compose passes `PUBLIC_URL` / `ADMIN_URL` as build
  arguments, `make start` the local ones). Built without it, an app guesses
  https from the headers and the console refuses its own forms over http;
  changing an address means building the images again.
- **Redis is optional.** Every path must work with a nil cache, and Redis is
  never the only copy of anything.
- **The shell is zsh.** An unquoted glob such as `--include=*.go` fails with
  "no matches found" — quote it.
- **TypeScript stays on 6** until SvelteKit, `svelte-check` and
  `typescript-eslint` support 7.

## Definition of done

A change is finished when all of this holds, and the summary says which checks
ran and their result — or which were skipped and why:

1. `make check` passes: formatting, go vet, staticcheck, both apps' lint and
   type-check, and every unit test.
2. If `compose.yaml`, `compose.env.example` or a Dockerfile changed: `make docker-config` passes.
3. If a migration, the store, hand-written SQL or a route changed:
   `make test-integration` passes.
4. Every new problem code is in the catalogs, every new key in every language,
   every new permission in `types.ts`.
5. New behaviour has a test; a security case has one without exception.
6. A user-visible change was looked at in a browser (`/preview`), and a change
   to a flow web/e2e covers passes `make e2e`.
7. `README.md` and this file say nothing the change made untrue.

## Git

- **Never commit, push, branch, tag or open a pull request unless asked** in
  this conversation. Leave changes in the working tree and say what changed.
- When asked to commit: never `--amend`, `--force` or `--no-verify`, and never
  rewrite history that has been pushed. Branch off `main` first if on it.
- `.claude/settings.local.json` is personal and is not committed.
