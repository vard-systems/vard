package roles

import (
	"net/http"

	"vard/api/kit/respond"
	"vard/api/kit/validate"
	"vard/core/model"
)

var (
	scopeInvalid       = respond.Define(http.StatusBadRequest, "user_role_scope_invalid", respond.Admin)
	applicationUnknown = respond.Define(http.StatusBadRequest, "user_role_application_unknown", respond.Admin)
	roleNameTaken      = respond.Define(http.StatusConflict, "user_role_name_taken", respond.Admin)
	includedUnknown    = respond.Define(http.StatusBadRequest, "user_role_inherits_unknown", respond.Admin)
	includeForbidden   = respond.Define(http.StatusForbidden, "user_role_inherits_forbidden", respond.Admin)
	includedForeign    = respond.Define(http.StatusBadRequest, "user_role_inherits_foreign", respond.Admin)
	includeCycle       = respond.Define(http.StatusBadRequest, "user_role_inherits_cycle", respond.Admin)
	scopesForbidden    = respond.Define(http.StatusForbidden, "user_role_api_scopes_forbidden", respond.Admin)
	scopeUnknown       = respond.Define(http.StatusBadRequest, "user_role_api_scope_unknown", respond.Admin)
)

func init() {
	validate.Register("rolename", model.RoleNamePattern.MatchString)
}

func (r *roleRequest) validate() error {
	r.clean()

	return validate.Struct(r)
}
