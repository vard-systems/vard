package admins

import (
	"fmt"
	"unicode/utf8"

	"loginer/internal/api/respond"
	"loginer/internal/api/validate"
	"loginer/internal/model"
)

const minPasswordLength = model.MinAdminPasswordLength

func (r *adminRequest) validate(creating bool) error {
	r.clean()

	if err := validate.Struct(r); err != nil {
		return err
	}

	switch {
	case r.Password == "" && creating:
		return respond.Invalid("password is required")
	case r.Password != "" && utf8.RuneCountInString(r.Password) < minPasswordLength:
		return respond.Invalid(fmt.Sprintf("password must be at least %d characters", minPasswordLength))
	}

	return nil
}
