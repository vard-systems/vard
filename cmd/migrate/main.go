// Command migrate applies, rolls back, lists and creates the database migrations
// in migrations/.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pressly/goose/v3"

	"loginer/internal/config"
	"loginer/internal/database"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(log); err != nil {
		log.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	// Writing a new, empty migration needs no database and no configuration.
	if len(os.Args) == 3 && os.Args[1] == "new" {
		return goose.Create(nil, "migrations", os.Args[2], "go")
	}

	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s up|down|status, or new <name>", os.Args[0])
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.DB)
	if err != nil {
		return err
	}
	defer database.Close(db)

	switch command := os.Args[1]; command {
	case "up":
		return database.Migrate(db, cfg.DB, log)
	case "down":
		return database.MigrateDown(db, cfg.DB, log)
	case "status":
		return database.MigrateStatus(db, cfg.DB)
	default:
		return fmt.Errorf("unknown command %q: want up, down or status", command)
	}
}
