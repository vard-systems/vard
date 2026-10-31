package caching

import (
	"encoding/json"
	"slices"
	"strconv"

	"vard/internal/infrastructure/cache"
)

// overviewResponse is both databases, or neither when there is no Redis.
type overviewResponse struct {
	Configured bool               `json:"configured"`
	Databases  []databaseResponse `json:"databases"`
}

type groupResponse struct {
	Name       string `json:"name"`
	Generation int64  `json:"generation"`
	Entries    int64  `json:"entries"`
	Stale      int64  `json:"stale"`
	// Known says the server reads the group; one it does not is holding
	// values from before, left to expire.
	Known bool `json:"known"`
}

type databaseResponse struct {
	Name      string           `json:"name"`
	Number    int              `json:"number"`
	Available bool             `json:"available"`
	Keys      int64            `json:"keys"`
	Kinds     map[string]int64 `json:"kinds"`
	Groups    []groupResponse  `json:"groups"`
	Sessions  map[string]int64 `json:"sessions"`
	// Editable says whether its cached values may be replaced by hand: the
	// cache database's may, the session database's never.
	Editable bool   `json:"editable"`
	Memory   int64  `json:"memory_bytes"`
	Version  string `json:"version"`
}

func newDatabaseResponse(stats cache.Stats) databaseResponse {
	out := databaseResponse{
		Name:      stats.Name,
		Number:    stats.Number,
		Available: stats.Available,
		Keys:      stats.Keys,
		Kinds:     stats.Kinds,
		Groups:    make([]groupResponse, 0, len(stats.Groups)),
		Sessions:  stats.Sessions,
		Editable:  stats.Name == cache.CacheDatabase,
		Memory:    stats.Memory,
		Version:   stats.Version,
	}
	if out.Kinds == nil {
		out.Kinds = map[string]int64{}
	}
	if out.Sessions == nil {
		out.Sessions = map[string]int64{}
	}

	for _, group := range stats.Groups {
		out.Groups = append(out.Groups, groupResponse{
			Name:       group.Name,
			Generation: group.Generation,
			Entries:    group.Entries,
			Stale:      group.Stale,
			Known:      slices.Contains(cache.Groups[stats.Name], group.Name),
		})
	}

	return out
}

type keyResponse struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Group string `json:"group"`
	Entry string `json:"entry"`
	// TTLSeconds is how long it has left, or -1 for a key that does not
	// expire.
	TTLSeconds int64 `json:"ttl_seconds"`
	Size       int64 `json:"size_bytes"`
	Editable   bool  `json:"editable"`
}

func newKeyResponse(key cache.Key) keyResponse {
	ttl := int64(-1)
	if key.TTL >= 0 {
		ttl = int64(key.TTL.Seconds())
	}

	return keyResponse{
		Name:       key.Name,
		Kind:       key.Kind,
		Group:      key.Group,
		Entry:      key.Entry,
		TTLSeconds: ttl,
		Size:       key.Size,
		Editable:   key.Editable,
	}
}

// keysResponse is a page of keys and where the next one starts: an empty
// cursor after the last page.
type keysResponse struct {
	Keys   []keyResponse `json:"keys"`
	Cursor string        `json:"cursor"`
}

func newKeysResponse(keys []cache.Key, next uint64) keysResponse {
	out := keysResponse{Keys: make([]keyResponse, 0, len(keys))}
	for _, key := range keys {
		out.Keys = append(out.Keys, newKeyResponse(key))
	}
	if next != 0 {
		out.Cursor = strconv.FormatUint(next, 10)
	}

	return out
}

// valueResponse is a key with what it holds.
type valueResponse struct {
	keyResponse
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

func newValueResponse(value cache.Value) valueResponse {
	return valueResponse{keyResponse: newKeyResponse(value.Key), Type: value.Type, Value: value.Value}
}
