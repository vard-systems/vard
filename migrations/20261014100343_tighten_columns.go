package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upTightenColumns, downTightenColumns)
}

// Three things the columns said that the code never meant.
//
//   - deleted_at. model.Base carried GORM's soft-delete column, but every
//     delete here is a hard one (the indexes on it went in query_indexes). A
//     column that is always null still makes GORM add "deleted_at IS NULL" to
//     every query, and it suggests that a deleted user's record is kept, which
//     it is not.
//   - created_at and updated_at could be null. GORM always writes them.
//   - Text columns could be null, while the field they are read into is a
//     plain string: one null, written by hand or by a script, and every read
//     of that row fails. They become not null, defaulting to the empty string
//     the code already means by "not set".
//
// The lists are written out rather than read from the models, so this says
// the same thing however the models change later.
var softDeleted = []string{
	"admin_roles",
	"admin_security",
	"admin_sessions",
	"admins",
	"api_scopes",
	"apis",
	"applications",
	"authorization_codes",
	"authorization_requests",
	"email_verifications",
	"languages",
	"login_codes",
	"login_flows",
	"mail_settings",
	"mfa_factors",
	"organizations",
	"otp_settings",
	"password_resets",
	"refresh_tokens",
	"social_identities",
	"social_logins",
	"social_providers",
	"sso_connections",
	"sso_identities",
	"sso_logins",
	"translations",
	"user_fields",
	"user_roles",
	"user_sessions",
	"users",
}

var stamped = map[string][]string{
	"admin_role_assignments": {"created_at"},
	"admin_roles":            {"created_at", "updated_at"},
	"admin_security":         {"created_at", "updated_at"},
	"admin_sessions":         {"created_at", "updated_at"},
	"admins":                 {"created_at", "updated_at"},
	"api_scopes":             {"created_at", "updated_at"},
	"apis":                   {"created_at", "updated_at"},
	"application_apis":       {"created_at"},
	"applications":           {"created_at", "updated_at"},
	"authorization_codes":    {"created_at", "updated_at"},
	"authorization_requests": {"created_at", "updated_at"},
	"email_verifications":    {"created_at", "updated_at"},
	"languages":              {"created_at", "updated_at"},
	"login_codes":            {"created_at", "updated_at"},
	"login_flows":            {"created_at", "updated_at"},
	"mail_settings":          {"created_at", "updated_at"},
	"mfa_factors":            {"created_at", "updated_at"},
	"organizations":          {"created_at", "updated_at"},
	"otp_settings":           {"created_at", "updated_at"},
	"password_resets":        {"created_at", "updated_at"},
	"refresh_tokens":         {"created_at", "updated_at"},
	"social_identities":      {"created_at", "updated_at"},
	"social_logins":          {"created_at", "updated_at"},
	"social_providers":       {"created_at", "updated_at"},
	"sso_connections":        {"created_at", "updated_at"},
	"sso_identities":         {"created_at", "updated_at"},
	"sso_logins":             {"created_at", "updated_at"},
	"translations":           {"created_at", "updated_at"},
	"user_fields":            {"created_at", "updated_at"},
	"user_roles":             {"created_at", "updated_at"},
	"user_sessions":          {"created_at", "updated_at"},
	"users":                  {"created_at", "updated_at"},
}

var textColumns = map[string][]string{
	"admin_roles":            {"description"},
	"admin_sessions":         {"ip", "user_agent"},
	"admins":                 {"last_login_ip"},
	"api_scopes":             {"description"},
	"apis":                   {"description"},
	"applications":           {"client_secret_hash", "description", "logo_url", "privacy_url", "secret_hint", "terms_url", "website_url"},
	"audit_logs":             {"actor_email", "ip", "request_id", "target_id", "target_type", "user_agent"},
	"authorization_codes":    {"audience", "code_challenge", "code_challenge_method", "nonce"},
	"authorization_requests": {"audience", "code_challenge", "code_challenge_method", "login_hint", "nonce", "state"},
	"email_verifications":    {"new_email"},
	"login_codes":            {"request"},
	"login_flows":            {"description"},
	"mail_settings":          {"from_address", "from_name", "host", "username"},
	"mfa_factors":            {"label"},
	"organizations":          {"logo_url", "privacy_url", "support_email", "support_phone", "terms_url"},
	"refresh_tokens":         {"audience"},
	"social_identities":      {"email"},
	"social_logins":          {"next", "request", "verifier"},
	"social_providers":       {"authorize_url", "key_id", "team_id", "token_auth", "token_url", "userinfo_url"},
	"sso_connections":        {"client_id", "email_attribute", "first_name_attribute", "groups_attribute", "issuer", "last_name_attribute", "metadata", "metadata_url", "name_id_format", "sp_certificate"},
	"sso_identities":         {"email"},
	"sso_logins":             {"next", "nonce", "request", "saml_request_id", "verifier"},
	"user_roles":             {"description"},
	"user_sessions":          {"ip", "user_agent"},
	"users":                  {"first_name", "last_name"},
}

func upTightenColumns(ctx context.Context, tx *sql.Tx) error {
	for _, table := range softDeleted {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %q DROP COLUMN IF EXISTS deleted_at`, table)); err != nil {
			return fmt.Errorf("drop %s.deleted_at: %w", table, err)
		}
	}

	for table, columns := range stamped {
		for _, column := range columns {
			fill := fmt.Sprintf(`UPDATE %q SET %q = now() WHERE %q IS NULL`, table, column, column)
			if _, err := tx.ExecContext(ctx, fill); err != nil {
				return fmt.Errorf("fill %s.%s: %w", table, column, err)
			}
		}
		if err := alter(ctx, tx, table, columns, "SET NOT NULL"); err != nil {
			return err
		}
	}

	for table, columns := range textColumns {
		for _, column := range columns {
			fill := fmt.Sprintf(`UPDATE %q SET %q = '' WHERE %q IS NULL`, table, column, column)
			if _, err := tx.ExecContext(ctx, fill); err != nil {
				return fmt.Errorf("fill %s.%s: %w", table, column, err)
			}
		}
		if err := alter(ctx, tx, table, columns, "SET DEFAULT ''", "SET NOT NULL"); err != nil {
			return err
		}
	}

	return nil
}

// downTightenColumns lets the columns be null again and gives the tables an
// empty deleted_at back. Nothing is lost: no row is changed.
func downTightenColumns(ctx context.Context, tx *sql.Tx) error {
	for table, columns := range textColumns {
		if err := alter(ctx, tx, table, columns, "DROP NOT NULL", "DROP DEFAULT"); err != nil {
			return err
		}
	}

	for table, columns := range stamped {
		if err := alter(ctx, tx, table, columns, "DROP NOT NULL"); err != nil {
			return err
		}
	}

	for _, table := range softDeleted {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %q ADD COLUMN IF NOT EXISTS deleted_at timestamptz`, table)); err != nil {
			return fmt.Errorf("add %s.deleted_at: %w", table, err)
		}
	}

	return nil
}

// alter applies every change to every column in one statement, so a table is
// rewritten at most once.
func alter(ctx context.Context, tx *sql.Tx, table string, columns []string, changes ...string) error {
	actions := make([]string, 0, len(columns)*len(changes))
	for _, column := range columns {
		for _, change := range changes {
			actions = append(actions, fmt.Sprintf("ALTER COLUMN %q %s", column, change))
		}
	}

	statement := fmt.Sprintf("ALTER TABLE %q %s", table, strings.Join(actions, ", "))
	if _, err := tx.ExecContext(ctx, statement); err != nil {
		return fmt.Errorf("alter %s: %w", table, err)
	}

	return nil
}
