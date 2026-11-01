package mfa

import (
	"net/http"

	"vard/api/kit/respond"
)

var (
	codeWrong    = respond.Define(http.StatusBadRequest, "mfa_code_wrong", respond.Admin)
	notEnrolling = respond.Define(http.StatusBadRequest, "mfa_not_enrolling", respond.Admin)
	alreadyOff   = respond.Define(http.StatusBadRequest, "mfa_disabled", respond.Admin)
	alreadyOn    = respond.Define(http.StatusConflict, "mfa_enabled", respond.Admin)
	mustStayOn   = respond.Define(http.StatusConflict, "mfa_required", respond.Admin)

	passkeyLimit   = respond.Define(http.StatusConflict, "mfa_passkey_limit", respond.Admin)
	passkeyRefused = respond.Define(http.StatusBadRequest, "mfa_passkey_refused", respond.Admin)
	passkeyAbsent  = respond.Define(http.StatusNotFound, "mfa_passkey_not_found", respond.Admin)
)
