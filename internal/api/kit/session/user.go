package session

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"vard/internal/brand"
)

// UserCookie carries a user's own session for single sign-on. It is
// deliberately not the administrators' cookie: signing in to an application
// must never sign anyone in to the panel.
const UserCookie = brand.UserSessionCookie

func SetUser(c *gin.Context, token string, expires time.Time, remember, secure bool) {
	maxAge := 0
	if remember {
		maxAge = int(time.Until(expires).Seconds())
	}

	writeUser(c, token, maxAge, secure)
}

func ClearUser(c *gin.Context, secure bool) {
	writeUser(c, "", -1, secure)
}

func writeUser(c *gin.Context, value string, maxAge int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     UserCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		// Lax: sent on the top-level navigation to the authorization endpoint,
		// not on other sites' script requests.
		SameSite: http.SameSiteLaxMode,
	})
}

// SignInStateCookie binds a provider sign-in to the browser that started it.
// Without it, an attacker could walk a victim's browser through the attacker's
// own state and code and sign the victim into the attacker's account.
const SignInStateCookie = brand.SignInStateCookie

// signInStatePath keeps the cookie to the provider endpoints, which are the
// only ones that ever read it.
const signInStatePath = "/oauth2"

func SetSignInState(c *gin.Context, state string, secure bool) {
	writeSignInState(c, state, 0, secure)
}

func ClearSignInState(c *gin.Context, secure bool) {
	writeSignInState(c, "", -1, secure)
}

// SignInState is the browser's state cookie, or "". Two cookies of the same
// name (which a sibling subdomain can plant) count as none, so the sign-in
// fails closed.
func SignInState(c *gin.Context) string {
	return single(c, SignInStateCookie)
}

// UserToken is the user's session cookie, or "" when absent or duplicated (a
// planted sibling-subdomain cookie cannot stand in).
func UserToken(c *gin.Context) string {
	return single(c, UserCookie)
}

func single(c *gin.Context, name string) string {
	named := c.Request.CookiesNamed(name)
	if len(named) != 1 {
		return ""
	}

	return named[0].Value
}

func writeSignInState(c *gin.Context, value string, maxAge int, secure bool) {
	// Apple posts its answer cross-site, which only SameSite=None cookies
	// survive, and None requires Secure. Over plain http (development) it
	// stays Lax. (SAML does not need it: its ACS holds the post and reads
	// this cookie on the GET it redirects to.)
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SignInStateCookie,
		Value:    value,
		Path:     signInStatePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
}

// SecondFactorCookie holds the handle of a sign-in waiting for its second
// factor. The page never sees it: only the endpoints that finish the sign-in
// read it.
const SecondFactorCookie = brand.SecondFactorCookie

const secondFactorPath = "/api/v1/account/login/second-factor"

func SetSecondFactor(c *gin.Context, handle string, expires time.Time, secure bool) {
	writeSecondFactor(c, handle, int(time.Until(expires).Seconds()), secure)
}

func ClearSecondFactor(c *gin.Context, secure bool) {
	writeSecondFactor(c, "", -1, secure)
}

// SecondFactor is the waiting sign-in's handle, or "" when absent or
// duplicated (a planted sibling-subdomain cookie cannot stand in).
func SecondFactor(c *gin.Context) string {
	return single(c, SecondFactorCookie)
}

func writeSecondFactor(c *gin.Context, value string, maxAge int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SecondFactorCookie,
		Value:    value,
		Path:     secondFactorPath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// KnownDeviceCookie is the accounts this browser has signed in to, which
// sign-ins are throttled by apart from strangers' attempts.
const KnownDeviceCookie = brand.KnownDeviceCookie

// knownDevicePath keeps it to the account endpoints, the only ones that read
// it.
const knownDevicePath = "/api/v1/account"

// knownDeviceLifetime is how long a browser stays known without signing in.
const knownDeviceLifetime = 180 * 24 * time.Hour

// KnownDevice is the browser's known-device cookie, or "" when absent or
// duplicated.
func KnownDevice(c *gin.Context) string {
	return single(c, KnownDeviceCookie)
}

func SetKnownDevice(c *gin.Context, value string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     KnownDeviceCookie,
		Value:    value,
		Path:     knownDevicePath,
		MaxAge:   int(knownDeviceLifetime.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
