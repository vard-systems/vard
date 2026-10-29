// Package middleware is what every request passes through, in order: the
// request log, recovery, security headers, CORS and the body limit.
package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

// Chain returns the middleware in order: logging (so panics are reported),
// recovery, the security headers (so even a preflight carries them), CORS from
// the caller, then the body limit, after preflight so a browser's check is
// never refused.
//
//	r.Use(middleware.Chain(log, cors.New(origins), https)...)
func Chain(log *slog.Logger, cors gin.HandlerFunc, https bool) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		RequestLogger(log),
		gin.Recovery(),
		SecurityHeaders(https),
		cors,
		BodyLimit(MaxBodyBytes),
	}
}
