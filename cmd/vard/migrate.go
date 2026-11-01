package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"vard/core/config"
	"vard/core/database"
	"vard/migrations"
)

// migrateCommand applies, rolls back, lists and resets the database
// migrations, and writes the next changelog.
func migrateCommand(args []string) int {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := migrate(log, args); err != nil {
		log.Error("fatal", "error", err)
		return 1
	}
	return 0
}

const migrateUsage = "usage: %s migrate up|down|status, reset -yes, or new <name>"

func migrate(log *slog.Logger, args []string) error {
	// Writing a new, empty changelog needs no database and no configuration.
	if len(args) == 2 && args[0] == "new" {
		path, err := newChangelog(changelogsDir, args[1])
		if err != nil {
			return err
		}
		fmt.Println("created", path)
		return nil
	}

	command := ""
	if len(args) > 0 {
		command = args[0]
	}
	switch {
	case len(args) == 1 && command != "reset":
	case len(args) == 2 && command == "reset" && args[1] == "-yes":
	case command == "reset":
		return errors.New("reset deletes every row in the database; run it as migrate reset -yes")
	default:
		return fmt.Errorf(migrateUsage, os.Args[0])
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

	switch command {
	case "up":
		if cfg.DB.MigrateDSN != "" {
			return database.MigrateAsOwner(cfg.DB, log)
		}
		return database.Migrate(owner, log)
	case "down":
		return database.MigrateDown(owner, log)
	case "status":
		states, err := database.MigrateStatus(owner)
		if err != nil {
			return err
		}
		for _, state := range states {
			fmt.Println(state)
		}
		return nil
	case "reset":
		if err := database.MigrateReset(owner, log); err != nil {
			return err
		}
		// The tables are new, so the runtime role is granted them again.
		if cfg.DB.MigrateDSN != "" {
			return database.MigrateAsOwner(cfg.DB, log)
		}
		return nil
	default:
		return fmt.Errorf("unknown command %q: want up, down, status or reset", command)
	}
}

const changelogsDir = "migrations/changelogs"

var changelogName = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// newChangelog writes the next changelog, numbered after the newest
// migration, and says where.
func newChangelog(dir, name string) (string, error) {
	if !changelogName.MatchString(name) {
		return "", fmt.Errorf("name the changelog in lower case with underscores (add_x), not %q", name)
	}

	all := migrations.All()
	version := all[len(all)-1].Version + 1

	words := strings.Split(name, "_")
	for i, word := range words {
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}

	var out bytes.Buffer
	err := changelogTemplate.Execute(&out, map[string]any{"Version": version, "Name": strings.Join(words, "")})
	if err != nil {
		return "", err
	}
	source, err := format.Source(out.Bytes())
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, fmt.Sprintf("%05d_%s.go", version, name))
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(source); err != nil {
		_ = file.Close()
		return "", err
	}
	return path, file.Close()
}

var changelogTemplate = template.Must(template.New("changelog").Parse(`package changelogs

import (
	"context"
	"database/sql"
)

func init() {
	add({{.Version}}, up{{.Name}}, down{{.Name}})
}

// up{{.Name}} …
func up{{.Name}}(_ context.Context, tx *sql.Tx) error {
	return nil
}

// down{{.Name}} undoes it, and says what is lost.
func down{{.Name}}(_ context.Context, tx *sql.Tx) error {
	return nil
}
`))
