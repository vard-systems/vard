package account

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"loginer/internal/api/kit/respond"
	"loginer/internal/model"
	"loginer/internal/oidc"
)

var (
	passwordTooShort = respond.Define(http.StatusBadRequest, "password_too_short", respond.Public)
	noSSOConnection  = respond.Define(http.StatusNotFound, "no_sso_connection", respond.Public)
)

// provided maps every provider problem (oidc.Problems) to a public API problem
// by code; only the status differs, so new problems need just a catalog
// sentence.
var provided = func() map[string]respond.Problem {
	statuses := map[string]int{
		oidc.ErrInvalidCredentials.Code:    http.StatusUnauthorized,
		oidc.ErrRequestExpired.Code:        http.StatusGone,
		oidc.ErrResetInvalid.Code:          http.StatusGone,
		oidc.ErrRegistrationClosed.Code:    http.StatusForbidden,
		oidc.ErrEmailTaken.Code:            http.StatusConflict,
		oidc.ErrNotYours.Code:              http.StatusNotFound,
		oidc.ErrSSORequired.Code:           http.StatusForbidden,
		oidc.ErrSSOUnknown.Code:            http.StatusNotFound,
		oidc.ErrPasswordNotOffered.Code:    http.StatusForbidden,
		oidc.ErrEmailNotVerified.Code:      http.StatusForbidden,
		oidc.ErrVerificationInvalid.Code:   http.StatusGone,
		oidc.ErrCodeExpired.Code:           http.StatusGone,
		oidc.ErrCodeAttemptsUsed.Code:      http.StatusGone,
		oidc.ErrCodeTooSoon.Code:           http.StatusTooManyRequests,
		oidc.ErrSignInClosed.Code:          http.StatusForbidden,
		oidc.ErrEmailChangeNotOffered.Code: http.StatusForbidden,
	}

	out := map[string]respond.Problem{}
	for _, refused := range oidc.Problems() {
		status, ok := statuses[refused.Code]
		if !ok {
			status = http.StatusBadRequest
		}

		out[refused.Code] = respond.Define(status, refused.Code, respond.Public)
	}

	return out
}()

// requestResponse is a sign-in under way, as the sign-in page shows it.
type requestResponse struct {
	Application oidc.PublicApplication `json:"application"`
	LoginHint   string                 `json:"login_hint"`
	ExpiresAt   time.Time              `json:"expires_at"`
}

func newRequestResponse(pending *oidc.PendingRequest) requestResponse {
	return requestResponse{
		Application: oidc.Public(pending.Application),
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
}

// userResponse is built by hand so new columns, roles and organisation fields
// are never published to the user by accident.
type userResponse struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLoginAt   *time.Time `json:"last_login_at"`
}

func newUserResponse(user *model.User) userResponse {
	return userResponse{
		ID:            user.ID,
		Email:         user.Email,
		EmailVerified: user.IsEmailVerified,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		CreatedAt:     user.CreatedAt,
		LastLoginAt:   user.LastLoginAt,
	}
}
