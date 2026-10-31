package database_test

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/config"
	"vard/internal/domain/model"
	"vard/internal/infrastructure/database"
	"vard/internal/infrastructure/database/databasetest"
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

// Going down, one step at a time, takes away everything the migrations
// made, and up builds it again.
func TestLiveTheMigrationsGoDownAndUpAgain(t *testing.T) {
	db, _ := databasetest.Open(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	applied := func() int64 {
		var version int64
		if err := db.Raw("SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied").Scan(&version).Error; err != nil {
			t.Fatal(err)
		}
		return version
	}
	for applied() > 0 {
		if err := database.MigrateDown(db, log); err != nil {
			t.Fatalf("down from %d: %v", applied(), err)
		}
	}
	if left := rows(t, db, `SELECT tablename FROM pg_tables
		WHERE schemaname = 'public' AND tablename <> 'goose_db_version'`); len(left) != 0 {
		t.Errorf("down left %v", left)
	}

	if err := database.Migrate(db, log); err != nil {
		t.Fatalf("up again: %v", err)
	}
	var roles int64
	if err := db.Table("admin_roles").Count(&roles).Error; err != nil || roles == 0 {
		t.Errorf("up again seeded %d admin roles (%v), want the starting ones", roles, err)
	}
}

// A reset leaves what a new installation starts with: every row added since
// is gone, and the seeded ones are back.
func TestLiveAResetStartsTheDatabaseAgain(t *testing.T) {
	db, _ := databasetest.Open(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := db.Exec("UPDATE organizations SET name = 'Renamed'").Error; err != nil {
		t.Fatal(err)
	}
	if err := database.MigrateReset(db, log); err != nil {
		t.Fatalf("reset: %v", err)
	}

	if names := rows(t, db, "SELECT name FROM organizations"); slices.Contains(names, "Renamed") || len(names) != 1 {
		t.Errorf("after a reset the organizations are %v, want the one a new installation has", names)
	}
	states, err := database.MigrateStatus(db)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range states {
		if !state.Applied {
			t.Errorf("after a reset migration %d is pending, want every one applied", state.Version)
		}
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
	db, _ := databasetest.Empty(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	errs := make(chan error, 4)
	var started sync.WaitGroup
	for range 4 {
		started.Go(func() { errs <- database.Migrate(db, log) })
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

// With the owner's DSN to migrate, the server runs as a role made for it that
// reads and writes rows and nothing else: a flaw in the running server cannot
// change the schema, whatever SQL it is tricked into.
func TestLiveTheServerRunsAsARoleThatCannotChangeTheSchema(t *testing.T) {
	role := fmt.Sprintf("app_test_%d", time.Now().UnixNano())
	server, err := database.Open(config.DB{Driver: "postgres", DSN: os.Getenv(databasetest.Env), TimeZone: "UTC"},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() {
		_ = server.Exec(`DROP ROLE IF EXISTS "` + role + `"`).Error
		_ = database.Close(server)
	})

	owner, cfg := databasetest.Empty(t)
	runtime, _ := url.Parse(cfg.DSN)
	runtime.User = url.UserPassword(role, "runtime-password")
	cfg.MigrateDSN, cfg.DSN = cfg.DSN, runtime.String()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	for range 2 { // the second start finds the role already made
		if err := database.MigrateAsOwner(cfg, log); err != nil {
			t.Fatalf("migrate as the owner: %v", err)
		}
	}

	db, err := database.Open(cfg, log)
	if err != nil {
		t.Fatalf("connect as the runtime role: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close(db)
		_ = owner.Exec(`DROP OWNED BY "` + role + `"`).Error
	})

	if err := db.Exec("UPDATE organizations SET name = 'Renamed'").Error; err != nil {
		t.Errorf("writing a row as the runtime role = %v, want it allowed", err)
	}
	for _, statement := range []string{
		"CREATE TABLE stolen (id int)",
		"ALTER TABLE users ADD COLUMN stolen text",
		"DROP TABLE audit_logs",
		"DELETE FROM goose_db_version",
	} {
		if err := db.Exec(statement).Error; err == nil || !strings.Contains(err.Error(), "permission denied") &&
			!strings.Contains(err.Error(), "must be owner") {
			t.Errorf("%s as the runtime role = %v, want it refused", statement, err)
		}
	}
}

// A fresh installation starts with no runtime role: the server connects as
// the owner first, which makes it, and only then as the role — in that
// order, or it can never start.
func TestLiveAFreshInstallationStartsAsItsRuntimeRole(t *testing.T) {
	role := fmt.Sprintf("app_fresh_%d", time.Now().UnixNano())
	server, err := database.Open(config.DB{Driver: "postgres", DSN: os.Getenv(databasetest.Env), TimeZone: "UTC"},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() {
		_ = server.Exec(`DROP ROLE IF EXISTS "` + role + `"`).Error
		_ = database.Close(server)
	})

	owner, cfg := databasetest.Empty(t)
	runtime, _ := url.Parse(cfg.DSN)
	runtime.User = url.UserPassword(role, "runtime-password")
	cfg.MigrateDSN, cfg.DSN, cfg.Migrate = cfg.DSN, runtime.String(), true

	db, err := database.Connect(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("a fresh installation could not start: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close(db)
		_ = owner.Exec(`DROP OWNED BY "` + role + `"`).Error
	})

	var user string
	if err := db.Raw("SELECT current_user").Scan(&user).Error; err != nil || user != role {
		t.Errorf("connected as %q (%v), want the runtime role %q", user, err, role)
	}
}

// A database the migrations from before the squash made says by its goose
// versions that it is up to date; it is refused rather than started on.
func TestLiveADatabaseFromBeforeTheSquashIsRefused(t *testing.T) {
	db, _ := databasetest.Empty(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	for _, statement := range []string{
		"CREATE TABLE goose_db_version (id serial PRIMARY KEY, version_id bigint NOT NULL, is_applied boolean NOT NULL, tstamp timestamp DEFAULT now())",
		"INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, true), (1, true), (6, true)",
		"CREATE TABLE users (id uuid PRIMARY KEY)",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := database.Migrate(db, log); !errors.Is(err, database.ErrSquashedHistory) {
		t.Errorf("migrating it = %v, want ErrSquashedHistory", err)
	}
}
