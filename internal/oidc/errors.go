package oidc

import (
	"net/http"
)

const (
	ErrInvalidRequest          = "invalid_request"
	ErrInvalidClient           = "invalid_client"
	ErrInvalidGrant            = "invalid_grant"
	ErrUnauthorizedClient      = "unauthorized_client"
	ErrUnsupportedGrantType    = "unsupported_grant_type"
	ErrUnsupportedResponseType = "unsupported_response_type"
	ErrInvalidScope            = "invalid_scope"
	ErrAccessDenied            = "access_denied"
	ErrServerError             = "server_error"
	ErrLoginRequired           = "login_required"
	ErrInvalidToken            = "invalid_token"
	ErrInsufficientScope       = "insufficient_scope"
)

type Error struct {
	Code        string
	Description string
	Status      int
}

func (e *Error) Error() string {
	return e.Code + ": " + e.Description
}

func oauthError(code, description string) *Error {
	status := http.StatusBadRequest
	switch code {
	case ErrInvalidClient, ErrInvalidToken:
		status = http.StatusUnauthorized
	case ErrInsufficientScope:
		status = http.StatusForbidden
	}

	return &Error{Code: code, Description: description, Status: status}
}

type Problem struct {
	Code    string
	message string
}

func (p *Problem) Error() string {
	return p.message
}

var problems []*Problem

func problem(code, message string) *Problem {
	p := &Problem{Code: code, message: message}
	problems = append(problems, p)

	return p
}

func Problems() []*Problem {
	return problems
}

// The errors of signing a user in. They are what the sign-in pages show, so
// they say what went wrong without saying which accounts exist.
var (
	ErrInvalidCredentials    = problem("invalid_credentials", "wrong email or password")
	ErrRequestExpired        = problem("request_expired", "the sign-in handle has expired")
	ErrRegistrationClosed    = problem("registration_closed", "the application does not allow registration")
	ErrEmailTaken            = problem("email_taken", "an account with this email already exists")
	ErrResetInvalid          = problem("reset_invalid", "the reset link has expired or been used")
	ErrTermsRequired         = problem("terms_required", "the terms were not accepted")
	ErrDetailsRequired       = problem("details_required", "accounts need fields the page cannot ask for")
	ErrPasswordTooLong       = problem("password_too_long", "the password is too long")
	ErrPasswordNotOffered    = problem("password_not_offered", "this sign-in does not take a password")
	ErrEmailNotVerified      = problem("email_not_verified", "the address is not verified; a link to verify it was sent")
	ErrVerificationInvalid   = problem("verification_invalid", "the verification link has expired or been used")
	ErrSignInClosed          = problem("sign_in_closed", "sign-ins are closed for this application")
	ErrEmailChangeNotOffered = problem("email_change_not_offered", "this sign-in does not allow changing the address")

	ErrCodeInvalid      = problem("code_invalid", "that code is not the one that was sent")
	ErrCodeExpired      = problem("code_expired", "the code has expired; sign in again")
	ErrCodeAttemptsUsed = problem("code_attempts_used", "too many wrong codes; sign in again")
	ErrCodeTooSoon      = problem("code_too_soon", "another code cannot be sent yet")

	ErrSocialUnknown            = problem("social_unknown", "the provider is not available")
	ErrSocialExpired            = problem("social_expired", "the social sign-in expired or was used")
	ErrSocialUpstream           = problem("social_upstream", "the provider did not complete the sign-in")
	ErrSocialNoEmail            = problem("social_no_email", "the provider shared no email address")
	ErrSocialLinkRefused        = problem("social_link_refused", "the address has an account the provider does not prove")
	ErrSocialRegistrationClosed = problem("social_registration_closed", "the provider cannot create accounts here")
	ErrSocialSSORequired        = problem("social_sso_required", "the address has to sign in through single sign-on")
	ErrSocialBlocked            = problem("social_blocked", "the account may not sign in")
	ErrSocialNotOffered         = problem("social_not_offered", "this sign-in does not offer other accounts")
)
