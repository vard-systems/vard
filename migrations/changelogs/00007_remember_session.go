package changelogs

import (
	"context"
	"database/sql"
	"fmt"
)

func init() {
	add(7, upRememberSession, downRememberSession)
}

// upRememberSession records whether a session's cookie outlives the browser
// ("stay signed in"), so a step-up can write its new cookie the same way.
// Sessions already open are recorded as not remembered: a step-up of one
// writes a cookie for the browser's lifetime, as before.
func upRememberSession(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`ALTER TABLE user_sessions ADD COLUMN remember boolean NOT NULL DEFAULT false`,
		`ALTER TABLE user_sessions ALTER COLUMN remember DROP DEFAULT`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("remember session: %w", err)
		}
	}
	return nil
}

// downRememberSession drops it. Sessions stay open; a step-up of any of them
// then writes a cookie for the browser's lifetime.
func downRememberSession(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `ALTER TABLE user_sessions DROP COLUMN IF EXISTS remember`); err != nil {
		return fmt.Errorf("undo remember session: %w", err)
	}
	return nil
}
