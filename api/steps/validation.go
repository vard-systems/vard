package steps

import (
	"strings"

	"vard/internal/interfaces/api/kit/validate"
)

func (r *stepRequest) validate() error {
	r.Request = strings.TrimSpace(r.Request)
	return validate.Struct(r)
}

func (r *deviceRequest) validate() error {
	r.UserCode = strings.TrimSpace(r.UserCode)
	return validate.Struct(r)
}
