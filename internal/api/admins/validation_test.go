package admins

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"vard/internal/api/kit/respond"
)

// TestAdminRequestValidate covers the form an administrator is made and
// edited with. An admin account opens the panel, so its password keeps the
// same floor as the first super admin's.
func TestAdminRequestValidate(t *testing.T) {
	good := func() adminRequest {
		return adminRequest{
			Email:           "  Mira@Example.com ",
			FirstName:       " Mira ",
			Status:          "active",
			Password:        "long-enough-1",
			ConfirmPassword: "long-enough-1",
		}
	}

	tests := []struct {
		name     string
		creating bool
		change   func(*adminRequest)
		want     string // the message, or "" when the form is fine
	}{
		{name: "a new administrator", creating: true, change: func(*adminRequest) {}},
		{
			name:   "an update keeping the password",
			change: func(r *adminRequest) { r.Password, r.ConfirmPassword = "", "" },
		},
		{
			name:     "a new administrator without a password",
			creating: true,
			change:   func(r *adminRequest) { r.Password, r.ConfirmPassword = "", "" },
			want:     "A password is required.",
		},
		{
			name:   "a short password",
			change: func(r *adminRequest) { r.Password, r.ConfirmPassword = "short", "short" },
			want:   "The password must be at least 10 characters.",
		},
		{
			name:   "passwords that differ",
			change: func(r *adminRequest) { r.ConfirmPassword = "something-else" },
			want:   "confirm_password must match password.",
		},
		{
			name:   "no name",
			change: func(r *adminRequest) { r.FirstName = "  " },
			want:   "first_name is required.",
		},
		{
			name:   "a status that is not one",
			change: func(r *adminRequest) { r.Status = "invited" },
			want:   "status must be one of: active, suspended, disabled.",
		},
		{
			name:   "an address that is too long",
			change: func(r *adminRequest) { r.Email = strings.Repeat("m", 250) + "@example.com" },
			want:   "email must be at most 255 characters.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := good()
			tt.change(&request)

			err := request.validate(tt.creating)

			if tt.want == "" {
				if err != nil {
					t.Fatalf("validate() = %v, want nothing", err)
				}
				if request.Email != "mira@example.com" || request.FirstName != "Mira" {
					t.Errorf("email = %q, first name = %q, want them tidied", request.Email, request.FirstName)
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
