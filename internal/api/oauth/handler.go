// Package oauth answers the OAuth 2.0 and OpenID Connect endpoints (discovery,
// keys, authorize, token, userinfo, logout, revocation, introspection) and the
// redirects of social sign-in and single sign-on.
package oauth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"loginer/internal/api/kit/respond"
	"loginer/internal/api/kit/session"
	"loginer/internal/brand"
	"loginer/internal/oidc"
)

type Handler struct {
	provider *oidc.Service
	log      *slog.Logger
	secure   bool
}

// New returns a Handler. `secure` sets the Secure flag on the user session
// cookie the logout endpoint clears.
func New(provider *oidc.Service, log *slog.Logger, secure bool) *Handler {
	return &Handler{provider: provider, log: log, secure: secure}
}

// Discovery answers the OpenID Connect discovery document.
func (h *Handler) Discovery(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, h.provider.Discovery(c.Request.Context()))
}

// JWKS answers the public keys tokens are signed with.
func (h *Handler) JWKS(c *gin.Context) {
	keys, err := h.provider.JWKS(c.Request.Context())
	if err != nil {
		h.serverError(c, err, "loading the signing keys failed")
		return
	}

	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, keys)
}

// Authorize starts a sign-in for an application, and redirects: to the
// sign-in page, or straight back with a code for a user already signed in.
func (h *Handler) Authorize(c *gin.Context) {
	ctx := c.Request.Context()

	params, err := authorizeParams(c)
	if err != nil {
		c.String(http.StatusBadRequest, "the request could not be read")
		return
	}

	current, err := h.provider.SessionFor(ctx, cookie(c))
	if err != nil {
		h.serverError(c, err, "loading the user session failed")
		return
	}

	location, err := h.provider.Authorize(ctx, params, current)
	if err != nil {
		h.serverError(c, err, "authorization failed")
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, location)
}

// SocialStart sends the browser to a provider and leaves the state in a cookie,
// binding the callback to this browser.
func (h *Handler) SocialStart(c *gin.Context) {
	location, state, err := h.provider.StartSocial(
		c.Request.Context(),
		c.Param("slug"),
		c.Query("request"),
		c.Query("next"),
	)
	if err != nil {
		h.socialFailed(c, err, "starting a social sign-in failed")
		return
	}

	session.SetSignInState(c, state, h.secure)

	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, location)
}

// SocialCallback is where the provider sends the browser back. It answers GET
// and POST, since Apple posts its answer.
func (h *Handler) SocialCallback(c *gin.Context) {
	ctx := c.Request.Context()

	// However this ends, the sign-in the browser was carrying is over.
	binding := session.SignInState(c)
	session.ClearSignInState(c, h.secure)

	code, state := c.Query("code"), c.Query("state")
	if c.Request.Method == http.MethodPost {
		code, state = c.PostForm("code"), c.PostForm("state")
	}

	if refused := c.Query("error"); refused != "" {
		h.log.Info("a social provider refused a sign-in",
			"provider", c.Param("slug"), "error", refused,
			"description", c.Query("error_description"))

		c.Redirect(http.StatusFound, h.provider.SocialErrorPage(oidc.ErrSocialUpstream))
		return
	}

	result, err := h.provider.CompleteSocial(ctx, c.Param("slug"), code, state, binding, client(c))
	if err != nil {
		h.socialFailed(c, err, "completing a social sign-in failed")
		return
	}

	session.SetUser(c, result.SignIn.Token, result.SignIn.Session.Record.ExpiresAt, result.SignIn.Remember, h.secure)

	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, h.provider.SocialLanding(ctx, result))
}

func (h *Handler) socialFailed(c *gin.Context, err error, note string) {
	if !oidc.IsSocialFailure(err) {
		h.log.Error(note, "provider", c.Param("slug"), "error", err)
	}

	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, h.provider.SocialErrorPage(err))
}

