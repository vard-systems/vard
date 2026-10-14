package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"loginer/internal/model"
)

type UserQuery struct {
	Search   string
	Verified *bool
	Role     *uuid.UUID
	Limit    int
	Offset   int
}

// userSearch is the record as one lower-cased text, searched by one box. It
// must match idx_users_search exactly, because Postgres uses a trigram index
// only for the expression it was built on.
const userSearch = `LOWER(email || ' ' || COALESCE(first_name, '') || ' ' || COALESCE(last_name, '') || ' ' || COALESCE(data, ''))`

func (s *Store) Users(ctx context.Context, q UserQuery) ([]model.User, int64, error) {
	query := s.db.WithContext(ctx).Model(&model.User{})

	if search := strings.TrimSpace(q.Search); search != "" {
		query = query.Where(userSearch+" LIKE ?", contains(search))
	}

	if q.Verified != nil {
		query = query.Where("is_email_verified = ?", *q.Verified)
	}

	if q.Role != nil {
		query = query.Where(
			"id IN (SELECT user_id FROM user_role_members WHERE user_role_id = ?)", *q.Role,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []model.User
	err := query.
		Preload("Roles", byName).
		Order("created_at DESC").
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&users).Error

	return users, total, err
}

func (s *Store) User(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).Preload("Roles", byName).First(&user, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}

	return &user, nil
}

func (s *Store) CreateUser(ctx context.Context, user *model.User) error {
	return translate(s.db.WithContext(ctx).Omit("Roles.*").Create(user).Error)
}

func (s *Store) SaveUser(ctx context.Context, user *model.User) error {
	return translate(s.db.WithContext(ctx).Omit(clause.Associations).Save(user).Error)
}

// SaveUserProfile writes only the name, so a stale snapshot cannot roll back a
// reset or deactivation committed meanwhile.
func (s *Store) SaveUserProfile(ctx context.Context, user *model.User) error {
	return translate(s.db.WithContext(ctx).Model(user).Updates(map[string]any{
		"first_name": user.FirstName,
		"last_name":  user.LastName,
	}).Error)
}

func (s *Store) DeleteUser(ctx context.Context, user *model.User) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM user_role_members WHERE user_id = ?", user.ID).Error; err != nil {
			return err
		}

		return tx.Delete(user).Error
	})
}

func (s *Store) FieldValueTaken(ctx context.Context, field string, value any, except uuid.UUID) (bool, error) {
	text := fmt.Sprintf("%v", value)
	if number, ok := value.(float64); ok {
		text = strconv.FormatFloat(number, 'f', -1, 64)
	}

	query := s.db.WithContext(ctx).
		Model(&model.User{}).
		Where("data::jsonb ->> ? = ?", field, text)

	if except != uuid.Nil {
		query = query.Where("id <> ?", except)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := s.db.WithContext(ctx).Preload("Roles", byName).First(&user, "email = ?", model.NormalizeEmail(email)).Error
	if err != nil {
		return nil, translate(err)
	}

	return &user, nil
}

// MarkUserSignedIn records the sign-in and clears failed attempts. It writes
// only these columns, so roles removed by SSO role sync during the sign-in are
// not written back.
func (s *Store) MarkUserSignedIn(ctx context.Context, user *model.User, at time.Time) error {
	return s.db.WithContext(ctx).Model(user).Omit(clause.Associations).Updates(map[string]any{
		"last_login_at":      at,
		"failed_login_count": 0,
		"locked_until":       nil,
	}).Error
}

func (s *Store) ReserveUserLogin(ctx context.Context, user *model.User, at time.Time, max int, lockFor time.Duration) (bool, error) {
	var rows []struct {
		LockedUntil *time.Time
	}

	// The WHERE matches no row while locked, so nothing is counted; otherwise
	// the attempt is counted and the `max`th locks.
	err := s.db.WithContext(ctx).Raw(`
		UPDATE users SET
			locked_until = CASE WHEN failed_login_count + 1 >= @max THEN @until ELSE locked_until END,
			failed_login_count = CASE WHEN failed_login_count + 1 >= @max THEN 0 ELSE failed_login_count + 1 END
		WHERE id = @id AND (locked_until IS NULL OR locked_until <= @now)
		RETURNING locked_until`,
		map[string]any{"max": max, "until": at.Add(lockFor), "id": user.ID, "now": at},
	).Scan(&rows).Error
	if err != nil {
		return false, err
	}

	return len(rows) > 0, nil
}

func (s *Store) ClearUserFailedLogins(ctx context.Context, user *model.User, at time.Time) error {
	return s.db.WithContext(ctx).Model(user).Omit(clause.Associations).Updates(map[string]any{
		"failed_login_count": 0,
		"locked_until":       nil,
	}).Error
}

func (s *Store) ChangeUserPassword(ctx context.Context, user *model.User, keep uuid.UUID, at time.Time) error {
	var ended []model.UserSession

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(user).Updates(map[string]any{
			"password_hash":         user.PasswordHash,
			"is_password_temporary": false,
		}).Error
		if err != nil {
			return err
		}

		if ended, err = revokeUserSessionsIn(tx, at, "user_id = ? AND id <> ?", user.ID, keep); err != nil {
			return err
		}

		statements := []string{
			"UPDATE refresh_tokens SET revoked_at = @at WHERE user_id = @user AND revoked_at IS NULL",
			"UPDATE password_resets SET used_at = @at WHERE user_id = @user AND used_at IS NULL",
			pendingEmailChangesUsed,
		}
		for _, statement := range statements {
			if err := tx.Exec(statement, map[string]any{"at": at, "user": user.ID, "keep": keep}).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.putUserSessions(ctx, ended...)

	return nil
}
