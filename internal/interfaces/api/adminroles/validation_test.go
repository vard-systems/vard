package adminroles

import (
	"errors"
	"reflect"
	"testing"

	"vard/internal/domain/model"
	"vard/internal/interfaces/api/kit/respond"
)

func TestRoleRequestApplyTo(t *testing.T) {
	tests := []struct {
		name    string
		request roleRequest
		want    string   // the message, or "" when the request is fine
		granted []string // what the role should grant when it is
	}{
		{
			name: "a role, its permissions put in catalog order once each",
			request: roleRequest{
				Name: " Moderator ",
				Permissions: []string{
					model.PermUsersWrite, model.PermActivityRead, model.PermUsersWrite,
				},
			},
			granted: []string{model.PermActivityRead, model.PermUsersWrite},
		},
		{
			name:    "a role that grants nothing",
			request: roleRequest{Name: "viewer"},
			granted: []string{},
		},
		{name: "no name", request: roleRequest{}, want: "name is required."},
		{
			name:    "a name with a space",
			request: roleRequest{Name: "head moderator"},
			want:    "name must start with a letter and hold only lower case letters, numbers, dashes and underscores.",
		},
		{
			name:    "the built-in name",
			request: roleRequest{Name: "super_admin"},
			want:    "super_admin is built in. Choose another name.",
		},
		{
			name:    "a permission the catalog does not have",
			request: roleRequest{Name: "moderator", Permissions: []string{"admins.manage"}},
			want:    "admins.manage is not a permission.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request
			var role model.AdminRole

			err := request.applyTo(&role)

			if tt.want == "" {
				if err != nil {
					t.Fatalf("applyTo() = %v, want nothing", err)
				}
				if !reflect.DeepEqual(role.Permissions, tt.granted) {
					t.Errorf("permissions = %v, want %v", role.Permissions, tt.granted)
				}
				return
			}

			var fault respond.Fault
			if !errors.As(err, &fault) {
				t.Fatalf("applyTo() = %v, want a respond.Fault", err)
			}
			if fault.Message != tt.want {
				t.Errorf("message = %q, want %q", fault.Message, tt.want)
			}
		})
	}
}
