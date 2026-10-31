// Package changelogs holds every change to the schema after the setup, one
// file each, numbered in the order they run: 00002_x.go is version 2. Each
// adds itself from its init function with add; make migrate-new name=x
// writes the next one. A changelog that has run anywhere is never edited.
package changelogs

import (
	"context"
	"database/sql"
	"slices"

	"github.com/pressly/goose/v3"
)

var all []*goose.Migration

// add registers a changelog under its file's number. Both directions run in
// one transaction, so a change that fails half way leaves nothing behind.
func add(version int64, up, down func(context.Context, *sql.Tx) error) {
	all = append(all, goose.NewGoMigration(version, &goose.GoFunc{RunTx: up}, &goose.GoFunc{RunTx: down}))
}

// All is every changelog, oldest first.
func All() []*goose.Migration {
	sorted := slices.Clone(all)
	slices.SortFunc(sorted, func(a, b *goose.Migration) int { return int(a.Version - b.Version) })
	return sorted
}
