package store

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"loginer/internal/cache"
	"loginer/internal/model"
)

type ApplicationQuery struct {
	Search  string
	Type    model.ApplicationType
	Enabled *bool
	Only    []uuid.UUID
	Limit   int
	Offset  int
}

func (s *Store) Applications(ctx context.Context, q ApplicationQuery) ([]model.Application, int64, error) {
	query := s.db.WithContext(ctx).Model(&model.Application{})

	if search := strings.TrimSpace(q.Search); search != "" {
		like := contains(search)
		query = query.Where(
			"LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(client_id) LIKE ?",
			like, like, like,
		)
	}

	if q.Type != "" {
		query = query.Where("type = ?", q.Type)
	}

	if q.Enabled != nil {
		query = query.Where("is_enabled = ?", *q.Enabled)
	}

	if q.Only != nil {
		if len(q.Only) == 0 {
			return []model.Application{}, 0, nil
		}
		query = query.Where("id IN ?", q.Only)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var apps []model.Application
	err := query.Order("LOWER(name), created_at").Limit(q.Limit).Offset(q.Offset).Find(&apps).Error

	return apps, total, err
}

func (s *Store) Application(ctx context.Context, id uuid.UUID) (*model.Application, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).First(&app, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}

	return &app, nil
}

func (s *Store) ApplicationRoleCounts(ctx context.Context) (map[uuid.UUID]int64, error) {
	return s.countBy(ctx, "user_roles", "application_id", nil)
}

func (s *Store) CreateApplication(ctx context.Context, app *model.Application) error {
	return translate(s.db.WithContext(ctx).Create(app).Error)
}

func (s *Store) SaveApplication(ctx context.Context, app *model.Application) error {
	return s.forgetting(ctx, translate(s.db.WithContext(ctx).Save(app).Error), cache.Clients, cache.Admins)
}

func (s *Store) DeleteApplication(ctx context.Context, app *model.Application) error {
	// Administrators with a role scoped to it lose that role, so every one of
	// them is read again.
	return s.forgetting(ctx, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roles := "SELECT id FROM user_roles WHERE application_id = @id"

		statements := []string{
			"DELETE FROM user_role_members WHERE user_role_id IN (" + roles + ")",
			"DELETE FROM user_role_inherits WHERE role_id IN (" + roles + ") OR inherited_role_id IN (" + roles + ")",
			"DELETE FROM user_role_api_scopes WHERE user_role_id IN (" + roles + ")",
			"DELETE FROM user_roles WHERE application_id = @id",
			"DELETE FROM admin_role_assignments WHERE application_id = @id",
			"DELETE FROM application_api_scopes WHERE application_id = @id",
			"DELETE FROM application_apis WHERE application_id = @id",
		}

		for _, statement := range statements {
			if err := tx.Exec(statement, map[string]any{"id": app.ID}).Error; err != nil {
				return err
			}
		}

		return tx.Delete(app).Error
	}), cache.Clients, cache.Admins, cache.Grants)
}

func (s *Store) ApplicationByClientID(ctx context.Context, clientID string) (*model.Application, error) {
	kept, err := cached(ctx, s, cache.Clients, "client_id:"+clientID, func() (client, error) {
		var app model.Application
		if err := s.db.WithContext(ctx).First(&app, "client_id = ?", clientID).Error; err != nil {
			return client{}, translate(err)
		}

		return client{Application: app, SecretHash: app.ClientSecretHash}, nil
	})
	if err != nil {
		return nil, err
	}

	app := kept.Application
	app.ClientSecretHash = kept.SecretHash

	return &app, nil
}

// client is how an application is cached, including its secret hash (a SHA-256
// of a random secret) which the model omits from JSON.
type client struct {
	Application model.Application `json:"application"`
	SecretHash  string            `json:"secret_hash"`
}
