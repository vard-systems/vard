package model

import (
	"strings"
	"unicode/utf8"
)

// Truncate drops invalid UTF-8 (Postgres refuses it) and cuts to `max` bytes on
// a rune boundary. It is for recorded outside input, where losing the tail
// beats losing the row.
func Truncate(value string, max int) string {
	value = strings.ToValidUTF8(value, "")
	if len(value) <= max {
		return value
	}

	for max > 0 && !utf8.RuneStart(value[max]) {
		max--
	}

	return value[:max]
}
