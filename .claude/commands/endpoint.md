---
description: Add an admin API endpoint, the way this repository adds one
argument-hint: "<what it should do>"
---

Add an endpoint for: $ARGUMENTS

Follow the shape the other packages already have, and read the nearest one
first — `internal/interfaces/api/fields/` is the smallest complete example.

1. The handler method in the package under `internal/interfaces/api/` the subject belongs
   to, or a new package of the same four files: `handler.go`, `request.go`,
   `response.go`, `validation.go`. A handler reads the request, asks the store
   and answers; it decides no statuses of its own beyond the happy one, passing
   everything else to `respond.Failure`.
2. Queries in `internal/infrastructure/store` — never in the handler — returning models, and
   `store.ErrNotFound` / `store.ErrDuplicate` for what the caller has to know.
3. The rules in `validation.go`, or in the model when they are what the thing
   *is* rather than what this request may say. A broken rule is a problem
   from `respond.Define`, with its sentence in `i18n/console/server.en.json`
   (see "An error" in CLAUDE.md) — never a sentence written in Go.
4. Mount it in `registerAdminRoutes` in `internal/interfaces/api/server.go`, behind
   `session.Require` and the permission it needs, on the admin server unless it
   is genuinely public.
5. `h.audit.Record(...)` for anything that changes something, and a line in
   `web/console/src/lib/components/activity/actions.ts` for the new action.
6. Tests: table-driven ones for the validation, and a `TestLive…` in
   `internal/interfaces/api/integration_test.go` when the route, the store or a permission
   is what you want to prove.

Then run `/check`.
