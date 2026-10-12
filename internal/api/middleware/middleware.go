package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

// Chain returns the middleware in order: logging (so panics are reported),
// recovery, CORS from the caller, then the body limit, after preflight so a
// browser's check is never refused.
//
//	r.Use(middleware.Chain(log, cors.New(origins))...)
func Chain(log *slog.Logger, cors gin.HandlerFunc) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		RequestLogger(log),
		gin.Recovery(),
		cors,
		BodyLimit(MaxBodyBytes),
	}
}
