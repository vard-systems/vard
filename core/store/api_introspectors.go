package store

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"vard/internal/domain/model"
)

// APIIntrospector is an application with whether it may introspect an API's
// tokens.
type APIIntrospector struct {
	Application model.Application
	Listed      bool
}

// APIIntrospectors lists the applications in `only` (every one when nil),
// each saying whether it may introspect the API's tokens.
func (s *Store) APIIntrospectors(ctx context.Context, apiID uuid.UUID, only []uuid.UUID) ([]APIIntrospector, error) {
	apps, _, err := s.Applications(ctx, ApplicationQuery{Only: only, Limit: 1000})
	if err != nil {
		return nil, err
	}

	var listed []uuid.UUID
	if err := s.db.WithContext(ctx).Model(&model.APIIntrospector{}).
		Where("api_id = ?", apiID).Pluck("application_id", &listed).Error; err != nil {
		return nil, err
	}

	out := make([]APIIntrospector, 0, len(apps))
	for _, app := range apps {
		out = append(out, APIIntrospector{Application: app, Listed: slices.Contains(listed, app.ID)})
	}
	return out, nil
}

// SetAPIIntrospectors makes `applications` the ones that may introspect the
// API's tokens, in place of whatever were.
func (s *Store) SetAPIIntrospectors(ctx context.Context, apiID uuid.UUID, applications []uuid.UUID, at time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("api_id = ?", apiID).Delete(&model.APIIntrospector{}).Error; err != nil {
			return err
		}
		if len(applications) == 0 {
			return nil
		}

		rows := make([]model.APIIntrospector, len(applications))
		for i, id := range applications {
			rows[i] = model.APIIntrospector{APIID: apiID, ApplicationID: id, CreatedAt: at}
		}
		return tx.Omit("API", "Application").Create(&rows).Error
	})
}

// MayIntrospect says whether the application is listed for any of the APIs a
// token names in its aud. Read from the database every time: a removal ends
// what the application may read at once.
func (s *Store) MayIntrospect(ctx context.Context, applicationID uuid.UUID, audiences []string) (bool, error) {
	if len(audiences) == 0 {
		return false, nil
	}

	var count int64
	err := s.db.WithContext(ctx).Model(&model.APIIntrospector{}).
		Joins("JOIN apis ON apis.id = api_introspectors.api_id").
		Where("api_introspectors.application_id = ? AND apis.identifier IN ?", applicationID, audiences).
		Limit(1).Count(&count).Error

	return count > 0, err
}
