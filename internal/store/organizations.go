package store

import (
	"context"
	"errors"

	"loginer/internal/cache"
	"loginer/internal/model"
)

func (s *Store) Organization(ctx context.Context) (*model.Organization, error) {
	var organization model.Organization
	if s.cache.Get(ctx, cache.Organization, "settings", &organization) {
		return &organization, nil
	}

	loaded, err := s.OrganizationForUpdate(ctx)
	if err != nil {
		return nil, err
	}

	s.cache.Set(ctx, cache.Organization, "settings", *loaded)

	return loaded, nil
}

// OrganizationForUpdate reads the database, never the cache, so a stale cached
// copy is never written back over the row.
func (s *Store) OrganizationForUpdate(ctx context.Context) (*model.Organization, error) {
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

// SaveOrganization updates the existing row only; unlike GORM's Save it never
// inserts a second one, answering ErrNotFound instead.
func (s *Store) SaveOrganization(ctx context.Context, organization *model.Organization) error {
	result := s.db.WithContext(ctx).Model(organization).Select("*").Omit("created_at").Updates(organization)
	if err := translate(result.Error); err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	s.forget(ctx, cache.Organization)

	return nil
}
