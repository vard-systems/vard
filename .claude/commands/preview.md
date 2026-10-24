---
description: Run the app on spare ports and look at a page in the browser
argument-hint: "[path, e.g. /settings]"
---

Look at $ARGUMENTS (default: `/settings`) in a running
panel, without disturbing whatever is already running.

First check whether the normal stack is up — anything answering on 8080, 8081,
5173 or 5174. If nothing is, `make dev` is the simple way and you can stop
reading. If something is, leave it alone and raise a throwaway instance beside
it:

1. A database of its own:
   `psql "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" -c "CREATE DATABASE loginer_preview;"`
2. The API, built from the working tree into `bin/` (which is ignored), on
   spare ports, applying its migrations on start:

   ```sh
   go build -o bin/loginer-preview ./cmd/loginer
   LOGINER_ADDR=:8090 LOGINER_ADMIN_ADDR=:8091 \
     LOGINER_DB_DSN="postgres://postgres:postgres@localhost:5432/loginer_preview?sslmode=disable" \
     LOGINER_DB_DRIVER=postgres LOGINER_DB_MIGRATE=true LOGINER_DB_MIGRATE_DIR=./migrations \
     LOGINER_ISSUER=http://localhost:5183 LOGINER_ACCOUNT_URL=http://localhost:5183 \
     LOGINER_ADMIN_URL=http://localhost:5184 LOGINER_ADMIN_MFA=optional \
     LOGINER_TRUSTED_PROXIES=127.0.0.1/32,::1/128 \
     LOGINER_SECRET_KEY=preview-secret-key-0123456789abcdef \
     LOGINER_REDIS_PREFIX=loginer-preview: \
     bin/loginer-preview &
   ```

   `LOGINER_ADMIN_MFA=optional` is what makes signing in one step instead of
   needing an authenticator. `LOGINER_REDIS_PREFIX` keeps its cache apart
   from the running stack's: the binary reads `.env`, so without it the two
   share one Redis, each reads the other's cached rows — an organisation
   whose id is not in its own database, say — and saves go wrong.

3. The panel against it, from `web/console` — its server renders the pages
   and passes `/api/v1/admin` on to `API_URL`, so the browser calls the API
   on the panel's own origin:
   `API_URL=http://localhost:8091 ./node_modules/.bin/vite dev --port 5184 --strictPort`
   (and, for the sign-in pages, from `web/id`:
   `API_URL=http://localhost:8090 ./node_modules/.bin/vite dev --port 5183 --strictPort`).
   The app's own Vite, not `bunx vite`: bunx can resolve a copy of its own,
   which starts without the SvelteKit plugin and answers every page 404.

4. The first administrator, since the database is empty:

   ```sh
   curl -s -X POST http://localhost:5184/api/v1/admin/setup \
     -H "Content-Type: application/json" -H "Origin: http://localhost:5184" \
     -d '{"email":"preview@example.com","password":"preview-password-1","first_name":"Preview"}'
   ```

5. Sign in at `http://localhost:5184/login` with the browser tools and
   go to the path. Look at it in both themes if the change is visual — the
   header's toggle switches them.

   Cookies on `localhost` are shared by every port, so signing in here
   replaces the running stack's session in this browser: say so when you
   report, since its owner will be signed out of it.

Then take it all down: stop every process you started — the API and the Vite
servers, found by their spare ports, never the running stack's —
`DROP DATABASE loginer_preview WITH (FORCE)`, clear its cache
(`redis-cli --scan --pattern 'loginer-preview:*' | xargs redis-cli del`), and
close the tab. Say what you saw, and what you changed if you fixed something.
