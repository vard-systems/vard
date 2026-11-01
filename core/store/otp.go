package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"vard/internal/domain/model"
	"vard/internal/infrastructure/cache"
)

func (s *Store) OTPSettings(ctx context.Context) (*model.OTPSettings, error) {
	settings, err := cached(ctx, s, cache.OTPSettings, "settings", func() (model.OTPSettings, error) {
		var settings model.OTPSettings

		err := translate(s.db.WithContext(ctx).Order("created_at").First(&settings).Error)
		switch {
		case err == nil:
			return settings, nil
		case !errors.Is(err, ErrNotFound):
			return settings, err
		}

		return model.DefaultOTPSettings(), nil
	})
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

func (s *Store) EnsureOTPSettings(ctx context.Context) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.OTPSettings{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	settings := model.DefaultOTPSettings()

	return s.forgetting(ctx, translate(s.db.WithContext(ctx).Create(&settings).Error), cache.OTPSettings)
}

// UpdateOTPSettings changes the settings through updateSettings: OTPSettings
// may answer from the cache, which must never be written back.
func (s *Store) UpdateOTPSettings(ctx context.Context, change func(*model.OTPSettings) (bool, error)) (*model.OTPSettings, error) {
	settings, err := updateSettings(ctx, s, model.DefaultOTPSettings, change)
	if err != nil {
		return nil, err
	}

	s.forget(ctx, cache.OTPSettings)

	return settings, nil
}

func (s *Store) CreateLoginCode(ctx context.Context, code *model.LoginCode) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// One sign-in at a time per user, so two started at once leave one
		// live code rather than two.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", code.UserID.String()).Error; err != nil {
			return translate(err)
		}

		// A live code's guess count carries over, so restarting the sign-in
		// does not grant fresh guesses.
		var prior model.LoginCode
		err := tx.Where("user_id = ? AND used_at IS NULL AND expires_at > ?", code.UserID, code.SentAt).
			Order("created_at DESC").First(&prior).Error
		switch {
		case err == nil:
			code.Attempts = prior.Attempts
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return translate(err)
		}

		if err := tx.
			Where("user_id = ? AND used_at IS NULL", code.UserID).
			Delete(&model.LoginCode{}).Error; err != nil {
			return translate(err)
		}

		return translate(tx.Create(code).Error)
	})
}

func (s *Store) LoginCodeByHash(ctx context.Context, hash string) (*model.LoginCode, error) {
	var code model.LoginCode

	err := translate(s.db.WithContext(ctx).
		Preload("User").
		Where("handle_hash = ?", hash).
		First(&code).Error)
	if err != nil {
		return nil, err
	}

	return &code, nil
}

// RecordLoginCodeAttempt counts a guess before it is compared, atomically and
// only while under `max`, so parallel guesses are each counted and at most
// `max` are ever compared. It returns the count and false when none are left.
func (s *Store) RecordLoginCodeAttempt(ctx context.Context, code *model.LoginCode, max int) (int, bool, error) {
	var rows []struct {
		Attempts int
	}

	err := s.db.WithContext(ctx).Raw(
		`UPDATE login_codes SET attempts = attempts + 1
		WHERE id = ? AND used_at IS NULL AND attempts < ?
		RETURNING attempts`,
		code.ID, max,
	).Scan(&rows).Error
	if err != nil {
		return 0, false, translate(err)
	}
	if len(rows) == 0 {
		return code.Attempts, false, nil
	}

	code.Attempts = rows[0].Attempts

	return code.Attempts, true, nil
}

// ConsumeLoginCode marks a code used conditionally, so of two concurrent right
// answers only one makes a session.
func (s *Store) ConsumeLoginCode(ctx context.Context, code *model.LoginCode, now time.Time) (bool, error) {
	result := s.db.WithContext(ctx).
		Model(&model.LoginCode{}).
		Where("id = ? AND used_at IS NULL", code.ID).
		Update("used_at", now)
	if result.Error != nil {
		return false, translate(result.Error)
	}

	if result.RowsAffected == 0 {
		return false, nil
	}

	code.UsedAt = &now

	return true, nil
}

func (s *Store) ResendLoginCode(ctx context.Context, code *model.LoginCode, hash string, sentAt time.Time) error {
	// expires_at is left alone: a code asked for again and again would
	// otherwise never expire.
	err := translate(s.db.WithContext(ctx).Model(code).Updates(map[string]any{
		"code_hash": hash,
		"sent_at":   sentAt,
	}).Error)
	if err != nil {
		return err
	}

	code.CodeHash = hash
	code.SentAt = sentAt

	return nil
}
