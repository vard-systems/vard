package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	"vard/internal/model"
)

func init() {
	goose.AddMigrationContext(upOrganizationLogo, downOrganizationLogo)
}

// upOrganizationLogo replaces the organisation's logo address with an uploaded
// app logo: the file in organization_logos, its version on the organisation's
// row. On a fresh database the schema already has all of it, and every step
// here is a no-op.
func upOrganizationLogo(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	if err := db.AutoMigrate(&model.OrganizationLogo{}); err != nil {
		return fmt.Errorf("create organization_logos: %w", err)
	}

	for _, statement := range []string{
		`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS logo_version varchar(16) NOT NULL DEFAULT ''`,
		// An address to someone else's server is not kept: the logo is now a
		// file the panel uploads.
		`ALTER TABLE organizations DROP COLUMN IF EXISTS logo_url`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("change organizations: %w", err)
		}
	}

	return nil
}

// downOrganizationLogo loses the uploaded logo; the logo address comes back
// empty.
func downOrganizationLogo(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	for _, statement := range []string{
		`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS logo_url varchar(512) NOT NULL DEFAULT ''`,
		`ALTER TABLE organizations DROP COLUMN IF EXISTS logo_version`,
		`DROP TABLE IF EXISTS organization_logos`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("undo the organization logo: %w", err)
		}
	}

	return nil
}
