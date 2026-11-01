// Package signedin is what the packages answering a person about their own
// account share: the session the request is signed in with, who is calling
// as the provider sees them, and how the provider's refusals are answered.
package signedin

import (
	"errors"
	"log/slog"
	"net/http"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"vard/internal/application/oidc"
	"vard/internal/domain/model"
	"vard/internal/interfaces/api/kit/respond"
	"vard/internal/interfaces/api/kit/session"
)

const sessionKey = "user_session"

var passwordTooShort = respond.Define(http.StatusBadRequest, "password_too_short", respond.Public)

// CheckPassword enforces length only: what else a password must be is a rule
// that pushes people to worse ones.
func CheckPassword(password string) error {
	switch {
	case utf8.RuneCountInString(password) < model.MinUserPasswordLength:
		return passwordTooShort.With("min", model.MinUserPasswordLength)
	case utf8.RuneCountInString(password) > model.MaxPasswordLength:
		return Problem(oidc.ErrPasswordTooLong).Fault(nil)
	}

	return nil
}

// Set keeps the request's signed-in session for the handlers after it.
func Set(c *gin.Context, session *oidc.Session) {
	c.Set(sessionKey, session)
}

// Session is the request's signed-in session, as a route guard set it.
func Session(c *gin.Context) *oidc.Session {
	value, _ := c.Get(sessionKey)
	current, _ := value.(*oidc.Session)
	return current
}

// Client is who is calling, as the provider records it.
func Client(c *gin.Context) oidc.Client {
	return oidc.Client{IP: c.ClientIP(), UserAgent: c.Request.UserAgent(), Language: session.Language(c)}
}

// Problem is the API problem a provider problem is answered with.
func Problem(refused *oidc.Problem) respond.Problem {
	return problems[refused.Code]
}

// Fail answers an error from the provider: a refusal as its problem, single
// sign-on required with the connection to use, anything else as a failure.
func Fail(c *gin.Context, log *slog.Logger, err error, note string) {
	if sso, ok := errors.AsType[*oidc.SSORequired](err); ok {
		respond.Fail(c, problems[oidc.ErrSSORequired.Code], "slug", sso.Slug, "name", sso.Name)
		return
	}

	if refused, ok := errors.AsType[*oidc.Problem](err); ok {
		if problem, ok := problems[refused.Code]; ok {
			respond.Write(c, problem.Fault(nil))
			return
		}
	}

	respond.Failure(c, log, err, note)
}

// problems maps every provider problem (oidc.Problems) to a public API problem
// by code; only the status differs, so new problems need just a catalog
// sentence. Defined once here, since a code can only be defined once.
var problems = func() map[string]respond.Problem {
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
		oidc.ErrBrowserRequired.Code:       http.StatusForbidden,
		oidc.ErrSecondFactorExpired.Code:   http.StatusGone,
		oidc.ErrTwoStepNotOffered.Code:     http.StatusForbidden,
		oidc.ErrAuthenticatorExists.Code:   http.StatusConflict,
		oidc.ErrPasskeyLimit.Code:          http.StatusConflict,
		oidc.ErrDeviceCodeInvalid.Code:     http.StatusNotFound,
		oidc.ErrDeviceRefused.Code:         http.StatusForbidden,
		oidc.ErrSignInAgain.Code:           http.StatusForbidden,
		oidc.ErrSecondFactorRequired.Code:  http.StatusForbidden,
		oidc.ErrStepUpRequired.Code:        http.StatusForbidden,
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
