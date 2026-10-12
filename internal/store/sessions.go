package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"loginer/internal/cache"
	"loginer/internal/model"
)

// Administrator sessions are cached by token hash. Every write puts the changed
// sessions (revoked ones included) after commit; see internal/cache/sessions.go
// for why an in-flight reader cannot undo that.

func (s *Store) CreateSession(ctx context.Context, session *model.AdminSession) error {
	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return err
	}

	s.putAdminSessions(ctx, *session)

	return nil
}

func (s *Store) SessionByTokenHash(ctx context.Context, hash string) (*model.AdminSession, error) {
	var session model.AdminSession
	if s.sessions.Session(ctx, cache.AdminSession, hash, &session) {
		// The hash is never written into Redis, and a revocation needs it.
		session.TokenHash = hash
		return &session, nil
	}

	if err := s.db.WithContext(ctx).Where("token_hash = ?", hash).First(&session).Error; err != nil {
		return nil, translate(err)
	}

	s.sessions.FillSession(ctx, cache.AdminSession, hash, session, session.ExpiresAt)

	return &session, nil
}

func (s *Store) RevokeSession(ctx context.Context, session *model.AdminSession, at time.Time) error {
	_, err := s.revokeAdminSessions(ctx, at, "id = ?", session.ID)
	return err
}

func (s *Store) SessionsFor(ctx context.Context, adminID uuid.UUID, limit int) ([]model.AdminSession, error) {
	var sessions []model.AdminSession
	err := s.db.WithContext(ctx).
		Where("admin_id = ?", adminID).
		Order("created_at DESC").
		Limit(limit).
		Find(&sessions).Error

	return sessions, err
}

func (s *Store) RevokeSessionsFor(ctx context.Context, adminID uuid.UUID, at time.Time) error {
	_, err := s.revokeAdminSessions(ctx, at, "admin_id = ?", adminID)
	return err
}

func (s *Store) RevokeOtherSessionsFor(ctx context.Context, adminID, keep uuid.UUID, at time.Time) (int64, error) {
	ended, err := s.revokeAdminSessions(ctx, at, "admin_id = ? AND id <> ?", adminID, keep)
	return int64(len(ended)), err
}

func (s *Store) RevokeOwnSession(ctx context.Context, adminID, id uuid.UUID, at time.Time) (bool, error) {
	ended, err := s.revokeAdminSessions(ctx, at, "id = ? AND admin_id = ?", id, adminID)
	return len(ended) > 0, err
}

func (s *Store) revokeAdminSessions(ctx context.Context, at time.Time, condition string, args ...any) ([]model.AdminSession, error) {
	ended, err := revokeAdminSessionsIn(s.db.WithContext(ctx), at, condition, args...)
	if err != nil {
		return nil, err
	}

	s.putAdminSessions(ctx, ended...)

	return ended, nil
}

func revokeAdminSessionsIn(tx *gorm.DB, at time.Time, condition string, args ...any) ([]model.AdminSession, error) {
	var ended []model.AdminSession
	err := tx.Model(&ended).
		Clauses(clause.Returning{}).
		Where("revoked_at IS NULL").
		Where(condition, args...).
		Update("revoked_at", at).Error

	return ended, err
}

func (s *Store) putAdminSessions(ctx context.Context, sessions ...model.AdminSession) {
	ctx, cancel := afterCommit(ctx)
	defer cancel()

	for _, session := range sessions {
		s.sessions.PutSession(ctx, cache.AdminSession, session.TokenHash, session, session.ExpiresAt)
	}
}

type userSession struct {
	ID              uuid.UUID  `json:"id"`
	CreatedAt       time.Time  `json:"created_at"`
	UserID          uuid.UUID  `json:"user_id"`
	AuthenticatedAt time.Time  `json:"authenticated_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
	RevokedAt       *time.Time `json:"revoked_at"`
	IP              string     `json:"ip"`
	UserAgent       string     `json:"user_agent"`
}

func newUserSession(session model.UserSession) userSession {
	return userSession{
		ID:              session.ID,
		CreatedAt:       session.CreatedAt,
		UserID:          session.UserID,
		AuthenticatedAt: session.AuthenticatedAt,
		ExpiresAt:       session.ExpiresAt,
		RevokedAt:       session.RevokedAt,
		IP:              session.IP,
		UserAgent:       session.UserAgent,
	}
}

func (u userSession) model(hash string) model.UserSession {
	session := model.UserSession{
		TokenHash:       hash,
		UserID:          u.UserID,
		AuthenticatedAt: u.AuthenticatedAt,
		ExpiresAt:       u.ExpiresAt,
		RevokedAt:       u.RevokedAt,
		IP:              u.IP,
		UserAgent:       u.UserAgent,
	}
	session.ID = u.ID
	session.CreatedAt = u.CreatedAt

	return session
}

func revokeUserSessionsIn(tx *gorm.DB, at time.Time, condition string, args ...any) ([]model.UserSession, error) {
	var ended []model.UserSession
	err := tx.Model(&ended).
		Clauses(clause.Returning{}).
		Where("revoked_at IS NULL").
		Where(condition, args...).
		Update("revoked_at", at).Error

	return ended, err
}

func (s *Store) putUserSessions(ctx context.Context, sessions ...model.UserSession) {
	ctx, cancel := afterCommit(ctx)
	defer cancel()

	for _, session := range sessions {
		s.sessions.PutSession(ctx, cache.UserSession, session.TokenHash, newUserSession(session), session.ExpiresAt)
	}
}
