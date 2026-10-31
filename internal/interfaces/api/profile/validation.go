package profile

import (
	"strings"

	"vard/internal/interfaces/api/kit/signedin"
	"vard/internal/interfaces/api/kit/validate"
)

func (r *profileRequest) validate() error {
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	return validate.Struct(r)
}

func (r *emailRequest) validate() error {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	return validate.Struct(r)
}

func (r *passwordRequest) validate() error {
	if err := validate.Struct(r); err != nil {
		return err
	}

	return signedin.CheckPassword(r.NewPassword)
}
