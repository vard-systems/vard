package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upTokenRevocations, downTokenRevocations)
}

// upTokenRevocations adds what ends access tokens before they expire: the
// sessions, refresh families, grants and tokens revoked, each kept as long
// as a token it ends could live. Tokens issued before the upgrade name no
// session or family, so only revoking them by jti reaches them. On a fresh
// database the schema already has the table.
func upTokenRevocations(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS token_revocations (
			id uuid PRIMARY KEY,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL,
			key varchar(160) NOT NULL,
			revoked_at timestamptz NOT NULL,
			expires_at timestamptz NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_token_revocations_key ON token_revocations (key)`,
		`CREATE INDEX IF NOT EXISTS idx_token_revocations_expires_at ON token_revocations (expires_at)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("add the token revocations: %w", err)
		}
	}

	return nil
}

// downTokenRevocations forgets every revocation: an access token revoked
// before is taken again until it expires.
func downTokenRevocations(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	if err := db.Exec(`DROP TABLE IF EXISTS token_revocations`).Error; err != nil {
		return fmt.Errorf("drop the token revocations: %w", err)
	}

	return nil
}
