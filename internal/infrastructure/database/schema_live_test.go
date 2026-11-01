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

// An installation upgraded to the DPoP binding keeps the requests and codes
// it had, each bound to no key, which is what they were; going back down
// keeps them too, without the column.
func TestLiveTheDPoPBindingUpgradesAnExistingDatabase(t *testing.T) {
	db, _ := databasetest.Open(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	migrateDownTo(t, db, log, 4)
	if has := rows(t, db, `SELECT table_name FROM information_schema.columns
		WHERE column_name = 'dpop_key' AND table_name IN ('authorization_requests', 'authorization_codes')`); len(has) != 0 {
		t.Fatalf("before the binding, dpop_key is on %v", has)
	}

	user := &model.User{Email: "ada@example.com", IsActive: true}
	app := &model.Application{Name: "Shop", ClientID: "shop", Type: model.AppWeb, TokenAuthMethod: model.AuthNone, IsEnabled: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	request := &model.AuthorizationRequest{HandleHash: "request", ApplicationID: app.ID, RedirectURI: "https://shop.example/cb", Scope: "openid", ExpiresAt: later}
	code := &model.AuthorizationCode{CodeHash: "code", ApplicationID: app.ID, UserID: user.ID, RedirectURI: "https://shop.example/cb", Scope: "openid", AuthenticatedAt: time.Now(), ExpiresAt: later}
	for _, row := range []any{request, code} {
		if err := db.Omit(clause.Associations, "DPoPKey").Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := database.Migrate(db, log); err != nil {
		t.Fatalf("up to the binding: %v", err)
	}
	for _, table := range []string{"authorization_requests", "authorization_codes"} {
		if keys := rows(t, db, "SELECT dpop_key FROM "+table); !slices.Equal(keys, []string{""}) {
			t.Errorf("%s after the upgrade holds keys %q, want the one row, bound to none", table, keys)
		}
	}

	migrateDownTo(t, db, log, 4)
	if has := rows(t, db, `SELECT table_name FROM information_schema.columns
		WHERE column_name = 'dpop_key' AND table_name IN ('authorization_requests', 'authorization_codes')`); len(has) != 0 {
		t.Errorf("down left dpop_key on %v", has)
	}
	for _, table := range []string{"authorization_requests", "authorization_codes"} {
		if left := rows(t, db, "SELECT id::text FROM "+table); len(left) != 1 {
			t.Errorf("down left %d rows in %s, want 1", len(left), table)
		}
	}
}

// migrateDownTo takes the migrations down, one at a time, until version is
// the last one applied.
func migrateDownTo(t *testing.T, db *gorm.DB, log *slog.Logger, version int64) {
	t.Helper()

	applied := func() int64 {
		var at int64
		if err := db.Raw("SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied").Scan(&at).Error; err != nil {
			t.Fatal(err)
		}
		return at
	}
	for applied() > version {
		if err := database.MigrateDown(db, log); err != nil {
			t.Fatalf("down from %d: %v", applied(), err)
		}
	}
	if at := applied(); at != version {
		t.Fatalf("the database is at %d, want %d", at, version)
	}
}

// An installation from before a flow could require a second factor upgrades
// with every flow requiring none and every waiting sign-in its own; going
// back down removes the step-ups still waiting, which would otherwise read
// as sign-ins their first factor never proved.
func TestLiveStepUpUpgradesAnExistingDatabase(t *testing.T) {
	db, _ := databasetest.Open(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	migrateDownTo(t, db, log, 5)
	stepUpColumns := `SELECT table_name || '.' || column_name FROM information_schema.columns
		WHERE (table_name = 'login_flows' AND column_name = 'require_second_factor')
			OR (table_name = 'pending_sign_ins' AND column_name = 'step_up_session_id')
		ORDER BY 1`
	if has := rows(t, db, stepUpColumns); len(has) != 0 {
		t.Fatalf("before step-up, the database has %v", has)
	}

	user := &model.User{Email: "ada@example.com", IsActive: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	session := &model.UserSession{TokenHash: "session", UserID: user.ID, AuthenticatedAt: time.Now(), ExpiresAt: later}
	waiting := &model.PendingSignIn{HandleHash: "waiting", UserID: user.ID, ExpiresAt: later}
	if err := db.Omit(clause.Associations, "Remember").Create(session).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Omit(clause.Associations, "StepUpSessionID", "ApplicationID", "DeviceAuthorizationID").Create(waiting).Error; err != nil {
		t.Fatal(err)
	}
	flows := rows(t, db, "SELECT id::text FROM login_flows")
	if len(flows) == 0 {
		t.Fatal("the database has no login flow to upgrade")
	}

	if err := database.Migrate(db, log); err != nil {
		t.Fatalf("up to step-up: %v", err)
	}
	if required := rows(t, db, "SELECT id::text FROM login_flows WHERE require_second_factor"); len(required) != 0 {
		t.Errorf("after the upgrade, flows %v require a second factor", required)
	}
	if steps := rows(t, db, "SELECT handle_hash FROM pending_sign_ins WHERE step_up_session_id IS NOT NULL"); len(steps) != 0 {
		t.Errorf("after the upgrade, %v are step-ups", steps)
	}

	stepUp := &model.PendingSignIn{HandleHash: "step-up", UserID: user.ID, StepUpSessionID: &session.ID, ExpiresAt: later}
	if err := db.Omit(clause.Associations).Create(stepUp).Error; err != nil {
		t.Fatal(err)
	}

	migrateDownTo(t, db, log, 5)
	if has := rows(t, db, stepUpColumns); len(has) != 0 {
		t.Errorf("down left %v", has)
	}
	if left := rows(t, db, "SELECT handle_hash FROM pending_sign_ins ORDER BY 1"); !slices.Equal(left, []string{"waiting"}) {
		t.Errorf("down left the waiting sign-ins %v, want only the one that was not a step-up", left)
	}
	if left := rows(t, db, "SELECT id::text FROM login_flows"); len(left) != len(flows) {
		t.Errorf("down left %d login flows, want %d", len(left), len(flows))
	}
}

// Sessions open before 00007 are recorded as not remembered, so a step-up of
// one writes a cookie for the browser's lifetime, as before; going down keeps
// every session.
func TestLiveRememberSessionUpgradesAnExistingDatabase(t *testing.T) {
	db, _ := databasetest.Open(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	migrateDownTo(t, db, log, 6)
	column := `SELECT column_name FROM information_schema.columns
		WHERE table_name = 'user_sessions' AND column_name = 'remember'`
	if has := rows(t, db, column); len(has) != 0 {
		t.Fatalf("before 00007, user_sessions has %v", has)
	}

	user := &model.User{Email: "ada@example.com", IsActive: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	open := &model.UserSession{TokenHash: "open", UserID: user.ID, AuthenticatedAt: time.Now(), ExpiresAt: later}
	if err := db.Omit(clause.Associations, "Remember").Create(open).Error; err != nil {
		t.Fatal(err)
	}

	if err := database.Migrate(db, log); err != nil {
		t.Fatalf("up to 00007: %v", err)
	}
	if remembered := rows(t, db, "SELECT token_hash FROM user_sessions WHERE remember"); len(remembered) != 0 {
		t.Errorf("after the upgrade, %v are remembered", remembered)
	}

	kept := &model.UserSession{TokenHash: "kept", UserID: user.ID, AuthenticatedAt: time.Now(), ExpiresAt: later, Remember: true}
	if err := db.Omit(clause.Associations).Create(kept).Error; err != nil {
		t.Fatal(err)
	}

	migrateDownTo(t, db, log, 6)
	if has := rows(t, db, column); len(has) != 0 {
		t.Errorf("down left %v", has)
	}
	if left := rows(t, db, "SELECT token_hash FROM user_sessions ORDER BY 1"); !slices.Equal(left, []string{"kept", "open"}) {
		t.Errorf("down left the sessions %v, want both", left)
	}
}

// Sign-ins waiting before 00008 name no application; going down deletes only
// those that name one, which the default flow would otherwise hold.
func TestLiveSignInApplicationUpgradesAnExistingDatabase(t *testing.T) {
	db, _ := databasetest.Open(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	migrateDownTo(t, db, log, 7)
	column := `SELECT column_name FROM information_schema.columns
		WHERE table_name = 'pending_sign_ins' AND column_name = 'application_id'`
	if has := rows(t, db, column); len(has) != 0 {
		t.Fatalf("before 00008, pending_sign_ins has %v", has)
	}

	user := &model.User{Email: "ada@example.com", IsActive: true}
	app := &model.Application{Name: "Telly", ClientID: "telly", Type: model.AppNative, TokenAuthMethod: model.AuthNone, IsEnabled: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	waiting := &model.PendingSignIn{HandleHash: "waiting", UserID: user.ID, ExpiresAt: later}
	if err := db.Omit(clause.Associations, "ApplicationID", "DeviceAuthorizationID").Create(waiting).Error; err != nil {
		t.Fatal(err)
	}

	if err := database.Migrate(db, log); err != nil {
		t.Fatalf("up to 00008: %v", err)
	}
	if named := rows(t, db, "SELECT handle_hash FROM pending_sign_ins WHERE application_id IS NOT NULL"); len(named) != 0 {
		t.Errorf("after the upgrade, %v name an application", named)
	}

	device := &model.PendingSignIn{HandleHash: "device", UserID: user.ID, ApplicationID: &app.ID, ExpiresAt: later}
	if err := db.Omit(clause.Associations, "DeviceAuthorizationID").Create(device).Error; err != nil {
		t.Fatal(err)
	}

	migrateDownTo(t, db, log, 7)
	if has := rows(t, db, column); len(has) != 0 {
		t.Errorf("down left %v", has)
	}
	if left := rows(t, db, "SELECT handle_hash FROM pending_sign_ins ORDER BY 1"); !slices.Equal(left, []string{"waiting"}) {
		t.Errorf("down left the waiting sign-ins %v, want only the one naming no application", left)
	}
}

// Step-ups waiting before 00009 name no device; going down keeps every
// waiting sign-in, which then only stops naming its device.
func TestLiveSignInDeviceUpgradesAnExistingDatabase(t *testing.T) {
	db, _ := databasetest.Open(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	migrateDownTo(t, db, log, 8)
	column := `SELECT column_name FROM information_schema.columns
		WHERE table_name = 'pending_sign_ins' AND column_name = 'device_authorization_id'`
	if has := rows(t, db, column); len(has) != 0 {
		t.Fatalf("before 00009, pending_sign_ins has %v", has)
	}

	user := &model.User{Email: "ada@example.com", IsActive: true}
	app := &model.Application{Name: "Telly", ClientID: "telly", Type: model.AppNative, TokenAuthMethod: model.AuthNone, IsEnabled: true}
	for _, row := range []any{user, app} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	later := time.Now().Add(time.Hour)
	waiting := &model.PendingSignIn{HandleHash: "waiting", UserID: user.ID, ApplicationID: &app.ID, ExpiresAt: later}
	if err := db.Omit(clause.Associations, "DeviceAuthorizationID").Create(waiting).Error; err != nil {
		t.Fatal(err)
	}

	if err := database.Migrate(db, log); err != nil {
		t.Fatalf("up to 00009: %v", err)
	}
	if named := rows(t, db, "SELECT handle_hash FROM pending_sign_ins WHERE device_authorization_id IS NOT NULL"); len(named) != 0 {
		t.Errorf("after the upgrade, %v name a device", named)
	}

	device := &model.DeviceAuthorization{DeviceCodeHash: "device", UserCodeHash: "user", ApplicationID: app.ID, Scope: "openid",
		Status: model.DevicePending, Interval: 5, ExpiresAt: later}
	if err := db.Omit(clause.Associations).Create(device).Error; err != nil {
		t.Fatal(err)
	}
	stepUp := &model.PendingSignIn{HandleHash: "step-up", UserID: user.ID, ApplicationID: &app.ID, DeviceAuthorizationID: &device.ID, ExpiresAt: later}
	if err := db.Omit(clause.Associations).Create(stepUp).Error; err != nil {
		t.Fatal(err)
	}

	migrateDownTo(t, db, log, 8)
	if has := rows(t, db, column); len(has) != 0 {
		t.Errorf("down left %v", has)
	}
	if left := rows(t, db, "SELECT handle_hash FROM pending_sign_ins ORDER BY 1"); !slices.Equal(left, []string{"step-up", "waiting"}) {
		t.Errorf("down left the waiting sign-ins %v, want both", left)
	}
}

// Connections from before 00010 answer for nobody's second factor: each is
// local, accepts amr mfa should it be switched to its provider, and lists
// no SAML context. Going down keeps every connection.
func TestLiveSSOSecondFactorUpgradesAnExistingDatabase(t *testing.T) {
	db, _ := databasetest.Open(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	migrateDownTo(t, db, log, 9)
	columns := `SELECT column_name FROM information_schema.columns
		WHERE table_name = 'sso_connections' AND column_name IN ('second_factor', 'accepted_amr', 'mfa_contexts')`
	if has := rows(t, db, columns); len(has) != 0 {
		t.Fatalf("before 00010, sso_connections has %v", has)
	}

	acme := &model.SSOConnection{Slug: "acme", Name: "Acme", Protocol: model.SSOProtocolOIDC, Domains: model.StringList{"acme.test"},
		Issuer: "https://acme.test", ClientID: "acme", Matching: model.SSOMatchLink, RoleMappings: model.SSORoleMappings{}}
	if err := db.Omit(clause.Associations, "SecondFactor", "AcceptedAMR", "MFAContexts").Create(acme).Error; err != nil {
		t.Fatal(err)
	}

	if err := database.Migrate(db, log); err != nil {
		t.Fatalf("up to 00010: %v", err)
	}
	got := rows(t, db, "SELECT second_factor || ' ' || accepted_amr::text || ' ' || mfa_contexts::text FROM sso_connections")
	if want := []string{`local ["mfa"] []`}; !slices.Equal(got, want) {
		t.Errorf("after the upgrade the connection is %v, want %v", got, want)
	}
	if err := db.Exec(`UPDATE sso_connections SET second_factor = 'both'`).Error; err == nil {
		t.Error("a second factor nobody named was stored")
	}
	if err := db.Exec(`UPDATE sso_connections SET second_factor = 'provider'`).Error; err != nil {
		t.Fatal(err)
	}

	migrateDownTo(t, db, log, 9)
	if has := rows(t, db, columns); len(has) != 0 {
		t.Errorf("down left %v", has)
	}
	if left := rows(t, db, "SELECT slug FROM sso_connections"); !slices.Equal(left, []string{"acme"}) {
		t.Errorf("down left the connections %v, want acme", left)
	}
}
