package caching

import (
	"bytes"
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"loginer/internal/api/kit/respond"
	"loginer/internal/cache"
)

var (
	notConfigured    = respond.Define(http.StatusServiceUnavailable, "redis_not_configured", respond.Admin)
	databaseNotFound = respond.Define(http.StatusNotFound, "cache_database_not_found", respond.Admin)
	groupNotFound    = respond.Define(http.StatusNotFound, "cache_group_not_found", respond.Admin)
	keyNotFound      = respond.Define(http.StatusNotFound, "cache_key_not_found", respond.Admin)
	keyNotEditable   = respond.Define(http.StatusConflict, "cache_key_not_editable", respond.Admin)
	generationKey    = respond.Define(http.StatusConflict, "cache_generation_key", respond.Admin)
	invalidKeyName   = respond.Define(http.StatusBadRequest, "cache_key_invalid", respond.Admin)
	invalidKind      = respond.Define(http.StatusBadRequest, "cache_kind_invalid", respond.Admin)
	invalidCursor    = respond.Define(http.StatusBadRequest, "cache_cursor_invalid", respond.Admin)
	invalidValue     = respond.Define(http.StatusBadRequest, "cache_value_invalid", respond.Admin)
	invalidTTL       = respond.Define(http.StatusBadRequest, "cache_ttl_invalid", respond.Admin)
)

const maxTTL = 24 * time.Hour

var kinds = []string{
	cache.KindEntry, cache.KindStale, cache.KindGeneration,
	cache.KindSession, cache.KindRateLimit, cache.KindOther,
}

func checkKind(kind string) error {
	if kind != "" && !slices.Contains(kinds, kind) {
		return invalidKind.With()
	}
	return nil
}

func checkName(name string) error {
	if name == "" || len(name) > 1024 {
		return invalidKeyName.With()
	}
	return nil
}

func (r writeRequest) check() (json.RawMessage, time.Duration, error) {
	// null is JSON, but it decodes into an empty record without complaint, so
	// the pages would show nothing where the value was.
	if len(r.Value) == 0 || !json.Valid(r.Value) || bytes.Equal(bytes.TrimSpace(r.Value), []byte("null")) {
		return nil, 0, invalidValue.With()
	}

	ttl := time.Duration(r.TTLSeconds) * time.Second
	if ttl < 0 || ttl > maxTTL {
		return nil, 0, invalidTTL.With("max", int(maxTTL.Seconds()))
	}

	return r.Value, ttl, nil
}
