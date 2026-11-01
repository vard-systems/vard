package store

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/core/cache"
	"vard/core/model"
)

type RoleQuery struct {
	Global       bool
	Applications []uuid.UUID
	Search       string
	Default      *bool
	Limit        int
	Offset       int
}

func byName(db *gorm.DB) *gorm.DB {
	return db.Order("name")
}

func (s *Store) UserRoles(ctx context.Context, q RoleQuery) ([]model.UserRole, int64, error) {
	query := s.db.WithContext(ctx).Model(&model.UserRole{})

	scopes := s.db.Where("1 = 0")
	if q.Global {
		scopes = scopes.Or("application_id IS NULL")
	}
	switch {
	case q.Applications == nil:
		scopes = scopes.Or("application_id IS NOT NULL")
	case len(q.Applications) > 0:
		scopes = scopes.Or("application_id IN ?", q.Applications)
	}
	query = query.Where(scopes)

	if search := strings.TrimSpace(q.Search); search != "" {
		like := contains(search)
		query = query.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ?", like, like)
	}

	if q.Default != nil {
		query = query.Where("is_default = ?", *q.Default)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var roles []model.UserRole
	err := query.
		Preload("Inherits", byName).
		Preload("APIScopes", scopesByName).
		Order("application_id IS NOT NULL").
		Order("(SELECT LOWER(applications.name) FROM applications WHERE applications.id = user_roles.application_id)").
		Order("name").
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&roles).Error

	return roles, total, err
}

func (s *Store) UserRole(ctx context.Context, id uuid.UUID) (*model.UserRole, error) {
	var role model.UserRole
	err := s.db.WithContext(ctx).
		Preload("Inherits", byName).
		Preload("APIScopes", scopesByName).
		First(&role, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}

	return &role, nil
}

func (s *Store) UserRolesByID(ctx context.Context, ids []uuid.UUID) ([]model.UserRole, error) {
	roles := []model.UserRole{}

	if err := s.byID(ctx, ids, &roles, func() int { return len(roles) }); err != nil {
		return nil, err
	}

	return roles, nil
}

func (s *Store) DefaultUserRoles(ctx context.Context) ([]model.UserRole, error) {
	var roles []model.UserRole
	err := s.db.WithContext(ctx).
		Where("is_default = ?", true).
		Where("application_id IS NULL OR application_id IN (SELECT id FROM applications WHERE is_enabled)").
		Order("application_id NULLS FIRST, name").
		Find(&roles).Error

	return roles, err
}

func (s *Store) AddUserRoles(ctx context.Context, userID uuid.UUID, roles []model.UserRole) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, role := range roles {
			err := tx.Exec(
				"INSERT INTO user_role_members (user_id, user_role_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
				userID, role.ID,
			).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Store) RemoveUserRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.db.WithContext(ctx).
		Exec("DELETE FROM user_role_members WHERE user_id = ? AND user_role_id = ?", userID, roleID).
		Error
}

func (s *Store) RoleGraph(ctx context.Context) (model.RoleGraph, error) {
	return cached(ctx, s, cache.Grants, "role_graph", func() (model.RoleGraph, error) {
		var roles []model.UserRole
		err := s.db.WithContext(ctx).
			Preload("Inherits", byName).
			Preload("APIScopes", scopesByName).
			Find(&roles).Error
		if err != nil {
			return nil, err
		}

		graph := make(model.RoleGraph, len(roles))
		for _, role := range roles {
			graph[role.ID] = role
		}

		return graph, nil
	})
}

func (s *Store) RoleMemberCounts(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]int64, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]int64{}, nil
	}

	return s.countBy(ctx, "user_role_members", "user_role_id", ids)
}

func (s *Store) CreateUserRole(ctx context.Context, role *model.UserRole) error {
	// The role and the rows of what it includes go together.
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Omit("Inherits.*", "APIScopes.*").Create(role).Error
	})
	return s.forgetting(ctx, translate(err), cache.Grants)
}

func (s *Store) SaveUserRole(ctx context.Context, role *model.UserRole) error {
	return s.forgetting(ctx, translate(s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Save(role).Error; err != nil {
			return err
		}

		if err := tx.Model(role).Association("Inherits").Replace(role.Inherits); err != nil {
			return err
		}

		return tx.Model(role).Association("APIScopes").Replace(role.APIScopes)
	})), cache.Grants)
}

func (s *Store) DeleteUserRole(ctx context.Context, role *model.UserRole) error {
	return s.forgetting(ctx, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		statements := []string{
			"DELETE FROM user_role_inherits WHERE role_id = @id OR inherited_role_id = @id",
			"DELETE FROM user_role_members WHERE user_role_id = @id",
			"DELETE FROM user_role_api_scopes WHERE user_role_id = @id",
		}

		for _, statement := range statements {
			if err := tx.Exec(statement, map[string]any{"id": role.ID}).Error; err != nil {
				return err
			}
		}

		return tx.Delete(role).Error
	}), cache.Grants)
}

func (s *Store) byID(ctx context.Context, ids []uuid.UUID, into any, found func() int) error {
	seen := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		seen[id] = true
	}

	if len(seen) == 0 {
		return nil
	}

	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Order("name").Find(into).Error; err != nil {
		return err
	}

	if found() != len(seen) {
		return ErrNotFound
	}

	return nil
}

func (s *Store) countBy(ctx context.Context, table, column string, only []uuid.UUID) (map[uuid.UUID]int64, error) {
	var rows []struct {
		ID    uuid.UUID
		Count int64
	}

	query := s.db.WithContext(ctx).Table(table).Select(column + " AS id, COUNT(*) AS count").Group(column)
	if only != nil {
		query = query.Where(column+" IN ?", only)
	}

	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	counts := make(map[uuid.UUID]int64, len(rows))
	for _, row := range rows {
		counts[row.ID] = row.Count
	}

	return counts, nil
}

func (s *Store) EffectiveRoles(ctx context.Context, user *model.User) ([]model.UserRole, error) {
	graph, err := s.RoleGraph(ctx)
	if err != nil {
		return nil, err
	}

	held := make([]uuid.UUID, 0, len(user.Roles))
	for _, role := range user.Roles {
		held = append(held, role.ID)
	}

	return graph.Effective(held), nil
}
