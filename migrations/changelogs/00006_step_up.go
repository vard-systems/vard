package changelogs

import (
	"context"
	"database/sql"
	"fmt"
)

func init() {
	add(6, upStepUp, downStepUp)
}

// upStepUp lets a login flow require a second factor, and a waiting sign-in
// name the session a step-up adds its factor to. Every flow there requires
// none, as before.
func upStepUp(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`ALTER TABLE login_flows ADD COLUMN require_second_factor boolean NOT NULL DEFAULT false`,
		`ALTER TABLE login_flows ALTER COLUMN require_second_factor DROP DEFAULT`,
		`ALTER TABLE pending_sign_ins ADD COLUMN step_up_session_id uuid`,
		`ALTER TABLE pending_sign_ins ADD CONSTRAINT fk_pending_sign_ins_step_up_session
			FOREIGN KEY (step_up_session_id) REFERENCES user_sessions(id) ON DELETE CASCADE`,
		`CREATE INDEX idx_pending_sign_ins_step_up_session_id ON pending_sign_ins USING btree (step_up_session_id)`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("step-up: %w", err)
		}
	}
	return nil
}

// downStepUp drops both. Step-ups still waiting are deleted first: without
// their session they would read as sign-ins of their own, which their first
// factor never proved. Flows that required a second factor ask it only of
// people who turned one on again.
func downStepUp(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`DELETE FROM pending_sign_ins WHERE step_up_session_id IS NOT NULL`,
		`DROP INDEX IF EXISTS idx_pending_sign_ins_step_up_session_id`,
		`ALTER TABLE pending_sign_ins DROP COLUMN IF EXISTS step_up_session_id`,
		`ALTER TABLE login_flows DROP COLUMN IF EXISTS require_second_factor`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("undo step-up: %w", err)
		}
	}
	return nil
}
