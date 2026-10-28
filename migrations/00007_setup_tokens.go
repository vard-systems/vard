package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upSetupTokens, downSetupTokens)
}

// upSetupTokens adds the one-time tokens that creating the first
// administrator takes. An installation that has one already never makes a
// token. On a fresh database the schema already has the table.
func upSetupTokens(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS setup_tokens (
			id uuid PRIMARY KEY,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL,
			token_hash varchar(64) NOT NULL,
			expires_at timestamptz NOT NULL,
			used_at timestamptz
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_setup_tokens_token_hash ON setup_tokens (token_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_setup_tokens_expires_at ON setup_tokens (expires_at)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("add the setup tokens: %w", err)
		}
	}

	return nil
}

// downSetupTokens loses any token not yet used; a server started without
// the table cannot be set up until it is migrated up again.
func downSetupTokens(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	if err := db.Exec(`DROP TABLE IF EXISTS setup_tokens`).Error; err != nil {
		return fmt.Errorf("drop the setup tokens: %w", err)
	}

	return nil
}
