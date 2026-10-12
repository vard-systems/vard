package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"loginer/internal/model"
)

type UserQuery struct {
	Search   string
	Verified *bool
	Role     *uuid.UUID
	Limit    int
	Offset   int
}

// userSearch is the record as one lower-cased text, searched by one box. It
// must match idx_users_search exactly, because Postgres uses a trigram index
// only for the expression it was built on.
const userSearch = `LOWER(email || ' ' || COALESCE(first_name, '') || ' ' || COALESCE(last_name, '') || ' ' || COALESCE(data, ''))`

func (s *Store) Users(ctx context.Context, q UserQuery) ([]model.User, int64, error) {
	query := s.db.WithContext(ctx).Model(&model.User{})

	if search := strings.TrimSpace(q.Search); search != "" {
		query = query.Where(userSearch+" LIKE ?", contains(search))
	}

	if q.Verified != nil {
		query = query.Where("is_email_verified = ?", *q.Verified)
	}

	if q.Role != nil {
		query = query.Where(
			"id IN (SELECT user_id FROM user_role_members WHERE user_role_id = ?)", *q.Role,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []model.User
	err := query.
		Preload("Roles", byName).
		Order("created_at DESC").
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&users).Error

	return users, total, err
}

func (s *Store) User(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).Preload("Roles", byName).First(&user, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}

	return &user, nil
}

func (s *Store) CreateUser(ctx context.Context, user *model.User) error {
	return translate(s.db.WithContext(ctx).Omit("Roles.*").Create(user).Error)
}

func (s *Store) SaveUser(ctx context.Context, user *model.User) error {
	return translate(s.db.WithContext(ctx).Omit(clause.Associations).Save(user).Error)
}

// SaveUserProfile writes only the name, so a stale snapshot cannot roll back a
// reset or deactivation committed meanwhile.
func (s *Store) SaveUserProfile(ctx context.Context, user *model.User) error {
	return translate(s.db.WithContext(ctx).Model(user).Updates(map[string]any{
		"first_name": user.FirstName,
		"last_name":  user.LastName,
	}).Error)
}

func (s *Store) DeleteUser(ctx context.Context, user *model.User) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM user_role_members WHERE user_id = ?", user.ID).Error; err != nil {
			return err
		}

		return tx.Unscoped().Delete(user).Error
	})
}

func (s *Store) FieldValueTaken(ctx context.Context, field string, value any, except uuid.UUID) (bool, error) {
	text := fmt.Sprintf("%v", value)
	if number, ok := value.(float64); ok {
		text = strconv.FormatFloat(number, 'f', -1, 64)
	}

	query := s.db.WithContext(ctx).
		Model(&model.User{}).
		Where("data::jsonb ->> ? = ?", field, text)

	if except != uuid.Nil {
		query = query.Where("id <> ?", except)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
