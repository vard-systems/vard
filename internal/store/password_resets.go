package store

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"loginer/internal/model"
)

func (s *Store) CreatePasswordReset(ctx context.Context, reset *model.PasswordReset) error {
	return translate(s.db.WithContext(ctx).Omit(clause.Associations).Create(reset).Error)
}

const passwordResetsPerHour = 5

// CreatePasswordResetThrottled stores a reset link only if the account got none
// in the last minute and fewer than passwordResetsPerHour in the hour. A
// per-account advisory lock stops a burst from all passing the check.
func (s *Store) CreatePasswordResetThrottled(ctx context.Context, reset *model.PasswordReset, now time.Time) (bool, error) {
	created := false

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", reset.UserID.String()).Error; err != nil {
			return translate(err)
		}

		var lastMinute, lastHour int64
		if err := tx.Model(&model.PasswordReset{}).
			Where("user_id = ? AND created_at > ?", reset.UserID, now.Add(-time.Minute)).
			Count(&lastMinute).Error; err != nil {
			return translate(err)
		}
		if err := tx.Model(&model.PasswordReset{}).
			Where("user_id = ? AND created_at > ?", reset.UserID, now.Add(-time.Hour)).
			Count(&lastHour).Error; err != nil {
			return translate(err)
		}
		if lastMinute > 0 || lastHour >= passwordResetsPerHour {
			return nil
		}

		created = true

		return translate(tx.Omit(clause.Associations).Create(reset).Error)
	})

	return created, err
}

func (s *Store) PasswordResetByHash(ctx context.Context, hash string) (*model.PasswordReset, error) {
	var reset model.PasswordReset
	if err := s.db.WithContext(ctx).First(&reset, "token_hash = ?", hash).Error; err != nil {
		return nil, translate(err)
	}

	return &reset, nil
}

func (s *Store) ResetPassword(ctx context.Context, reset *model.PasswordReset, user *model.User, at time.Time) error {
	var ended endedSessions

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.PasswordReset{}).
			Where("id = ? AND used_at IS NULL", reset.ID).
			Update("used_at", at)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAlreadyUsed
		}

		changes := map[string]any{
			"password_hash":         user.PasswordHash,
			"is_password_temporary": false,
			"failed_login_count":    0,
			"locked_until":          nil,
		}
		// An emailed link proves the address; the link a temporary password
		// hands out does not.
		if !user.IsPasswordTemporary {
			changes["is_email_verified"] = true
		}
		err := tx.Model(user).Updates(changes).Error
		if err != nil {
			return err
		}

		if ended, err = revokeUserSessionsIn(tx, at, "user_id = ?", user.ID); err != nil {
			return err
		}

		statements := []string{
			"UPDATE refresh_tokens SET revoked_at = @at WHERE user_id = @user AND revoked_at IS NULL",
			"UPDATE password_resets SET used_at = @at WHERE user_id = @user AND used_at IS NULL",
			pendingEmailChangesUsed,
		}
		for _, statement := range statements {
			if err := tx.Exec(statement, map[string]any{"at": at, "user": user.ID}).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.sessionsEnded(ctx, ended)

	return nil
}
