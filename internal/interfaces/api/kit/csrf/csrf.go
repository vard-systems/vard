// Package csrf stops other sites changing anything with a signed-in cookie.
// SameSite=Lax is not enough: any subdomain counts as the same site, and a
// text/plain form reaches JSON endpoints without a preflight.
//
// On every state-changing request:
//
//   - a body must be application/json, which a form cannot send;
//   - the browser's Origin (or Sec-Fetch-Site) must be allowed. A request with
//     neither is not a browser's and carries no victim's cookie, and Bearer
//     requests are exempt (see New).
package csrf

import (
	"mime"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"vard/internal/interfaces/api/kit/respond"
)

var (
	unsupportedBody = respond.Define(http.StatusUnsupportedMediaType, "unsupported_body", respond.Both)
	crossOrigin     = respond.Define(http.StatusForbidden, "cross_origin", respond.Both)
	crossSite       = respond.Define(http.StatusForbidden, "cross_site", respond.Both)
)

func New(origins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		if hasBody(c.Request) {
			media, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
			if media != "application/json" {
				respond.Abort(c, unsupportedBody)
				return
			}
		}

		// Bearer-token requests carry no victim's cookie and are judged by the
		// token; forms cannot send Authorization, and other origins only can if
		// CORS allows.
		if hasBearer(c.Request) {
			c.Next()
			return
		}

		if origin := c.GetHeader("Origin"); origin != "" {
			if !slices.Contains(origins, origin) {
				respond.Abort(c, crossOrigin, "origin", origin)
				return
			}
		} else if site := c.GetHeader("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
			respond.Abort(c, crossSite)
			return
		}

		c.Next()
	}
}

func hasBearer(r *http.Request) bool {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	return found && strings.EqualFold(scheme, "Bearer") && strings.TrimSpace(token) != ""
}

func hasBody(r *http.Request) bool {
	return r.ContentLength > 0 || len(r.TransferEncoding) > 0
}