// SSOStart sends the browser to a connection's identity provider, passing the
// typed address as `login_hint`.
func (h *Handler) SSOStart(c *gin.Context) {
	location, state, err := h.provider.StartSSO(
		c.Request.Context(),
		c.Param("slug"),
		c.Query("request"),
		c.Query("next"),
		c.Query("login_hint"),
	)
	if err != nil {
		h.ssoFailed(c, err, "starting a single sign-on failed")
		return
	}

	session.SetSignInState(c, state, h.secure)

	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, location)
}

// SSOCallback is where an OpenID Connect provider sends the browser back to.
func (h *Handler) SSOCallback(c *gin.Context) {
	binding := session.SignInState(c)
	session.ClearSignInState(c, h.secure)

	if refused := c.Query("error"); refused != "" {
		err := h.provider.RefuseSSO(c.Request.Context(), c.Param("slug"), c.Query("state"), binding,
			refused, c.Query("error_description"), client(c))
		h.ssoFailed(c, err, "recording an identity provider's refusal failed")
		return
	}

	result, err := h.provider.CompleteSSOCallback(c.Request.Context(), c.Param("slug"), c.Query("code"), c.Query("state"), binding, client(c))
	h.ssoFinished(c, result, err)
}

// samlResponseLimit is the most a provider's POST may be. An assertion with
// many groups and certificates is tens of kilobytes.
const samlResponseLimit = 1 << 20

// SSOAssertion is the assertion consumer service a SAML provider posts its
// response to. It only holds the response and sends the browser on with 303
// (so the next request is a GET) to SSOAssertionReturn, on this origin, where
// the browser's state cookie arrives even over plain http: a SameSite=Lax
// cookie is sent on a top-level GET, never on another site's POST. The
// cookie is left alone here, for that request to read.
func (h *Handler) SSOAssertion(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, samlResponseLimit)

	location, err := h.provider.HoldSSOAssertion(c.Request.Context(), c.Param("slug"), c.Request)
	if err != nil {
		h.ssoFailed(c, err, "holding a SAML response failed")
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusSeeOther, location)
}

// SSOAssertionReturn finishes a SAML sign-in with the response SSOAssertion
// held for this browser's own sign-in.
func (h *Handler) SSOAssertionReturn(c *gin.Context) {
	result, err := h.provider.CompleteSSOAssertion(c.Request.Context(), c.Param("slug"), session.SignInState(c), client(c))

	// Anyone can link a browser here before its provider has answered; that
	// must not cost it the sign-in under way. Once one was used, the cookie goes.
	if !errors.Is(err, oidc.ErrSSOExpired) {
		session.ClearSignInState(c, h.secure)
	}

	h.ssoFinished(c, result, err)
}

// SSOMetadata is this server's SAML metadata as one connection's service
// provider: the file an identity provider is set up from.
func (h *Handler) SSOMetadata(c *gin.Context) {
	document, err := h.provider.SSOMetadata(c.Request.Context(), c.Param("slug"))
	if err != nil {
		respond.Fail(c, respond.NotFoundAny)
		return
	}

	c.Header("Content-Disposition", `inline; filename="`+c.Param("slug")+`-metadata.xml"`)
	c.Data(http.StatusOK, "application/samlmetadata+xml; charset=utf-8", document)
}

func (h *Handler) ssoFinished(c *gin.Context, result *oidc.SocialResult, err error) {
	if err != nil {
		h.ssoFailed(c, err, "completing a single sign-on failed")
		return
	}

	session.SetUser(c, result.SignIn.Token, result.SignIn.Session.Record.ExpiresAt, result.SignIn.Remember, h.secure)

	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, h.provider.SocialLanding(c.Request.Context(), result))
}

func (h *Handler) ssoFailed(c *gin.Context, err error, note string) {
	if !oidc.IsSSOFailure(err) {
		h.log.Error(note, "connection", c.Param("slug"), "error", err)
	}

	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, h.provider.SSOErrorPage(err))
}

