package setup

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"vard/internal/api/kit/respond"
)

// TestSetupRequestValidate covers the form the whole panel is opened with.
// A super admin is everything, so this is the one password in the project
// with a length rule.
func TestSetupRequestValidate(t *testing.T) {
	good := func() setupRequest {
		return setupRequest{
			Email:     "mira@example.com",
			Password:  "a-long-enough-password",
			FirstName: "Mira",
			LastName:  "Testova",
		}
	}

	tests := []struct {
		name    string
		change  func(*setupRequest)
		want    string // the message, or "" when the form is fine
		checkOn func(setupRequest) string
	}{
		{name: "a filled in form", change: func(*setupRequest) {}},
		{
			name:   "spaces and capitals are tidied away",
			change: func(r *setupRequest) { r.Email = "  Mira@Example.com "; r.FirstName = " Mira " },
			checkOn: func(r setupRequest) string {
				if r.Email != "mira@example.com" || r.FirstName != "Mira" {
					return "email = " + r.Email + ", first name = " + r.FirstName
				}
				return ""
			},
		},
		{
			name:   "no address",
			change: func(r *setupRequest) { r.Email = "" },
			want:   "email is required.",
		},
		{
			name:   "not an address",
			change: func(r *setupRequest) { r.Email = "mira at example" },
			want:   "email must be an email address.",
		},
		{
			name:   "a password anyone could guess the length of",
			change: func(r *setupRequest) { r.Password = "short" },
			want:   "The password must be at least 14 characters.",
		},
		{
			name:   "no name",
			change: func(r *setupRequest) { r.FirstName = "   " },
			want:   "first_name is required.",
		},
		{
			name:   "no last name is fine",
			change: func(r *setupRequest) { r.LastName = "" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := good()
			tt.change(&request)

			err := request.validate()

			if tt.want == "" {
				if err != nil {
					t.Fatalf("validate() = %v, want nothing", err)
				}
				if tt.checkOn != nil {
					if problem := tt.checkOn(request); problem != "" {
						t.Errorf("form was not tidied: %s", problem)
					}
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

// The password is taken as it was typed: trimming it would quietly change
// what someone chose, and spaces are as good a character as any.
func TestSetupKeepsThePasswordAsTyped(t *testing.T) {
	request := setupRequest{
		Email:     "mira@example.com",
		Password:  "  spaces  matter  ",
		FirstName: "Mira",
	}

	if err := request.validate(); err != nil {
		t.Fatalf("validate() = %v", err)
	}

	if !strings.HasPrefix(request.Password, "  ") {
		t.Errorf("password = %q, want it untouched", request.Password)
	}
}
