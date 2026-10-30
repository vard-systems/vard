package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/domain/model"
)

// revokeTokensIn records that the access tokens under each key are revoked at
// `at`, in the transaction that ended what they were issued in. A key revoked
// again keeps its later time and expiry.
func revokeTokensIn(tx *gorm.DB, at time.Time, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	rows := make([]model.TokenRevocation, len(keys))
	for i, key := range keys {
		rows[i] = model.NewTokenRevocation(key, at)
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"revoked_at": gorm.Expr("GREATEST(token_revocations.revoked_at, EXCLUDED.revoked_at)"),
			"expires_at": gorm.Expr("GREATEST(token_revocations.expires_at, EXCLUDED.expires_at)"),
		}),
	}).Create(&rows).Error
}

// RevokeAccessToken ends one access token by its jti.
func (s *Store) RevokeAccessToken(ctx context.Context, jti string, at time.Time) error {
	return revokeTokensIn(s.db.WithContext(ctx), at, model.RevokedToken(jti))
}

// AccessTokenRevoked says whether any of an access token's keys was revoked
// at or after it was issued. It reads the database every time: a revocation
// is never only in a cache.
func (s *Store) AccessTokenRevoked(ctx context.Context, keys []string, issued time.Time) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.TokenRevocation{}).
		Where("key IN ? AND revoked_at >= ?", keys, issued).
		Limit(1).Count(&count).Error

	return count > 0, err
}

// familiesRevokedIn records the refresh families of the tokens matched as
// revoked, so the access tokens issued with them end too.
func familiesRevokedIn(tx *gorm.DB, at time.Time, condition string, args ...any) error {
	var families []uuid.UUID
	if err := tx.Model(&model.RefreshToken{}).Distinct("family_id").Where(condition, args...).Pluck("family_id", &families).Error; err != nil {
		return err
	}

	keys := make([]string, len(families))
	for i, family := range families {
		keys[i] = model.RevokedFamily(family)
	}

	return revokeTokensIn(tx, at, keys...)
}

// revokeUserRefreshIn revokes every live refresh token of a user and the
// access tokens of their families: a session that ended long ago may have
// left a family that outlived it.
func revokeUserRefreshIn(tx *gorm.DB, at time.Time, user uuid.UUID) (int64, error) {
	if err := familiesRevokedIn(tx, at, "user_id = ? AND revoked_at IS NULL", user); err != nil {
		return 0, err
	}

	result := tx.Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", user).
		Update("revoked_at", at)

	return result.RowsAffected, result.Error
}
