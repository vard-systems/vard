package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/model"
)

func (s *Store) CreateRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	return translate(s.db.WithContext(ctx).Omit(clause.Associations).Create(token).Error)
}

func (s *Store) RefreshTokenByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	if err := s.db.WithContext(ctx).First(&token, "token_hash = ?", hash).Error; err != nil {
		return nil, translate(err)
	}

	return &token, nil
}

// RotateRefreshToken revokes `old` and stores `next` in one transaction; if
// `old` was already revoked it returns ErrAlreadyUsed.
func (s *Store) RotateRefreshToken(ctx context.Context, old, next *model.RefreshToken, at time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.RefreshToken{}).
			Where("id = ? AND revoked_at IS NULL", old.ID).
			Update("revoked_at", at)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAlreadyUsed
		}

		return translate(tx.Omit(clause.Associations).Create(next).Error)
	})
}

func (s *Store) RevokeRefreshFamily(ctx context.Context, family uuid.UUID, at time.Time) error {
	return s.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", family).
		Update("revoked_at", at).Error
}

func (s *Store) RevokeRefreshTokensForCode(ctx context.Context, code uuid.UUID, at time.Time) error {
	return s.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("code_id = ? AND revoked_at IS NULL", code).
		Update("revoked_at", at).Error
}

func (s *Store) RevokeRefreshTokensForUser(ctx context.Context, user uuid.UUID, at time.Time) error {
	return s.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", user).
		Update("revoked_at", at).Error
}

type Grant struct {
	Application model.Application
	Scopes      []string
	FirstIssued time.Time
	LastIssued  time.Time
}

func (s *Store) UserGrants(ctx context.Context, user uuid.UUID, now time.Time) ([]Grant, error) {
	var tokens []model.RefreshToken
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", user, now).
		Order("created_at DESC").
		Find(&tokens).Error
	if err != nil {
		return nil, err
	}

	byApp := map[uuid.UUID]*Grant{}
	order := []uuid.UUID{}
	for _, token := range tokens {
		grant, seen := byApp[token.ApplicationID]
		if !seen {
			grant = &Grant{FirstIssued: token.AuthenticatedAt, LastIssued: token.CreatedAt}
			byApp[token.ApplicationID] = grant
			order = append(order, token.ApplicationID)
		}

		grant.Scopes = append(grant.Scopes, token.Scope)
		if token.AuthenticatedAt.Before(grant.FirstIssued) {
			grant.FirstIssued = token.AuthenticatedAt
		}
	}

	if len(order) == 0 {
		return []Grant{}, nil
	}

	var apps []model.Application
	if err := s.db.WithContext(ctx).Where("id IN ?", order).Find(&apps).Error; err != nil {
		return nil, err
	}
	found := make(map[uuid.UUID]model.Application, len(apps))
	for _, app := range apps {
		found[app.ID] = app
	}

	out := make([]Grant, 0, len(order))
	for _, id := range order {
		app, ok := found[id]
		if !ok {
			continue
		}

		grant := byApp[id]
		grant.Application = app
		out = append(out, *grant)
	}

	return out, nil
}

func (s *Store) RevokeUserGrant(ctx context.Context, user, application uuid.UUID, at time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("user_id = ? AND application_id = ? AND revoked_at IS NULL", user, application).
		Update("revoked_at", at)

	return result.RowsAffected, result.Error
}
