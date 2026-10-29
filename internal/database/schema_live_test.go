package database_test

import (
	"io"
	"log/slog"
	"slices"
	"testing"

	"gorm.io/gorm"

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
