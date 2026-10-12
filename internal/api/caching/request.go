package caching

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"

	"loginer/internal/cache"
)

// targetType is what these rows are called in the activity log. The target
// is the database, by name; the key or group is in the metadata.
const targetType = "cache"

// writeRequest is the body of the endpoint that replaces a cached value.
type writeRequest struct {
	// Value is the new value, as JSON of any shape.
	Value json.RawMessage `json:"value"`
	// TTLSeconds is how long it should live; left out or zero keeps what it
	// had left.
	TTLSeconds int `json:"ttl_seconds"`
}

func parseKeyQuery(c *gin.Context) (cache.KeyQuery, error) {
	query := cache.KeyQuery{
		Kind:   c.Query("kind"),
		Group:  c.Query("group"),
		Search: c.Query("search"),
	}

	if err := checkKind(query.Kind); err != nil {
		return query, err
	}

	if cursor := c.Query("cursor"); cursor != "" {
		// The cursor travels as text: it is a 64-bit number, which a
		// JavaScript number cannot hold exactly.
		parsed, err := strconv.ParseUint(cursor, 10, 64)
		if err != nil {
			return query, invalidCursor.With()
		}
		query.Cursor = parsed
	}

	if limit, err := strconv.Atoi(c.Query("limit")); err == nil {
		query.Limit = limit
	}

	return query, nil
}

func keyName(c *gin.Context) (string, error) {
	name := c.Query("name")
	if err := checkName(name); err != nil {
		return "", err
	}

	return name, nil
}
