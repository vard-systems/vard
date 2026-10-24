package twostep

import "encoding/json"

type confirmAuthenticatorRequest struct {
	Code string `json:"code" validate:"required,max=16"`
}

type finishPasskeyRequest struct {
	Label    string          `json:"label" validate:"max=100"`
	Response json.RawMessage `json:"response"`
}

// presenceRequest carries the password that changes to how one signs in ask
// for, when the account has one.
// presenceRequest carries the password that changes to how one signs in ask
// for, when the account has one.
type presenceRequest struct {
	CurrentPassword string `json:"current_password" validate:"max=72"`
}
