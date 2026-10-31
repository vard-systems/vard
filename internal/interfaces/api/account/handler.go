// Package account is the API the sign-in pages call: signing in and up, codes,
// password resets, and each user's own profile, sessions and connected
// applications.
package account

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"

	"vard/internal/application/oidc"
	"vard/internal/brand"
	"vard/internal/domain/model"
	"vard/internal/infrastructure/store"
	"vard/internal/interfaces/api/kit/respond"
	"vard/internal/interfaces/api/kit/session"
	"vard/internal/interfaces/api/kit/signedin"
	"vard/internal/interfaces/api/kit/validate"
)

type Handler struct {
	provider *oidc.Service
	log      *slog.Logger
	secure   bool
}

// New returns a Handler. `secure` sets the Secure flag on the session cookie.
func New(provider *oidc.Service, log *slog.Logger, secure bool) *Handler {
	return &Handler{provider: provider, log: log, secure: secure}
}

// Request describes a sign-in under way: the application it is for, as its
// sign-in page shows it.
func (h *Handler) Request(c *gin.Context) {
	pending, err := h.provider.Pending(c.Request.Context(), c.Param("handle"))
	if err != nil {
		h.fail(c, err, "loading a sign-in request failed")
		return
	}

	c.JSON(http.StatusOK, newRequestResponse(pending))
}

