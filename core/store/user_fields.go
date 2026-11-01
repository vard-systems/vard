package store

import (
	"context"

	"github.com/google/uuid"

	"vard/core/model"
)

func (s *Store) UserFields(ctx context.Context) ([]model.UserField, error) {
	var fields []model.UserField
	err := s.db.WithContext(ctx).Order("position, created_at").Find(&fields).Error

	return fields, err
}

func (s *Store) UserField(ctx context.Context, id uuid.UUID) (*model.UserField, error) {
	var field model.UserField
	if err := s.db.WithContext(ctx).First(&field, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}

	return &field, nil
}

func (s *Store) CreateUserField(ctx context.Context, field *model.UserField) error {
	return translate(s.db.WithContext(ctx).Create(field).Error)
}

func (s *Store) SaveUserField(ctx context.Context, field *model.UserField) error {
	return translate(s.db.WithContext(ctx).Save(field).Error)
}

func (s *Store) DeleteUserField(ctx context.Context, field *model.UserField) error {
	return s.db.WithContext(ctx).Delete(field).Error
}

func (s *Store) NextFieldPosition(ctx context.Context) int {
	var last model.UserField
	if err := s.db.WithContext(ctx).Order("position DESC").First(&last).Error; err != nil {
		return 1
	}

	return last.Position + 1
}
