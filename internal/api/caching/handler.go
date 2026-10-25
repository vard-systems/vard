// Package caching lets a super admin look into Redis: its databases, groups and
// keys, and clear or edit what the server wrote there.
package caching

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"

	"vard/internal/api/kit/audit"
	"vard/internal/api/kit/respond"
	"vard/internal/cache"
)

type Handler struct {
	redis *cache.Redis
	audit audit.Recorder
	log   *slog.Logger
}

// New returns a Handler. A nil Redis is a server running without one, which
// these endpoints say rather than fail on.
func New(redis *cache.Redis, recorder audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{redis: redis, audit: recorder, log: log}
}

// Overview sums up both databases: how many keys of each kind, each group's
// generation and what it holds, and the Redis server they share.
func (h *Handler) Overview(c *gin.Context) {
	databases := h.redis.Databases()
	if databases == nil {
		c.JSON(http.StatusOK, overviewResponse{Configured: false, Databases: []databaseResponse{}})
		return
	}

	out := overviewResponse{Configured: true}
	for _, database := range databases {
		stats, err := database.Stats(c.Request.Context())
		if err != nil {
			h.log.Warn("reading a redis database failed", "database", database.Name(), "error", err)
			stats = cache.Stats{Name: database.Name(), Number: database.Number()}
		}
		out.Databases = append(out.Databases, newDatabaseResponse(stats))
	}

	c.JSON(http.StatusOK, out)
}

// Keys lists a page of one database's keys.
func (h *Handler) Keys(c *gin.Context) {
	database, ok := h.database(c)
	if !ok {
		return
	}

	query, err := parseKeyQuery(c)
	if err != nil {
		respond.Failure(c, h.log, err, "reading a key listing failed")
		return
	}

	keys, next, err := database.Keys(c.Request.Context(), query)
	if err != nil {
		respond.Failure(c, h.log, err, "listing redis keys failed")
		return
	}

	c.JSON(http.StatusOK, newKeysResponse(keys, next))
}

// Key answers one key and what it holds.
func (h *Handler) Key(c *gin.Context) {
	database, name, ok := h.key(c)
	if !ok {
		return
	}

	value, err := database.Read(c.Request.Context(), name)
	if err != nil {
		h.fail(c, err, "reading a redis key failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": newValueResponse(value)})
}

// UpdateKey replaces a cached value by hand. Only a value of a group's
// current generation in the cache database can be: see cache.Write.
func (h *Handler) UpdateKey(c *gin.Context) {
	database, name, ok := h.key(c)
	if !ok {
		return
	}

	var req writeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	value, ttl, err := req.check()
	if err != nil {
		respond.Failure(c, h.log, err, "checking a cache value failed")
		return
	}

	ctx := c.Request.Context()
	if err := database.Write(ctx, name, value, ttl); err != nil {
		h.fail(c, err, "writing a redis key failed")
		return
	}

	h.audit.RecordWith(c, "cache.entry_updated", targetType, database.Name(), map[string]any{"key": name})

	written, err := database.Read(ctx, name)
	if err != nil {
		h.fail(c, err, "reading a redis key failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": newValueResponse(written)})
}

// DeleteKey removes one key. The next request that wanted it reads the
// database again.
func (h *Handler) DeleteKey(c *gin.Context) {
	database, name, ok := h.key(c)
	if !ok {
		return
	}

	removed, err := database.Delete(c.Request.Context(), name)
	if err != nil {
		h.fail(c, err, "removing a redis key failed")
		return
	}
	if !removed {
		respond.Fail(c, keyNotFound)
		return
	}

	h.audit.RecordWith(c, "cache.entry_deleted", targetType, database.Name(), map[string]any{"key": name})

	c.Status(http.StatusNoContent)
}

// ClearGroup forgets everything cached in one group, for every server
// process at once.
func (h *Handler) ClearGroup(c *gin.Context) {
	database, ok := h.database(c)
	if !ok {
		return
	}

	group := c.Param("group")
	if !slices.Contains(cache.Groups[database.Name()], group) {
		respond.Fail(c, groupNotFound)
		return
	}

	if err := database.Clear(c.Request.Context(), group); err != nil {
		respond.Failure(c, h.log, err, "clearing a cache group failed")
		return
	}

	h.audit.RecordWith(c, "cache.group_cleared", targetType, database.Name(), map[string]any{"group": group})

	c.Status(http.StatusNoContent)
}

// Flush removes every key this server keeps in one database. Flushing sessions
// signs nobody out, but resets every rate limit.
func (h *Handler) Flush(c *gin.Context) {
	database, ok := h.database(c)
	if !ok {
		return
	}

	if err := database.Flush(c.Request.Context()); err != nil {
		respond.Failure(c, h.log, err, "flushing a redis database failed")
		return
	}

	h.audit.Record(c, "cache.flushed", targetType, database.Name())

	c.Status(http.StatusNoContent)
}

func (h *Handler) database(c *gin.Context) (*cache.Cache, bool) {
	if h.redis.Databases() == nil {
		respond.Fail(c, notConfigured)
		return nil, false
	}

	switch c.Param("database") {
	case cache.CacheDatabase:
		return h.redis.Cache, true
	case cache.SessionDatabase:
		return h.redis.Sessions, true
	}

	respond.Fail(c, databaseNotFound)
	return nil, false
}

func (h *Handler) key(c *gin.Context) (*cache.Cache, string, bool) {
	database, ok := h.database(c)
	if !ok {
		return nil, "", false
	}

	name, err := keyName(c)
	if err != nil {
		respond.Failure(c, h.log, err, "reading a key name failed")
		return nil, "", false
	}

	return database, name, true
}

func (h *Handler) fail(c *gin.Context, err error, message string) {
	switch {
	case errors.Is(err, cache.ErrNoKey):
		respond.Fail(c, keyNotFound)
	case errors.Is(err, cache.ErrNotEditable):
		respond.Fail(c, keyNotEditable)
	case errors.Is(err, cache.ErrGenerationKey):
		respond.Fail(c, generationKey)
	case errors.Is(err, cache.ErrNoRedis):
		respond.Fail(c, notConfigured)
	default:
		respond.Failure(c, h.log, err, message)
	}
}
