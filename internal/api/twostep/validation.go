package twostep

import (
	"strings"

	"vard/internal/api/kit/respond"
	"vard/internal/api/kit/validate"
)

// maxPasskeyResponse is far more than any authenticator's answer, and keeps a
// body that is not one from being parsed at length.
const maxPasskeyResponse = 16 << 10

func (r *confirmAuthenticatorRequest) validate() error {
	r.Code = strings.ReplaceAll(strings.TrimSpace(r.Code), " ", "")
	return validate.Struct(r)
}

func (r *finishPasskeyRequest) validate() error {
	r.Label = strings.TrimSpace(r.Label)
	if len(r.Response) > maxPasskeyResponse {
		return respond.InvalidBody.With()
	}
	return validate.Struct(r)
}

func (r *presenceRequest) validate() error {
	return validate.Struct(r)
}
