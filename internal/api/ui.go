package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// What only the API answers on each listener. A path under one of these that
// no route matches is a JSON problem like any other unknown API path, never
// the app: a client asking the API for something that is not there must not
// be handed a page.
var (
	publicAPIPaths = []string{"/api/", "/oauth2/", "/.well-known/"}
	adminAPIPaths  = []string{"/api/"}
)

// WithPublicUI serves the id app on the public listener, on every path the
// API does not answer.
func WithPublicUI(r *gin.Engine, ui http.Handler) {
	withUI(r, ui, publicAPIPaths)
}

// WithAdminUI serves the console on the admin listener, on every path the
// admin API does not answer.
func WithAdminUI(r *gin.Engine, ui http.Handler) {
	withUI(r, ui, adminAPIPaths)
}

func withUI(r *gin.Engine, ui http.Handler, apiPaths []string) {
	r.NoRoute(func(c *gin.Context) {
		method := c.Request.Method
		if (method != http.MethodGet && method != http.MethodHead) || isAPIPath(c.Request.URL.Path, apiPaths) {
			notFound(c)
			return
		}

		ui.ServeHTTP(c.Writer, c.Request)
	})
}

func isAPIPath(path string, apiPaths []string) bool {
	for _, prefix := range apiPaths {
		if strings.HasPrefix(path, prefix) || path == strings.TrimSuffix(prefix, "/") {
			return true
		}
	}
	return false
}
