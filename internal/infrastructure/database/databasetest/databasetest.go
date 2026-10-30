// Package databasetest makes a throwaway Postgres database for a test,
// migrated to the newest schema or left empty, and dropped when the test
// ends.
package databasetest

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"gorm.io/gorm"

	"vard/internal/brand"
	"vard/internal/config"
	"vard/internal/infrastructure/database"
)

// Env names a Postgres server the tests may create and drop databases on, as
// a URL: postgres://user:password@localhost:5432/postgres?sslmode=disable.
const Env = "VARD_TEST_DB_DSN"

// Open makes the database and returns it with the configuration that opens
// it, which a test starting a second server on the same database needs. A
// test without Env is skipped, so `go test ./...` passes anywhere.
func Open(t testing.TB) (*gorm.DB, config.DB) {
	t.Helper()

	db, cfg := Empty(t)
	if err := database.Migrate(db, cfg, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return db, cfg
}

// Empty makes the database with nothing in it, not even the migrations.
func Empty(t testing.TB) (*gorm.DB, config.DB) {
	t.Helper()

	dsn := os.Getenv(Env)
	if dsn == "" {
		t.Skipf("%s is not set", Env)
	}

	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	server, err := database.Open(config.DB{Driver: "postgres", DSN: dsn, TimeZone: "UTC"}, quiet)
	if err != nil {
		t.Fatalf("open the test database server: %v", err)
	}
	t.Cleanup(func() { _ = database.Close(server) })

	suffix := make([]byte, 6)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	name := brand.Slug + "_test_" + hex.EncodeToString(suffix)

	if err := server.Exec("CREATE DATABASE " + name).Error; err != nil {
		t.Fatalf("create %s: %v", name, err)
	}

	address, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("%s is not a URL: %v", Env, err)
	}
	address.Path = "/" + name

	cfg := config.DB{Driver: "postgres", DSN: address.String(), TimeZone: "UTC", MigrateDir: migrationsDir(t)}

	db, err := database.Open(cfg, quiet)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	t.Cleanup(func() {
		_ = database.Close(db)
		_ = server.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)").Error
	})

	return db, cfg
}

// migrationsDir is found from this file rather than from wherever the tests
// run.
func migrationsDir(t testing.TB) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot find the migrations")
	}

	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "migrations")
}
