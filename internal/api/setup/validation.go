package setup

import (
	"strings"

	"loginer/internal/api/validate"
)

func (r *setupRequest) validate() error {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)

	return validate.Struct(r)
}