// Application describes an application by its client id, for the signed-out
// page to offer a way back to it.
func (h *Handler) Application(c *gin.Context) {
	app, err := h.provider.ApplicationByClientID(c.Request.Context(), c.Param("client_id"))
	if errors.Is(err, store.ErrNotFound) {
		respond.Fail(c, respond.ApplicationNotFound)
		return
	}
	if err != nil {
		h.fail(c, err, "loading an application failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"application": app})
}

// Organization describes the organisation the sign-in pages speak for. Public,
// so it needs no session.
func (h *Handler) Organization(c *gin.Context) {
	organization, err := h.provider.Organization(c.Request.Context())
	if err != nil {
		h.fail(c, err, "loading the organization failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"organization": organization})
}

// OrganizationLogo sends the organisation's app logo to anyone: the sign-in
// pages show it before anybody signs in.
func (h *Handler) OrganizationLogo(c *gin.Context) {
	logo, err := h.provider.OrganizationLogo(c.Request.Context())
	if errors.Is(err, store.ErrNotFound) {
		respond.Fail(c, respond.OrganizationLogoNotFound)
		return
	}
	if err != nil {
		h.fail(c, err, "loading the organization logo failed")
		return
	}

	respond.Image(c, logo.ContentType, logo.Data, logo.Version, false)
}

// LoginOptions returns the options of the login flow for the sign-in handle
// `request`, or the default flow.
func (h *Handler) LoginOptions(c *gin.Context) {
	options, err := h.provider.LoginOptions(c.Request.Context(), c.Query("request"))
	if err != nil {
		h.fail(c, err, "loading the login options failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"login": options})
}

// Languages says which languages these pages may be shown in, and which one
// somebody gets before they have chosen.
func (h *Handler) Languages(c *gin.Context) {
	languages, fallback, err := h.provider.Languages(c.Request.Context())
	if err != nil {
		h.fail(c, err, "listing languages failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"languages": languages, "default": fallback})
}

// LanguageText returns the sign-in pages' text in one offered language with
// every key filled in.
func (h *Handler) LanguageText(c *gin.Context) {
	language, messages, err := h.provider.LanguageText(c.Request.Context(), c.Param("code"))
	if errors.Is(err, store.ErrNotFound) {
		respond.Fail(c, respond.LanguageNotFound)
		return
	}
	if err != nil {
		h.fail(c, err, "loading a language failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"language": language, "messages": messages})
}

// SocialProviders lists the sign-in buttons, with only what a button needs.
func (h *Handler) SocialProviders(c *gin.Context) {
	providers, err := h.provider.SocialButtons(c.Request.Context())
	if err != nil {
		h.fail(c, err, "listing social providers failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

// SSOButtons lists the organisations' identity providers the sign-in page
// offers a button for.
func (h *Handler) SSOButtons(c *gin.Context) {
	buttons, available, err := h.provider.SSOButtons(c.Request.Context())
	if err != nil {
		h.fail(c, err, "listing single sign-on connections failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"connections": buttons, "available": available})
}

// DiscoverSSO says which connection an address signs in through, for "Sign in
// with SSO": the one that owns its domain, or no_sso_connection.
func (h *Handler) DiscoverSSO(c *gin.Context) {
	var req discoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := validate.Struct(req); err != nil {
		respond.Failure(c, h.log, err, "validating a single sign-on lookup failed")
		return
	}

	connection, enforced, err := h.provider.DiscoverSSO(c.Request.Context(), req.Email)
	if errors.Is(err, store.ErrNotFound) {
		respond.Fail(c, noSSOConnection)
		return
	}
	if err != nil {
		h.fail(c, err, "finding a single sign-on connection failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"connection": connection, "enforced": enforced})
}

// Login signs a user in and, for a sign-in under way, says where to go next.
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a sign-in failed")
		return
	}

	result, err := h.provider.SignIn(c.Request.Context(), req.Email, req.Password, req.Request, req.Remember, signedin.Client(c))
	if err != nil {
		h.fail(c, err, "signing a user in failed")
		return
	}

	h.signedIn(c, req.Request, result)
}

// Register creates an account for a sign-in under way, and signs it in.
func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a registration failed")
		return
	}

	result, err := h.provider.Register(c.Request.Context(), oidc.Registration{
		Request:       req.Request,
		Email:         req.Email,
		Password:      req.Password,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		AcceptedTerms: req.AcceptTerms,
		Remember:      req.Remember,
	}, signedin.Client(c))
	if err != nil {
		h.fail(c, err, "registering a user failed")
		return
	}

	h.signedIn(c, req.Request, result)
}

func (h *Handler) signedIn(c *gin.Context, request string, result *oidc.SignInResult) {
	if result.ResetToken != "" {
		c.JSON(http.StatusOK, signedInResponse{PasswordChangeRequired: true, ResetToken: result.ResetToken})
		return
	}

	// Held for an emailed code: no cookie is set and no session exists yet,
	// so nothing but the code finishes this sign-in.
	if result.Code != nil {
		c.JSON(http.StatusOK, signedInResponse{Code: result.Code})
		return
	}

	// Held for the user's second factor: the cookie names the sign-in, and
	// the second-factor page finishes it.
	if result.SecondFactorHandle != "" {
		session.SetSecondFactor(c, result.SecondFactorHandle, time.Now().Add(model.PendingSignInLifetime), h.secure)
		c.JSON(http.StatusOK, signedInResponse{SecondFactor: true})
		return
	}

	session.SetUser(c, result.Token, result.Session.Record.ExpiresAt, result.Remember, h.secure)

	if request == "" {
		c.JSON(http.StatusOK, signedInResponse{})
		return
	}

	location, err := h.provider.Continue(c.Request.Context(), request, result.Session)
	if err != nil {
		h.fail(c, err, "continuing a sign-in failed")
		return
	}

	c.JSON(http.StatusOK, signedInResponse{RedirectTo: location})
}

// Code finishes a sign-in that was waiting for an emailed code, and answers
// as the sign-in itself does: a cookie, and where to go next.
func (h *Handler) Code(c *gin.Context) {
	var req codeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a one-time code failed")
		return
	}

	result, err := h.provider.SubmitLoginCode(c.Request.Context(), req.Handle, req.Code, signedin.Client(c))
	if err != nil {
		h.fail(c, err, "checking a one-time code failed")
		return
	}

	h.signedIn(c, result.Request, result)
}

// ResendCode sends another code for a sign-in that is still waiting, and
// answers with the wait before the next one.
func (h *Handler) ResendCode(c *gin.Context) {
	var req resendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a resend failed")
		return
	}

	challenge, err := h.provider.ResendLoginCode(c.Request.Context(), req.Handle, signedin.Client(c))
	if err != nil {
		h.fail(c, err, "sending another one-time code failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": challenge})
}

// ForgotPassword sends a reset link. It answers the same whether or not the
// address has an account.
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req forgotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a password reset request failed")
		return
	}

	if err := h.provider.ForgotPassword(c.Request.Context(), req.Email, req.Request, req.Language, signedin.Client(c)); err != nil {
		h.fail(c, err, "starting a password reset failed")
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "sent"})
}

// CheckReset says whether a reset link still works.
func (h *Handler) CheckReset(c *gin.Context) {
	err := h.provider.CheckReset(c.Request.Context(), c.Query("token"))
	if err != nil && !errors.Is(err, oidc.ErrResetInvalid) {
		h.fail(c, err, "checking a reset link failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": err == nil})
}

// ResetPassword sets a new password through a reset link.
func (h *Handler) ResetPassword(c *gin.Context) {
	var req resetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a password reset failed")
		return
	}

	if err := h.provider.ResetPassword(c.Request.Context(), req.Token, req.Password, signedin.Client(c)); err != nil {
		h.fail(c, err, "resetting a password failed")
		return
	}

	session.ClearUser(c, h.secure)

	c.JSON(http.StatusOK, gin.H{"status": "reset"})
}

// VerifyEmail uses a verification link. It is a POST made when the person
// presses the button, so mail scanners that follow links cannot spend it.
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req verifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := validate.Struct(req); err != nil {
		respond.Failure(c, h.log, err, "validating an email verification failed")
		return
	}

	if err := h.provider.VerifyEmail(c.Request.Context(), req.Token, signedin.Client(c)); err != nil {
		h.fail(c, err, "verifying an email failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "verified"})
}

// Logout signs the user out of this server in this browser, and says where to
// go next: the application's registered post-logout URI when the confirmation
// page names one, otherwise the signed-out page.
func (h *Handler) Logout(c *gin.Context) {
	var req logoutRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			respond.Fail(c, respond.InvalidBody)
			return
		}
	}
	if err := validate.Struct(req); err != nil {
		respond.Failure(c, h.log, err, "validating a logout failed")
		return
	}

	location, signedOut, err := h.provider.Logout(c.Request.Context(), oidc.LogoutParams{
		ClientID:              req.ClientID,
		PostLogoutRedirectURI: req.PostLogoutRedirectURI,
		State:                 req.State,
		Confirmed:             true,
	}, session.UserToken(c), signedin.Client(c))
	if err != nil {
		h.log.Error("signing a user out failed", "error", err)
		signedOut = true
	}

	if signedOut {
		session.ClearUser(c, h.secure)
	}

	c.JSON(http.StatusOK, gin.H{"status": "signed out", "location": location})
}

var (
	accountTokenRefused = respond.Define(http.StatusUnauthorized, "account_token_refused", respond.Public)
	accountScopeMissing = respond.Define(http.StatusForbidden, "account_scope_missing", respond.Public)
)

// RequireSession requires a user's session: the id app's cookie, or an account
// API access token with account.read (account.write to change anything). A
// request with a token is judged by the token alone and has no current browser
// session.
func (h *Handler) RequireSession(c *gin.Context) {
	if bearer := session.Bearer(c); bearer != "" {
		h.requireToken(c, bearer)
		return
	}

	token := session.UserToken(c)

	current, err := h.provider.SessionFor(c.Request.Context(), token)
	if err != nil {
		h.fail(c, err, "loading the user session failed")
		c.Abort()
		return
	}
	if current == nil {
		respond.Abort(c, respond.NotSignedIn)
		return
	}

	signedin.Set(c, current)
	c.Next()
}

func (h *Handler) requireToken(c *gin.Context, token string) {
	caller, err := h.provider.AccountCaller(c.Request.Context(), token)
	if errors.Is(err, oidc.ErrTokenRefused) {
		c.Header("WWW-Authenticate", `Bearer realm="`+brand.Realm+`", error="invalid_token"`)
		respond.Abort(c, accountTokenRefused)
		return
	}
	if err != nil {
		h.fail(c, err, "checking an account API token failed")
		c.Abort()
		return
	}

	needed := []string{model.ScopeAccountWrite}
	if c.Request.Method == http.MethodGet {
		needed = append(needed, model.ScopeAccountRead)
	}
	if !slices.ContainsFunc(caller.Scopes, func(scope string) bool { return slices.Contains(needed, scope) }) {
		c.Header("WWW-Authenticate", `Bearer realm="`+brand.Realm+`", error="insufficient_scope", scope="`+needed[len(needed)-1]+`"`)
		respond.Abort(c, accountScopeMissing, "scope", needed[len(needed)-1])
		return
	}

	signedin.Set(c, caller.Session)
	c.Next()
}

func (h *Handler) fail(c *gin.Context, err error, note string) {
	signedin.Fail(c, h.log, err, note)
}

// SecondFactor is what the second-factor page offers for the sign-in the
// browser's cookie names.
func (h *Handler) SecondFactor(c *gin.Context) {
	challenge, err := h.provider.SecondFactorFor(c.Request.Context(), session.SecondFactor(c))
	if err != nil {
		h.fail(c, err, "loading a second-factor challenge failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"second_factor": challenge})
}

// SubmitSecondFactor finishes a waiting sign-in, and answers as the sign-in
// itself does: a cookie, and where to go next.
func (h *Handler) SubmitSecondFactor(c *gin.Context) {
	var req secondFactorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a second factor failed")
		return
	}

	ctx := c.Request.Context()
	current, err := h.provider.SessionFor(ctx, session.UserToken(c))
	if err != nil {
		h.fail(c, err, "loading the user session failed")
		return
	}

	result, err := h.provider.SubmitSecondFactor(ctx, session.SecondFactor(c), current, oidc.SecondFactorAnswer{
		Code:    req.Code,
		Passkey: req.Passkey,
	}, signedin.Client(c))
	if err != nil {
		h.fail(c, err, "checking a second factor failed")
		return
	}

	session.ClearSecondFactor(c, h.secure)
	h.signedIn(c, result.Request, result)
}

// BeginPasskey starts a sign-in with a passkey alone.
func (h *Handler) BeginPasskey(c *gin.Context) {
	var req passkeyStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a passkey sign-in failed")
		return
	}

	offer, err := h.provider.BeginPasskeySignIn(c.Request.Context(), req.Request, req.Remember)
	if err != nil {
		h.fail(c, err, "starting a passkey sign-in failed")
		return
	}

	c.JSON(http.StatusOK, offer)
}

// FinishPasskey signs in with the passkey's answer, and answers as a password
// sign-in does: a cookie, and where to go next.
func (h *Handler) FinishPasskey(c *gin.Context) {
	var req passkeyFinishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a passkey answer failed")
		return
	}

	result, err := h.provider.FinishPasskeySignIn(c.Request.Context(), req.Handle, req.Response, signedin.Client(c))
	if err != nil {
		h.fail(c, err, "checking a passkey failed")
		return
	}

	h.signedIn(c, result.Request, result)
}
