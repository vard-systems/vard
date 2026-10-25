package setup

import (
	"net/http"
	"strings"

	"vard/internal/api/kit/respond"
	"vard/internal/api/kit/validate"
)

var (
	alreadySetUp = respond.Define(http.StatusConflict, "setup_done", respond.Admin)
)

func (r *setupRequest) validate() error {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)

	return validate.Struct(r)
}
