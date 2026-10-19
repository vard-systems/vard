package roles

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"loginer/internal/api/kit/respond"
)

// TestRoleRequestValidate covers the name a role is checked for by, which is
// why it is held to one shape.
func TestRoleRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request roleRequest
		want    string // the message, or "" when the request is fine
	}{
		{name: "a role", request: roleRequest{Name: "editor"}},
		{name: "spaces and capitals are tidied away", request: roleRequest{Name: "  Editor "}},
		{name: "no name", request: roleRequest{}, want: "name is required."},
		{
			name:    "a space in the name",
			request: roleRequest{Name: "senior editor"},
			want:    "name must start with a letter and hold only lower case letters, numbers, dashes and underscores.",
		},
		{
			name:    "a description too long",
			request: roleRequest{Name: "editor", Description: strings.Repeat("d", 256)},
			want:    "description must be at most 255 characters.",
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
				if request.Name != "editor" {
					t.Errorf("name = %q, want it trimmed and lowered", request.Name)
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
