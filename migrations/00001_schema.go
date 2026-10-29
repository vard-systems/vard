// Package migrations holds the database migrations, each registering itself
// with goose from its init function.
//
// Importing this package is what makes the migrations exist; internal/database
// imports it for that reason alone.
//
// This file is the whole schema in one step: schema/up.sql, written out once
// and never changed, and the rows an installation starts with. A later change
// is a new file beside this one (make migrate-new name=x), never an edit
// here: a migration that has run anywhere says the same thing forever.
package migrations

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"vard/internal/model"
)

func init() {
	goose.AddMigrationContext(upSchema, downSchema)
}

var (
	//go:embed schema/up.sql
	schemaUp string
	//go:embed schema/down.sql
	schemaDown string
)

// gormTx wraps the transaction goose hands a migration in GORM, so a
// migration can use the migrator instead of writing every statement by hand.
// Everything it does runs in goose's transaction, so a migration that fails
// half way leaves nothing behind.
func gormTx(tx *sql.Tx) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: tx}), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("wrap transaction: %w", err)
	}
	return db, nil
}

// upSchema builds the whole database: every table, constraint and index, the
// search index where the database allows it, and the few rows an installation
// starts with.
func upSchema(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	if err := db.Exec(schemaUp).Error; err != nil {
		return fmt.Errorf("create the schema: %w", err)
	}

	if err := userSearchIndex(db); err != nil {
		return err
	}

	return seed(db)
}

// downSchema drops everything again.
func downSchema(_ context.Context, tx *sql.Tx) error {
	if _, err := tx.Exec(schemaDown); err != nil {
		return fmt.Errorf("drop the schema: %w", err)
	}
	return nil
}

// userSearch lets the users page find a word anywhere in a user's address,
// name or fields. It needs pg_trgm, which a managed database may not let us
// install; without it the search still works, by scanning.
const userSearch = `CREATE INDEX idx_users_search ON users USING gin (
	LOWER(email || ' ' || COALESCE(first_name, '') || ' ' || COALESCE(last_name, '') || ' ' || COALESCE(data, ''))
	gin_trgm_ops)`

func userSearchIndex(db *gorm.DB) error {
	if err := db.Exec("SAVEPOINT trigram").Error; err != nil {
		return err
	}

	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
		return db.Exec("ROLLBACK TO SAVEPOINT trigram").Error
	}

	if err := db.Exec(userSearch).Error; err != nil {
		return fmt.Errorf("create idx_users_search: %w", err)
	}

	return nil
}

// seed writes the rows an installation cannot start without: the admin roles
// there are to hold, the organisation the panel edits, the login flow every
// application falls back to, and the language every page falls back to. Each
// names its columns, as up.sql made them, so a later change to a model cannot
// change what this writes.
//
// It seeds nothing else. There are no applications, no user fields and no
// user roles: what a record is made of and who may hold what is decided in
// the panel, so a new installation starts with none of it.
//
// There is no administrator either. The first one is made on
// /new-super-admin with the setup token the server logs while there is none,
// or with `admin create` — so a deployment has no password written down
// anywhere, and no default one to forget to change. How administrators are
// then made to sign in (admin_security) is written on the server's first
// start, from the configuration, for the same reason: it is a setting, not a
// schema.
//
// English is the only language here, offered and the default. Its text is not
// stored: it ships in i18n/ and changes with every release, and a migration has
// to say the same thing forever. Other languages are added from the panel.
func seed(db *gorm.DB) error {
	now := time.Now()
	row := func(what, statement string, values ...any) error {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		if err := db.Exec(statement, append([]any{id, now, now}, values...)...).Error; err != nil {
			return fmt.Errorf("create %s: %w", what, err)
		}
		return nil
	}

	for _, role := range startingRoles {
		var permissions any
		if role.Permissions != nil {
			encoded, err := json.Marshal(role.Permissions)
			if err != nil {
				return err
			}
			permissions = string(encoded)
		}
		err := row("the admin role "+role.Name, `INSERT INTO admin_roles
			(id, created_at, updated_at, name, description, permissions) VALUES (?, ?, ?, ?, ?, ?)`,
			role.Name, role.Description, permissions)
		if err != nil {
			return err
		}
	}

	organization := model.DefaultOrganization()
	err := row("the organization", `INSERT INTO organizations
		(id, created_at, updated_at, name, slug, timezone) VALUES (?, ?, ?, ?, ?, ?)`,
		organization.Name, organization.Slug, organization.Timezone)
	if err != nil {
		return err
	}

	flow := model.DefaultLoginFlow()
	steps, err := json.Marshal(flow.Steps)
	if err != nil {
		return err
	}
	err = row("the default login flow", `INSERT INTO login_flows
		(id, created_at, updated_at, name, slug, description, is_default, is_enabled, steps,
		 allow_sign_in, allow_registration, allow_password_reset, allow_remember_me,
		 verify_email_on_register, require_verified_email, allow_email_change, session_lifetime_hours)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		flow.Name, flow.Slug, flow.Description, flow.IsDefault, flow.IsEnabled, string(steps),
		flow.AllowSignIn, flow.AllowRegistration, flow.AllowPasswordReset, flow.AllowRememberMe,
		flow.VerifyEmailOnRegister, flow.RequireVerifiedEmail, flow.AllowEmailChange, flow.SessionLifetimeHours)
	if err != nil {
		return err
	}

	language := model.DefaultLanguage()
	return row("the default language", `INSERT INTO languages
		(id, created_at, updated_at, code, name, native_name, is_enabled, is_default, position)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		language.Code, language.Name, language.NativeName, language.IsEnabled, language.IsDefault, language.Position)
}

// startingRoles are the admin roles a panel is given, and what each grants.
//
// super_admin is built in: it grants every permission by name, so it lists
// none, and it is the only role that can manage administrators. A super admin
// can change or remove any of the others. app_manager is meant to be held for
// one application: every one of its permissions can be scoped.
var startingRoles = []model.AdminRole{
	{
		Name:        model.RoleSuperAdmin,
		Description: "Everything, including managing administrators",
	},
	{
		Name:        "admin",
		Description: "Everything but managing administrators",
		Permissions: model.AdminPermissionNames(),
	},
	{
		Name:        "moderator",
		Description: "Manage users, application roles and who holds them, and read the activity log",
		Permissions: []string{
			model.PermActivityRead, model.PermUsersRead, model.PermUsersWrite, model.PermAPIsRead,
			model.PermApplicationsRead, model.PermUserRolesWrite, model.PermRoleAssignmentsWrite,
		},
	},
	{
		Name:        "app_manager",
		Description: "Look after an application: its settings, its roles, and who holds them",
		Permissions: []string{
			model.PermApplicationsRead, model.PermApplicationsWrite,
			model.PermUserRolesWrite, model.PermRoleAssignmentsWrite,
		},
	},
	{
		Name:        "support",
		Description: "Look users up, fix their records and give them application roles",
		Permissions: []string{
			model.PermUsersRead, model.PermUsersWrite, model.PermApplicationsRead, model.PermRoleAssignmentsWrite,
		},
	},
	{
		Name:        "auditor",
		Description: "Read only, including the activity log",
		Permissions: []string{
			model.PermActivityRead, model.PermUsersRead, model.PermAPIsRead, model.PermApplicationsRead,
		},
	},
}
