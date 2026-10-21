package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	"loginer/internal/model"
)

func init() {
	goose.AddMigrationContext(upSAMLTwoStep, downSAMLTwoStep)
}

// upSAMLTwoStep makes room for a SAML sign-in that no longer needs a
// cross-site cookie: the provider's POST is held on the sign-in
// (sso_logins.saml_response), found by its own handle (relay_hash), until the
// browser comes back for it; and saml_assertions remembers the assertions
// already used, so none is used twice.
func upSAMLTwoStep(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	login := &model.SSOLogin{}
	for _, field := range []string{"RelayHash", "SAMLResponse"} {
		if db.Migrator().HasColumn(login, field) {
			continue
		}
		if err := db.Migrator().AddColumn(login, field); err != nil {
			return fmt.Errorf("add sso_logins.%s: %w", field, err)
		}
	}
	if !db.Migrator().HasIndex(login, "RelayHash") {
		if err := db.Migrator().CreateIndex(login, "RelayHash"); err != nil {
			return fmt.Errorf("index sso_logins.relay_hash: %w", err)
		}
	}

	if !db.Migrator().HasTable(&model.SAMLAssertion{}) {
		if err := db.Migrator().CreateTable(&model.SAMLAssertion{}); err != nil {
			return fmt.Errorf("create saml_assertions: %w", err)
		}
	}

	return nil
}

// downSAMLTwoStep drops both. SAML sign-ins under way at that moment fail and
// have to be started again; the record of used assertions goes, which only
// ever held what had expired within minutes.
func downSAMLTwoStep(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	if err := db.Migrator().DropTable("saml_assertions"); err != nil {
		return fmt.Errorf("drop saml_assertions: %w", err)
	}
	for _, column := range []string{"relay_hash", "saml_response"} {
		if db.Migrator().HasColumn("sso_logins", column) {
			if err := db.Migrator().DropColumn("sso_logins", column); err != nil {
				return fmt.Errorf("drop sso_logins.%s: %w", column, err)
			}
		}
	}

	return nil
}
