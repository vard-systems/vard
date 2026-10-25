package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upSealedSuccessorIndex, downSealedSuccessorIndex)
}

// upSealedSuccessorIndex indexes the refresh tokens still holding a sealed
// successor, which are cleared every grace: the few rows in it, not the
// table. On a fresh database the schema already has it.
func upSealedSuccessorIndex(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_sealed ON refresh_tokens (revoked_at) WHERE successor_sealed IS NOT NULL`).Error; err != nil {
		return fmt.Errorf("index the sealed refresh tokens: %w", err)
	}

	return nil
}

// downSealedSuccessorIndex loses only the index; clearing them reads the
// table instead.
func downSealedSuccessorIndex(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	if err := db.Exec(`DROP INDEX IF EXISTS idx_refresh_tokens_sealed`).Error; err != nil {
		return fmt.Errorf("drop the sealed refresh tokens' index: %w", err)
	}

	return nil
}
