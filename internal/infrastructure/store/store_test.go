package store

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// TestTranslate covers the one piece of the store that is worth testing
// without a database: turning what the driver says into the errors the rest
// of the project handles.
func TestTranslate(t *testing.T) {
	tests := []struct {
		name string
		in   error
		want error
	}{
		{name: "nothing went wrong", in: nil, want: nil},
		{name: "no such row", in: gorm.ErrRecordNotFound, want: ErrNotFound},
		{
			name: "a wrapped missing row still counts",
			in:   errors.Join(errors.New("loading user"), gorm.ErrRecordNotFound),
			want: ErrNotFound,
		},
		{
			name: "a unique index refuses the row",
			in:   &pgconn.PgError{Code: "23505", Message: `duplicate key value violates unique constraint "idx_users_email"`},
			want: ErrDuplicate,
		},
		{
			name: "a unique index refuses the row in another language",
			in:   &pgconn.PgError{Code: "23505", Message: `повторяющееся значение ключа нарушает ограничение уникальности "idx_users_email"`},
			want: ErrDuplicate,
		},
		{
			name: "a wrapped unique violation still counts",
			in:   fmt.Errorf("creating a user: %w", &pgconn.PgError{Code: "23505"}),
			want: ErrDuplicate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translate(tt.in)

			if !errors.Is(got, tt.want) {
				t.Fatalf("translate(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// Anything the store does not recognise has to reach the caller unchanged: a
// connection that dropped is not a missing row.
func TestTranslateKeepsOtherErrors(t *testing.T) {
	tests := []struct {
		name string
		in   error
	}{
		{name: "a dropped connection", in: errors.New("connection refused")},
		{name: "a foreign key violation", in: &pgconn.PgError{Code: "23503", Message: "violates foreign key constraint"}},
		{name: "a message that only mentions a duplicate", in: errors.New("duplicate column name in the query")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translate(tt.in)

			if !errors.Is(got, tt.in) || errors.Is(got, ErrDuplicate) || errors.Is(got, ErrNotFound) {
				t.Fatalf("translate(%v) = %v, want it unchanged", tt.in, got)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := map[string]string{
		"Ann":        "%ann%",
		"50%":        `%50\%%`,
		"first_name": `%first\_name%`,
		`a\b`:        `%a\\b%`,
	}

	for in, want := range tests {
		if got := contains(in); got != want {
			t.Errorf("contains(%q) = %q, want %q", in, got, want)
		}
	}
}
