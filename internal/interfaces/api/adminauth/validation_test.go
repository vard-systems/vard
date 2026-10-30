package adminauth

import (
	"errors"
	"net/http"
	"testing"

	"vard/internal/interfaces/api/kit/respond"
)

// TestLoginRequestValidate covers the sign-in form: both fields are wanted,
// and a username of nothing but spaces is nothing.
func TestLoginRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request loginRequest
		want    string // the message, or "" when the form is fine
	}{
		{
			name:    "a filled in form",
			request: loginRequest{Username: "admin", Password: "hunter2"},
		},
		{
			name:    "the username is trimmed",
			request: loginRequest{Username: "  admin  ", Password: "hunter2"},
		},
		{
			name:    "no username",
			request: loginRequest{Password: "hunter2"},
			want:    "username is required.",
		},
		{
			name:    "a username of spaces",
			request: loginRequest{Username: "   ", Password: "hunter2"},
			want:    "username is required.",
		},
		{
			name:    "no password",
			request: loginRequest{Username: "admin"},
			want:    "password is required.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request
			err := request.validate()

			if tt.want == "" {
				if err != nil {
					t.Fatalf("validate() = %v, want nothing", err)
				}
				if request.Username != "admin" {
					t.Errorf("username = %q, want it trimmed", request.Username)
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

// TestProfileRequestValidate covers the form an administrator changes their
// own name and address with: the address is tidied the way the admins
// endpoints tidy it, and a first name and an address are wanted.
func TestProfileRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request profileRequest
		want    string // the problem's code, or "" when the form is fine
	}{
		{
			name:    "a filled in form",
			request: profileRequest{FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"},
		},
		{
			name:    "no last name is fine",
			request: profileRequest{FirstName: "Ada", Email: "ada@example.com"},
		},
		{
			name:    "an address with spaces and capitals",
			request: profileRequest{FirstName: " Ada ", Email: "  Ada@Example.com "},
		},
		{
			name:    "no first name",
			request: profileRequest{Email: "ada@example.com"},
			want:    "validation.required",
		},
		{
			name:    "a first name of spaces",
			request: profileRequest{FirstName: "   ", Email: "ada@example.com"},
			want:    "validation.required",
		},
		{
			name:    "no address",
			request: profileRequest{FirstName: "Ada"},
			want:    "validation.required",
		},
		{
			name:    "an address that is not one",
			request: profileRequest{FirstName: "Ada", Email: "ada"},
			want:    "validation.email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request
			err := request.validate()

			if tt.want == "" {
				if err != nil {
					t.Fatalf("validate() = %v, want nothing", err)
				}
				if request.Email != "ada@example.com" || request.FirstName != "Ada" {
					t.Errorf("tidied to %q, %q; want the name trimmed and the address lower case", request.FirstName, request.Email)
				}
				return
			}

			var fault respond.Fault
			if !errors.As(err, &fault) {
				t.Fatalf("validate() = %v, want a respond.Fault", err)
			}
			if fault.Code != tt.want {
				t.Errorf("code = %q, want %q", fault.Code, tt.want)
			}
		})
	}
}

// TestPasswordRequestValidate covers changing one's own password: both
// passwords are wanted, and the new one is held to an administrator's floor.
func TestPasswordRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request passwordRequest
		want    string // the problem's code, or "" when the form is fine
	}{
		{
			name:    "a long enough new password",
			request: passwordRequest{CurrentPassword: "old-password", NewPassword: "a-new-password-1"},
		},
		{
			name:    "exactly the shortest allowed",
			request: passwordRequest{CurrentPassword: "old-password", NewPassword: "01234567890123"},
		},
		{
			name:    "one character short",
			request: passwordRequest{CurrentPassword: "old-password", NewPassword: "0123456789012"},
			want:    "admin_password_too_short",
		},
		{
			name:    "no current password",
			request: passwordRequest{NewPassword: "a-new-password"},
			want:    "validation.required",
		},
		{
			name:    "no new password",
			request: passwordRequest{CurrentPassword: "old-password"},
			want:    "validation.required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.validate()

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
			if fault.Code != tt.want {
				t.Errorf("code = %q, want %q", fault.Code, tt.want)
			}
		})
	}
}
