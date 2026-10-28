package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAdminPasskeys, downAdminPasskeys)
}

// upAdminPasskeys lets an administrator's factors hold a passkey: a sealed
// credential is longer than a sealed seed, and is found by its id; and a
// session waiting for its second factor keeps the passkey ceremony it was
// offered. Existing factors and sessions are unchanged. On a fresh database
// the schema already has all of it.
func upAdminPasskeys(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`ALTER TABLE mfa_factors ALTER COLUMN secret TYPE text`,
		`ALTER TABLE mfa_factors ADD COLUMN IF NOT EXISTS credential_id varchar(255) NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_mfa_factors_credential_id ON mfa_factors (credential_id)`,
		`ALTER TABLE admin_sessions ADD COLUMN IF NOT EXISTS passkey_ceremony text NOT NULL DEFAULT ''`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("add administrators' passkeys: %w", err)
		}
	}

	return nil
}

// downAdminPasskeys removes every administrator's passkey, since the column
// that is left cannot hold one; their authenticators stay.
func downAdminPasskeys(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`DELETE FROM mfa_factors WHERE method = 'webauthn'`,
		`ALTER TABLE admin_sessions DROP COLUMN IF EXISTS passkey_ceremony`,
		`DROP INDEX IF EXISTS idx_mfa_factors_credential_id`,
		`ALTER TABLE mfa_factors DROP COLUMN IF EXISTS credential_id`,
		`ALTER TABLE mfa_factors ALTER COLUMN secret TYPE varchar(255)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("drop administrators' passkeys: %w", err)
		}
	}

	return nil
}
