package setup

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"vard/internal/api/kit/respond"
	"vard/internal/api/kit/validate"
	"vard/internal/model"
)

var (
	alreadySetUp = respond.Define(http.StatusConflict, "setup_done", respond.Admin)
	// tokenRefused answers a missing, wrong, used or expired setup token alike.
	tokenRefused = respond.Define(http.StatusForbidden, "setup_token_invalid", respond.Admin)
)

func (r *setupRequest) validate() error {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)

	if err := validate.Struct(r); err != nil {
		return err
	}
	if utf8.RuneCountInString(r.Password) < model.MinAdminPasswordLength {
		return respond.AdminPasswordTooShort.With("min", model.MinAdminPasswordLength)
	}

	return nil
}
