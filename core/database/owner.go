package database

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"vard/core/config"
)

// rolePattern is what a runtime role's name may be, so it can be written into
// GRANT, which takes no parameters.
var rolePattern = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

// Connect opens the connection the server runs on, migrating first when
// cfg.Migrate says to. With a MigrateDSN the owner migrates before the
// runtime role connects: on a fresh installation that migration is what makes
// the role, so the other order could never start.
func Connect(cfg config.DB, log *slog.Logger) (*gorm.DB, error) {
	if cfg.Migrate && cfg.MigrateDSN != "" {
		if err := MigrateAsOwner(cfg, log); err != nil {
			return nil, err
		}
	}

	db, err := Open(cfg, log)
	if err != nil {
		return nil, err
	}

	if cfg.Migrate && cfg.MigrateDSN == "" {
		if err := Migrate(db, log); err != nil {
			_ = Close(db)
			return nil, err
		}
	}

	return db, nil
}

// MigrateAsOwner connects as the schema's owner (cfg.MigrateDSN) and migrates,
// makes the runtime role named by cfg.DSN if there is none, and grants that
// role the rows of every table and nothing else, so the running server cannot
// change the schema or read what other roles keep.
func MigrateAsOwner(cfg config.DB, log *slog.Logger) error {
	ownerCfg := cfg
	ownerCfg.DSN = cfg.MigrateDSN
	owner, err := Open(ownerCfg, log)
	if err != nil {
		return fmt.Errorf("connect as the schema owner: %w", err)
	}
	defer Close(owner)

	if err := Migrate(owner, log); err != nil {
		return err
	}

	return grantRuntime(owner, cfg.DSN, log)
}

func grantRuntime(owner *gorm.DB, runtimeDSN string, log *slog.Logger) error {
	address, err := url.Parse(runtimeDSN)
	if err != nil || address.User == nil {
		return errors.New("the runtime DSN names no user to grant to")
	}
	role := address.User.Username()
	if !rolePattern.MatchString(role) {
		return fmt.Errorf("the runtime role %q is not a plain lower-case name", role)
	}
	password, _ := address.User.Password()

	return owner.Transaction(func(tx *gorm.DB) error {
		var exists bool
		if err := tx.Raw("SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = ?)", role).Scan(&exists).Error; err != nil {
			return err
		}
		if !exists {
			// CREATE ROLE takes no parameters, so Postgres quotes the
			// password itself; the statement holds it, so it is never logged.
			var create string
			if err := tx.Raw("SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', ?::text, ?::text)", role, password).Scan(&create).Error; err != nil {
				return err
			}
			silent := tx.Session(&gorm.Session{Logger: logger.Discard})
			if err := silent.Exec(create).Error; err != nil {
				return fmt.Errorf("create the runtime role: %w", err)
			}
			log.Info("created the runtime database role", "role", role)
		}

		for _, statement := range []string{
			// Before Postgres 15 every role may create tables in public.
			`REVOKE CREATE ON SCHEMA public FROM PUBLIC`,
			`GRANT USAGE ON SCHEMA public TO "` + role + `"`,
			`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO "` + role + `"`,
			`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO "` + role + `"`,
			// Which migrations ran is the owner's to say.
			`REVOKE ALL ON goose_db_version FROM "` + role + `"`,
		} {
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("grant the runtime role: %w", err)
			}
		}
		return nil
	})
}
