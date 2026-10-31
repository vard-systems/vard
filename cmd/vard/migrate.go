package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pressly/goose/v3"

	"vard/internal/config"
	"vard/internal/infrastructure/database"
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

	// Migrating is the schema owner's: with a MigrateDSN it connects as the
	// owner, and never as the runtime role, which may not exist yet and may
	// not drop a table or read goose's.
	ownerCfg := cfg.DB
	if cfg.DB.MigrateDSN != "" {
		ownerCfg.DSN = cfg.DB.MigrateDSN
	}
	owner, err := database.Open(ownerCfg, log)
	if err != nil {
		return err
	}
	defer database.Close(owner)

	switch command := args[0]; command {
	case "up":
		if cfg.DB.MigrateDSN != "" {
			return database.MigrateAsOwner(cfg.DB, log)
		}
		return database.Migrate(owner, ownerCfg, log)
	case "down":
		return database.MigrateDown(owner, ownerCfg, log)
	case "status":
		return database.MigrateStatus(owner, ownerCfg)
	default:
		return fmt.Errorf("unknown command %q: want up, down or status", command)
	}
}
