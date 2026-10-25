package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upDevelopmentRedirects, downDevelopmentRedirects)
}

// upDevelopmentRedirects adds the flag that lets a native app in development
// register Expo Go's exp:// addresses. Every existing application starts
// without it. On a fresh database the schema already has the column.
func upDevelopmentRedirects(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	if err := db.Exec(`ALTER TABLE applications ADD COLUMN IF NOT EXISTS allow_development_redirects boolean NOT NULL DEFAULT false`).Error; err != nil {
		return fmt.Errorf("add allow_development_redirects: %w", err)
	}

	return nil
}

// downDevelopmentRedirects loses the flag; an application that registered an
// exp:// address keeps it, but no longer validates until it is removed.
func downDevelopmentRedirects(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	if err := db.Exec(`ALTER TABLE applications DROP COLUMN IF EXISTS allow_development_redirects`).Error; err != nil {
		return fmt.Errorf("drop allow_development_redirects: %w", err)
	}

	return nil
}
