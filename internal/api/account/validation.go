package account

import (
	"strings"

	"loginer/internal/api/kit/respond"
	"loginer/internal/api/kit/signedin"
	"loginer/internal/api/kit/validate"
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

	return signedin.CheckPassword(r.Password)
}

func (r *codeRequest) validate() error {
	r.Code = strings.TrimSpace(r.Code)
	return validate.Struct(r)
}

func (r *resendRequest) validate() error {
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

	return signedin.CheckPassword(r.Password)
}

// maxPasskeyResponse is far more than any authenticator's answer, and keeps a
// body that is not one from being parsed at length.
const maxPasskeyResponse = 16 << 10

func (r *secondFactorRequest) validate() error {
	r.Code = strings.TrimSpace(r.Code)
	if len(r.Passkey) > maxPasskeyResponse {
		return respond.InvalidBody.With()
	}
	return validate.Struct(r)
}

func (r *passkeyStartRequest) validate() error {
	return validate.Struct(r)
}

func (r *passkeyFinishRequest) validate() error {
	if len(r.Response) == 0 || len(r.Response) > maxPasskeyResponse {
		return respond.InvalidBody.With()
	}
	return validate.Struct(r)
}
