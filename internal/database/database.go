// Package database opens the Postgres connection and runs the migrations.
package database

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"vard/internal/config"
)

func Open(cfg config.DB, log *slog.Logger) (*gorm.DB, error) {
	if cfg.Driver != "postgres" {
		return nil, fmt.Errorf("unsupported database driver %q", cfg.Driver)
	}

	level := logger.Warn
	if cfg.LogQueries {
		level = logger.Info
	}

	// A lookup that finds nothing is not logged as an error: the store uses
	// First to ask "is there one?" and turns the answer into store.ErrNotFound.
	// Queries are logged without their values, which are hashes, sealed
	// secrets and people's addresses.
	db, err := gorm.Open(postgres.Open(dsn(cfg)), &gorm.Config{
		Logger: logger.NewSlogLogger(log, logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  level,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
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
	// behind a proxy is picked up; a quarter are kept idle, enough for the
	// usual load without holding the rest of Postgres's connections.
	if cfg.MaxConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxConns)
		sqlDB.SetMaxIdleConns(max(2, cfg.MaxConns/4))
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
		sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	}

	if unencrypted(cfg.DSN) {
		log.Warn("the database connection is not encrypted; set sslmode=verify-full for a database reached over a network")
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

// dsn adds the time zone and the server-side limits, each only where the DSN
// does not set its own. They are appended by hand: URL escaping would turn
// "Asia/Bishkek" into Asia%2FBishkek, which Postgres rejects.
func dsn(cfg config.DB) string {
	out := cfg.DSN
	add := func(name, value string) {
		if strings.Contains(out, name+"=") {
			return
		}
		separator := "?"
		if strings.Contains(out, "?") {
			separator = "&"
		}
		out += separator + name + "=" + value
	}

	add("TimeZone", cfg.TimeZone)
	add("statement_timeout", strconv.FormatInt(cfg.StatementTimeout.Milliseconds(), 10))
	add("idle_in_transaction_session_timeout", strconv.FormatInt(cfg.IdleInTransactionTimeout.Milliseconds(), 10))

	return out
}

// unencrypted says a DSN reaches a database over a network without TLS. A
// host on this machine, or a bare name such as a container's on its own
// network, is not counted.
func unencrypted(dsn string) bool {
	address, err := url.Parse(dsn)
	if err != nil {
		return false
	}

	switch address.Query().Get("sslmode") {
	case "require", "verify-ca", "verify-full":
		return false
	}

	host := address.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		return !ip.IsLoopback()
	}
	return host != "" && host != "localhost" && strings.Contains(host, ".")
}
