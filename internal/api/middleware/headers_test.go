package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders(t *testing.T) {
	want := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "no-referrer",
		"X-Frame-Options":         "DENY",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
	}

	cases := []struct {
		name  string
		route gin.HandlerFunc
	}{
		{name: "an answer", route: func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) }},
		{name: "a redirect", route: func(c *gin.Context) { c.Redirect(http.StatusFound, "https://app.example/callback?code=x") }},
		{name: "a stopped request", route: func(c *gin.Context) { c.AbortWithStatus(http.StatusForbidden) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(SecurityHeaders())
			r.GET("/thing", tc.route)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/thing", nil))

			for name, value := range want {
				if got := w.Header().Get(name); got != value {
					t.Errorf("%s = %q, want %q", name, got, value)
				}
			}
		})
	}
}
