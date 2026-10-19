package users

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"loginer/internal/api/kit/respond"
)

// TestUserRequestValidate covers the one column a user record has of its own.
// It is what someone signs in with, so an address that is not one has no
// business being stored.
func TestUserRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request userRequest
		want    string // the message, or "" when the record is fine
	}{
		{
			name:    "an address",
			request: userRequest{Email: "mira@example.com"},
		},
		{
			name:    "spaces and capitals are tidied away",
			request: userRequest{Email: "  Mira@Example.com "},
		},
		{
			name:    "nothing at all",
			request: userRequest{},
			want:    "email is required.",
		},
		{
			name:    "spaces only",
			request: userRequest{Email: "   "},
			want:    "email is required.",
		},
		{
			name:    "not an address",
			request: userRequest{Email: "mira at example"},
			want:    "email must be an email address.",
		},
		{
			name:    "longer than the column",
			request: userRequest{Email: strings.Repeat("m", 250) + "@example.com"},
			want:    "email must be at most 255 characters.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request
			err := request.validate(false)

			if tt.want == "" {
				if err != nil {
					t.Fatalf("validate() = %v, want nothing", err)
				}
				if request.Email != "mira@example.com" {
					t.Errorf("email = %q, want it trimmed and lowered", request.Email)
				}
				return
			}

			var fault respond.Fault
			if !errors.As(err, &fault) {
				t.Fatalf("validate() = %v, want a respond.Fault", err)
			}

			if fault.Status != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", fault.Status)
			}
			if fault.Message != tt.want {
				t.Errorf("message = %q, want %q", fault.Message, tt.want)
			}
		})
	}
}

// TestUserRequestPassword covers the password an administrator gives a user:
// required when the user is made, optional after, and typed the same twice.
func TestUserRequestPassword(t *testing.T) {
	tests := []struct {
		name     string
		creating bool
		password string
		confirm  string
		want     string // the message, or "" when the request is fine
	}{
		{name: "a new user with a password", creating: true, password: "long-enough", confirm: "long-enough"},
		{name: "a new user without one", creating: true, want: "A password is required."},
		{name: "an update keeping the password", creating: false},
		{name: "an update changing it", creating: false, password: "long-enough", confirm: "long-enough"},
		{
			name:     "too short",
			creating: true,
			password: "short",
			confirm:  "short",
			want:     "The password must be at least 8 characters.",
		},
		{
			name:     "longer than bcrypt reads",
			creating: true,
			password: strings.Repeat("p", 73),
			confirm:  strings.Repeat("p", 73),
			want:     "password must be at most 72 characters.",
		},
		{
			name:     "typed differently the second time",
			creating: false,
			password: "long-enough",
			confirm:  "long-enougg",
			want:     "confirm_password must match password.",
		},
		{
			name:     "not confirmed",
			creating: true,
			password: "long-enough",
			want:     "confirm_password must match password.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := userRequest{
				Email:           "mira@example.com",
				Password:        tt.password,
				ConfirmPassword: tt.confirm,
			}

			err := request.validate(tt.creating)

			if tt.want == "" {
				if err != nil {
					t.Fatalf("validate() = %v, want nothing", err)
				}
				return
			}

			var fault respond.Fault
			if !errors.As(err, &fault) {
				t.Fatalf("validate() = %v, want a respond.Fault", err)
			}
			if fault.Message != tt.want {
				t.Errorf("message = %q, want %q", fault.Message, tt.want)
			}
		})
	}
}
