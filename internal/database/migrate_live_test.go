package database_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"vard/internal/database"
	"vard/internal/database/databasetest"
	"vard/internal/model"
	"vard/internal/store"
)

// A database made before these changes is stepped forward with its
// applications as they were — no certificate method, nothing bound, no
// development redirects — and each down step takes its columns away again.
func TestLiveTheNewColumnsUpgradeAnExistingDatabase(t *testing.T) {
	db, cfg := databasetest.Open(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	app := &model.Application{
		Name:                 "Before",
		Type:                 model.AppWeb,
		ClientID:             "before",
		TokenAuthMethod:      model.AuthClientSecretBasic,
		GrantTypes:           []string{model.GrantAuthorizationCode},
		RedirectURIs:         []string{"https://app.example.com/callback"},
		AccessTokenLifetime:  model.DefaultAccessTokenLifetime,
		IDTokenLifetime:      model.DefaultIDTokenLifetime,
		RefreshTokenLifetime: model.DefaultRefreshTokenLifetime,
		IsEnabled:            true,
	}
	if err := store.New(db).CreateApplication(ctx, app); err != nil {
		t.Fatal(err)
	}

	type column struct{ table, name string }
	// Newest first, the order the down steps take them away.
	steps := []struct {
		name    string
		columns []column
		index   string
		table   string
	}{
		{name: "SSO domain proofs", table: "sso_domain_verifications", columns: []column{
			{"pending_sign_ins", "link_connection_id"}, {"pending_sign_ins", "link_subject"}, {"pending_sign_ins", "link_email"},
		}},
		{name: "setup tokens", table: "setup_tokens"},
		{name: "sealed successor index", index: "idx_refresh_tokens_sealed"},
		{name: "mutual TLS", columns: []column{
			{"applications", "tls_subject_kind"}, {"applications", "tls_subject"},
			{"applications", "tls_certificates"}, {"applications", "certificate_bound_tokens"},
			{"refresh_tokens", "certificate_thumbprint"},
		}},
		{name: "development redirects", columns: []column{{"applications", "allow_development_redirects"}}},
		{name: "refresh successor", columns: []column{{"refresh_tokens", "successor_id"}, {"refresh_tokens", "successor_sealed"}}},
	}

	// A fresh database has the index from the models, as the migration makes
	// it on an old one: on the rows still holding a sealed successor alone.
	var definition string
	if err := db.Raw("SELECT indexdef FROM pg_indexes WHERE indexname = 'idx_refresh_tokens_sealed'").Scan(&definition).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(definition, "WHERE (successor_sealed IS NOT NULL)") {
		t.Errorf("a fresh database's index = %q, want it partial on successor_sealed", definition)
	}

	for _, step := range steps {
		if err := database.MigrateDown(db, cfg, log); err != nil {
			t.Fatalf("down %s: %v", step.name, err)
		}
		for _, c := range step.columns {
			if db.Migrator().HasColumn(c.table, c.name) {
				t.Errorf("after down %s, %s.%s is still there", step.name, c.table, c.name)
			}
		}
		if step.index != "" && db.Migrator().HasIndex("refresh_tokens", step.index) {
			t.Errorf("after down %s, %s is still there", step.name, step.index)
		}
		if step.table != "" && db.Migrator().HasTable(step.table) {
			t.Errorf("after down %s, %s is still there", step.name, step.table)
		}
	}

	// A connection made before domains were proved keeps linking at the
	// domains it listed: they are carried over as held.
	before := &model.SSOConnection{
		Slug: "acme", Name: "Acme", Protocol: model.SSOProtocolOIDC, Issuer: "https://idp.acme.test",
		ClientID: "client", Domains: model.StringList{"acme.test", "acme.example"}, Matching: model.SSOMatchLink,
		ShowOnLogin: true, Scopes: model.StringList{}, RoleMappings: model.SSORoleMappings{},
	}
	if err := db.Create(before).Error; err != nil {
		t.Fatalf("a connection from before: %v", err)
	}

	if err := database.Migrate(db, cfg, log); err != nil {
		t.Fatalf("up: %v", err)
	}
	for _, step := range steps {
		for _, c := range step.columns {
			if !db.Migrator().HasColumn(c.table, c.name) {
				t.Errorf("after up, %s.%s is missing", c.table, c.name)
			}
		}
		if step.index != "" && !db.Migrator().HasIndex("refresh_tokens", step.index) {
			t.Errorf("after up, %s is missing", step.index)
		}
		if step.table != "" && !db.Migrator().HasTable(step.table) {
			t.Errorf("after up, %s is missing", step.table)
		}
	}

	var held int64
	if err := db.Raw(`SELECT count(*) FROM sso_domain_verifications
		WHERE method = 'existing' AND verified_at IS NOT NULL AND domain IN ('acme.test', 'acme.example')`).Scan(&held).Error; err != nil {
		t.Fatal(err)
	}
	if held != 2 {
		t.Errorf("%d of the existing connection's 2 domains were carried over as held", held)
	}

	upgraded, err := store.New(db).Application(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if upgraded.TLSSubjectKind != "" || upgraded.TLSSubject != "" || len(upgraded.TLSCertificates) != 0 ||
		upgraded.CertificateBoundTokens || upgraded.AllowDevelopmentRedirects {
		t.Errorf("upgraded application = %+v, want no certificate, nothing bound, no development redirects", upgraded)
	}
}
