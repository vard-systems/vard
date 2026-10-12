package adminroles

import (
	"net/http"

	"loginer/internal/api/respond"
	"loginer/internal/api/validate"
	"loginer/internal/model"
)

func init() {
	validate.Register("adminrole", model.RoleNamePattern.MatchString)
}

func (r *roleRequest) applyTo(role *model.AdminRole) error {
	r.clean()

	if err := validate.Struct(r); err != nil {
		return err
	}

	if r.Name == model.RoleSuperAdmin {
		return respond.Fault{Status: http.StatusConflict, Message: "super_admin is built in: choose another name"}
	}

	asked := make(map[string]bool, len(r.Permissions))
	for _, name := range r.Permissions {
		if !model.IsAdminPermission(name) {
			return respond.Fault{
				Status:  http.StatusBadRequest,
				Message: "permissions: " + name + " is not a permission",
			}
		}
		asked[name] = true
	}

	permissions := []string{}
	for _, name := range model.AdminPermissionNames() {
		if asked[name] {
			permissions = append(permissions, name)
		}
	}

	role.Name = r.Name
	role.Description = r.Description
	role.Permissions = permissions

	return nil
}
