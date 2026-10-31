---
description: Write a database migration the way this repository writes them
argument-hint: "<what it should change>"
---

Write a migration for: $ARGUMENTS

A change to the schema is a changelog: a Go file in `migrations/changelogs`,
numbered in order (`00004_x.go` is version 4), that registers itself from
`init` with `add(4, upX, downX)`. `make migrate-new name=x` writes it with the
next number. `migrations/setup` is version 1, the whole first schema
(`schema/up.sql`, frozen, and `seed.go`), and is never touched;
`migrations.All()` hands goose the setup and every changelog.

- `up` takes goose's transaction and runs statements written out in full —
  `CREATE TABLE`, `ALTER TABLE … ADD COLUMN`, `CREATE INDEX` — never
  `AutoMigrate` or anything else read from the models, so the migration says
  the same thing however the models change later.
- Write each statement as the model would make it: the drift test,
  `TestLiveTheMigrationsMakeWhatTheModelsDescribe`, compares every column,
  constraint and index of a migrated database with what `AutoMigrate` of
  `model.All()` makes, and names whatever differs.
- A new `not null` column needs a value for the rows that already exist: add
  it with a default, and drop the default after if the model has none.
- `down` undoes it — `DROP TABLE`, `DROP COLUMN` — and says in a comment what
  is lost.
- A new model goes in `model.All()`, and `TestAllListsEveryModel` counts them,
  so bump the count.
- An index on a table that grows (refresh_tokens, audit_logs, users) is made
  `CREATE INDEX CONCURRENTLY`, which cannot run in a transaction: register
  that changelog with `goose.GoFunc{RunDB: …}` instead of `add`, so an upgrade does not
  hold up writes while it builds.
- The server's connections end a statement after `VARD_DB_STATEMENT_TIMEOUT`
  (15 s). A step that may take longer on a large database starts with
  `SET LOCAL statement_timeout = 0` in its transaction.
- Never edit a migration that has already run anywhere.

Then `make migrate-up`, `make migrate-status`, and `make test-integration`,
which applies every migration to a throwaway database and runs the drift
test.
