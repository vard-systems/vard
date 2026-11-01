// Package store is the only place that writes queries: everything above it gets
// models and this package's errors.
package store

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"vard/internal/infrastructure/cache"
)

type Store struct {
	db       *gorm.DB
	cache    *cache.Cache
	sessions *cache.Cache

	onLogout func(context.Context, []LogoutNotice)
}

func New(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) WithCache(r *cache.Redis) *Store {
	s.cache = r.CacheDB()
	s.sessions = r.SessionDB()
	return s
}

func (s *Store) in(group string) *cache.Cache {
	if slices.Contains(cache.Groups[cache.SessionDatabase], group) {
		return s.sessions
	}
	return s.cache
}

func cached[T any](ctx context.Context, s *Store, group, field string, load func() (T, error)) (T, error) {
	var value T
	// Write back into the generation read, so a write that forgets the group
	// during `load` leaves this value unread.
	generation, ok := s.in(group).GetAt(ctx, group, field, &value)
	if ok {
		return value, nil
	}

	value, err := load()
	if err != nil {
		return value, err
	}

	s.in(group).SetAt(ctx, group, field, value, generation)

	return value, nil
}

// forget is called only after commit; forgetting earlier would let a reader
// cache the old row again.
func (s *Store) forget(ctx context.Context, groups ...string) {
	ctx, cancel := afterCommit(ctx)
	defer cancel()

	for _, group := range groups {
		s.in(group).Forget(ctx, group)
	}
}

const afterCommitTimeout = 2 * time.Second

// afterCommit is the context for invalidating after a commit: it keeps the
// request's values but not its cancellation, with a short timeout, so a caller
// hanging up cannot leave other processes serving stale data.
func afterCommit(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), afterCommitTimeout)
}

func (s *Store) forgetting(ctx context.Context, err error, groups ...string) error {
	if err == nil {
		s.forget(ctx, groups...)
	}

	return err
}

var ErrNotFound = errors.New("not found")

var ErrDuplicate = errors.New("already exists")

func translate(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return ErrNotFound
	case isUniqueViolation(err):
		return fmt.Errorf("%w: %w", ErrDuplicate, err)
	default:
		return err
	}
}

// uniqueViolation is the SQLSTATE of a clash with a unique index. The code,
// not the message, is read: the message is in the server's lc_messages.
const uniqueViolation = "23505"

func isUniqueViolation(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == uniqueViolation
}

// likeEscaper escapes what LIKE treats specially, so a search for "50%" or
// "first_name" finds those characters rather than matching anything.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func contains(search string) string {
	return "%" + likeEscaper.Replace(strings.ToLower(search)) + "%"
}

// ErrAlreadyUsed is a single-use secret presented again; it differs from
// ErrNotFound because a replay revokes what the first use issued.
var ErrAlreadyUsed = errors.New("already used")

// Ready says Postgres answers. Redis is left out: everything works without
// it, so its being down is no reason to take the server out of service.
func (s *Store) Ready(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
