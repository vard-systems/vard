package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/model"
)

// UserFactors are a user's second factors, confirmed or still being set up,
// oldest first.
func (s *Store) UserFactors(ctx context.Context, user uuid.UUID) ([]model.UserFactor, error) {
	var factors []model.UserFactor
	err := s.db.WithContext(ctx).Where("user_id = ?", user).Order("created_at").Find(&factors).Error

	return factors, err
}

// HasUserFactor says whether a user has a confirmed second factor, which is
// what makes their sign-ins ask for one.
func (s *Store) HasUserFactor(ctx context.Context, user uuid.UUID) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.UserFactor{}).
		Where("user_id = ? AND confirmed_at IS NOT NULL", user).
		Count(&count).Error

	return count > 0, err
}

// StartUserFactor stores a factor being set up, in place of any other of its
// kind left unfinished.
func (s *Store) StartUserFactor(ctx context.Context, factor *model.UserFactor) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("user_id = ? AND method = ? AND confirmed_at IS NULL", factor.UserID, factor.Method).
			Delete(&model.UserFactor{}).Error
		if err != nil {
			return err
		}

		return tx.Omit(clause.Associations).Create(factor).Error
	})
}

// ConfirmUserFactor finishes setting a factor up with what it holds from now
// on. Only an unconfirmed one is changed, so a ceremony answered twice
// confirms once: ErrAlreadyUsed otherwise.
func (s *Store) ConfirmUserFactor(ctx context.Context, factor *model.UserFactor, at time.Time) error {
	result := s.db.WithContext(ctx).Model(&model.UserFactor{}).
		Where("id = ? AND confirmed_at IS NULL", factor.ID).
		Updates(map[string]any{
			"secret":        factor.Secret,
			"credential_id": factor.CredentialID,
			"label":         factor.Label,
			"confirmed_at":  at,
			"last_used_at":  factor.LastUsedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAlreadyUsed
	}

	factor.ConfirmedAt = &at

	return nil
}

// ClaimUserFactorStep takes an authenticator's step, once: of two sign-ins
// with the same code only one gets it.
func (s *Store) ClaimUserFactorStep(ctx context.Context, factor uuid.UUID, at time.Time) (bool, error) {
	result := s.db.WithContext(ctx).Model(&model.UserFactor{}).
		Where("id = ? AND confirmed_at IS NOT NULL AND (last_used_at IS NULL OR last_used_at < ?)", factor, at).
		Update("last_used_at", at)

	return result.RowsAffected == 1, result.Error
}

// UsedPasskey keeps what a passkey sign-in changed about the credential (its
// counter) and when it was used.
func (s *Store) UsedPasskey(ctx context.Context, factor uuid.UUID, secret string, at time.Time) error {
	return s.db.WithContext(ctx).Model(&model.UserFactor{}).
		Where("id = ?", factor).
		Updates(map[string]any{"secret": secret, "last_used_at": at}).Error
}

// RemoveUserFactor removes one of a user's factors. When it was their last
// confirmed one, their recovery codes go too: with nothing to recover there
// is nothing for them to stand in for. It reports whether anything was
// removed.
func (s *Store) RemoveUserFactor(ctx context.Context, user, factor uuid.UUID) (bool, error) {
	removed := false

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ? AND user_id = ?", factor, user).Delete(&model.UserFactor{})
		if result.Error != nil {
			return result.Error
		}
		removed = result.RowsAffected == 1

		var left int64
		if err := tx.Model(&model.UserFactor{}).
			Where("user_id = ? AND confirmed_at IS NOT NULL", user).
			Count(&left).Error; err != nil {
			return err
		}
		if left > 0 {
			return nil
		}

		return tx.Where("user_id = ?", user).Delete(&model.UserRecoveryCode{}).Error
	})

	return removed, err
}

// ReplaceRecoveryCodes gives a user a new set of recovery codes; the old ones
// stop working.
func (s *Store) ReplaceRecoveryCodes(ctx context.Context, user uuid.UUID, hashes []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", user).Delete(&model.UserRecoveryCode{}).Error; err != nil {
			return err
		}

		codes := make([]model.UserRecoveryCode, len(hashes))
		for i, hash := range hashes {
			codes[i] = model.UserRecoveryCode{UserID: user, CodeHash: hash}
		}

		return tx.Omit(clause.Associations).Create(&codes).Error
	})
}

// ConsumeUserRecoveryCode spends one of a user's recovery codes. It is one
// UPDATE, so a code typed twice at once works once.
// ConsumeUserRecoveryCode spends one of the user's recovery codes, once.
func (s *Store) ConsumeUserRecoveryCode(ctx context.Context, user uuid.UUID, hash string, at time.Time) (bool, error) {
	result := s.db.WithContext(ctx).Model(&model.UserRecoveryCode{}).
		Where("user_id = ? AND code_hash = ? AND used_at IS NULL", user, hash).
		Update("used_at", at)

	return result.RowsAffected == 1, result.Error
}

