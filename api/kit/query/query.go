// Package query reads paging and filter values from a request's query string,
// within bounds.
package query

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const MaxOffset = 1_000_000

func Int(c *gin.Context, name string, fallback, max int) int {
	raw := c.Query(name)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}

	return min(value, max)
}

func Bool(c *gin.Context, name string) *bool {
	switch c.Query(name) {
	case "true":
		value := true
		return &value
	case "false":
		value := false
		return &value
	default:
		return nil
	}
}
