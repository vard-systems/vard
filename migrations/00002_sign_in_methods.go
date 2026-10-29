package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upSignInMethods, downSignInMethods)
}

// signInMethodTables are where how a person signed in is kept, from the
// sign-in to the tokens it leads to, for the ID token's amr claim.
var signInMethodTables = []string{
	"user_sessions", "pending_sign_ins", "authorization_codes", "device_authorizations", "refresh_tokens",
}

// upSignInMethods adds the methods column to each. Rows from before keep
// none, and their tokens carry no amr claim.
func upSignInMethods(_ context.Context, tx *sql.Tx) error {
	for _, table := range signInMethodTables {
		if _, err := tx.Exec(`ALTER TABLE ` + table + ` ADD COLUMN IF NOT EXISTS methods jsonb`); err != nil {
			return fmt.Errorf("add %s.methods: %w", table, err)
		}
	}
	return nil
}

// downSignInMethods loses how existing sessions and tokens were signed in;
// they go on working, without an amr claim.
func downSignInMethods(_ context.Context, tx *sql.Tx) error {
	for _, table := range signInMethodTables {
		if _, err := tx.Exec(`ALTER TABLE ` + table + ` DROP COLUMN IF EXISTS methods`); err != nil {
			return fmt.Errorf("drop %s.methods: %w", table, err)
		}
	}
	return nil
}
