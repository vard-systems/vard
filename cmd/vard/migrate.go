package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pressly/goose/v3"

	"vard/internal/config"
	"vard/internal/database"
)

// migrateCommand applies, rolls back, lists and creates the database
// migrations in migrations/.
func migrateCommand(args []string) int {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := migrate(log, args); err != nil {
		log.Error("fatal", "error", err)
		return 1
	}
	return 0
}

func migrate(log *slog.Logger, args []string) error {
	// Writing a new, empty migration needs no database and no configuration.
	if len(args) == 2 && args[0] == "new" {
		// Numbered in order (00002_x.go), not by timestamp, so the folder reads
		// as the sequence it is applied in.
		goose.SetSequential(true)
		return goose.Create(nil, "migrations", args[1], "go")
	}

	if len(args) != 1 {
		return fmt.Errorf("usage: %s migrate up|down|status, or new <name>", os.Args[0])
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.DB, log)
	if err != nil {
		return err
	}
	defer database.Close(db)

	switch command := args[0]; command {
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
