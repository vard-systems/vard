package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"

	"vard/internal/brand"
	"vard/internal/config"
	_ "vard/migrations"
)

func Migrate(db *gorm.DB, cfg config.DB, log *slog.Logger) error {
	sqlDB, err := prepareGoose(db)
	if err != nil {
		return err
	}

	unlock, err := migrationLock(sqlDB)
	if err != nil {
		return err
	}
	defer unlock()

	before, err := goose.GetDBVersion(sqlDB)
	if err != nil {
		return err
	}

	if err := goose.Up(sqlDB, cfg.MigrateDir); err != nil {
		return err
	}

	after, err := goose.GetDBVersion(sqlDB)
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

func MigrateDown(db *gorm.DB, cfg config.DB, log *slog.Logger) error {
	sqlDB, err := prepareGoose(db)
	if err != nil {
		return err
	}

	unlock, err := migrationLock(sqlDB)
	if err != nil {
		return err
	}
	defer unlock()

	if err := goose.Down(sqlDB, cfg.MigrateDir); err != nil {
		return err
	}

	version, err := goose.GetDBVersion(sqlDB)
	if err != nil {
		return err
	}

	log.Info("rolled back", "version", version)

	return nil
}

func MigrateStatus(db *gorm.DB, cfg config.DB) error {
	sqlDB, err := prepareGoose(db)
	if err != nil {
		return err
	}

	goose.SetLogger(printLogger{})

	return goose.Status(sqlDB, cfg.MigrateDir)
}

type printLogger struct{}

// Printf ends every line, as the log package goose writes through by default
// does; goose's own lines carry no newline, so a status would be one line.
func (printLogger) Printf(format string, v ...any) {
	fmt.Println(strings.TrimRight(fmt.Sprintf(format, v...), "\n"))
}

func (printLogger) Fatalf(format string, v ...any) {
	fmt.Fprintf(os.Stderr, format, v...)
	os.Exit(1)
}

// migrationLock makes instances started together, as replicas are, migrate
// one at a time: the rest wait, then find nothing left to do. The lock is a
// session's, so it is held on a connection of its own until unlock.
func migrationLock(db *sql.DB) (unlock func(), err error) {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	const lock = "SELECT pg_advisory_lock(hashtext($1))"
	if _, err := conn.ExecContext(ctx, lock, brand.Slug+" migrations"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("wait for other instances' migrations: %w", err)
	}

	return func() {
		_, _ = conn.ExecContext(ctx, "SELECT pg_advisory_unlock(hashtext($1))", brand.Slug+" migrations")
		_ = conn.Close()
	}, nil
}

func prepareGoose(db *gorm.DB) (*sql.DB, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return nil, err
	}

	goose.SetLogger(goose.NopLogger())

	return sqlDB, nil
}
