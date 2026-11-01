package changelogs

import (
	"context"
	"database/sql"
	"fmt"
)

func init() {
	add(8, upSignInApplication, downSignInApplication)
}

// upSignInApplication lets a waiting sign-in with no authorization request
// name the application it is for — a device being allowed — so it is held
// to that application's login flow. Waiting sign-ins there name none.
func upSignInApplication(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`ALTER TABLE pending_sign_ins ADD COLUMN application_id uuid`,
		`ALTER TABLE pending_sign_ins ADD CONSTRAINT fk_pending_sign_ins_application
			FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE`,
		`CREATE INDEX idx_pending_sign_ins_application_id ON pending_sign_ins USING btree (application_id)`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("sign-in application: %w", err)
		}
	}
	return nil
}

// downSignInApplication drops it. Sign-ins still waiting for an application
// are deleted first: without it they would be held to the default flow,
// which may ask less than the application's.
func downSignInApplication(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`DELETE FROM pending_sign_ins WHERE application_id IS NOT NULL`,
		`DROP INDEX IF EXISTS idx_pending_sign_ins_application_id`,
		`ALTER TABLE pending_sign_ins DROP COLUMN IF EXISTS application_id`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("undo sign-in application: %w", err)
		}
	}
	return nil
}
