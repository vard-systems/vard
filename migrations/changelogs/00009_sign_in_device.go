package changelogs

import (
	"context"
	"database/sql"
	"fmt"
)

func init() {
	add(9, upSignInDevice, downSignInDevice)
}

// upSignInDevice lets a step-up started to allow a device name it, so leaving
// the step-up denies that device. Waiting sign-ins there name none.
func upSignInDevice(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`ALTER TABLE pending_sign_ins ADD COLUMN device_authorization_id uuid`,
		`ALTER TABLE pending_sign_ins ADD CONSTRAINT fk_pending_sign_ins_device_authorization
			FOREIGN KEY (device_authorization_id) REFERENCES device_authorizations(id) ON DELETE CASCADE`,
		`CREATE INDEX idx_pending_sign_ins_device_authorization_id ON pending_sign_ins USING btree (device_authorization_id)`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("sign-in device: %w", err)
		}
	}
	return nil
}

// downSignInDevice drops it. Waiting step-ups keep their application; leaving
// one then leaves the device waiting until it expires, as before.
func downSignInDevice(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`DROP INDEX IF EXISTS idx_pending_sign_ins_device_authorization_id`,
		`ALTER TABLE pending_sign_ins DROP COLUMN IF EXISTS device_authorization_id`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("undo sign-in device: %w", err)
		}
	}
	return nil
}
