---
description: Write a database migration the way this repository writes them
argument-hint: "<what it should change>"
---

Write a migration for: $ARGUMENTS

Migrations are Go files in `migrations/`, one per change, each registering
itself with goose from `init`. `make migrate-new name=x` makes the file;
`migrations/00001_schema.go` holds `gormTx` and shows the shape.

- `up` takes goose's transaction, wraps it with `gormTx`, and uses the models:
  `db.AutoMigrate(&model.Thing{})` rather than hand-written DDL, so the table
  and the struct cannot drift.
- A new column that is `not null` needs a value for the rows that already
  exist: add it with a default first, then drop the default, or GORM will
  refuse to store a false.
- `down` undoes it — `Migrator().DropTable`, `DropColumn` — and says in a
  comment what is lost.
- A new model goes in `model.All()`, and `TestAllListsEveryModel` counts them,
  so bump the count.
- Never edit a migration that has already run anywhere.

Then `make migrate-up`, `make migrate-status`, and `make test-integration`,
which applies every migration to a throwaway database.
