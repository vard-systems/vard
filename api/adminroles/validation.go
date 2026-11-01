package adminroles

import (
	"net/http"

	"vard/internal/domain/model"
	"vard/internal/interfaces/api/kit/respond"
	"vard/internal/interfaces/api/kit/validate"
)

var (
	roleAbsent        = respond.Define(http.StatusNotFound, "admin_role_not_found", respond.Admin)
	superAdminFixed   = respond.Define(http.StatusForbidden, "admin_role_built_in", respond.Admin)
	roleNameTaken     = respond.Define(http.StatusConflict, "admin_role_name_taken", respond.Admin)
	nameReserved      = respond.Define(http.StatusConflict, "admin_role_name_reserved", respond.Admin)
	permissionUnknown = respond.Define(http.StatusBadRequest, "admin_role_permission_unknown", respond.Admin)
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
		return nameReserved.Fault(nil)
	}

	asked := make(map[string]bool, len(r.Permissions))
	for _, name := range r.Permissions {
		if !model.IsAdminPermission(name) {
			return permissionUnknown.With("permission", name)
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
