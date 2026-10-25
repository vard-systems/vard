package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"vard/internal/api/kit/respond"
)

// MaxBodyBytes caps every request body. The largest legitimate body (a language
// or SAML metadata) is tens of kilobytes; the cap stops unauthenticated callers
// filling memory.
const MaxBodyBytes = 1 << 20

var BodyTooLarge = respond.Define(http.StatusRequestEntityTooLarge, "body_too_large", respond.Both)

// BodyLimit refuses a declared oversized body up front and cuts off an
// undeclared or lying one while it is read.
func BodyLimit(max int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > max {
			respond.Abort(c, BodyTooLarge)
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)

		c.Next()
	}
}
