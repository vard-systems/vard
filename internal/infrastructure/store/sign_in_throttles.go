package store

import (
	"context"
	"slices"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/domain/model"
)

// ReserveSignIn counts an attempt against every key before the attempt is
// checked, so parallel guesses cannot slip past the count, and answers whether
// it may go ahead. While any key is waiting the attempt is refused and counted
// against none. Keys are locked in order, so two attempts sharing keys never
// deadlock.
func (s *Store) ReserveSignIn(ctx context.Context, keys []model.ThrottleKey, now time.Time) (bool, error) {
	sorted := slices.Clone(keys)
	slices.SortFunc(sorted, func(a, b model.ThrottleKey) int {
		switch {
		case a.Key < b.Key:
			return -1
		case a.Key > b.Key:
			return 1
		}
		return 0
	})

	allowed := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows := make([]model.SignInThrottle, len(sorted))
		for i, key := range sorted {
			fresh := model.SignInThrottle{Key: key.Key, ExpiresAt: now.Add(model.ThrottleMemory)}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&fresh).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&rows[i], "key = ?", key.Key).Error; err != nil {
				return err
			}

			// A count past its memory starts again.
			if !now.Before(rows[i].ExpiresAt) {
				rows[i].Failures, rows[i].BlockedUntil = 0, nil
			}
			if rows[i].BlockedUntil != nil && now.Before(*rows[i].BlockedUntil) {
				return nil
			}
		}

		allowed = true
		for i, key := range sorted {
			row := &rows[i]
			row.Failures++
			row.ExpiresAt = now.Add(model.ThrottleMemory)
			if wait := model.ThrottleWait(row.Failures, key.Free); wait > 0 {
				until := now.Add(wait)
				row.BlockedUntil = &until
			}

			err := tx.Model(&model.SignInThrottle{}).Where("id = ?", row.ID).Updates(map[string]any{
				"failures": row.Failures, "blocked_until": row.BlockedUntil, "expires_at": row.ExpiresAt,
			}).Error
			if err != nil {
				return err
			}
		}

		return nil
	})

	return allowed, err
}

// ClearSignIn forgets the counts of keys an attempt has just succeeded
// against.
func (s *Store) ClearSignIn(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Where("key IN ?", keys).Delete(&model.SignInThrottle{}).Error
}

// ReturnSignIn gives back the attempt a successful sign-in was counted for,
// on a key a success must not clear: a network address, where a stranger's
// own account would otherwise wipe the count of their guesses at others. A
// wait that very attempt started ends with it: the count is back under it.
func (s *Store) ReturnSignIn(ctx context.Context, key model.ThrottleKey) error {
	return s.db.WithContext(ctx).Model(&model.SignInThrottle{}).
		Where("key = ? AND failures > 0", key.Key).
		Updates(map[string]any{
			"failures":      gorm.Expr("failures - 1"),
			"blocked_until": gorm.Expr("CASE WHEN failures - 1 < ? THEN NULL ELSE blocked_until END", key.Free),
		}).Error
}
