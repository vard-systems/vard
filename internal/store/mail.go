package store

import (
	"context"
	"errors"

	"vard/internal/model"
)

// MailSettings returns the oldest (seeded) row, or the default, which sends
// nothing. It is not cached: email is rare, and the row carries a sealed
// password.
func (s *Store) MailSettings(ctx context.Context) (*model.MailSettings, error) {
	var settings model.MailSettings

	err := translate(s.db.WithContext(ctx).Order("created_at").First(&settings).Error)
	switch {
	case err == nil:
		return &settings, nil
	case !errors.Is(err, ErrNotFound):
		return nil, err
	}

	settings = model.DefaultMailSettings()

	return &settings, nil
}

func (s *Store) EnsureMailSettings(ctx context.Context, settings model.MailSettings) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.MailSettings{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	return translate(s.db.WithContext(ctx).Create(&settings).Error)
}

func (s *Store) UpdateMailSettings(ctx context.Context, change func(*model.MailSettings) (bool, error)) (*model.MailSettings, error) {
	return updateSettings(ctx, s, model.DefaultMailSettings, change)
}
