package store

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/cache"
	"vard/internal/model"
)

func scopesByName(db *gorm.DB) *gorm.DB {
	return db.Order("name")
}

func (s *Store) APIs(ctx context.Context, search string) ([]model.API, error) {
	query := s.db.WithContext(ctx).Model(&model.API{}).Preload("Scopes", scopesByName)

	if search = strings.TrimSpace(search); search != "" {
		like := contains(search)
		query = query.Where("LOWER(name) LIKE ? OR LOWER(identifier) LIKE ? OR LOWER(description) LIKE ?", like, like, like)
	}

	var apis []model.API
	err := query.Order("LOWER(name)").Find(&apis).Error

	return apis, err
}

func (s *Store) API(ctx context.Context, id uuid.UUID) (*model.API, error) {
	var api model.API
	if err := s.db.WithContext(ctx).Preload("Scopes", scopesByName).First(&api, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}

	return &api, nil
}

func (s *Store) APIByIdentifier(ctx context.Context, identifier string) (*model.API, error) {
	var api model.API
	err := s.db.WithContext(ctx).Preload("Scopes", scopesByName).First(&api, "identifier = ?", identifier).Error
	if err != nil {
		return nil, translate(err)
	}

	return &api, nil
}

func (s *Store) APIApplicationCounts(ctx context.Context) (map[uuid.UUID]int64, error) {
	return s.countBy(ctx, "application_apis", "api_id", nil)
}

func (s *Store) APIRoleCounts(ctx context.Context) (map[uuid.UUID]int64, error) {
	var rows []struct {
		ID    uuid.UUID
		Count int64
	}

	err := s.db.WithContext(ctx).
		Table("user_role_api_scopes").
		Joins("JOIN api_scopes ON api_scopes.id = user_role_api_scopes.api_scope_id").
		Select("api_scopes.api_id AS id, COUNT(DISTINCT user_role_api_scopes.user_role_id) AS count").
		Group("api_scopes.api_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[uuid.UUID]int64, len(rows))
	for _, row := range rows {
		counts[row.ID] = row.Count
	}

	return counts, nil
}

type APIApplication struct {
	Application model.Application
	Authorized  bool
	Allowed     []uuid.UUID
}

func (s *Store) APIApplications(ctx context.Context, apiID uuid.UUID, only []uuid.UUID) ([]APIApplication, error) {
	apps, _, err := s.Applications(ctx, ApplicationQuery{Only: only, Limit: 1000})
	if err != nil {
		return nil, err
	}

	var authorized []uuid.UUID
	if err := s.db.WithContext(ctx).Model(&model.ApplicationAPI{}).
		Where("api_id = ?", apiID).Pluck("application_id", &authorized).Error; err != nil {
		return nil, err
	}

	var rows []model.ApplicationAPIScope
	err = s.db.WithContext(ctx).
		Where("api_scope_id IN (SELECT id FROM api_scopes WHERE api_id = ?)", apiID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]APIApplication, 0, len(apps))
	for _, app := range apps {
		entry := APIApplication{Application: app, Authorized: slices.Contains(authorized, app.ID), Allowed: []uuid.UUID{}}
		for _, row := range rows {
			if row.ApplicationID == app.ID {
				entry.Allowed = append(entry.Allowed, row.APIScopeID)
			}
		}
		out = append(out, entry)
	}

	return out, nil
}

func (s *Store) APIAuditLog(ctx context.Context, apiID uuid.UUID, limit int) ([]model.AuditLog, error) {
	var events []model.AuditLog
	err := s.db.WithContext(ctx).
		// Each side of the OR matches an index (idx_audit_logs_target,
		// idx_audit_logs_api), so this stays an index read.
		Where(`(target_type = ? AND target_id = ?)
			OR (metadata LIKE '%"api_id"%' AND (metadata::jsonb) ->> 'api_id' = ?)`,
			"api", apiID.String(), apiID.String()).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error

	return events, err
}

func (s *Store) APIScopesByID(ctx context.Context, ids []uuid.UUID) ([]model.APIScope, error) {
	scopes := []model.APIScope{}
	if err := s.byID(ctx, ids, &scopes, func() int { return len(scopes) }); err != nil {
		return nil, err
	}

	return scopes, nil
}

func (s *Store) CreateAPI(ctx context.Context, api *model.API) error {
	return s.forgetting(ctx, translate(s.db.WithContext(ctx).Create(api).Error), cache.Grants)
}

