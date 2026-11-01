package account

import "encoding/json"

// loginRequest is the body of POST /account/login. Request is the handle of
// the sign-in under way, when there is one.
type loginRequest struct {
	Request  string `json:"request" validate:"max=64"`
	Email    string `json:"email" validate:"required,max=255"`
	Password string `json:"password" validate:"required,max=128"`
	// Remember is the "stay signed in" box. A flow that does not offer it
	// ignores whatever is sent.
	Remember bool `json:"remember"`
}

type registerRequest struct {
	Request     string `json:"request" validate:"required,max=64"`
	Email       string `json:"email" validate:"required,email,max=255"`
	Password    string `json:"password" validate:"required,max=128"`
	FirstName   string `json:"first_name" validate:"max=100"`
	LastName    string `json:"last_name" validate:"max=100"`
	AcceptTerms bool   `json:"accept_terms"`
	Remember    bool   `json:"remember"`
}

// codeRequest is the body of POST /account/login/code: the handle the sign-in
// was held under, and the code from the message.
type codeRequest struct {
	Handle string `json:"handle" validate:"required,max=64"`
	Code   string `json:"code" validate:"required,max=16"`
}

type resendRequest struct {
	Handle string `json:"handle" validate:"required,max=64"`
}

// discoverRequest is the address "Sign in with SSO" looks a connection up by.
type discoverRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
}

type forgotRequest struct {
	Request string `json:"request" validate:"max=64"`
	Email   string `json:"email" validate:"required,email,max=255"`
	// Language is the one the page was shown in, which the email is written
	// in. One that is not offered, or none, is the default.
	Language string `json:"language" validate:"max=16"`
}

// resetRequest is the body of POST /account/reset-password.
// verifyRequest is the body of POST /account/verify-email.
type verifyRequest struct {
	Token string `json:"token" validate:"required,max=64"`
}

type resetRequest struct {
	Token    string `json:"token" validate:"required,max=64"`
	Password string `json:"password" validate:"required,max=128"`
}

// logoutRequest is the body of POST /account/logout, all of it optional: the
// confirmation page passes on what the application's logout request named.
type logoutRequest struct {
	ClientID              string `json:"client_id" validate:"max=255"`
	PostLogoutRedirectURI string `json:"post_logout_redirect_uri" validate:"max=2048"`
	State                 string `json:"state" validate:"max=512"`
}

// secondFactorRequest answers a waiting sign-in: a code (from the
// authenticator, or a recovery code), or a passkey's response as the browser
// gave it.
type secondFactorRequest struct {
	Code    string          `json:"code" validate:"max=32"`
	Passkey json.RawMessage `json:"passkey"`
}

// passkeyStartRequest starts a sign-in with a passkey alone, for the sign-in
// under way when there is one.
type passkeyStartRequest struct {
	Request  string `json:"request" validate:"max=64"`
	Remember bool   `json:"remember"`
}

// passkeyFinishRequest is the browser's answer, with the handle it was given.
type passkeyFinishRequest struct {
	Handle   string          `json:"handle" validate:"required,max=64"`
	Response json.RawMessage `json:"response"`
}

// enrolAuthenticatorRequest is a code from the authenticator being set up.
type enrolAuthenticatorRequest struct {
	Code string `json:"code" validate:"required,max=16"`
}

// enrolPasskeyRequest is the passkey the browser made, under the name the
// person gave it.
type enrolPasskeyRequest struct {
	Label    string          `json:"label" validate:"max=100"`
	Response json.RawMessage `json:"response"`
}
