package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upSignInThrottles, downSignInThrottles)
}

// upSignInThrottles adds the counts users' sign-ins are throttled by, which
// take the place of the users' lock (failed_login_count and locked_until,
// which stay for the administrators). A user locked during the upgrade stays
// locked until that lock runs out. On a fresh database the schema already
// has the table.
func upSignInThrottles(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS sign_in_throttles (
			id uuid PRIMARY KEY,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL,
			key varchar(160) NOT NULL,
			failures bigint NOT NULL,
			blocked_until timestamptz,
			expires_at timestamptz NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_sign_in_throttles_key ON sign_in_throttles (key)`,
		`CREATE INDEX IF NOT EXISTS idx_sign_in_throttles_expires_at ON sign_in_throttles (expires_at)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("add the sign-in throttles: %w", err)
		}
	}

	return nil
}

// downSignInThrottles forgets every count; the server it goes back to locks
// users again by failed_login_count.
func downSignInThrottles(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	if err := db.Exec(`DROP TABLE IF EXISTS sign_in_throttles`).Error; err != nil {
		return fmt.Errorf("drop the sign-in throttles: %w", err)
	}

	return nil
}
