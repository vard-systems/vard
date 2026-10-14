package store

import (
	"context"
	"time"

	"gorm.io/gorm"

	"loginer/internal/brand"
	"loginer/internal/model"
)

func (s *Store) SigningKeys(ctx context.Context) ([]model.SigningKey, error) {
	var keys []model.SigningKey
	err := s.db.WithContext(ctx).Order("created_at DESC").Find(&keys).Error

	return keys, err
}

func (s *Store) CreateSigningKey(ctx context.Context, key *model.SigningKey) error {
	return translate(s.db.WithContext(ctx).Create(key).Error)
}

const keyLock = brand.SigningKeysLock

// AddSigningKeyIfDue stores `key` unless its algorithm already has an unretired
// key newer than `dueBefore`. An advisory lock makes one of several servers
// create it; it reports whether this one did.
func (s *Store) AddSigningKeyIfDue(ctx context.Context, key *model.SigningKey, dueBefore time.Time) (bool, error) {
	added := false

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", keyLock).Error; err != nil {
			return err
		}

		var fresh int64
		err := tx.Model(&model.SigningKey{}).
			Where("algorithm = ? AND retired_at IS NULL AND created_at > ?", key.Algorithm, dueBefore).
			Count(&fresh).Error
		if err != nil || fresh > 0 {
			return err
		}

		added = true
		return translate(tx.Create(key).Error)
	})

	return added, err
}

func (s *Store) RetireSigningKeys(ctx context.Context, algorithm string, before, at time.Time) error {
	return s.db.WithContext(ctx).Model(&model.SigningKey{}).
		Where("algorithm = ? AND retired_at IS NULL AND created_at < ?", algorithm, before).
		Update("retired_at", at).Error
}

func (s *Store) DeleteSigningKeys(ctx context.Context, before time.Time) error {
	return s.db.WithContext(ctx).
		Where("retired_at IS NOT NULL AND retired_at < ?", before).
		Delete(&model.SigningKey{}).Error
}

// DeleteSigningKeysExcept removes every key but these: what revoking a
// compromised key means, since a published key is one APIs still trust.
func (s *Store) DeleteSigningKeysExcept(ctx context.Context, keep []string) error {
	return s.db.WithContext(ctx).Where("kid NOT IN ?", keep).Delete(&model.SigningKey{}).Error
}