// RecoveryCodesLeft is how many of a user's recovery codes are unused.
func (s *Store) RecoveryCodesLeft(ctx context.Context, user uuid.UUID) (int, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.UserRecoveryCode{}).
		Where("user_id = ? AND used_at IS NULL", user).
		Count(&count).Error

	return int(count), err
}

func (s *Store) CreatePendingSignIn(ctx context.Context, pending *model.PendingSignIn) error {
	return s.db.WithContext(ctx).Omit(clause.Associations).Create(pending).Error
}

// PendingSignInByHash is a sign-in waiting for its second factor, with its
// user; ErrNotFound if there is none.
func (s *Store) PendingSignInByHash(ctx context.Context, hash string) (*model.PendingSignIn, error) {
	var pending model.PendingSignIn
	if err := s.db.WithContext(ctx).Preload("User").First(&pending, "handle_hash = ?", hash).Error; err != nil {
		return nil, translate(err)
	}

	return &pending, nil
}

// ConsumePendingSignIn spends a waiting sign-in, once: false if it was
// already spent.
func (s *Store) ConsumePendingSignIn(ctx context.Context, pending *model.PendingSignIn, at time.Time) (bool, error) {
	result := s.db.WithContext(ctx).Model(&model.PendingSignIn{}).
		Where("id = ? AND used_at IS NULL", pending.ID).
		Update("used_at", at)
	if result.Error != nil {
		return false, result.Error
	}

	pending.UsedAt = &at

	return result.RowsAffected == 1, nil
}

// OfferPasskey keeps the passkey ceremony a waiting sign-in now offers, in
// place of any it offered before.
func (s *Store) OfferPasskey(ctx context.Context, pending *model.PendingSignIn, ceremony string) error {
	err := s.db.WithContext(ctx).Model(&model.PendingSignIn{}).
		Where("id = ?", pending.ID).
		Update("passkey", ceremony).Error
	if err == nil {
		pending.Passkey = ceremony
	}

	return err
}

// ResetTwoStep removes a user's every factor, recovery code and sign-in
// waiting for one: an administrator's way back in for someone who lost them
// all. It reports how many factors went.
func (s *Store) ResetTwoStep(ctx context.Context, user uuid.UUID) (int64, error) {
	var removed int64

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("user_id = ?", user).Delete(&model.UserFactor{})
		if result.Error != nil {
			return result.Error
		}
		removed = result.RowsAffected

		if err := tx.Where("user_id = ?", user).Delete(&model.UserRecoveryCode{}).Error; err != nil {
			return err
		}

		return tx.Where("user_id = ?", user).Delete(&model.PendingSignIn{}).Error
	})

	return removed, err
}

// UsersWithTwoStep is which of these users have a confirmed second factor.
func (s *Store) UsersWithTwoStep(ctx context.Context, users []uuid.UUID) (map[uuid.UUID]bool, error) {
	var ids []uuid.UUID
	err := s.db.WithContext(ctx).Model(&model.UserFactor{}).
		Distinct("user_id").
		Where("user_id IN ? AND confirmed_at IS NOT NULL", users).
		Pluck("user_id", &ids).Error

	with := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		with[id] = true
	}

	return with, err
}

func (s *Store) CreatePasskeyLogin(ctx context.Context, login *model.PasskeyLogin) error {
	return s.db.WithContext(ctx).Create(login).Error
}

// ClaimPasskeyLogin takes a passkey sign-in's ceremony, once: of two answers
// sent at the same moment only one is checked. ErrNotFound when there is
// none, ErrAlreadyUsed when it was taken or has expired.
func (s *Store) ClaimPasskeyLogin(ctx context.Context, hash string, at time.Time) (*model.PasskeyLogin, error) {
	var login model.PasskeyLogin
	if err := s.db.WithContext(ctx).First(&login, "handle_hash = ?", hash).Error; err != nil {
		return nil, translate(err)
	}

	result := s.db.WithContext(ctx).Model(&model.PasskeyLogin{}).
		Where("id = ? AND used_at IS NULL AND expires_at > ?", login.ID, at).
		Update("used_at", at)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrAlreadyUsed
	}

	return &login, nil
}

// UserFactorByCredential is the confirmed passkey with this credential id.
func (s *Store) UserFactorByCredential(ctx context.Context, user uuid.UUID, credential string) (*model.UserFactor, error) {
	var factor model.UserFactor
	err := s.db.WithContext(ctx).
		First(&factor, "user_id = ? AND credential_id = ? AND confirmed_at IS NOT NULL", user, credential).Error
	if err != nil {
		return nil, translate(err)
	}

	return &factor, nil
}