// Token issues tokens.
func (h *Handler) Token(c *gin.Context) {
	noStore(c)

	params, err := tokenParams(c)
	if err != nil {
		h.oauthError(c, err)
		return
	}

	tokens, err := h.provider.Token(c.Request.Context(), params)
	if err != nil {
		h.oauthError(c, err)
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// UserInfo answers the claims about the user an access token is for.
func (h *Handler) UserInfo(c *gin.Context) {
	noStore(c)

	token, err := bearer(c)
	if err != nil {
		h.bearerError(c, err)
		return
	}

	claims, err := h.provider.UserInfo(c.Request.Context(), token)
	if err != nil {
		h.bearerError(c, err)
		return
	}

	c.JSON(http.StatusOK, claims)
}

// Logout signs the user out of this server, and redirects.
func (h *Handler) Logout(c *gin.Context) {
	params, err := logoutParams(c)
	if err != nil {
		c.String(http.StatusBadRequest, "the request could not be read")
		return
	}

	location, signedOut, err := h.provider.Logout(c.Request.Context(), params, cookie(c), client(c))
	if err != nil {
		h.serverError(c, err, "logout failed")
		return
	}

	// Clear the cookie only for a logout about this browser's session; anyone
	// can put this address in a link.
	if signedOut {
		session.ClearUser(c, h.secure)
	}
	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, location)
}

// Revoke revokes a refresh token.
func (h *Handler) Revoke(c *gin.Context) {
	noStore(c)

	auth, token, err := tokenOnlyParams(c)
	if err != nil {
		h.oauthError(c, err)
		return
	}

	if err := h.provider.Revoke(c.Request.Context(), auth, token); err != nil {
		h.oauthError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

// Introspect says whether a token is active.
func (h *Handler) Introspect(c *gin.Context) {
	noStore(c)

	auth, token, err := tokenOnlyParams(c)
	if err != nil {
		h.oauthError(c, err)
		return
	}

	result, err := h.provider.Introspect(c.Request.Context(), auth, token)
	if err != nil {
		h.oauthError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) oauthError(c *gin.Context, err error) {
	var failure *oidc.Error
	if !errors.As(err, &failure) {
		h.log.Error("an oauth endpoint failed", "error", err, "path", c.FullPath())
		c.JSON(http.StatusInternalServerError, errorResponse{Error: oidc.ErrServerError, Description: "something went wrong"})
		return
	}

	// A client that tried HTTP Basic is told which scheme to use again
	// (RFC 6749 section 5.2).
	if failure.Status == http.StatusUnauthorized {
		c.Header("WWW-Authenticate", `Basic realm="`+brand.Realm+`"`)
	}

	c.JSON(failure.Status, errorResponse{Error: failure.Code, Description: failure.Description})
}

func (h *Handler) bearerError(c *gin.Context, err error) {
	var failure *oidc.Error
	if !errors.As(err, &failure) {
		h.log.Error("userinfo failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: oidc.ErrServerError, Description: "something went wrong"})
		return
	}

	c.Header("WWW-Authenticate", `Bearer realm="`+brand.Realm+`", error="`+failure.Code+`", error_description="`+quoteSafe(failure.Description)+`"`)
	c.JSON(failure.Status, errorResponse{Error: failure.Code, Description: failure.Description})
}

func (h *Handler) serverError(c *gin.Context, err error, note string) {
	h.log.Error(note, "error", err)
	c.String(http.StatusInternalServerError, "something went wrong")
}

func noStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
}

func cookie(c *gin.Context) string {
	return session.UserToken(c)
}

func client(c *gin.Context) oidc.Client {
	language, _ := c.Cookie(oidc.LanguageCookie)
	return oidc.Client{IP: c.ClientIP(), UserAgent: c.Request.UserAgent(), Language: language}
}
