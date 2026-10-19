package mfa

import (
	"net/http"

	"loginer/internal/api/kit/respond"
)

var (
	codeWrong    = respond.Define(http.StatusBadRequest, "mfa_code_wrong", respond.Admin)
	notEnrolling = respond.Define(http.StatusBadRequest, "mfa_not_enrolling", respond.Admin)
	alreadyOff   = respond.Define(http.StatusBadRequest, "mfa_disabled", respond.Admin)
	alreadyOn    = respond.Define(http.StatusConflict, "mfa_enabled", respond.Admin)
	mustStayOn   = respond.Define(http.StatusConflict, "mfa_required", respond.Admin)
)
