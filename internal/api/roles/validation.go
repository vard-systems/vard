package roles

import (
	"loginer/internal/api/validate"
	"loginer/internal/model"
)

func init() {
	validate.Register("rolename", model.RoleNamePattern.MatchString)
}

func (r *roleRequest) validate() error {
	r.clean()

	return validate.Struct(r)
}
