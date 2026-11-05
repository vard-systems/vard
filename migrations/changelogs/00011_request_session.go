package changelogs

import (
	"context"
	"database/sql"
	"fmt"
)

func init() {
	add(11, upRequestSession, downRequestSession)
}

// upRequestSession binds a sign-in waiting on its Agreements or Consent page
// to the session that reached it, so only that session may answer the page.
// Requests already waiting name none and are refused there; their user
// starts the sign-in again.
func upRequestSession(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`ALTER TABLE authorization_requests ADD COLUMN session_id uuid`,
		`ALTER TABLE authorization_requests ADD CONSTRAINT fk_authorization_requests_session
			FOREIGN KEY (session_id) REFERENCES user_sessions(id) ON DELETE SET NULL`,
		`CREATE INDEX idx_authorization_requests_session_id ON authorization_requests USING btree (session_id)`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("request session: %w", err)
		}
	}
	return nil
}

// downRequestSession drops it. What session reached each waiting step page is
// lost, and any session may answer one again.
func downRequestSession(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`DROP INDEX IF EXISTS idx_authorization_requests_session_id`,
		`ALTER TABLE authorization_requests DROP COLUMN IF EXISTS session_id`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("undo request session: %w", err)
		}
	}
	return nil
}
