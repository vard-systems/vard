package adminauth

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"vard/internal/api/kit/respond"
	"vard/internal/api/kit/validate"
	"vard/internal/model"
)

var (
	wrongCredentials = respond.Define(http.StatusUnauthorized, "admin_wrong_credentials", respond.Admin)
	signInExpired    = respond.Define(http.StatusUnauthorized, "admin_sign_in_expired", respond.Admin)
	signInCodeWrong  = respond.Define(http.StatusUnauthorized, "admin_sign_in_code_wrong", respond.Admin)
)

func (r *loginRequest) validate() error {
	r.Username = strings.TrimSpace(r.Username)

	return validate.Struct(r)
}

func (r *profileRequest) validate() error {
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Email = model.NormalizeEmail(r.Email)
	r.AvatarURL = strings.TrimSpace(r.AvatarURL)

	if err := validate.Struct(r); err != nil {
		return err
	}

	// The picture is put in an img tag on every page of the panel, so it is
	// held to the rule every such link is: a full http(s) address, or none.
	if model.ValidLink("avatar_url", r.AvatarURL) != nil {
		return avatarInvalid.With()
	}

	return nil
}

func (r *passwordRequest) validate() error {
	if err := validate.Struct(r); err != nil {
		return err
	}

	if utf8.RuneCountInString(r.NewPassword) < model.MinAdminPasswordLength {
		return respond.AdminPasswordTooShort.With("min", model.MinAdminPasswordLength)
	}

	return nil
}
