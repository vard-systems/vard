package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"vard/internal/config"
	"vard/internal/database"
	"vard/internal/jose"
	"vard/internal/store"
)

// secretsCommand seals every stored secret again under the current
// VARD_SECRET_KEY: what the key VARD_SECRET_KEY_PREVIOUS names sealed is
// written under the current one, after which the previous key can be
// dropped. The server does the same as it starts mid-rotation; this is for
// doing it without starting one.
func secretsCommand(args []string) int {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	if err := secrets(args, os.Stdout); err != nil {
		log.Error("fatal", "error", err)
		return 1
	}
	return 0
}

func secrets(args []string, stdout io.Writer) error {
	if len(args) != 1 || args[0] != "reseal" {
		return fmt.Errorf("usage: %s secrets reseal", os.Args[0])
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	sealer, err := jose.NewSealer(cfg.SecretKey, cfg.SecretKeyPrevious)
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.DB)
	if err != nil {
		return err
	}
	defer database.Close(db)

	changed, err := store.New(db).ResealAll(context.Background(), resealWith(sealer))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "sealed %d values again under the current key\n", changed)
	return err
}

// resealWith opens a value with any key the sealer knows and, when it was
// not sealed under the current one where it is, seals it again.
func resealWith(sealer *jose.Sealer) store.Reseal {
	return func(where jose.Context, sealed []byte) ([]byte, bool, error) {
		plain, stale, err := sealer.OpenFor(where, sealed)
		if err != nil || !stale {
			return nil, false, err
		}

		resealed, err := sealer.SealFor(where, plain)
		return resealed, true, err
	}
}
