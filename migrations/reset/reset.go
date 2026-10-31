// Package reset takes a database back to nothing and builds it again: every
// migration's down, newest first, to before the setup, then every up. What
// is left is what a new installation starts with, and every row that was
// there is gone. It is for a development database (make db-reset); the
// command that runs it asks first.
package reset

import (
	"context"
	"fmt"

	"github.com/pressly/goose/v3"
)

// Run resets the database the provider migrates.
func Run(ctx context.Context, provider *goose.Provider) error {
	if _, err := provider.DownTo(ctx, 0); err != nil {
		return fmt.Errorf("take the database down: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("build the database again: %w", err)
	}
	return nil
}
