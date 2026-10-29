---
description: Write a database migration the way this repository writes them
argument-hint: "<what it should change>"
---

Write a migration for: $ARGUMENTS

Migrations are Go files in `migrations/`, one per change, each registering
itself with goose from `init`. `make migrate-new name=x` makes the file.
`migrations/00001_schema.go` is the whole first schema (`schema/up.sql`,
frozen) and holds `gormTx`.

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
- Never edit a migration that has already run anywhere.

Then `make migrate-up`, `make migrate-status`, and `make test-integration`,
which applies every migration to a throwaway database and runs the drift
test.
