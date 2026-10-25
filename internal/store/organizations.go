package store

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"vard/internal/cache"
	"vard/internal/model"
)

func (s *Store) Organization(ctx context.Context) (*model.Organization, error) {
	var organization model.Organization
	if s.cache.Get(ctx, cache.Organization, "settings", &organization) {
		return &organization, nil
	}

	loaded, err := s.loadOrganization(ctx)
	if err != nil {
		return nil, err
	}

	s.cache.Set(ctx, cache.Organization, "settings", *loaded)

	return loaded, nil
}

// loadOrganization reads the database, creating the row on first use.
func (s *Store) loadOrganization(ctx context.Context) (*model.Organization, error) {
	var organization model.Organization

	err := translate(s.db.WithContext(ctx).Order("created_at").First(&organization).Error)
	switch {
	case err == nil:
		return &organization, nil
	case !errors.Is(err, ErrNotFound):
		return nil, err
	}

	organization = model.DefaultOrganization()
	if err := s.db.WithContext(ctx).Create(&organization).Error; err != nil {
		return nil, translate(err)
	}

	s.forget(ctx, cache.Organization)

	return &organization, nil
}

// UpdateOrganization changes the organisation through updateSettings, so the
// row changed is the stored one, never a cached copy.
func (s *Store) UpdateOrganization(ctx context.Context, change func(*model.Organization) (bool, error)) (*model.Organization, error) {
	organization, err := updateSettings(ctx, s, model.DefaultOrganization, change)
	if err != nil {
		return nil, err
	}

	s.forget(ctx, cache.Organization)

	return organization, nil
}

// OrganizationLogo is the organisation's app logo, or ErrNotFound. The bytes
// are never cached: the organisation's cached row carries only their version,
// and a browser keeps the logo by it.
func (s *Store) OrganizationLogo(ctx context.Context) (*model.OrganizationLogo, error) {
	var logo model.OrganizationLogo
	if err := s.db.WithContext(ctx).Order("created_at").First(&logo).Error; err != nil {
		return nil, translate(err)
	}

	return &logo, nil
}

// SetOrganizationLogo stores a logo in place of the one there, and the
// organisation's record of its version, together.
func (s *Store) SetOrganizationLogo(ctx context.Context, logo model.OrganizationLogo) (*model.Organization, error) {
	organization, err := updateSettingsWith(ctx, s, model.DefaultOrganization, func(tx *gorm.DB, organization *model.Organization) (bool, error) {
		// There is one logo at most: the new one replaces whatever is there.
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.OrganizationLogo{}).Error; err != nil {
			return false, err
		}
		if err := tx.Create(&logo).Error; err != nil {
			return false, err
		}

		organization.LogoVersion = logo.Version

		return true, nil
	})
	if err != nil {
		return nil, err
	}

	s.forget(ctx, cache.Organization)

	return organization, nil
}

// RemoveOrganizationLogo removes the logo, and says whether there was one.
func (s *Store) RemoveOrganizationLogo(ctx context.Context) (*model.Organization, bool, error) {
	removed := false

	organization, err := updateSettingsWith(ctx, s, model.DefaultOrganization, func(tx *gorm.DB, organization *model.Organization) (bool, error) {
		result := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.OrganizationLogo{})
		if result.Error != nil {
			return false, result.Error
		}

		removed = result.RowsAffected > 0 || organization.LogoVersion != ""
		organization.LogoVersion = ""

		return removed, nil
	})
	if err != nil {
		return nil, false, err
	}

	if removed {
		s.forget(ctx, cache.Organization)
	}

	return organization, removed, nil
}
