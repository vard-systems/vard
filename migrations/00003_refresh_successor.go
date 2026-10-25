package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upRefreshSuccessor, downRefreshSuccessor)
}

// upRefreshSuccessor lets a replaced refresh token name what replaced it,
// and keep that token sealed through the grace, so a retry is answered with
// the same token. Existing replaced tokens have none and stay replays, as
// they were. On a fresh database the schema already has both columns.
func upRefreshSuccessor(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS successor_id uuid`,
		`ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS successor_sealed bytea`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("add the refresh token successor: %w", err)
		}
	}

	return nil
}

// downRefreshSuccessor loses only the grace: a refresh token replaced in the
// last half minute is then a replay, as before.
func downRefreshSuccessor(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS successor_sealed`,
		`ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS successor_id`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("drop the refresh token successor: %w", err)
		}
	}

	return nil
}
