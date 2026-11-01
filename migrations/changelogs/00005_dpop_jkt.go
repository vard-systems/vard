package changelogs

import (
	"context"
	"database/sql"
	"fmt"
)

func init() {
	add(5, upDPoPJKT, downDPoPJKT)
}

// upDPoPJKT gives an authorization request and its code the DPoP key the code
// is bound to (dpop_jkt, RFC 9449 §10): a key thumbprint, always 43
// characters, or ” for a code any client holding it may redeem, which is
// what every row there already is.
func upDPoPJKT(ctx context.Context, tx *sql.Tx) error {
	for _, table := range []string{"authorization_requests", "authorization_codes"} {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE `+table+` ADD COLUMN dpop_key varchar(43) NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add %s.dpop_key: %w", table, err)
		}
	}
	return nil
}

// downDPoPJKT drops the binding. A request or code made while it was there
// loses its key, and its code can then be redeemed without a proof from it.
func downDPoPJKT(ctx context.Context, tx *sql.Tx) error {
	for _, table := range []string{"authorization_codes", "authorization_requests"} {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE `+table+` DROP COLUMN IF EXISTS dpop_key`); err != nil {
			return fmt.Errorf("drop %s.dpop_key: %w", table, err)
		}
	}
	return nil
}
