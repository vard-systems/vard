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
	// Lax: sent on the top-level navigation to the authorization endpoint,
	// not on other sites' script requests.
	writeCookie(c, cookie{name: UserCookie, path: "/", sameSite: http.SameSiteLaxMode}, value, maxAge, secure)
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
	return read(c, SignInStateCookie)
}

// UserToken is the user's session cookie, or "" when absent or duplicated (a
// planted sibling-subdomain cookie cannot stand in).
func UserToken(c *gin.Context) string {
	return read(c, UserCookie)
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

	writeCookie(c, cookie{name: SignInStateCookie, path: signInStatePath, sameSite: sameSite}, value, maxAge, secure)
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
	return read(c, SecondFactorCookie)
}

func writeSecondFactor(c *gin.Context, value string, maxAge int, secure bool) {
	writeCookie(c, cookie{name: SecondFactorCookie, path: secondFactorPath, sameSite: http.SameSiteLaxMode}, value, maxAge, secure)
}
