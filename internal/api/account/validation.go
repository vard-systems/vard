package account

import (
	"strings"
	"unicode/utf8"

	"loginer/internal/api/kit/validate"
	"loginer/internal/oidc"
)

func (r *loginRequest) validate() error {
	r.Email = strings.TrimSpace(r.Email)
	return validate.Struct(r)
}

func (r *registerRequest) validate() error {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)

	if err := validate.Struct(r); err != nil {
		return err
	}

	return checkPassword(r.Password)
}

func (r *codeRequest) validate() error {
	r.Code = strings.TrimSpace(r.Code)
	return validate.Struct(r)
}

func (r *resendRequest) validate() error {
	return validate.Struct(r)
}

func (r *emailRequest) validate() error {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	return validate.Struct(r)
}

func (r *forgotRequest) validate() error {
	r.Email = strings.TrimSpace(r.Email)
	return validate.Struct(r)
}

func (r *resetRequest) validate() error {
	if err := validate.Struct(r); err != nil {
		return err
	}

	return checkPassword(r.Password)
}

func (r *profileRequest) validate() error {
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	return validate.Struct(r)
}

func (r *passwordRequest) validate() error {
	if err := validate.Struct(r); err != nil {
		return err
	}

	return checkPassword(r.NewPassword)
}

// checkPassword enforces length only; over 72 bytes is refused since bcrypt
// would silently truncate.
func checkPassword(password string) error {
	switch {
	case utf8.RuneCountInString(password) < oidc.MinPasswordLength:
		return passwordTooShort.With("min", oidc.MinPasswordLength)
	case len(password) > 72:
		return provided[oidc.ErrPasswordTooLong.Code].Fault(nil)
	}

	return nil
}
