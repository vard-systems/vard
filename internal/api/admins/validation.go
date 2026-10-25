package admins

import (
	"net/http"
	"unicode/utf8"

	"vard/internal/api/kit/respond"
	"vard/internal/api/kit/validate"
	"vard/internal/model"
)

var (
	ownMFA      = respond.Define(http.StatusConflict, "admin_own_mfa", respond.Admin)
	ownAccount  = respond.Define(http.StatusConflict, "admin_own_account", respond.Admin)
	adminAbsent = respond.Define(http.StatusNotFound, "admin_not_found", respond.Admin)
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
