package changelogs

import (
	"context"
	"database/sql"
	"fmt"
)

func init() {
	add(4, upAPIIntrospectors, downAPIIntrospectors)
}

// upAPIIntrospectors makes the list of applications that may introspect an
// API's tokens. It starts empty: until an administrator lists one, an API's
// tokens are answered only to the client they were issued to, as before.
func upAPIIntrospectors(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`CREATE TABLE api_introspectors (
			api_id uuid NOT NULL,
			application_id uuid NOT NULL,
			created_at timestamp with time zone NOT NULL,
			CONSTRAINT api_introspectors_pkey PRIMARY KEY (api_id, application_id),
			CONSTRAINT fk_api_introspectors_api FOREIGN KEY (api_id) REFERENCES apis(id) ON DELETE CASCADE,
			CONSTRAINT fk_api_introspectors_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX idx_api_introspectors_application ON api_introspectors USING btree (application_id)`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("make api_introspectors: %w", err)
		}
	}
	return nil
}

// downAPIIntrospectors drops the list: every API's tokens are answered only to
// their own client again, and who was listed is lost.
func downAPIIntrospectors(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS api_introspectors`); err != nil {
		return fmt.Errorf("drop api_introspectors: %w", err)
	}
	return nil
}
