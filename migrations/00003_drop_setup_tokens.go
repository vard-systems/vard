package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upDropSetupTokens, downDropSetupTokens)
}

// upDropSetupTokens drops the one-time tokens the first administrator was
// made with: setting the panel up no longer takes one.
func upDropSetupTokens(_ context.Context, tx *sql.Tx) error {
	if _, err := tx.Exec(`DROP TABLE IF EXISTS setup_tokens`); err != nil {
		return fmt.Errorf("drop setup_tokens: %w", err)
	}
	return nil
}

// downDropSetupTokens makes the table again, empty: a token issued before is
// not brought back.
func downDropSetupTokens(_ context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`CREATE TABLE setup_tokens (
			id uuid NOT NULL,
			created_at timestamp with time zone NOT NULL,
			updated_at timestamp with time zone NOT NULL,
			token_hash character varying(64) NOT NULL,
			expires_at timestamp with time zone NOT NULL,
			used_at timestamp with time zone,
			CONSTRAINT setup_tokens_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_setup_tokens_expires_at ON setup_tokens USING btree (expires_at)`,
		`CREATE UNIQUE INDEX idx_setup_tokens_token_hash ON setup_tokens USING btree (token_hash)`,
	} {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("make setup_tokens again: %w", err)
		}
	}
	return nil
}
