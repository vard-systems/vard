package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upSSODomainProof, downSSODomainProof)
}

// upSSODomainProof adds the proof a connection holds each of its domains,
// which linking to an existing account takes, and the identity a waiting
// sign-in links once its second factor is answered. Every domain a
// connection already lists is taken as held, so an existing connection links
// as it did; a domain added from now on is proved in DNS first. On a fresh
// database the schema already has the table and columns, and there is no
// connection to carry over.
func upSSODomainProof(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS sso_domain_verifications (
			id uuid PRIMARY KEY,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL,
			connection_id uuid NOT NULL CONSTRAINT fk_sso_domain_verifications_connection REFERENCES sso_connections (id) ON DELETE CASCADE,
			domain varchar(253) NOT NULL,
			token varchar(64) NOT NULL,
			method varchar(16) NOT NULL DEFAULT '',
			verified_at timestamptz,
			checked_at timestamptz
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_sso_domain_verifications_domain ON sso_domain_verifications (connection_id, domain)`,
		`INSERT INTO sso_domain_verifications (id, created_at, updated_at, connection_id, domain, token, method, verified_at)
			SELECT gen_random_uuid(), now(), now(), connections.id, listed.domain,
				encode(sha256(gen_random_uuid()::text::bytea), 'hex'), 'existing', now()
			FROM sso_connections connections, jsonb_array_elements_text(connections.domains) AS listed (domain)
			ON CONFLICT (connection_id, domain) DO NOTHING`,
		`ALTER TABLE pending_sign_ins ADD COLUMN IF NOT EXISTS link_connection_id uuid CONSTRAINT fk_pending_sign_ins_link_connection REFERENCES sso_connections (id) ON DELETE CASCADE`,
		`ALTER TABLE pending_sign_ins ADD COLUMN IF NOT EXISTS link_subject varchar(512) NOT NULL DEFAULT ''`,
		`ALTER TABLE pending_sign_ins ADD COLUMN IF NOT EXISTS link_email varchar(255) NOT NULL DEFAULT ''`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("add the SSO domain proofs: %w", err)
		}
	}

	return nil
}

// downSSODomainProof loses every proof, and the links waiting sign-ins were
// to make; those sign-ins then finish without linking.
func downSSODomainProof(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`ALTER TABLE pending_sign_ins DROP COLUMN IF EXISTS link_email`,
		`ALTER TABLE pending_sign_ins DROP COLUMN IF EXISTS link_subject`,
		`ALTER TABLE pending_sign_ins DROP COLUMN IF EXISTS link_connection_id`,
		`DROP TABLE IF EXISTS sso_domain_verifications`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("drop the SSO domain proofs: %w", err)
		}
	}

	return nil
}
