package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/mail"
	"os"
	"strings"
	"unicode/utf8"

	"vard/core/config"
	"vard/core/database"
	"vard/core/model"
	"vard/core/store"
)

// adminCommand sets the panel up from the command line, for an installation
// nobody can open a browser on yet.
func adminCommand(args []string) int {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	if err := admin(args, os.Stdin, os.Stdout); err != nil {
		log.Error("fatal", "error", err)
		return 1
	}
	return 0
}

func admin(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "create" {
		return fmt.Errorf("usage: %s admin create -email <address> -first-name <name> [-last-name <name>], with the password on standard input", os.Args[0])
	}

	flags := flag.NewFlagSet("admin create", flag.ContinueOnError)
	email := flags.String("email", "", "the first administrator's address, which is also their username")
	first := flags.String("first-name", "", "their first name")
	last := flags.String("last-name", "", "their last name")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	address, err := mail.ParseAddress(strings.TrimSpace(*email))
	if err != nil || address.Name != "" {
		return errors.New("-email must be one plain address")
	}
	if strings.TrimSpace(*first) == "" {
		return errors.New("-first-name is required")
	}

	// The password comes on standard input, never as a flag, which the
	// shell's history and the process list would keep.
	password, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	password = strings.TrimRight(password, "\r\n")
	if utf8.RuneCountInString(password) < model.MinAdminPasswordLength {
		return fmt.Errorf("the password on standard input must be at least %d characters", model.MinAdminPasswordLength)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Open(cfg.DB, slog.Default())
	if err != nil {
		return err
	}
	defer database.Close(db)

	ctx := context.Background()
	st := store.New(db)

	created := model.Admin{
		Username:  strings.ToLower(address.Address),
		Email:     strings.ToLower(address.Address),
		FirstName: strings.TrimSpace(*first),
		LastName:  strings.TrimSpace(*last),
		Status:    model.StatusActive,
	}
	if err := created.SetPassword(password); err != nil {
		return err
	}

	if err := st.CreateFirstAdmin(ctx, &created); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "created %s, a super admin; sign in at %s\n", created.Email, cfg.AdminURL)
	return err
}
