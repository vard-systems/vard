package main

import (
	"io"
	"strings"
	"testing"
)

// Creating the first administrator from the command line refuses what it
// would refuse over the web, before it reaches for the database.
func TestAdminCreateRefusesBadInput(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		password string
		want     string
	}{
		{"no subcommand", []string{}, "", "usage"},
		{"an unknown subcommand", []string{"delete"}, "", "usage"},
		{"no address", []string{"create", "-first-name", "Root"}, "a-long-password-1\n", "-email"},
		{"an address with a name in it", []string{"create", "-email", "Root <root@example.com>", "-first-name", "Root"}, "a-long-password-1\n", "-email"},
		{"no first name", []string{"create", "-email", "root@example.com"}, "a-long-password-1\n", "-first-name"},
		{"a short password", []string{"create", "-email", "root@example.com", "-first-name", "Root"}, "short\n", "at least"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := admin(c.args, strings.NewReader(c.password), io.Discard)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("admin(%v) = %v, want an error about %q", c.args, err, c.want)
			}
		})
	}
}
