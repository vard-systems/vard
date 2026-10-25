package store

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/model"
)

// pendingEmailChangesUsed cancels pending address changes wherever access is
// reset (password reset or change, sign out everywhere); otherwise whoever
// requested one could take the account back.
const pendingEmailChangesUsed = "UPDATE email_verifications SET used_at = @at " +
	"WHERE user_id = @user AND used_at IS NULL AND new_email <> ''"

func (s *Store) CreateEmailVerification(ctx context.Context, verification *model.EmailVerification) error {
	return translate(s.db.WithContext(ctx).Omit(clause.Associations).Create(verification).Error)
}

func (s *Store) EmailVerificationByHash(ctx context.Context, hash string) (*model.EmailVerification, error) {
	var verification model.EmailVerification
	if err := s.db.WithContext(ctx).First(&verification, "token_hash = ?", hash).Error; err != nil {
		return nil, translate(err)
	}

	return &verification, nil
}

func (s *Store) VerifyEmail(ctx context.Context, verification *model.EmailVerification, at time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.EmailVerification{}).
			Where("id = ? AND used_at IS NULL", verification.ID).
			Update("used_at", at)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAlreadyUsed
		}

		err := tx.Model(&model.EmailVerification{}).
			Where("user_id = ? AND used_at IS NULL", verification.UserID).
			Update("used_at", at).Error
		if err != nil {
			return err
		}

		changes := map[string]any{"is_email_verified": true}
		if verification.IsChange() {
			changes["email"] = model.NormalizeEmail(verification.NewEmail)
		}

		err = tx.Model(&model.User{}).Where("id = ?", verification.UserID).Updates(changes).Error

		return translate(err)
	})
}
