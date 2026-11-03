package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/core/cache"
	"vard/core/model"
)

func (s *Store) CreateAdminSession(ctx context.Context, session *model.AdminSession) error {
	return s.db.WithContext(ctx).Create(session).Error
}

// AdminSessionByTokenHash reads a session from the database every time, never
// from Redis, as UserSessionByHash does.
func (s *Store) AdminSessionByTokenHash(ctx context.Context, hash string) (*model.AdminSession, error) {
	var session model.AdminSession
	if err := s.db.WithContext(ctx).Where("token_hash = ?", hash).First(&session).Error; err != nil {
		return nil, translate(err)
	}

	return &session, nil
}

func (s *Store) RevokeAdminSession(ctx context.Context, session *model.AdminSession, at time.Time) error {
	_, err := s.revokeAdminSessions(ctx, at, "id = ?", session.ID)
	return err
}

func (s *Store) AdminSessionsFor(ctx context.Context, adminID uuid.UUID, limit int) ([]model.AdminSession, error) {
	var sessions []model.AdminSession
	err := s.db.WithContext(ctx).
		Where("admin_id = ?", adminID).
		Order("created_at DESC").
		Limit(limit).
		Find(&sessions).Error

	return sessions, err
}

func (s *Store) RevokeAdminSessionsFor(ctx context.Context, adminID uuid.UUID, at time.Time) error {
	_, err := s.revokeAdminSessions(ctx, at, "admin_id = ?", adminID)
	return err
}

func (s *Store) RevokeOtherAdminSessionsFor(ctx context.Context, adminID, keep uuid.UUID, at time.Time) (int64, error) {
	ended, err := s.revokeAdminSessions(ctx, at, "admin_id = ? AND id <> ?", adminID, keep)
	return int64(len(ended)), err
}

// ChangeAdminPassword writes an administrator's new password and ends every
// other session of theirs in one transaction: a password changed while a
// session someone else holds lives on would leave them in.
func (s *Store) ChangeAdminPassword(ctx context.Context, admin *model.Admin, keep uuid.UUID, at time.Time) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(admin).Update("password_hash", admin.PasswordHash).Error; err != nil {
			return err
		}

		_, err := revokeAdminSessionsIn(tx, at, "admin_id = ? AND id <> ?", admin.ID, keep)
		return err
	})
	if err != nil {
		return err
	}

	s.forget(ctx, cache.Admins)

	return nil
}

func (s *Store) RevokeOwnAdminSession(ctx context.Context, adminID, id uuid.UUID, at time.Time) (bool, error) {
	ended, err := s.revokeAdminSessions(ctx, at, "id = ? AND admin_id = ?", id, adminID)
	return len(ended) > 0, err
}

func (s *Store) revokeAdminSessions(ctx context.Context, at time.Time, condition string, args ...any) ([]model.AdminSession, error) {
	return revokeAdminSessionsIn(s.db.WithContext(ctx), at, condition, args...)
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

// TouchAdminSession records that a session was used, at most once every
// model.SeenEvery, so its idle timeout runs from its last use.
func (s *Store) TouchAdminSession(ctx context.Context, session *model.AdminSession, at time.Time) error {
	err := s.db.WithContext(ctx).Model(&model.AdminSession{}).
		Where("id = ? AND revoked_at IS NULL AND last_seen_at < ?", session.ID, at.Add(-model.SeenEvery)).
		Update("last_seen_at", at).Error
	if err != nil {
		return err
	}

	session.LastSeenAt = at

	return nil
}

// PassAdminSessionMFA ends the session a password opened and starts `next`,
// its successor with the second factor passed, in one transaction: the token
// a sign-in had before its second factor is worth nothing after it.
// ErrAlreadyUsed when `current` has ended meanwhile.
func (s *Store) PassAdminSessionMFA(ctx context.Context, current, next *model.AdminSession, at time.Time) error {
	var ended []model.AdminSession

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		if ended, err = revokeAdminSessionsIn(tx, at, "id = ?", current.ID); err != nil {
			return err
		}
		if len(ended) == 0 {
			return ErrAlreadyUsed
		}

		next.IsMFAPassed = true
		return tx.Create(next).Error
	})
	return err
}
