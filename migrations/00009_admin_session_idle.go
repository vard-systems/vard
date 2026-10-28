package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAdminSessionIdle, downAdminSessionIdle)
}

// upAdminSessionIdle adds the administrators' idle timeout and when each of
// their sessions was last used. A session open during the upgrade counts as
// used then, so nobody is signed out by it. On a fresh database the schema
// already has both columns.
func upAdminSessionIdle(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`ALTER TABLE admin_security ADD COLUMN IF NOT EXISTS idle_timeout_minutes bigint NOT NULL DEFAULT 30`,
		`ALTER TABLE admin_sessions ADD COLUMN IF NOT EXISTS last_seen_at timestamptz NOT NULL DEFAULT now()`,
		`ALTER TABLE admin_sessions ALTER COLUMN last_seen_at DROP DEFAULT`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("add the admin idle timeout: %w", err)
		}
	}

	return nil
}

// downAdminSessionIdle loses the setting and the times; sessions then end
// only at their absolute lifetime.
func downAdminSessionIdle(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`ALTER TABLE admin_sessions DROP COLUMN IF EXISTS last_seen_at`,
		`ALTER TABLE admin_security DROP COLUMN IF EXISTS idle_timeout_minutes`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("drop the admin idle timeout: %w", err)
		}
	}

	return nil
}
