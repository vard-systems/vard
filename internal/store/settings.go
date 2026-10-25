package store

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// updateSettings changes the one row of a settings table. The row is read
// locked and written back in the same transaction, so two administrators
// saving different fields at once each keep theirs, and `change` validates
// what is really stored rather than a copy another save has overtaken. A table
// with no row yet starts from `fallback`.
//
// `change` edits the row and reports whether anything changed; nothing is
// written when it did not, or when it fails, and its error is returned as is.
func updateSettings[T any](ctx context.Context, s *Store, fallback func() T, change func(*T) (bool, error)) (*T, error) {
	return updateSettingsWith(ctx, s, fallback, func(_ *gorm.DB, row *T) (bool, error) {
		return change(row)
	})
}

// updateSettingsWith is updateSettings for a change that writes other rows
// too: it gets the transaction, so they are written with the settings row or
// not at all.
func updateSettingsWith[T any](ctx context.Context, s *Store, fallback func() T, change func(*gorm.DB, *T) (bool, error)) (*T, error) {
	var row T

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Order("created_at").First(&row).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			row = fallback()
		case err != nil:
			return err
		}

		changed, err := change(tx, &row)
		if err != nil || !changed {
			return err
		}

		return tx.Save(&row).Error
	})
	if err != nil {
		return nil, translate(err)
	}

	return &row, nil
}
