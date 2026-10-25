package store

import (
	"context"
	"errors"

	"vard/internal/cache"
	"vard/internal/model"
)

func (s *Store) AdminSecurity(ctx context.Context) (*model.AdminSecurity, error) {
	security, err := cached(ctx, s, cache.AdminSecurity, "settings", func() (model.AdminSecurity, error) {
		var security model.AdminSecurity

		err := translate(s.db.WithContext(ctx).Order("created_at").First(&security).Error)
		switch {
		case err == nil:
			return security, nil
		case !errors.Is(err, ErrNotFound):
			return security, err
		}

		return model.DefaultAdminSecurity(), nil
	})
	if err != nil {
		return nil, err
	}

	return &security, nil
}

func (s *Store) EnsureAdminSecurity(ctx context.Context, mfaRequired bool) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.AdminSecurity{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	security := model.AdminSecurity{RequireMFA: mfaRequired}

	return s.forgetting(ctx, translate(s.db.WithContext(ctx).Create(&security).Error), cache.AdminSecurity)
}

func (s *Store) SaveAdminSecurity(ctx context.Context, security *model.AdminSecurity) error {
	return s.forgetting(ctx, translate(s.db.WithContext(ctx).Save(security).Error), cache.AdminSecurity)
}

func (s *Store) AdminMFACounts(ctx context.Context) (total, withMFA int64, err error) {
	if err = s.db.WithContext(ctx).Model(&model.Admin{}).Count(&total).Error; err != nil {
		return 0, 0, err
	}

	err = s.db.WithContext(ctx).
		Model(&model.Admin{}).
		Where("id IN (?)", s.db.Model(&model.MFA{}).
			Select("admin_id").
			Where("confirmed_at IS NOT NULL")).
		Count(&withMFA).Error

	return total, withMFA, err
}
