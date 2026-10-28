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

// RotateRefreshToken revokes `old`, naming `next` as what replaced it with
// `sealed`, its token sealed, and stores `next`, in one transaction; if `old`
// was already revoked it returns ErrAlreadyUsed. The family's sealed
// successors older than the grace are cleared on the way.
func (s *Store) RotateRefreshToken(ctx context.Context, old, next *model.RefreshToken, sealed []byte, at time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.RefreshToken{}).
			Where("id = ? AND revoked_at IS NULL", old.ID).
			Updates(map[string]any{"revoked_at": at, "successor_id": next.ID, "successor_sealed": sealed})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAlreadyUsed
		}

		if err := tx.Model(&model.RefreshToken{}).
			Where("family_id = ? AND successor_sealed IS NOT NULL AND revoked_at < ?", old.FamilyID, at.Add(-model.RefreshReuseGrace)).
			Update("successor_sealed", nil).Error; err != nil {
			return err
		}

		return translate(tx.Omit(clause.Associations).Create(next).Error)
	})
}

// ForgetRefreshSuccessors clears every sealed successor whose grace is over,
// and says how many it cleared.
func (s *Store) ForgetRefreshSuccessors(ctx context.Context, now time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("successor_sealed IS NOT NULL AND revoked_at < ?", now.Add(-model.RefreshReuseGrace)).
		Update("successor_sealed", nil)
	return result.RowsAffected, result.Error
}

// ForgetRefreshSuccessor clears one replaced token's sealed successor.
func (s *Store) ForgetRefreshSuccessor(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("id = ?", id).
		Update("successor_sealed", nil).Error
}

// RefreshToken is one refresh token by id.
func (s *Store) RefreshToken(ctx context.Context, id uuid.UUID) (*model.RefreshToken, error) {
	var token model.RefreshToken
	if err := s.db.WithContext(ctx).First(&token, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}

	return &token, nil
}

// RevokeRefreshFamily revokes a family's refresh tokens and the access tokens
// issued with them.
func (s *Store) RevokeRefreshFamily(ctx context.Context, family uuid.UUID, at time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&model.RefreshToken{}).
			Where("family_id = ? AND revoked_at IS NULL", family).
			Update("revoked_at", at).Error
		if err != nil {
			return err
		}

		return revokeTokensIn(tx, at, model.RevokedFamily(family))
	})
}

// RevokeRefreshTokensForCode revokes what a code's first exchange issued,
// its access tokens included, once the code is presented again.
func (s *Store) RevokeRefreshTokensForCode(ctx context.Context, code uuid.UUID, at time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&model.RefreshToken{}).
			Where("code_id = ? AND revoked_at IS NULL", code).
			Update("revoked_at", at).Error
		if err != nil {
			return err
		}

		return familiesRevokedIn(tx, at, "code_id = ?", code)
	})
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

// RevokeUserGrant takes back what a user allowed an application: its
// refresh tokens, and every access token it holds for them.
func (s *Store) RevokeUserGrant(ctx context.Context, user, application uuid.UUID, at time.Time) (int64, error) {
	var revoked int64

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.RefreshToken{}).
			Where("user_id = ? AND application_id = ? AND revoked_at IS NULL", user, application).
			Update("revoked_at", at)
		if result.Error != nil {
			return result.Error
		}
		revoked = result.RowsAffected

		var clientID string
		if err := tx.Model(&model.Application{}).Where("id = ?", application).Pluck("client_id", &clientID).Error; err != nil {
			return err
		}

		return revokeTokensIn(tx, at, model.RevokedGrant(user, clientID))
	})

	return revoked, err
}