func (s *Store) SaveAPI(ctx context.Context, api *model.API) error {
	return s.forgetting(ctx, translate(s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Save(api).Error; err != nil {
			return err
		}

		kept := []uuid.UUID{}
		for _, scope := range api.Scopes {
			if scope.ID != uuid.Nil {
				kept = append(kept, scope.ID)
			}
		}

		gone := tx.Model(&model.APIScope{}).Select("id").Where("api_id = ?", api.ID)
		if len(kept) > 0 {
			gone = gone.Where("id NOT IN ?", kept)
		}

		var goneIDs []uuid.UUID
		if err := gone.Pluck("id", &goneIDs).Error; err != nil {
			return err
		}

		if err := removeScopes(tx, goneIDs); err != nil {
			return err
		}

		// Names are unique per row, so kept scopes are renamed to their own ids
		// first to let two scopes swap names.
		if len(kept) > 0 {
			err := tx.Exec("UPDATE api_scopes SET name = id::text WHERE api_id = ? AND id IN ?", api.ID, kept).Error
			if err != nil {
				return err
			}
		}

		for i := range api.Scopes {
			scope := &api.Scopes[i]
			scope.APIID = api.ID

			if scope.ID == uuid.Nil {
				if err := tx.Create(scope).Error; err != nil {
					return err
				}
				continue
			}

			err := tx.Model(&model.APIScope{}).
				Where("id = ? AND api_id = ?", scope.ID, api.ID).
				Updates(map[string]any{
					"name":        scope.Name,
					"description": scope.Description,
					"is_default":  scope.IsDefault,
				}).Error
			if err != nil {
				return err
			}
		}

		return nil
	})), cache.Grants)
}

func (s *Store) DeleteAPI(ctx context.Context, api *model.API) error {
	return s.forgetting(ctx, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var scopeIDs []uuid.UUID
		if err := tx.Model(&model.APIScope{}).Where("api_id = ?", api.ID).Pluck("id", &scopeIDs).Error; err != nil {
			return err
		}

		if err := removeScopes(tx, scopeIDs); err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM application_apis WHERE api_id = ?", api.ID).Error; err != nil {
			return err
		}

		return tx.Delete(api).Error
	}), cache.Grants)
}

func removeScopes(tx *gorm.DB, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	statements := []string{
		"DELETE FROM application_api_scopes WHERE api_scope_id IN ?",
		"DELETE FROM user_role_api_scopes WHERE api_scope_id IN ?",
		"DELETE FROM api_scopes WHERE id IN ?",
	}

	for _, statement := range statements {
		if err := tx.Exec(statement, ids).Error; err != nil {
			return err
		}
	}

	return nil
}

type APIAccess struct {
	API        model.API
	Authorized bool
	Allowed    []uuid.UUID
}

func (s *Store) ApplicationAPIAccess(ctx context.Context, applicationID uuid.UUID) ([]APIAccess, error) {
	apis, err := s.APIs(ctx, "")
	if err != nil {
		return nil, err
	}

	var authorized []uuid.UUID
	err = s.db.WithContext(ctx).Model(&model.ApplicationAPI{}).
		Where("application_id = ?", applicationID).Pluck("api_id", &authorized).Error
	if err != nil {
		return nil, err
	}

	var allowed []uuid.UUID
	err = s.db.WithContext(ctx).Model(&model.ApplicationAPIScope{}).
		Where("application_id = ?", applicationID).Pluck("api_scope_id", &allowed).Error
	if err != nil {
		return nil, err
	}

	out := make([]APIAccess, 0, len(apis))
	for _, api := range apis {
		access := APIAccess{API: api, Allowed: []uuid.UUID{}}

		for _, id := range authorized {
			if id == api.ID {
				access.Authorized = true
			}
		}

		for _, scope := range api.Scopes {
			for _, id := range allowed {
				if id == scope.ID {
					access.Allowed = append(access.Allowed, scope.ID)
				}
			}
		}

		out = append(out, access)
	}

	return out, nil
}

func (s *Store) AuthorizeApplicationAPI(ctx context.Context, applicationID, apiID uuid.UUID, scopes []uuid.UUID) error {
	return s.forgetting(ctx, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		grant := model.ApplicationAPI{ApplicationID: applicationID, APIID: apiID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&grant).Error; err != nil {
			return err
		}

		err := tx.Exec(
			"DELETE FROM application_api_scopes WHERE application_id = ? AND api_scope_id IN (SELECT id FROM api_scopes WHERE api_id = ?)",
			applicationID, apiID,
		).Error
		if err != nil {
			return err
		}

		for _, scope := range scopes {
			row := model.ApplicationAPIScope{ApplicationID: applicationID, APIScopeID: scope}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
		}

		return nil
	}), cache.Grants)
}

func (s *Store) RevokeApplicationAPI(ctx context.Context, applicationID, apiID uuid.UUID) error {
	return s.forgetting(ctx, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Exec(
			"DELETE FROM application_api_scopes WHERE application_id = ? AND api_scope_id IN (SELECT id FROM api_scopes WHERE api_id = ?)",
			applicationID, apiID,
		).Error
		if err != nil {
			return err
		}

		return tx.Exec("DELETE FROM application_apis WHERE application_id = ? AND api_id = ?", applicationID, apiID).Error
	}), cache.Grants)
}
