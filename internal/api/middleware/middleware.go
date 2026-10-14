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
//	r.Use(middleware.Chain(log, cors.New(origins))...)
func Chain(log *slog.Logger, cors gin.HandlerFunc) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		RequestLogger(log),
		gin.Recovery(),
		SecurityHeaders(),
		cors,
		BodyLimit(MaxBodyBytes),
	}
}
