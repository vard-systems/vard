package store

import (
	"context"
	"errors"

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
