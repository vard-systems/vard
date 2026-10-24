// Package migrations holds the database migrations, each registering itself
// with goose from its init function.
//
// Importing this package is what makes the migrations exist; internal/database
// imports it for that reason alone.
//
// This file is the whole schema in one step. A later change to it is a new
// file beside this one (make migrate-new name=x), never an edit here: a
// migration that has run anywhere says the same thing forever.
package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"loginer/internal/model"
)

func init() {
	goose.AddMigrationContext(upSchema, downSchema)
}

// gormTx wraps the transaction goose hands a migration in GORM, so a
// migration can use the model structs and the migrator instead of writing DDL
// by hand. Everything it does runs in goose's transaction, so a migration
// that fails half way leaves nothing behind.
func gormTx(tx *sql.Tx) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: tx}), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("wrap transaction: %w", err)
	}
	return db, nil
}

// upSchema builds the whole database: every table the models describe, the
// indexes they cannot describe themselves, and the few rows an installation
// starts with.
//
// The schema is the models — `model.All()` is the list, in the order a table
// has to exist before the tables that point at it, and each struct says what
// its columns are.
func upSchema(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	if err := db.AutoMigrate(model.All()...); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	for _, index := range indexes {
		if err := db.Exec(index.create).Error; err != nil {
			return fmt.Errorf("create %s: %w", index.name, err)
		}
	}

	return seed(db)
}

// downSchema drops everything again: the join tables GORM made for the
// many-to-many relations first, then the models in the reverse of the order
// they were made, so no foreign key blocks a drop.
func downSchema(_ context.Context, tx *sql.Tx) error {
	db, err := gormTx(tx)
	if err != nil {
		return err
	}

	tables := []any{"user_role_members", "user_role_inherits", "user_role_api_scopes"}

	models := model.All()
	for i := len(models) - 1; i >= 0; i-- {
		tables = append(tables, models[i])
	}

	return db.Migrator().DropTable(tables...)
}

// indexes are the ones a struct tag cannot describe: partial, ordered, or
// with an operator class.
var indexes = []struct{ name, create string }{
	// A null application means the whole panel for an admin role, and a
	// global role for a user role. Postgres treats every null as different,
	// so a unique index over a nullable column needs a partial index for the
	// null case.
	{"idx_user_roles_global_name", `CREATE UNIQUE INDEX idx_user_roles_global_name
		ON user_roles (name) WHERE application_id IS NULL`},
	{"idx_admin_role_assignments_global", `CREATE UNIQUE INDEX idx_admin_role_assignments_global
		ON admin_role_assignments (admin_id, role_id) WHERE application_id IS NULL`},
	{"idx_admin_role_assignments_scoped", `CREATE UNIQUE INDEX idx_admin_role_assignments_scoped
		ON admin_role_assignments (admin_id, role_id, application_id) WHERE application_id IS NOT NULL`},

	// The users page searches addresses by prefix; the unique index on email
	// compares with the database's collation, which a LIKE 'ab%' cannot use.
	{"idx_users_email_prefix", `CREATE INDEX idx_users_email_prefix
		ON users (email text_pattern_ops)`},

	// The lists that page newest first, and the activity log's filters, read
	// in these orders.
	{"idx_user_sessions_recent", `CREATE INDEX idx_user_sessions_recent
		ON user_sessions (created_at DESC, id DESC)`},
	{"idx_audit_logs_created_id", `CREATE INDEX idx_audit_logs_created_id
		ON audit_logs (created_at DESC, id DESC)`},
	{"idx_audit_logs_action_created", `CREATE INDEX idx_audit_logs_action_created
		ON audit_logs (action, created_at DESC)`},
	{"idx_audit_logs_actor_created", `CREATE INDEX idx_audit_logs_actor_created
		ON audit_logs (actor_email, created_at DESC)`},
}

// seed writes the rows an installation cannot start without: the admin roles
// there are to hold, the organisation the panel edits, the login flow every
// application falls back to, and the language every page falls back to.
//
// It seeds nothing else. There are no applications, no user fields and no
// user roles: what a record is made of and who may hold what is decided in
// the panel, so a new installation starts with none of it.
//
// There is no administrator either. The first one is made by whoever opens
// the panel, on /admin/new-super-admin, which is offered only while there is
// none — so a deployment has no password written down anywhere, and no
// default one to forget to change. How administrators are then made to sign
// in (admin_security) is written on the server's first start, from the
// configuration, for the same reason: it is a setting, not a schema.
//
// The base language's row is the only language here. Its text, and the other
// languages the server ships with, are imported on the first start by
// store.EnsureLanguages: the files change with every release, and a migration
// has to say the same thing forever.
func seed(db *gorm.DB) error {
	roles := append([]model.AdminRole(nil), startingRoles...)
	if err := db.Create(&roles).Error; err != nil {
		return fmt.Errorf("create the admin roles: %w", err)
	}

	organization := model.DefaultOrganization()
	if err := db.Create(&organization).Error; err != nil {
		return fmt.Errorf("create the organization: %w", err)
	}

	flow := model.DefaultLoginFlow()
	if err := db.Create(&flow).Error; err != nil {
		return fmt.Errorf("create the default login flow: %w", err)
	}

	language := model.DefaultLanguage()
	if err := db.Create(&language).Error; err != nil {
		return fmt.Errorf("create the default language: %w", err)
	}

	return nil
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
