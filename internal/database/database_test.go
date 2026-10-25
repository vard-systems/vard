package database

import (
	"strings"
	"testing"

	"vard/internal/config"
)

// The time zone is added by hand rather than through url encoding, because
// escaping turns the slash into %2F and Postgres rejects it.
func TestDSNAddsTimeZone(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		tz   string
		want string
	}{
		{
			name: "dsn already has parameters",
			dsn:  "postgres://u:p@localhost:5432/db?sslmode=disable",
			tz:   "Asia/Bishkek",
			want: "postgres://u:p@localhost:5432/db?sslmode=disable&TimeZone=Asia/Bishkek",
		},
		{
			name: "dsn has no parameters",
			dsn:  "postgres://u:p@localhost:5432/db",
			tz:   "Asia/Bishkek",
			want: "postgres://u:p@localhost:5432/db?TimeZone=Asia/Bishkek",
		},
		{
			name: "dsn already names a zone, so it is left alone",
			dsn:  "postgres://u:p@localhost:5432/db?TimeZone=UTC",
			tz:   "Asia/Bishkek",
			want: "postgres://u:p@localhost:5432/db?TimeZone=UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dsn(config.DB{DSN: tt.dsn, TimeZone: tt.tz})

			if got != tt.want {
				t.Errorf("dsn() = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "%2F") {
				t.Error("the slash in the zone name was escaped; Postgres rejects that")
			}
		})
	}
}

func TestOpenRejectsUnknownDriver(t *testing.T) {
	_, err := Open(config.DB{Driver: "mysql", DSN: "whatever"})

	if err == nil {
		t.Fatal("want an error for an unsupported driver, got none")
	}
	if !strings.Contains(err.Error(), "mysql") {
		t.Errorf("want the error to name the driver, got: %v", err)
	}
}
