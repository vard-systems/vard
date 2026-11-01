package account

import (
	"net/http"
	"time"

	"vard/internal/application/oidc"
	"vard/internal/interfaces/api/kit/respond"
)

var (
	noSSOConnection = respond.Define(http.StatusNotFound, "no_sso_connection", respond.Public)
)

// requestResponse is a sign-in under way, as the sign-in page shows it.
type requestResponse struct {
	Application oidc.PublicApplication `json:"application"`
	LoginHint   string                 `json:"login_hint"`
	ExpiresAt   time.Time              `json:"expires_at"`
}

func newRequestResponse(pending *oidc.PendingRequest) requestResponse {
	return requestResponse{
		Application: oidc.NewPublicApplication(pending.Application),
		LoginHint:   pending.LoginHint,
		ExpiresAt:   pending.ExpiresAt,
	}
}

// signedInResponse says what the page does next: follow RedirectTo, choose a
// new password with ResetToken (temporary password), or ask for the emailed
// Code. With none, the user is signed in and there is nowhere to go.
type signedInResponse struct {
	RedirectTo             string              `json:"redirect_to,omitempty"`
	PasswordChangeRequired bool                `json:"password_change_required,omitempty"`
	ResetToken             string              `json:"reset_token,omitempty"`
	Code                   *oidc.CodeChallenge `json:"code,omitempty"`
	// SecondFactor says the sign-in waits for the user's second factor, on
	// the second-factor page; the cookie set with it names the sign-in.
	SecondFactor bool `json:"second_factor,omitempty"`
	// RecoveryCodes come once, with a first factor set up while signing in.
	RecoveryCodes []string `json:"recovery_codes,omitempty"`
}
