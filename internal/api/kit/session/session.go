// Package session reads who is calling (an administrator, a user, or an admin
// API token) and guards routes by session and permission.
package session

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loginer/internal/api/kit/respond"
	"loginer/internal/auth"
	"loginer/internal/brand"
	"loginer/internal/model"
	"loginer/internal/oidc"
)

// Cookie carries the admin session token, HttpOnly so script (and an XSS bug)
// cannot read it.
const Cookie = brand.AdminSessionCookie

const (
	key        = "admin"
	sessionKey = "session_id"
)

// Set stores the token in the browser. `secure` adds the Secure flag, which
// can only be on once the API is served over HTTPS.
func Set(c *gin.Context, token string, secure bool) {
	write(c, token, int(auth.SessionLifetime.Seconds()), secure)
}

// Clear removes it again.
func Clear(c *gin.Context, secure bool) {
	write(c, "", -1, secure)
}

func write(c *gin.Context, value string, maxAge int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     Cookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// Require refuses the request unless it carries a session for an
// administrator who may sign in. Handlers behind it can call Admin without
// checking.
func Require(service *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := Token(c)

		user, current, err := service.Authenticate(c.Request.Context(), token)
		if err != nil {
			respond.Abort(c, respond.NotSignedIn)
			return
		}

		c.Set(key, user)
		c.Set(sessionKey, current.ID)

		c.Next()
	}
}

var tokenRefused = respond.Define(http.StatusUnauthorized, "token_refused", respond.Admin)

// RequireAny accepts an administrator's session cookie or an admin API access
// token. A token caller becomes a service admin holding its token's
// permissions, so every permission check works unchanged and RequireSuperAdmin
// refuses it. A token always wins over a cookie.
func RequireAny(service *auth.Service, provider *oidc.Service, log *slog.Logger) gin.HandlerFunc {
	cookie := Require(service)

	return func(c *gin.Context) {
		token := Bearer(c)
		if token == "" {
			cookie(c)
			return
		}

		caller, err := provider.AdminCaller(c.Request.Context(), token)
		if errors.Is(err, oidc.ErrTokenRefused) {
			c.Header("WWW-Authenticate", `Bearer realm="`+brand.Realm+`", error="invalid_token"`)
			respond.Abort(c, tokenRefused)
			return
		}
		if err != nil {
			respond.Failure(c, log, err, "checking an admin API token failed")
			c.Abort()
			return
		}

		c.Set(key, model.ServiceAdmin(*caller.Application, caller.Permissions))
		c.Next()
	}
}

// RequestOf is where a call came from, as the session and the log record it.
func RequestOf(c *gin.Context) auth.Request {
	return auth.Request{IP: c.ClientIP(), UserAgent: c.Request.UserAgent()}
}

// Bearer is the access token in the request's Authorization header, or ""
// when it carries none.
func Bearer(c *gin.Context) string {
	scheme, token, found := strings.Cut(c.GetHeader("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

// Token is the administrator's session cookie, or "" when absent or
// duplicated: a cookie planted from a sibling subdomain cannot stand in for it.
func Token(c *gin.Context) string {
	return single(c, Cookie)
}

// RequireSetup admits signed-in administrators and half-signed-in ones waiting
// to enrol a second factor; it guards the enrolment endpoints.
func RequireSetup(service *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := Token(c)

		user, _, state, err := service.Session(c.Request.Context(), token)
		if err != nil || (state != auth.StateSignedIn && state != auth.StateEnroll) {
			respond.Abort(c, respond.NotSignedIn)
			return
		}

		c.Set(key, user)

		c.Next()
	}
}

// Can refuses the request unless the signed-in administrator holds a role
// granting the permission. It goes after Require, which loads the roles.
func Can(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if admin := Admin(c); admin == nil || !admin.HasPermission(permission) {
			respond.Abort(c, respond.NotAllowed)
			return
		}

		c.Next()
	}
}

// CanAnywhere requires one of the permissions panel-wide or for at least one
// application; handlers then narrow with Allowed and Reach.
func CanAnywhere(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := Admin(c)

		for _, permission := range permissions {
			if admin != nil && admin.HasPermissionAnywhere(permission) {
				c.Next()
				return
			}
		}

		respond.Abort(c, respond.NotAllowed)
	}
}

// Allowed reports whether the administrator holds the permission for one
// application.
func Allowed(c *gin.Context, permission string, application uuid.UUID) bool {
	admin := Admin(c)

	return admin != nil && admin.HasPermissionFor(permission, &application)
}

// AllowedScope reports whether the administrator holds the permission for a
// role's scope (nil application means global).
func AllowedScope(c *gin.Context, permission string, application *uuid.UUID) bool {
	if application == nil {
		admin := Admin(c)
		return admin != nil && admin.HasPermission(permission)
	}

	return Allowed(c, permission, *application)
}

// SeesRole reports whether the administrator can see a role: global roles are
// visible to all who list roles; application roles to those who can read the
// application.
func SeesRole(c *gin.Context, role model.UserRole) bool {
	return role.Global() || Allowed(c, model.PermApplicationsRead, *role.ApplicationID)
}

// Reach is the applications the administrator holds the permission for, as a
// store filter: nil for all, otherwise their ids (possibly none).
func Reach(c *gin.Context, permission string) []uuid.UUID {
	admin := Admin(c)
	if admin == nil {
		return []uuid.UUID{}
	}

	all, ids := admin.ApplicationsWith(permission)
	if all {
		return nil
	}

	return ids
}

var superAdminOnly = respond.Define(http.StatusForbidden, "super_admin_only", respond.Admin)

// RequireSuperAdmin refuses the request unless the signed-in administrator is
// a super admin: managing administrators and their roles is theirs alone.
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if admin := Admin(c); admin == nil || !admin.IsSuperAdmin() {
			respond.Abort(c, superAdminOnly)
			return
		}

		c.Next()
	}
}

// ID returns the session the request came with. Like Admin, it is only valid
// behind Require.
func ID(c *gin.Context) uuid.UUID {
	id, _ := c.Get(sessionKey)
	current, _ := id.(uuid.UUID)

	return current
}

// Admin returns the administrator making the request. It is only valid behind
// Require, which is the only thing that sets it.
func Admin(c *gin.Context) *model.Admin {
	user, _ := c.Get(key)
	admin, _ := user.(*model.Admin)

	return admin
}
