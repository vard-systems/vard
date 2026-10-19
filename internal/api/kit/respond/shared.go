package respond

import "net/http"

// Problems more than one handler answers with. One that only a single package
// answers with is defined beside that package's handler instead.
var (
	ApplicationNotFound = Define(http.StatusNotFound, "application_not_found", Both)
	APINotFound         = Define(http.StatusNotFound, "api_not_found", Admin)
	UserNotFound        = Define(http.StatusNotFound, "user_not_found", Admin)
	UserRoleNotFound    = Define(http.StatusNotFound, "user_role_not_found", Admin)

	AdminEmailTaken       = Define(http.StatusConflict, "admin_email_taken", Admin)
	AdminPasswordRequired = Define(http.StatusBadRequest, "admin_password_required", Admin)
	AdminPasswordTooShort = Define(http.StatusBadRequest, "admin_password_too_short", Admin)
	AdminPasswordTooLong  = Define(http.StatusBadRequest, "admin_password_too_long", Admin)
)
