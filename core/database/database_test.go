package database

import (
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vard/core/config"
)

// The time zone and the server-side limits are added by hand rather than
// through url encoding, because escaping turns the zone's slash into %2F and
// Postgres rejects it; a DSN that sets its own keeps it.
func TestDSNAddsTheZoneAndLimits(t *testing.T) {
	cfg := config.DB{TimeZone: "Asia/Bishkek", StatementTimeout: 15 * time.Second, IdleInTransactionTimeout: time.Minute}
	const limits = "&statement_timeout=15000&idle_in_transaction_session_timeout=60000"

	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "dsn already has parameters",
			dsn:  "postgres://u:p@localhost:5432/db?sslmode=disable",
			want: "postgres://u:p@localhost:5432/db?sslmode=disable&TimeZone=Asia/Bishkek" + limits,
		},
		{
			name: "dsn has no parameters",
			dsn:  "postgres://u:p@localhost:5432/db",
			want: "postgres://u:p@localhost:5432/db?TimeZone=Asia/Bishkek" + limits,
		},
		{
			name: "dsn already names a zone and a timeout, so they are left alone",
			dsn:  "postgres://u:p@localhost:5432/db?TimeZone=UTC&statement_timeout=0",
			want: "postgres://u:p@localhost:5432/db?TimeZone=UTC&statement_timeout=0&idle_in_transaction_session_timeout=60000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			with := cfg
			with.DSN = tt.dsn
			got := dsn(with)

			if got != tt.want {
				t.Errorf("dsn() = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "%2F") {
				t.Error("the slash in the zone name was escaped; Postgres rejects that")
			}
		})
	}
}

// A database reached over a network without TLS is warned about; one on this
// machine, or a container's on its own network, is not.
func TestUnencryptedFindsAPlainConnectionOverANetwork(t *testing.T) {
	tests := []struct {
		dsn  string
		want bool
	}{
		{"postgres://u:p@localhost:5432/db?sslmode=disable", false},
		{"postgres://u:p@127.0.0.1:5432/db?sslmode=disable", false},
		{"postgres://u:p@postgres:5432/db?sslmode=disable", false},
		{"postgres://u:p@db.example.com:5432/db?sslmode=disable", true},
		{"postgres://u:p@db.example.com:5432/db", true},
		{"postgres://u:p@db.example.com:5432/db?sslmode=prefer", true},
		{"postgres://u:p@10.0.0.5:5432/db?sslmode=disable", true},
		{"postgres://u:p@db.example.com:5432/db?sslmode=verify-full", false},
	}

	for _, tt := range tests {
		t.Run(tt.dsn, func(t *testing.T) {
			if got := unencrypted(tt.dsn); got != tt.want {
				t.Errorf("unencrypted = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOpenRejectsUnknownDriver(t *testing.T) {
	_, err := Open(config.DB{Driver: "mysql", DSN: "whatever"}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err == nil {
		t.Fatal("want an error for an unsupported driver, got none")
	}
	if !strings.Contains(err.Error(), "mysql") {
		t.Errorf("want the error to name the driver, got: %v", err)
	}
}
