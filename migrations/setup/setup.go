// Package setup is the first migration: the whole schema in one step,
// schema/up.sql, written out once and never changed, and the rows an
// installation starts with (seed.go). A later change is a changelog
// (migrations/changelogs), never an edit here: a migration that has run
// anywhere says the same thing forever.
package setup

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Version is the setup's place in goose_db_version; every changelog comes
// after it.
const Version = 1

// Migration builds the database from nothing, and its down takes it all
// away again.
var Migration = goose.NewGoMigration(Version, &goose.GoFunc{RunTx: upSchema}, &goose.GoFunc{RunTx: downSchema})

var (
	//go:embed schema/up.sql
	schemaUp string
	//go:embed schema/down.sql
	schemaDown string
)

// gormTx wraps the transaction goose hands a migration in GORM, so a
// migration can use the migrator instead of writing every statement by hand.
// Everything it does runs in goose's transaction, so a migration that fails
// half way leaves nothing behind.
func gormTx(tx *sql.Tx) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: tx}), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("wrap transaction: %w", err)
	}
	return db, nil
}

// upSchema builds the whole database: every table, constraint and index, the
// search index where the database allows it, and the few rows an installation
// starts with.
func upSchema(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	if err := db.Exec(schemaUp).Error; err != nil {
		return fmt.Errorf("create the schema: %w", err)
	}

	if err := userSearchIndex(db); err != nil {
		return err
	}

	return seed(db)
}

// downSchema drops everything again.
func downSchema(_ context.Context, tx *sql.Tx) error {
	if _, err := tx.Exec(schemaDown); err != nil {
		return fmt.Errorf("drop the schema: %w", err)
	}
	return nil
}

// userSearch lets the users page find a word anywhere in a user's address,
// name or fields. It needs pg_trgm, which a managed database may not let us
// install; without it the search still works, by scanning.
const userSearch = `CREATE INDEX idx_users_search ON users USING gin (
	LOWER(email || ' ' || COALESCE(first_name, '') || ' ' || COALESCE(last_name, '') || ' ' || COALESCE(data::text, ''))
	gin_trgm_ops)`

func userSearchIndex(db *gorm.DB) error {
	if err := db.Exec("SAVEPOINT trigram").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
		return db.Exec("ROLLBACK TO SAVEPOINT trigram").Error
	}

	if err := db.Exec(userSearch).Error; err != nil {
		return fmt.Errorf("create idx_users_search: %w", err)
	}

	return nil
}
