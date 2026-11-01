package changelogs

import (
	"context"
	"database/sql"
	"fmt"
)

func init() {
	add(10, upSSOSecondFactor, downSSOSecondFactor)
}

// upSSOSecondFactor says, per connection, who answers for the second factor
// a login flow that requires one asks: this server (local, every existing
// connection) or the provider, by the amr values (OpenID Connect) or
// authentication contexts (SAML) an administrator lists. Existing
// connections accept amr "mfa" should they be switched to the provider.
func upSSOSecondFactor(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`ALTER TABLE sso_connections ADD COLUMN second_factor varchar(8) NOT NULL DEFAULT 'local'`,
		`ALTER TABLE sso_connections ADD CONSTRAINT chk_sso_connections_second_factor CHECK (second_factor IN ('local','provider'))`,
		`ALTER TABLE sso_connections ADD COLUMN accepted_amr jsonb NOT NULL DEFAULT '["mfa"]'`,
		`ALTER TABLE sso_connections ALTER COLUMN accepted_amr DROP DEFAULT`,
		`ALTER TABLE sso_connections ADD COLUMN mfa_contexts jsonb NOT NULL DEFAULT '[]'`,
		`ALTER TABLE sso_connections ALTER COLUMN mfa_contexts DROP DEFAULT`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("sso second factor: %w", err)
		}
	}
	return nil
}

// downSSOSecondFactor drops all three. A connection that trusted its
// provider's second factor forgets it, and its lists are lost; a flow that
// requires a factor then asks the account's own of everybody it signs in.
func downSSOSecondFactor(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range []string{
		`ALTER TABLE sso_connections DROP COLUMN IF EXISTS mfa_contexts`,
		`ALTER TABLE sso_connections DROP COLUMN IF EXISTS accepted_amr`,
		`ALTER TABLE sso_connections DROP COLUMN IF EXISTS second_factor`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("undo sso second factor: %w", err)
		}
	}
	return nil
}
