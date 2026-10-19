// Package cors allows browsers on an exact list of origins to call the API.
package cors

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	allowedMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	allowedHeaders = "Authorization, Content-Type"
)

// New allows exactly the listed origins, with credentials; no wildcards or
// suffix matching, since a loose match would let any site act as the signed-in
// user.
//
// The provider endpoints a browser app calls directly (token, userinfo,
// revocation, introspection, discovery, JWKS) are open to every origin without
// credentials: they never read a cookie.
func New(origins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if public(c.Request.URL.Path) {
			if origin != "" {
				h := c.Writer.Header()
				h.Set("Access-Control-Allow-Origin", "*")
				h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				h.Set("Access-Control-Allow-Headers", allowedHeaders)
			}

			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}

			c.Next()
			return
		}

		if origin != "" && slices.Contains(origins, origin) {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Methods", allowedMethods)
			h.Set("Access-Control-Allow-Headers", allowedHeaders)

			// The answer depends on which origin asked, so a cache must not
			// hand one origin's answer to another.
			h.Add("Vary", "Origin")
		}

		// Preflights are answered here for anyone; the browser enforces the
		// result.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// publicPaths are the provider endpoints any origin may call. Authorize and
// logout are excluded: browsers navigate there, and they read the session
// cookie.
var publicPaths = []string{
	"/oauth2/token",
	"/oauth2/userinfo",
	"/oauth2/revoke",
	"/oauth2/introspect",
}

func public(path string) bool {
	return slices.Contains(publicPaths, path) || strings.HasPrefix(path, "/.well-known/")
}
