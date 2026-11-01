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

// Problem is why a user's sign-in or change was refused. It carries only its
// code: what the user reads is `error.<code>` in i18n/id, and the HTTP status
// is chosen in api/account (`provided`), since this package knows
// nothing of HTTP.
type Problem struct {
	Code string
}

func (p *Problem) Error() string {
	return p.Code
}

var problems []*Problem

func problem(code string) *Problem {
	p := &Problem{Code: code}
	problems = append(problems, p)

	return p
}

func Problems() []*Problem {
	return problems
}

// The errors of signing a user in. They are what the sign-in pages show, so
// they say what went wrong without saying which accounts exist.
var (
	ErrInvalidCredentials    = problem("invalid_credentials")
	ErrRequestExpired        = problem("request_expired")
	ErrRegistrationClosed    = problem("registration_closed")
	ErrEmailTaken            = problem("email_taken")
	ErrResetInvalid          = problem("reset_invalid")
	ErrTermsRequired         = problem("terms_required")
	ErrDetailsRequired       = problem("details_required")
	ErrPasswordTooLong       = problem("password_too_long")
	ErrPasswordNotOffered    = problem("password_not_offered")
	ErrEmailNotVerified      = problem("email_not_verified")
	ErrVerificationInvalid   = problem("verification_invalid")
	ErrSignInClosed          = problem("sign_in_closed")
	ErrEmailChangeNotOffered = problem("email_change_not_offered")

	ErrCodeInvalid      = problem("code_invalid")
	ErrCodeExpired      = problem("code_expired")
	ErrCodeAttemptsUsed = problem("code_attempts_used")
	ErrCodeTooSoon      = problem("code_too_soon")

	ErrSocialUnknown            = problem("social_unknown")
	ErrSocialExpired            = problem("social_expired")
	ErrSocialUpstream           = problem("social_upstream")
	ErrSocialNoEmail            = problem("social_no_email")
	ErrSocialLinkRefused        = problem("social_link_refused")
	ErrSocialEmailUnverified    = problem("social_email_unverified")
	ErrSocialRegistrationClosed = problem("social_registration_closed")
	ErrSocialSSORequired        = problem("social_sso_required")
	ErrSocialBlocked            = problem("social_blocked")
	ErrSocialNotOffered         = problem("social_not_offered")
)
