package database

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"gorm.io/gorm"

	"vard/internal/brand"
	"vard/migrations"
	"vard/migrations/reset"
)

// Migrate applies every migration that has not run.
func Migrate(db *gorm.DB, log *slog.Logger) error {
	ctx := context.Background()
	if err := refuseSquashedHistory(db); err != nil {
		return err
	}
	provider, err := newProvider(db)
	if err != nil {
		return err
	}

	before, err := provider.GetDBVersion(ctx)
	if err != nil {
		return err
	}
	if _, err := provider.Up(ctx); err != nil {
		return err
	}
	after, err := provider.GetDBVersion(ctx)
	if err != nil {
		return err
	}

	if after == before {
		log.Info("database is up to date", "version", after)
	} else {
		log.Info("migrations applied", "from", before, "to", after)
	}
	return nil
}

// ErrSquashedHistory is a database migrated by the migrations from before
// they were squashed into migrations/setup. Its goose versions say it is up
// to date, so goose would run nothing and the server would start on a schema
// it does not know.
var ErrSquashedHistory = errors.New("this database was made by the migrations from before they were " +
	"squashed into one setup, and cannot be stepped forward: back it up and start the server on a new, " +
	"empty database")

// refuseSquashedHistory answers ErrSquashedHistory for such a database: it
// has run migrations, but not the setup, which is the only one that ever
// made token_revocations.
func refuseSquashedHistory(db *gorm.DB) error {
	var tables struct {
		Versions    bool
		Revocations bool
	}
	err := db.Raw(`SELECT to_regclass('goose_db_version') IS NOT NULL AS versions,
		to_regclass('token_revocations') IS NOT NULL AS revocations`).Scan(&tables).Error
	if err != nil || !tables.Versions || tables.Revocations {
		return err
	}

	// A table that may not exist cannot be named in the query above.
	var migrated bool
	err = db.Raw(`SELECT EXISTS (SELECT 1 FROM goose_db_version WHERE version_id > 0 AND is_applied)`).
		Scan(&migrated).Error
	if err != nil {
		return err
	}
	if migrated {
		return ErrSquashedHistory
	}
	return nil
}

// MigrateDown rolls back the newest migration.
func MigrateDown(db *gorm.DB, log *slog.Logger) error {
	ctx := context.Background()
	provider, err := newProvider(db)
	if err != nil {
		return err
	}

	if _, err := provider.Down(ctx); err != nil {
		return err
	}
	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		return err
	}

	log.Info("rolled back", "version", version)
	return nil
}

// MigrateReset takes every migration down and up again (migrations/reset).
func MigrateReset(db *gorm.DB, log *slog.Logger) error {
	provider, err := newProvider(db)
	if err != nil {
		return err
	}

	if err := reset.Run(context.Background(), provider); err != nil {
		return err
	}

	log.Info("database reset")
	return nil
}

// MigrationState is one migration and whether it has run.
type MigrationState struct {
	Version   int64
	Applied   bool
	AppliedAt time.Time
}

// MigrateStatus says which migrations have run, oldest first.
func MigrateStatus(db *gorm.DB) ([]MigrationState, error) {
	provider, err := newProvider(db)
	if err != nil {
		return nil, err
	}

	statuses, err := provider.Status(context.Background())
	if err != nil {
		return nil, err
	}

	states := make([]MigrationState, 0, len(statuses))
	for _, status := range statuses {
		states = append(states, MigrationState{
			Version:   status.Source.Version,
			Applied:   status.State == goose.StateApplied,
			AppliedAt: status.AppliedAt,
		})
	}
	return states, nil
}

// newProvider is goose over the migrations compiled in, and nothing in its
// global registry or on disk.
//
// Its lock makes instances started together, as replicas are, migrate one
// at a time: the rest wait, then find nothing left to do. The lock is a
// session's, held on a connection of its own while goose works.
func newProvider(db *gorm.DB) (*goose.Provider, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	locker, err := lock.NewPostgresSessionLocker(
		lock.WithLockID(migrationLockID()),
		// Asked for every second, for up to ten minutes: a long migration
		// on another instance is waited for, not failed.
		lock.WithLockTimeout(1, 600),
		lock.WithUnlockTimeout(1, 60),
	)
	if err != nil {
		return nil, err
	}

	return goose.NewProvider(goose.DialectPostgres, sqlDB, nil,
		goose.WithGoMigrations(migrations.All()...),
		goose.WithDisableGlobalRegistry(true),
		goose.WithSessionLocker(locker),
	)
}

// migrationLockID is this project's advisory lock, so another program
// migrating with goose on the same server waits for nothing of ours.
func migrationLockID() int64 {
	sum := fnv.New64a()
	_, _ = sum.Write([]byte(brand.Slug + " migrations"))
	return int64(sum.Sum64() >> 1)
}

// String prints a state as `migrate status` lists it.
func (s MigrationState) String() string {
	if !s.Applied {
		return fmt.Sprintf("%05d  pending", s.Version)
	}
	return fmt.Sprintf("%05d  applied %s", s.Version, s.AppliedAt.Local().Format(time.DateTime))
}
