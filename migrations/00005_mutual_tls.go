package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upMutualTLS, downMutualTLS)
}

// upMutualTLS adds what RFC 8705 needs: how an application's client
// certificate is recognised and whether its tokens are bound to it, and the
// certificate a public client's refresh family is bound to. Existing rows
// keep working as before: no certificate, nothing bound. On a fresh database
// the schema already has all of it, and every step here is a no-op.
func upMutualTLS(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`ALTER TABLE applications ADD COLUMN IF NOT EXISTS tls_subject_kind varchar(16) NOT NULL DEFAULT ''`,
		`ALTER TABLE applications ADD COLUMN IF NOT EXISTS tls_subject varchar(512) NOT NULL DEFAULT ''`,
		`ALTER TABLE applications ADD COLUMN IF NOT EXISTS tls_certificates text`,
		`ALTER TABLE applications ADD COLUMN IF NOT EXISTS certificate_bound_tokens boolean NOT NULL DEFAULT false`,
		`ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS certificate_thumbprint varchar(64) NOT NULL DEFAULT ''`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("add the mutual TLS columns: %w", err)
		}
	}

	return nil
}

// downMutualTLS loses every application's certificate settings, and the
// certificate binding of refresh tokens, which are then taken without one. An
// application left on a certificate method no longer validates; move it to a
// secret before going down.
func downMutualTLS(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS certificate_thumbprint`,
		`ALTER TABLE applications DROP COLUMN IF EXISTS certificate_bound_tokens`,
		`ALTER TABLE applications DROP COLUMN IF EXISTS tls_certificates`,
		`ALTER TABLE applications DROP COLUMN IF EXISTS tls_subject`,
		`ALTER TABLE applications DROP COLUMN IF EXISTS tls_subject_kind`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("drop the mutual TLS columns: %w", err)
		}
	}

	return nil
}
