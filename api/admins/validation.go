package admins

import (
	"net/http"
	"unicode/utf8"

	"vard/api/kit/respond"
	"vard/api/kit/validate"
	"vard/core/model"
)

var (
	ownMFA      = respond.Define(http.StatusConflict, "admin_own_mfa", respond.Admin)
	ownAccount  = respond.Define(http.StatusConflict, "admin_own_account", respond.Admin)
	adminAbsent = respond.Define(http.StatusNotFound, "admin_not_found", respond.Admin)
	ownAccess   = respond.Define(http.StatusConflict, "admin_own_access", respond.Admin)

	assignedRoleUnknown        = respond.Define(http.StatusBadRequest, "admin_assignment_role_unknown", respond.Admin)
	assignedApplicationUnknown = respond.Define(http.StatusBadRequest, "admin_assignment_application_unknown", respond.Admin)
	superAdminScoped           = respond.Define(http.StatusBadRequest, "admin_assignment_super_admin_scoped", respond.Admin)

	idleTimeoutOutOfRange = respond.Define(http.StatusBadRequest, "admin_idle_timeout_out_of_range", respond.Admin)
)

const minPasswordLength = model.MinAdminPasswordLength

func (r *adminRequest) validate(creating bool) error {
	r.clean()

	if err := validate.Struct(r); err != nil {
		return err
	}

	switch {
	case r.Password == "" && creating:
		return respond.AdminPasswordRequired.With()
	case r.Password != "" && utf8.RuneCountInString(r.Password) < minPasswordLength:
		return respond.AdminPasswordTooShort.With("min", minPasswordLength)
	}

	return nil
}
