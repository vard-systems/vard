package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders are set on every answer. The API sends JSON, XML and
// redirects, never a page: so nothing it sends may be sniffed into HTML,
// framed, or run as a document, and no address it redirects from — which can
// carry a code or a state — is passed on as a Referer. Served over https
// (whoever ends TLS in front), it also holds browsers to https for two years.
func SecurityHeaders(https bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		if https {
			h.Set("Strict-Transport-Security", "max-age=63072000")
		}
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		c.Next()
	}
}
