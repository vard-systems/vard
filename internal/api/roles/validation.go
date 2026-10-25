package roles

import (
	"net/http"

	"vard/internal/api/kit/respond"
	"vard/internal/api/kit/validate"
	"vard/internal/model"
)

var (
	scopeInvalid       = respond.Define(http.StatusBadRequest, "user_role_scope_invalid", respond.Admin)
	applicationUnknown = respond.Define(http.StatusBadRequest, "user_role_application_unknown", respond.Admin)
	roleNameTaken      = respond.Define(http.StatusConflict, "user_role_name_taken", respond.Admin)
)

func init() {
	validate.Register("rolename", model.RoleNamePattern.MatchString)
}

func (r *roleRequest) validate() error {
	r.clean()

	return validate.Struct(r)
}
