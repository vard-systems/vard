package store

import (
	"context"
	"errors"

	"gorm.io/gorm/clause"

	"vard/internal/domain/model"
)

// ServerSecret is the sealed secret by that name, made with make the first
// time it is asked for. Two servers starting at once keep the same one.
func (s *Store) ServerSecret(ctx context.Context, name string, make func() ([]byte, error)) ([]byte, error) {
	var found model.ServerSecret
	err := s.db.WithContext(ctx).First(&found, "name = ?", name).Error
	if err == nil {
		return found.Value, nil
	}
	if err := translate(err); !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	sealed, err := make()
	if err != nil {
		return nil, err
	}
	fresh := model.ServerSecret{Name: name, Value: sealed}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&fresh).Error; err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).First(&found, "name = ?", name).Error; err != nil {
		return nil, translate(err)
	}
	return found.Value, nil
}
