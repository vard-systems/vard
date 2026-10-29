package database_test

import (
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/database"
	"vard/internal/database/databasetest"
	"vard/internal/model"
)

// The migrations are written out by hand and the models are not, so nothing
// else stops a model changing without the migration that changes every
// existing database with it. Each column, foreign key and model-declared index
// a fresh database would get from the models is the same one the migrations
// make; the migrations may add indexes a struct tag cannot describe.
func TestLiveTheMigrationsMakeWhatTheModelsDescribe(t *testing.T) {
	migrated, _ := databasetest.Open(t)
	described, _ := databasetest.Empty(t)
	if err := described.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}

	for _, part := range []struct {
		name  string
		query string
		// subset is true where the migrations may have more than the models.
		subset bool
	}{
		{name: "columns", query: `SELECT table_name || '.' || column_name || ' ' || data_type
			|| COALESCE('(' || character_maximum_length || ')', '')
			|| CASE WHEN is_nullable = 'NO' THEN ' not null' ELSE '' END
			|| COALESCE(' default ' || column_default, '')
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name <> 'goose_db_version'`},
		{name: "constraints", query: `SELECT conrelid::regclass || ' ' || conname || ' ' || pg_get_constraintdef(oid)
			FROM pg_constraint
			WHERE connamespace = 'public'::regnamespace AND conrelid::regclass::text <> 'goose_db_version'`},
		{name: "indexes", subset: true, query: `SELECT indexdef FROM pg_indexes
			WHERE schemaname = 'public' AND tablename <> 'goose_db_version'`},
	} {
		t.Run(part.name, func(t *testing.T) {
			made, want := rows(t, migrated, part.query), rows(t, described, part.query)
			for _, line := range want {
				if !slices.Contains(made, line) {
					t.Errorf("the models describe what no migration makes: %s", line)
				}
			}
			if part.subset {
				return
			}
			for _, line := range made {
				if !slices.Contains(want, line) {
					t.Errorf("a migration makes what no model describes: %s", line)
				}
			}
		})
	}
}

// Going down takes away everything the migrations made, and up builds it again.
func TestLiveTheMigrationsGoDownAndUpAgain(t *testing.T) {
	db, cfg := databasetest.Open(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := database.MigrateDown(db, cfg, log); err != nil {
		t.Fatalf("down: %v", err)
	}
	if left := rows(t, db, `SELECT tablename FROM pg_tables
		WHERE schemaname = 'public' AND tablename <> 'goose_db_version'`); len(left) != 0 {
		t.Errorf("down left %v", left)
	}

	if err := database.Migrate(db, cfg, log); err != nil {
		t.Fatalf("up again: %v", err)
	}
	var roles int64
	if err := db.Table("admin_roles").Count(&roles).Error; err != nil || roles == 0 {
		t.Errorf("up again seeded %d admin roles (%v), want the starting ones", roles, err)
	}
}

func rows(t *testing.T, db *gorm.DB, query string) []string {
	t.Helper()
	var out []string
	if err := db.Raw(query).Scan(&out).Error; err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

// The database refuses what Go never writes, so a bug or a hand-run query
// cannot store it either. Each row is whole but for the one thing wrong.
func TestLiveTheDatabaseRefusesWhatGoNeverWrites(t *testing.T) {
	db, _ := databasetest.Open(t)

	user := &model.User{Email: "ada@example.com", IsActive: true}
	app := &model.Application{
		Name: "Shop", ClientID: "shop", Type: model.AppWeb, TokenAuthMethod: model.AuthNone, IsEnabled: true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatal(err)
	}
	nowhere := uuid.New()

	for _, tt := range []struct {
		name, refusedBy string
		row             any
	}{
		{"an administrator status no code knows", "chk_admins_status",
			&model.Admin{Username: "a@example.com", Email: "a@example.com", Status: "root"}},
		{"an administrator's address in capitals", "chk_admins_email_lower",
			&model.Admin{Username: "b@example.com", Email: "B@example.com", Status: model.StatusActive}},
		{"a second organization", "idx_organizations_one_row",
			&model.Organization{Name: "Another", Slug: "another", Timezone: "UTC"}},
		{"an application type no code knows", "chk_applications_type",
			&model.Application{Name: "X", ClientID: "x", Type: "mainframe", TokenAuthMethod: model.AuthNone}},
		{"a refresh token naming a session that never was", "fk_refresh_tokens_session",
			&model.RefreshToken{TokenHash: "h", FamilyID: uuid.New(), ApplicationID: app.ID, UserID: user.ID, SessionID: &nowhere}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.Omit(clause.Associations).Create(tt.row).Error; err == nil || !strings.Contains(err.Error(), tt.refusedBy) {
				t.Errorf("= %v, want it refused by %s", err, tt.refusedBy)
			}
		})
	}

	// Users' addresses are lower-cased by the model on the way in; the
	// check is for a write that goes round it.
	err := db.Exec("UPDATE users SET email = 'Ada@Example.com' WHERE id = ?", user.ID).Error
	if err == nil || !strings.Contains(err.Error(), "chk_users_email_lower") {
		t.Errorf("an address written in capitals = %v, want it refused by chk_users_email_lower", err)
	}
}

// Instances started together, as replicas are, migrate one at a time: the
// second waits for the first and finds nothing left to do.
func TestLiveInstancesStartingTogetherMigrateOnce(t *testing.T) {
	db, cfg := databasetest.Empty(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	errs := make(chan error, 4)
	var started sync.WaitGroup
	for range 4 {
		started.Go(func() { errs <- database.Migrate(db, cfg, log) })
	}
	started.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("an instance's migration = %v, want it to wait its turn", err)
		}
	}
}

// The limit is Postgres's own, so a query that runs past it is ended by the
// database even if nothing in Go is waiting to cancel it.
func TestLiveAQueryPastTheStatementTimeoutIsEnded(t *testing.T) {
	_, cfg := databasetest.Empty(t)
	cfg.StatementTimeout = 100 * time.Millisecond

	db, err := database.Open(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close(db) })

	err = db.Exec("SELECT pg_sleep(1)").Error
	if err == nil || !strings.Contains(err.Error(), "statement timeout") {
		t.Errorf("a one-second query under a 100ms limit = %v, want it ended", err)
	}
}
