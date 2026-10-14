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

type SessionQuery struct {
	// Search is the start of an address — "ada" finds ada@example.com — so it
	// can use the prefix index on users.email rather than read every user.
	Search string
	UserID uuid.UUID
	After  uuid.UUID
	Limit  int
}

type ActiveSession struct {
	model.UserSession
	Email     string
	FirstName string
	LastName  string
}

func (s *Store) ActiveSessions(ctx context.Context, q SessionQuery, now time.Time) ([]ActiveSession, error) {
	query := s.db.WithContext(ctx).
		Table("user_sessions").
		Select("user_sessions.*, users.email, users.first_name, users.last_name").
		Joins("JOIN users ON users.id = user_sessions.user_id").
		Where("user_sessions.revoked_at IS NULL AND user_sessions.expires_at > ?", now)

	if search := model.NormalizeEmail(q.Search); search != "" {
		query = query.Where("users.email LIKE ?", likeEscaper.Replace(search)+"%")
	}
	if q.UserID != uuid.Nil {
		query = query.Where("user_sessions.user_id = ?", q.UserID)
	}
	if q.After != uuid.Nil {
		query = query.Where(
			"(user_sessions.created_at, user_sessions.id) < (SELECT created_at, id FROM user_sessions WHERE id = ?)",
			q.After,
		)
	}

	var sessions []ActiveSession
	err := query.
		Order("user_sessions.created_at DESC, user_sessions.id DESC").
		Limit(q.Limit).
		Scan(&sessions).Error

	return sessions, err
}

func (s *Store) UserSession(ctx context.Context, id uuid.UUID) (*model.UserSession, error) {
	var session model.UserSession
	if err := s.db.WithContext(ctx).First(&session, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}

	return &session, nil
}

func (s *Store) SignOutUser(ctx context.Context, user uuid.UUID, at time.Time) (sessions, tokens int64, err error) {
	var ended []model.UserSession

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		if ended, err = revokeUserSessionsIn(tx, at, "user_id = ? AND expires_at > ?", user, at); err != nil {
			return err
		}
		sessions = int64(len(ended))

		result := tx.Model(&model.RefreshToken{}).
			Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", user, at).
			Update("revoked_at", at)
		if result.Error != nil {
			return result.Error
		}
		tokens = result.RowsAffected

		return tx.Exec(pendingEmailChangesUsed, map[string]any{"at": at, "user": user}).Error
	})
	if err == nil {
		s.putUserSessions(ctx, ended...)
	}

	return sessions, tokens, err
}

func (s *Store) CreateUserSession(ctx context.Context, session *model.UserSession) error {
	if err := translate(s.db.WithContext(ctx).Omit(clause.Associations).Create(session).Error); err != nil {
		return err
	}

	s.putUserSessions(ctx, *session)

	return nil
}

func (s *Store) UserSessionByHash(ctx context.Context, hash string) (*model.UserSession, error) {
	var kept userSession
	if s.sessions.Session(ctx, cache.UserSession, hash, &kept) {
		session := kept.model(hash)
		return &session, nil
	}

	var session model.UserSession
	if err := s.db.WithContext(ctx).First(&session, "token_hash = ?", hash).Error; err != nil {
		return nil, translate(err)
	}

	s.sessions.FillSession(ctx, cache.UserSession, hash, newUserSession(session), session.ExpiresAt)

	return &session, nil
}

func (s *Store) RevokeUserSession(ctx context.Context, id uuid.UUID, at time.Time) error {
	ended, err := revokeUserSessionsIn(s.db.WithContext(ctx), at, "id = ?", id)
	s.putUserSessions(ctx, ended...)

	return err
}

func (s *Store) RevokeUserSessionsFor(ctx context.Context, user uuid.UUID, at time.Time) error {
	ended, err := revokeUserSessionsIn(s.db.WithContext(ctx), at, "user_id = ?", user)
	s.putUserSessions(ctx, ended...)

	return err
}

func (s *Store) ActiveUserSessions(ctx context.Context, user uuid.UUID, now time.Time) ([]model.UserSession, error) {
	var sessions []model.UserSession
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", user, now).
		Order("authenticated_at DESC").
		Find(&sessions).Error

	return sessions, err
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

		ID:        u.ID,
		CreatedAt: u.CreatedAt}

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
