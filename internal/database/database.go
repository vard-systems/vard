// Package database opens the Postgres connection and runs the migrations.
package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"vard/internal/config"
)

func Open(cfg config.DB) (*gorm.DB, error) {
	if cfg.Driver != "postgres" {
		return nil, fmt.Errorf("unsupported database driver %q", cfg.Driver)
	}

	level := logger.Warn
	if cfg.LogQueries {
		level = logger.Info
	}

	// A lookup that finds nothing is not logged as an error: the store uses
	// First to ask "is there one?" and turns the answer into store.ErrNotFound.
	db, err := gorm.Open(postgres.Open(dsn(cfg)), &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  level,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Connections are recycled every half hour so a failover or a resized pool
	// behind a proxy is picked up. Unset keeps Go's defaults.
	if cfg.MaxConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxConns)
		sqlDB.SetMaxIdleConns(cfg.MaxConns)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
		sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	}

	return db, nil
}

func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// dsn appends the time zone by hand: URL escaping would turn "Asia/Bishkek"
// into Asia%2FBishkek, which Postgres rejects.
func dsn(cfg config.DB) string {
	if strings.Contains(cfg.DSN, "TimeZone=") {
		return cfg.DSN
	}

	separator := "?"
	if strings.Contains(cfg.DSN, "?") {
		separator = "&"
	}

	return cfg.DSN + separator + "TimeZone=" + cfg.TimeZone
}
